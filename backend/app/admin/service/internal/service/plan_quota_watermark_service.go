package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/tx7do/go-crud/viewer"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"

	internalMessageV1 "go-wind-admin/api/gen/go/internal_message/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/user"
	"go-wind-admin/pkg/mailtext"
	"go-wind-admin/pkg/task"
)

// PlanQuotaWatermarkService 套餐配额水位通知：定时扫描全部 ON 租户四类配额
// （USER_LIMIT / STORAGE / API_CALL / AI_TOKENS 月度）的用量水位（数据层
// ScanQuotaWatermarks，与计量同源），对达到阈值的租户按其管理员偏好语言渲染
// 站内信告警并投递给租户管理员（sys_tenants.admin_user_id）。
//
// 与硬执行（quota_enforcer / quota_gate，超限拒绝）互补：这里是到达上限之前的
// 提前告警。扫描频率即告警频率上限（每日一条，无状态：不记录"已通知"）。
type PlanQuotaWatermarkService struct {
	log *bLogger.Helper

	tenantUsageRepo  *data.TenantUsageRepo
	aiUsageLogRepo   *data.AiUsageLogRepo
	internalMessages *InternalMessageService
	internalMsgRepo  *data.InternalMessageRepo
	entClient        *entCrud.EntClient[*ent.Client]
}

func NewPlanQuotaWatermarkService(
	ctx *bootstrap.Context,
	tenantUsageRepo *data.TenantUsageRepo,
	aiUsageLogRepo *data.AiUsageLogRepo,
	internalMessages *InternalMessageService,
	internalMsgRepo *data.InternalMessageRepo,
	entClient *entCrud.EntClient[*ent.Client],
) *PlanQuotaWatermarkService {
	return &PlanQuotaWatermarkService{
		log:              ctx.NewLoggerHelper("plan-quota-watermark/service/admin-service"),
		tenantUsageRepo:  tenantUsageRepo,
		aiUsageLogRepo:   aiUsageLogRepo,
		internalMessages: internalMessages,
		internalMsgRepo:  internalMsgRepo,
		entClient:        entClient,
	}
}

// AsyncPlanQuotaWatermarkScan 水位扫描任务 handler（task_system.md §5.7）：
// 扫描 → 命中租户按租户归组（一租户一条消息，多维度命中合并为同一张清单）→
// 按租户管理员偏好语言渲染 → 站内信投递。
func (s *PlanQuotaWatermarkService) AsyncPlanQuotaWatermarkScan(taskType string, payload *task.PlanQuotaWatermarkTaskData) error {
	// asynq ctx 不带 viewer；跨租户扫描与站内信落库都需要 viewer，用系统查看器。
	ctx := viewer.WithSystemContext(context.Background())

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)

	hits, err := s.tenantUsageRepo.ScanQuotaWatermarks(ctx, s.aiUsageLogRepo, monthStart)
	if err != nil {
		s.log.Errorf(ctx, "quota watermark: scan failed: %v", err)
		return err
	}
	if len(hits) == 0 {
		s.log.Infof(ctx, "quota watermark: no tenant over watermark, nothing to deliver")
		return nil
	}

	grouped := make(map[int64][]data.QuotaWatermarkHit)
	adminOf := make(map[int64]int64)
	nameOf := make(map[int64]string)
	for _, h := range hits {
		grouped[h.TenantID] = append(grouped[h.TenantID], h)
		adminOf[h.TenantID] = h.AdminUserID
		nameOf[h.TenantID] = h.TenantName
	}

	// 投递失败语义（与审计日报 deliverToPlatformUsers 同型）：单租户失败记日志后
	// 继续其余租户、不向上报错重试——asynq 重试会整任务重跑，重复投递已成功的
	// 租户；全部尝试都失败才报错（此时无任何收件行落库，重试无重复副作用）。
	attempted, failed := 0, 0
	for tenantID, tenantHits := range grouped {
		adminID := adminOf[tenantID]
		if adminID == 0 {
			// 无可投递的管理员：留运维日志（平台应当知道的事），不计入投递尝试。
			s.log.Warnf(ctx, "quota watermark: tenant %d [%s] over watermark in %d dimension(s) but has no admin user, no notification sent",
				tenantID, nameOf[tenantID], len(tenantHits))
			continue
		}
		attempted++
		admin, uerr := s.entClient.Client().User.Query().
			Where(user.IDEQ(adminID), user.DeletedAtIsNil()).
			Only(ctx)
		if uerr != nil || admin == nil {
			s.log.Errorf(ctx, "quota watermark: query admin user %d of tenant %d failed: %v", adminID, tenantID, uerr)
			failed++
			continue
		}
		locale := mailtext.LocaleZhCN
		if admin.Locale != nil {
			if l, ok := mailtext.LocaleOfTag(*admin.Locale); ok {
				locale = l
			}
		}
		title := watermarkTitle(locale, nameOf[tenantID])
		content := watermarkBody(locale, nameOf[tenantID], tenantHits)
		if derr := s.deliverOne(ctx, adminID, title, content); derr != nil {
			s.log.Errorf(ctx, "quota watermark: deliver to admin %d of tenant %d failed: %v", adminID, tenantID, derr)
			failed++
		}
	}
	if attempted > 0 && failed == attempted {
		return fmt.Errorf("quota watermark: all %d tenant notifications failed", failed)
	}
	s.log.Infof(ctx, "quota watermark: %d/%d tenants over watermark, %d delivered", len(grouped), len(grouped), attempted-failed)
	return nil
}

