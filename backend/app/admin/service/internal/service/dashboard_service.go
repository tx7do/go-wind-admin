package service

import (
	"fmt"
	"go-wind-admin/pkg/middleware/auth"
	"strings"
	"unicode/utf8"

	"context"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
)

// DashboardService 为后台首页分析页提供只读聚合统计。
// 多租户隔离由 ent Policy + auth 中间件注入的 viewer 自动完成。
type DashboardService struct {
	adminV1.DashboardServiceHTTPServer

	log *bLogger.Helper

	dashboardRepo *data.DashboardRepo
	scriptRuntime *ScriptRuntime
}

func NewDashboardService(
	ctx *bootstrap.Context,
	dashboardRepo *data.DashboardRepo,
	scriptRuntime *ScriptRuntime,
) *DashboardService {
	return &DashboardService{
		log:           ctx.NewLoggerHelper("dashboard/service/admin-service"),
		scriptRuntime: scriptRuntime,
		dashboardRepo: dashboardRepo,
	}
}

// GetOverview 返回四张概览卡的计数。
func (s *DashboardService) GetOverview(ctx context.Context, _ *emptypb.Empty) (*adminV1.DashboardOverviewResponse, error) {
	userCount, err := s.dashboardRepo.CountActiveUsers(ctx)
	if err != nil {
		return nil, err
	}
	roleCount, err := s.dashboardRepo.CountRoles(ctx)
	if err != nil {
		return nil, err
	}
	todayLoginCount, err := s.dashboardRepo.CountTodayLogins(ctx)
	if err != nil {
		return nil, err
	}
	todayOperationCount, err := s.dashboardRepo.CountTodayOperations(ctx)
	if err != nil {
		return nil, err
	}

	return &adminV1.DashboardOverviewResponse{
		UserCount:           uint32(userCount),
		RoleCount:           uint32(roleCount),
		TodayLoginCount:     uint32(todayLoginCount),
		TodayOperationCount: uint32(todayOperationCount),
	}, nil
}

// GetLoginTrend 返回近 days 天每日登录次数趋势，按日期升序、缺日补零。
func (s *DashboardService) GetLoginTrend(ctx context.Context, req *adminV1.GetLoginTrendRequest) (*adminV1.LoginTrendResponse, error) {
	days := int(req.GetDays())
	rows, err := s.dashboardRepo.LoginTrend(ctx, days)
	if err != nil {
		return nil, err
	}

	points := make([]*adminV1.TrendPoint, 0, len(rows))
	for _, r := range rows {
		points = append(points, &adminV1.TrendPoint{
			Date:  r.Date,
			Count: uint32(r.Count),
		})
	}
	return &adminV1.LoginTrendResponse{Points: points}, nil
}

// GetOperationActionDistribution 返回操作审计按 action 的分布。
func (s *DashboardService) GetOperationActionDistribution(ctx context.Context, _ *emptypb.Empty) (*adminV1.ActionDistributionResponse, error) {
	rows, err := s.dashboardRepo.OperationActionDistribution(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]*adminV1.DistributionItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, &adminV1.DistributionItem{
			Label: r.Action,
			Count: uint32(r.Count),
		})
	}
	return &adminV1.ActionDistributionResponse{Items: items}, nil
}

// GetLoginStatusDistribution 返回登录审计按 status 的分布。
func (s *DashboardService) GetLoginStatusDistribution(ctx context.Context, _ *emptypb.Empty) (*adminV1.StatusDistributionResponse, error) {
	rows, err := s.dashboardRepo.LoginStatusDistribution(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]*adminV1.DistributionItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, &adminV1.DistributionItem{
			Label: r.Status,
			Count: uint32(r.Count),
		})
	}
	return &adminV1.StatusDistributionResponse{Items: items}, nil
}

// 异常信号阈值（24h 窗口）。
const (
	nightOpsAlertMin     = 1 // 深夜（本地 0-6 点）操作条数下限
	userFailedAlertMin   = 3 // 单用户失败操作条数下限
	loginFailAlertMin    = 3 // 单账号登录失败次数下限（疑似口令尝试）
	sensitiveOpsAlertMin = 1 // 敏感操作（DELETE/EXPORT/ASSIGN）条数下限
	sensitiveDetailMax   = 5 // 响应里最多携带的敏感操作明细条数
)

// 告警类型常量（前端 i18n 模板的 type 键，告警本身零文案）。
const (
	aiAlertBruteForce   = "BRUTE_FORCE"
	aiAlertNightOps     = "NIGHT_OPS"
	aiAlertFailedOps    = "FAILED_OPS"
	aiAlertSensitiveOps = "SENSITIVE_OPS"
)

