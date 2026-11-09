package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"github.com/tx7do/go-utils/trans"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	monitorAlertV1 "go-wind-admin/api/gen/go/monitor_alert/service/v1"
	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
	redisCacheV1 "go-wind-admin/api/gen/go/redis_cache/service/v1"
	serverMonitorV1 "go-wind-admin/api/gen/go/server_monitor/service/v1"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/user"
	"go-wind-admin/pkg/mailtext"
	"go-wind-admin/pkg/middleware/auth"
	"go-wind-admin/pkg/task"

	"github.com/tx7do/go-crud/viewer"
)

// MonitorAlertService 监控告警：规则 CRUD + 周期评估 + 告警通知分发。
//
// 评估内核 evaluateOnce 被两个入口共享：周期扫描任务（每 5 分钟）与管理页的
// 「立即评估」。「告警→恢复」成对通知：越限发告警、回到阈值内发恢复，
// 持续越限按 cooldown_minutes 重发（不是只发一次然后装死，也不是每次扫描都轰炸）。
type MonitorAlertService struct {
	adminV1.MonitorAlertServiceHTTPServer

	repo       *data.MonitorAlertRuleRepo
	serverRepo *data.ServerMonitorRepo
	redisRepo  *data.RedisCacheMonitorRepo
	notifier   Notifier
	entClient  *entCrud.EntClient[*ent.Client]
	log        *bLogger.Helper

	// 采集函数默认绑 repo；测试桩直接覆写这两个字段注入假样本
	collectServerFn func(ctx context.Context) (*serverMonitorV1.ServerMonitorInfo, error)
	collectRedisFn  func(ctx context.Context) (*redisCacheV1.RedisCacheMonitorInfo, error)
}

func NewMonitorAlertService(
	ctx *bootstrap.Context,
	repo *data.MonitorAlertRuleRepo,
	serverRepo *data.ServerMonitorRepo,
	redisRepo *data.RedisCacheMonitorRepo,
	entClient *entCrud.EntClient[*ent.Client],
) *MonitorAlertService {
	svc := &MonitorAlertService{
		repo:       repo,
		serverRepo: serverRepo,
		redisRepo:  redisRepo,
		entClient:  entClient,
		log:        ctx.NewLoggerHelper("monitor-alert/service/admin-service"),
		// notifier 默认"未装配"占位：装配期由 RegisterNotifier 换成 NotificationService
		// （与 InternalMessageService 同模式）。
		notifier: unwiredNotifier{},
	}
	svc.collectServerFn = svc.collectServer
	svc.collectRedisFn = svc.collectRedis
	return svc
}

// RegisterNotifier 装配通知出口（NotificationService）。
func (s *MonitorAlertService) RegisterNotifier(notifier Notifier) {
	s.notifier = notifier
}

// ==== CRUD（平台管理员） ====

func (s *MonitorAlertService) ListMonitorAlertRule(ctx context.Context, req *paginationV1.PagingRequest) (*monitorAlertV1.ListMonitorAlertRuleResponse, error) {
	if err := requirePlatformAdmin(ctx, s.log, "monitor-alert/list"); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, req)
}

func (s *MonitorAlertService) GetMonitorAlertRule(ctx context.Context, req *monitorAlertV1.GetMonitorAlertRuleRequest) (*monitorAlertV1.MonitorAlertRule, error) {
	if err := requirePlatformAdmin(ctx, s.log, "monitor-alert/get"); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, req.GetId())
}