// deliverOne 建一条站内信行并向单个收件人投递收件行（审计日报 deliverMessage 的
// 单收件人形态）。静音时段由 sendNotification 内核处理（只抑制实时推送，收件行
// 照落）。
func (s *PlanQuotaWatermarkService) deliverOne(ctx context.Context, recipientUserID int64, title, content string) error {
	now := time.Now()
	msg, err := s.internalMsgRepo.Create(ctx, &internalMessageV1.CreateInternalMessageRequest{
		Data: &internalMessageV1.InternalMessage{
			Title:     &title,
			Content:   trans.Ptr(content),
			Status:    trans.Ptr(internalMessageV1.InternalMessage_PUBLISHED),
			Type:      trans.Ptr(internalMessageV1.InternalMessage_NOTIFICATION),
			CreatedBy: trans.Ptr(int64(0)),
			CreatedAt: timeutil.TimeToTimestamppb(&now),
		},
	})
	if err != nil {
		return err
	}
	return s.internalMessages.sendNotification(ctx, msg.GetId(), recipientUserID, 0, &now, title, content)
}

// ---- 水位告警文案（站内信，按租户管理员偏好语言渲染） ----
//
// 与 ai_digest_service.go 的日报文案同一条链：站内信不是事务性邮件，文案不进
// pkg/mailtext；语言标签归一复用 mailtext.Locale/LocaleOfTag，文案表留在本
// 服务内、是其唯一出口，未设置或未识别回落中文。配额类型名不做翻译
// （USER_LIMIT 等技术名与用量页 / 闸门报错口径一致）。
type watermarkCopy struct {
	title      string // %s=租户名
	tenantLine string // %s=租户名
	quotaHead  string // %d=水位百分比
	quotaLine  string // %s=类型名 %d=已用 %d=上限 %d=百分比
	footer     string
}

var watermarkTables = map[mailtext.Locale]watermarkCopy{
	mailtext.LocaleZhCN: {
		title:      "套餐配额水位告警（%s）",
		tenantLine: "租户：%s\n",
		quotaHead:  "以下配额的用量已达到 %d%% 水位：\n",
		quotaLine:  "  - %s：已用 %d / 上限 %d（%d%%）\n",
		footer:     "请尽快检查用量并与平台管理员联系调整套餐配额；用量到达上限后，相关操作将被拒绝。\n",
	},
	mailtext.LocaleEnUS: {
		title:      "Plan Quota Watermark Alert (%s)",
		tenantLine: "Tenant: %s\n",
		quotaHead:  "The usage of the following quotas has reached the %d%% watermark:\n",
		quotaLine:  "  - %s: used %d / limit %d (%d%%)\n",
		footer:     "Please review your usage soon and contact the platform administrator to adjust the plan quotas; once a limit is reached, the related operations will be rejected.\n",
	},
}

func watermarkCopyOf(locale mailtext.Locale) watermarkCopy {
	if cp, ok := watermarkTables[locale]; ok {
		return cp
	}
	return watermarkTables[mailtext.LocaleZhCN]
}

func watermarkTitle(locale mailtext.Locale, tenantName string) string {
	return fmt.Sprintf(watermarkCopyOf(locale).title, tenantName)
}

func watermarkBody(locale mailtext.Locale, tenantName string, hits []data.QuotaWatermarkHit) string {
	cp := watermarkCopyOf(locale)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(cp.tenantLine, tenantName))
	sb.WriteString(fmt.Sprintf(cp.quotaHead, int(math.Round(data.QuotaWatermarkThreshold*100))))
	for _, h := range hits {
		sb.WriteString(fmt.Sprintf(cp.quotaLine, h.QuotaType, h.Used, h.Limit, h.RatioPct))
	}
	sb.WriteString(cp.footer)
	return sb.String()
}