// GetAiInsights 安全与异常洞察：规则预筛审计明细里的行为模式，
// 返回结构化告警事实（severity + type + facts），**不含任何人类文案**——
// 文案由前端 i18n 模板按界面语言渲染；LLM 总体评估按请求语言生成。
// 平台用户专属：明细日志为全平台数据，且会外发到模型端点。
func (s *DashboardService) GetAiInsights(ctx context.Context, req *adminV1.AiInsightsRequest) (*adminV1.AiInsightsResponse, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	if operator.GetTenantId() != 0 {
		return nil, adminV1.ErrorForbidden("ai insights is platform-only")
	}

	var alerts []*adminV1.AiInsightAlert
	addAlert := func(severity, alertType string, facts map[string]string) {
		alerts = append(alerts, &adminV1.AiInsightAlert{
			Severity: severity,
			Type:     alertType,
			Facts:    facts,
		})
	}

	// 1. 疑似口令尝试
	failAccounts, err := s.dashboardRepo.LoginFailAccounts(ctx, loginFailAlertMin)
	if err != nil {
		return nil, err
	}
	for _, acc := range failAccounts {
		addAlert("HIGH", aiAlertBruteForce, map[string]string{
			"username": acc.Username,
			"count":    fmt.Sprintf("%d", acc.Fails),
			"ip":       acc.LastIP,
		})
	}

	// 2. 用户行为模式
	behaviors, err := s.dashboardRepo.UserBehavior24h(ctx)
	if err != nil {
		return nil, err
	}
	for _, b := range behaviors {
		if b.Night >= nightOpsAlertMin {
			addAlert("MEDIUM", aiAlertNightOps, map[string]string{
				"username": b.Username,
				"count":    fmt.Sprintf("%d", b.Night),
			})
		}
		if b.Failed >= userFailedAlertMin {
			addAlert("MEDIUM", aiAlertFailedOps, map[string]string{
				"username": b.Username,
				"count":    fmt.Sprintf("%d", b.Failed),
				"total":    fmt.Sprintf("%d", b.Total),
			})
		}
	}

	// 3. 敏感操作明细
	sensitiveOps, err := s.dashboardRepo.SensitiveOps24h(ctx, sensitiveDetailMax)
	if err != nil {
		return nil, err
	}
	if len(sensitiveOps) >= sensitiveOpsAlertMin {
		facts := map[string]string{"count": fmt.Sprintf("%d", len(sensitiveOps))}
		alert := &adminV1.AiInsightAlert{
			Severity: "LOW",
			Type:     aiAlertSensitiveOps,
			Facts:    facts,
		}
		for _, op := range sensitiveOps {
			alert.Items = append(alert.Items, &adminV1.AiSensitiveOpItem{
				Username:     op.Username,
				Action:       op.Action,
				ResourceType: op.ResourceType,
				CreatedAt:    op.CreatedAt.Format("01-02 15:04"),
			})
		}
		alerts = append(alerts, alert)
	}

	// 4. 总体评估：有告警才调 LLM，并按请求语言输出；失败不阻断告警返回
	summary := ""
	if len(alerts) > 0 {
		lang := req.GetLang()
		if lang == "" {
			lang = "zh-CN"
		}
		var alertFacts strings.Builder
		for _, a := range alerts {
			alertFacts.WriteString("- [" + a.Severity + "] " + a.Type)
			for k, v := range a.Facts {
				alertFacts.WriteString(fmt.Sprintf(" %s=%s", k, v))
			}
			alertFacts.WriteString("\n")
		}
		langLine := "用中文输出。"
		if strings.HasPrefix(lang, "en") {
			langLine = "Respond in English."
		}
		assessment, err := s.scriptRuntime.ChatForScript(ctx, 0,
			"你是企业后台的安全运营助手。以下是从审计日志规则筛出的异常信号清单，"+
				"写一段 100 字以内的总体风险评估：概括风险等级与最需要优先处置的一项，"+
				"语气克制、只基于给定信号，不要编造未提及的信息。"+langLine,
			alertFacts.String())
		if err != nil {
			// LLM 不可用不阻断告警：告警本身是确定性事实
			s.log.Errorf(ctx, "ai insights assessment failed, alerts still returned: %v", err)
		} else {
			summary = assessment
		}
	}

	return &adminV1.AiInsightsResponse{
		Alerts:  alerts,
		Summary: summary,
	}, nil
}

// splitInsightPoints 把模型输出的要点文本拆成结构化条目：
// 识别 "- "/"* "/数字序号前缀；无多行时按句号拆分。最多 5 条。
func splitInsightPoints(summary string) []string {
	// stripPointPrefix 剥离单条要点的列表前缀与数字序号。
	stripPointPrefix := func(line string) string {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")
		line = strings.TrimPrefix(line, "• ")
		// 去数字序号（"1. " / "1、"）。分隔符含多字节字符（"、"占 3 字节），
		// 必须按 rune 宽度切：此前按字节偏移 +1 切会把中文顿号拦腰斩断，
		// 产出非法 UTF-8（"1、要点" → "\x80\x81要点"）。
		if idx := strings.IndexAny(line, ".、"); idx > 0 && idx <= 2 {
			_, size := utf8.DecodeRuneInString(line[idx:])
			if rest := strings.TrimSpace(line[idx+size:]); rest != "" {
				line = rest
			}
		}
		return line
	}

	var points []string
	for _, line := range strings.Split(summary, "\n") {
		if line = stripPointPrefix(line); line != "" {
			points = append(points, line)
		}
	}
	// 无多行时按句号拆分。覆盖（而非追加）行解析结果：行解析只会把整段当一个
	// 要点，不重置会把同一内容以列表形态和句号形态重复出两份。
	if len(points) <= 1 && summary != "" {
		points = nil
		for _, s := range strings.Split(summary, "。") {
			if s = strings.TrimSpace(stripPointPrefix(s)); s != "" {
				points = append(points, s+"。")
			}
		}
	}
	if len(points) > 5 {
		points = points[:5]
	}
	return points
}