func (s *MonitorAlertService) CreateMonitorAlertRule(ctx context.Context, req *monitorAlertV1.CreateMonitorAlertRuleRequest) (*monitorAlertV1.MonitorAlertRule, error) {
	if err := requirePlatformAdmin(ctx, s.log, "monitor-alert/create"); err != nil {
		return nil, err
	}
	if err := validateAlertRule(req.GetData()); err != nil {
		return nil, err
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := s.repo.Create(ctx, req.GetData(), operator.GetUserId())
	if err != nil {
		return nil, err
	}

	return s.repo.Get(ctx, id)
}

func (s *MonitorAlertService) UpdateMonitorAlertRule(ctx context.Context, req *monitorAlertV1.UpdateMonitorAlertRuleRequest) (*emptypb.Empty, error) {
	if err := requirePlatformAdmin(ctx, s.log, "monitor-alert/update"); err != nil {
		return nil, err
	}
	if err := validateAlertRule(req.GetData()); err != nil {
		return nil, err
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, req, operator.GetUserId()); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *MonitorAlertService) DeleteMonitorAlertRule(ctx context.Context, req *monitorAlertV1.DeleteMonitorAlertRuleRequest) (*emptypb.Empty, error) {
	if err := requirePlatformAdmin(ctx, s.log, "monitor-alert/delete"); err != nil {
		return nil, err
	}
	if err := s.repo.Delete(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// EvaluateMonitorAlerts 手动触发一轮评估（与周期扫描同一内核）。
func (s *MonitorAlertService) EvaluateMonitorAlerts(ctx context.Context, _ *monitorAlertV1.EvaluateMonitorAlertsRequest) (*monitorAlertV1.EvaluateMonitorAlertsResponse, error) {
	if err := requirePlatformAdmin(ctx, s.log, "monitor-alert/evaluate"); err != nil {
		return nil, err
	}

	outcomes := s.evaluateOnce(ctx)

	resp := &monitorAlertV1.EvaluateMonitorAlertsResponse{Outcomes: make([]*monitorAlertV1.EvaluateMonitorAlertsResponse_RuleOutcome, 0, len(outcomes))}
	for _, o := range outcomes {
		resp.Outcomes = append(resp.Outcomes, &monitorAlertV1.EvaluateMonitorAlertsResponse_RuleOutcome{
			RuleId:       o.ruleID,
			Name:         o.name,
			Firing:       o.firing,
			CurrentValue: o.currentValue,
			Notified:     o.notified,
			Reason:       trans.Ptr(o.reason),
		})
	}
	return resp, nil
}

// ==== 评估内核 ====

type alertOutcome struct {
	ruleID       uint64
	name         string
	firing       bool
	currentValue *float64
	notified     bool
	reason       string
}

// validateAlertRule 建/改规则的公共校验：非布尔指标必须有 op；渠道/目标合法。
func validateAlertRule(data *monitorAlertV1.MonitorAlertRule) error {
	if data == nil {
		return adminV1.ErrorBadRequest("invalid parameter")
	}
	if data.GetMetric() == monitorAlertV1.MonitorMetric_MONITOR_METRIC_UNSPECIFIED {
		return adminV1.ErrorBadRequest("metric is required")
	}
	if data.GetChannel() != notificationV1.Channel_EMAIL && data.GetChannel() != notificationV1.Channel_WEBHOOK {
		return adminV1.ErrorBadRequest("channel must be EMAIL or WEBHOOK")
	}
	if data.GetTarget() == "" {
		return adminV1.ErrorBadRequest("target is required")
	}
	// DB_PING_FAIL 是布尔指标（越限即 ping 失败），op/threshold 无意义可缺省；
	// 其余指标必须有比较运算与阈值。
	if data.GetMetric() != monitorAlertV1.MonitorMetric_DB_PING_FAIL {
		if data.GetOp() == monitorAlertV1.AlertOp_ALERT_OP_UNSPECIFIED {
			return adminV1.ErrorBadRequest("op is required for metric %s", data.GetMetric().String())
		}
		if data.GetThreshold() == 0 {
			return adminV1.ErrorBadRequest("threshold is required for metric %s", data.GetMetric().String())
		}
	}
	return nil
}

// collectServer 读服务器监控样本（一轮只读一次，全部 GO_/DB_ 指标共享）。
func (s *MonitorAlertService) collectServer(ctx context.Context) (*serverMonitorV1.ServerMonitorInfo, error) {
	return s.serverRepo.GetInfo(ctx)
}

// collectRedis 读 Redis 监控样本。
func (s *MonitorAlertService) collectRedis(ctx context.Context) (*redisCacheV1.RedisCacheMonitorInfo, error) {
	return s.redisRepo.GetInfo(ctx)
}

// metricValue 从样本里取规则的当前指标值。
// serverSample/redisSample 为 nil 表示对应来源本轮采集失败（调用方保证不是该来源的规则不会走到 nil 分支）。
func metricValue(rule *monitorAlertV1.MonitorAlertRule, serverSample *serverMonitorV1.ServerMonitorInfo, redisSample *redisCacheV1.RedisCacheMonitorInfo) (float64, error) {
	switch rule.GetMetric() {
	case monitorAlertV1.MonitorMetric_GO_GOROUTINES:
		return float64(serverSample.GetGo().GetNumGoroutine()), nil
	case monitorAlertV1.MonitorMetric_GO_MEM_ALLOC_MB:
		return float64(serverSample.GetGo().GetMemAllocBytes()) / 1024 / 1024, nil
	case monitorAlertV1.MonitorMetric_DB_OPEN_CONNECTIONS:
		return float64(serverSample.GetDatabase().GetOpenConnections()), nil
	case monitorAlertV1.MonitorMetric_DB_PING_FAIL:
		if serverSample.GetDatabase().GetPingOk() {
			return 0, nil
		}
		return 1, nil
	case monitorAlertV1.MonitorMetric_REDIS_DB_SIZE:
		return float64(redisSample.GetDbSize()), nil
	default:
		return 0, fmt.Errorf("unsupported metric %s", rule.GetMetric().String())
	}
}

// isBreached 阈值判定。DB_PING_FAIL 是布尔指标、绕过比较：ping 失败（值 1）即越限，
// ping 正常（值 0）永不越限——不绕过的话，未填 op/threshold 的规则会以零值兜底
// 走 GE 比较（0 >= 0），把"一切正常"判成"持续越限"。
func isBreached(rule *monitorAlertV1.MonitorAlertRule, value float64) bool {
	if rule.GetMetric() == monitorAlertV1.MonitorMetric_DB_PING_FAIL {
		return value >= 1
	}
	switch rule.GetOp() {
	case monitorAlertV1.AlertOp_LE:
		return value <= rule.GetThreshold()
	case monitorAlertV1.AlertOp_GE:
		fallthrough
	default:
		return value >= rule.GetThreshold()
	}
}

type sampleSet struct {
	server  *serverMonitorV1.ServerMonitorInfo
	redis   *redisCacheV1.RedisCacheMonitorInfo
	serverN bool // server 样本已成功采集
	redisOK bool
}

// evaluateOnce 评估一轮：逐条启用规则求值 → 决定通知 → 回写状态。
//
// 通知时机（cooldown 以 last_alerted_at 为锚）：
//   - 未越限且上次越限 → 「恢复」通知（不等冷却：恢复通知晚到没有意义）；
//   - 越限且（上次未越限 或 冷却已过）→ 「告警」通知；
//   - 越限但冷却未到 → 只回写 last_value，不重发。
//
// 采集失败轮：跳过该来源的规则且**不清 firing 状态**——采集抖动不是恢复，
// 把它当恢复会发假通知，把它当持续越限重发会发重复告警，两个都比"跳过一轮"糟。
func (s *MonitorAlertService) evaluateOnce(ctx context.Context) []alertOutcome {
	rules, err := s.repo.ListEnabled(ctx)
	if err != nil {
		s.log.Errorf(ctx, "monitor alert scan: list rules failed: %s", err.Error())
		return nil
	}
	if len(rules) == 0 {
		return nil
	}

	needServer, needRedis := false, false
	for _, rule := range rules {
		switch rule.GetMetric() {
		case monitorAlertV1.MonitorMetric_GO_GOROUTINES,
			monitorAlertV1.MonitorMetric_GO_MEM_ALLOC_MB,
			monitorAlertV1.MonitorMetric_DB_OPEN_CONNECTIONS,
			monitorAlertV1.MonitorMetric_DB_PING_FAIL:
			needServer = true
		case monitorAlertV1.MonitorMetric_REDIS_DB_SIZE:
			needRedis = true
		}
	}

	samples := sampleSet{}
	if needServer {
		if info, err := s.collectServerFn(ctx); err != nil {
			s.log.Errorf(ctx, "monitor alert scan: collect server sample failed: %s", err.Error())
		} else {
			samples.server = info
			samples.serverN = true
		}
	}
	if needRedis {
		if info, err := s.collectRedisFn(ctx); err != nil {
			s.log.Errorf(ctx, "monitor alert scan: collect redis sample failed: %s", err.Error())
		} else {
			samples.redis = info
			samples.redisOK = true
		}
	}

	outcomes := make([]alertOutcome, 0, len(rules))
	now := time.Now()
	for _, rule := range rules {
		outcome := alertOutcome{ruleID: rule.GetId(), name: rule.GetName()}

		// 来源采集失败：跳过且不清 firing（见函数头注释）
		serverSourced := rule.GetMetric() != monitorAlertV1.MonitorMetric_REDIS_DB_SIZE
		if (serverSourced && !samples.serverN) || (!serverSourced && !samples.redisOK) {
			outcome.firing = rule.GetLastFiring()
			outcome.reason = "sample collection failed, skipped"
			outcomes = append(outcomes, outcome)
			continue
		}

		value, err := metricValue(rule, samples.server, samples.redis)
		if err != nil {
			outcome.reason = err.Error()
			outcomes = append(outcomes, outcome)
			continue
		}
		outcome.currentValue = &value
		outcome.firing = isBreached(rule, value)

		switch {
		case outcome.firing && !rule.GetLastFiring():
			outcome.notified = s.sendAlert(ctx, rule, value, false, now)
			if !outcome.notified {
				outcome.reason = "alert dispatch failed"
			}
		case outcome.firing && rule.GetLastFiring() && cooldownElapsed(rule, now):
			outcome.notified = s.sendAlert(ctx, rule, value, false, now)
			outcome.reason = "cooldown elapsed, re-alerted"
		case !outcome.firing && rule.GetLastFiring():
			outcome.notified = s.sendAlert(ctx, rule, value, true, now)
			if !outcome.notified {
				outcome.reason = "resolve dispatch failed"
			}
		default:
			outcome.reason = "no state change"
		}

		// 状态回写：firing 如实反映本次求值（否则"恢复"判定会漂）；
		// 冷却锚 last_alerted_at 只在通知真正发出时推进——发失败下一轮会立即重试，
		// 而不是被假锚按进冷却窗口里漏发。
		alertedAt := &now
		if !outcome.notified {
			alertedAt = nil
		}
		if err := s.repo.MarkFiring(ctx, rule.GetId(), outcome.firing, outcome.currentValue, alertedAt); err != nil {
			s.log.Errorf(ctx, "monitor alert scan: mark rule [%d] failed: %s", rule.GetId(), err.Error())
		}
		outcomes = append(outcomes, outcome)
	}

	return outcomes
}

func cooldownElapsed(rule *monitorAlertV1.MonitorAlertRule, now time.Time) bool {
	last := rule.GetLastAlertedAt()
	if last == nil {
		return true
	}
	return now.Sub(last.AsTime()) >= time.Duration(rule.GetCooldownMinutes())*time.Minute
}

// sendAlert 发告警/恢复通知，返回是否真的发出去了（冷却锚只按真发推进）。
// 失败不中断评估循环——一次 SMTP/Webhook 失败不该拦住其余规则，
// 本轮的缺口由下一轮扫描补发（firing 状态没变，下次仍会命中发送分支）。
func (s *MonitorAlertService) sendAlert(ctx context.Context, rule *monitorAlertV1.MonitorAlertRule, value float64, resolved bool, now time.Time) bool {
	locale := s.alertLocale(ctx, rule)
	title, content := alertTextFor(locale, rule, value, resolved, now)
	resp, err := s.notifier.SendDirect(ctx, &notificationV1.SendDirectNotificationRequest{
		EventType:      notificationV1.EventType_MONITOR_ALERT,
		Channel:        rule.Channel,
		Target:         rule.GetTarget(),
		Title:          title,
		Content:        content,
		RelatedId:      rule.Id,
		OperatorUserId: trans.Ptr(uint64(0)),
	})
	if err != nil {
		s.log.Errorf(ctx, "monitor alert [%s] notify (%s → %s) failed (delivery=%d): %s",
			rule.GetName(), rule.GetChannel().String(), rule.GetTarget(), resp.GetDeliveryId(), err.Error())
		return false
	}
	s.log.Infof(ctx, "monitor alert [%s] %s: value=%.2f (delivery=%d)",
		rule.GetName(), map[bool]string{true: "RESOLVED", false: "FIRING"}[resolved], value, resp.GetDeliveryId())
	return true
}

// ---- 告警文案（按收件人偏好语言渲染，2026-10-04 接 lang） ----
//
// 与 ai_digest_service.go 的日报文案同一条链：语言标签归一复用
// mailtext.Locale/LocaleOfTag，文案表留在本服务内、是其唯一出口，
// 未设置/未识别回落中文。指标与比较符用枚举字面量（DB_PING_FAIL/GE），
// 与闸门报错、用量页的口径一致，两语言不做翻译。
type alertCopy struct {
	resolvedTitle string // %s=规则名
	resolvedBody  string // %s=指标 %.2f=当前值 %s=比较符 %.2f=阈值 %s=时间
	alertTitle    string // %s=规则名
	alertBody     string // %s=指标 %.2f=当前值 %s=比较符 %.2f=阈值 %s=时间 %d=冷却分钟
}

var alertTables = map[mailtext.Locale]alertCopy{
	mailtext.LocaleZhCN: {
		resolvedTitle: "[已恢复] %s",
		resolvedBody:  "监控指标 %s 已回到阈值内（当前值 %.2f，阈值 %s %.2f），时间 %s。",
		alertTitle:    "[告警] %s",
		alertBody:     "监控指标 %s 当前值 %.2f，越过阈值 %s %.2f，时间 %s。持续越限时每 %d 分钟重发一次。",
	},
	mailtext.LocaleEnUS: {
		resolvedTitle: "[Resolved] %s",
		resolvedBody:  "Metric %s is back within threshold (current value %.2f, threshold %s %.2f), at %s.",
		alertTitle:    "[Alert] %s",
		alertBody:     "Metric %s current value %.2f crossed the threshold %s %.2f at %s. Re-alerts every %d minutes while it keeps crossing.",
	},
}

func alertCopyOf(locale mailtext.Locale) alertCopy {
	if cp, ok := alertTables[locale]; ok {
		return cp
	}
	return alertTables[mailtext.LocaleZhCN]
}

// alertTextFor 按语言的告警/恢复文案（原固定中文 alertText 的多语言形态）。
func alertTextFor(locale mailtext.Locale, rule *monitorAlertV1.MonitorAlertRule, value float64, resolved bool, now time.Time) (string, string) {
	cp := alertCopyOf(locale)
	ts := now.Format("2006-01-02 15:04:05")
	if resolved {
		return fmt.Sprintf(cp.resolvedTitle, rule.GetName()),
			fmt.Sprintf(cp.resolvedBody, rule.GetMetric().String(), value, rule.GetOp().String(), rule.GetThreshold(), ts)
	}
	return fmt.Sprintf(cp.alertTitle, rule.GetName()),
		fmt.Sprintf(cp.alertBody, rule.GetMetric().String(), value, rule.GetOp().String(), rule.GetThreshold(), ts, rule.GetCooldownMinutes())
}

// alertLocale 解析告警收件人的偏好语言，解析不到一律回落默认（中文）：
//   - INTERNAL：target 为收件用户 ID 的十进制串，读该用户的 locale；
//   - EMAIL：target 按用户邮箱反查（收件邮箱是注册用户时能对上偏好）；
//   - WEBHOOK：对端没有用户身份，无偏好可循。
//
// 与日报的收件人分组同取向：语言解析是锦上添花，任何失败都回落默认、
// 绝不影响告警送达。
func (s *MonitorAlertService) alertLocale(ctx context.Context, rule *monitorAlertV1.MonitorAlertRule) mailtext.Locale {
	const fallback = mailtext.LocaleZhCN
	if s.entClient == nil {
		return fallback // 旧测试装配未注入 ent
	}

	var q *ent.UserQuery
	switch rule.GetChannel() {
	case notificationV1.Channel_INTERNAL:
		id, perr := strconv.ParseUint(strings.TrimSpace(rule.GetTarget()), 10, 32)
		if perr != nil || id == 0 {
			return fallback // target 不是用户 ID 形态，无偏好可循
		}
		q = s.entClient.Client().User.Query().Where(user.IDEQ(uint64(id)))
	case notificationV1.Channel_EMAIL:
		target := strings.TrimSpace(rule.GetTarget())
		if target == "" {
			return fallback
		}
		q = s.entClient.Client().User.Query().Where(user.EmailEQ(target))
	default:
		return fallback
	}

	u, err := q.Where(user.DeletedAtIsNil()).
		Only(viewer.WithSystemContext(ctx))
	if err != nil && !ent.IsNotFound(err) {
		// 真查询故障才记日志（带原始错误）；查无此人是预期回落，不刷屏
		s.log.Errorf(ctx, "monitor alert locale: query recipient user failed (channel=%s target=%s): %v",
			rule.GetChannel().String(), rule.GetTarget(), err)
	}
	if u == nil || u.Locale == nil {
		return fallback
	}
	if l, ok := mailtext.LocaleOfTag(*u.Locale); ok {
		return l
	}
	return fallback
}

// ==== 周期任务 ====

// AsyncMonitorAlertScan 周期扫描 handler（系统级常驻任务，SystemContext 上下文）。
func (s *MonitorAlertService) AsyncMonitorAlertScan(taskType string, _ *task.MonitorAlertScanTaskData) error {
	ctx := viewer.WithSystemContext(context.Background())
	s.log.Infof(ctx, "[%s] scan start", taskType)

	outcomes := s.evaluateOnce(ctx)

	var notified int
	for _, o := range outcomes {
		if o.notified {
			notified++
		}
	}
	s.log.Infof(ctx, "[%s] scan done: %d rules, %d notified", taskType, len(outcomes), notified)
	return nil
}
