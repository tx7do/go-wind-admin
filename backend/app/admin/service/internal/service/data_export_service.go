package service

import (
	"net/http"
	"strconv"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/pkg/middleware/auth"
)

// 非审计数据的服务端导出单次行数上限（与审计导出同额：防无界查询 OOM，
// 见 audit_export_service.go 的说明；更大规模走各自的归档通道）。
const dataExportMaxRows = 500000

// DataExportService 非审计数据的服务端导出（XLSX/CSV，手动路由 + 公共内核）。
//
// 权限语义与各自列表接口对齐：
//   - AI 用量流水：租户管理员可导出本租户（viewer 按操作人重建后由 mixin
//     谓词自动收窄，平台管理员导全量）——与 AiUsageLogService.List 同口径。
//   - 通知投递台账：平台管理员专属（表无 tenant_id，租户可见即全平台可见，
//     语义对齐 requirePlatformAdmin，见 notification_platform_guard.go）。
type DataExportService struct {
	aiUsageRepo  *data.AiUsageLogRepo
	deliveryRepo *data.NotificationDeliveryRepo
	log          *bLogger.Helper

	// tokenChecker 手动路由的鉴权锚：:export 不穿 kratos middleware，
	// auth 中间件的一套在此补齐（见 export_common.go 公共样板）。
	tokenChecker auth.AccessTokenChecker

	// tenantAccessChecker 手动路由的租户闸（P1 修复补齐）：:export 游离在
	// kratos 中间件链之外，此前套餐模块白名单/租户状态/配额四段语义全部失效。
	tenantAccessChecker auth.TenantAccessChecker
}

func NewDataExportService(
	ctx *bootstrap.Context,
	aiUsageRepo *data.AiUsageLogRepo,
	deliveryRepo *data.NotificationDeliveryRepo,
	tokenChecker auth.AccessTokenChecker,
	tenantAccessChecker auth.TenantAccessChecker,
) *DataExportService {
	return &DataExportService{
		aiUsageRepo:         aiUsageRepo,
		deliveryRepo:        deliveryRepo,
		log:                 ctx.NewLoggerHelper("data-export/service/admin-service"),
		tokenChecker:        tokenChecker,
		tenantAccessChecker: tenantAccessChecker,
	}
}

// ServeAiUsageExport GET /admin/v1/ai/usage-logs:export —— AI 用量流水导出。
// 列与用量页展示列一致（"导出的就是页面看到的"）；行范围由 viewer 收窄语义决定。
// 租户操作者须经租户闸（按同源列表端点 /admin/v1/ai/usage-logs 的 Api 行带出
// 模块白名单/状态/配额语义）——此前套餐不含 AI 模块的租户也能导出且不占配额。
func (s *DataExportService) ServeAiUsageExport(w http.ResponseWriter, r *http.Request) error {
	ctx, operator, ok := authorizeExportRequest(w, r, s.tokenChecker)
	if !ok {
		return nil
	}
	if operator.GetTenantId() > 0 {
		if !requireExportTenantModule(w, s.tenantAccessChecker, ctx, operator,
			"/admin/v1/ai/usage-logs", "GET") {
			return nil
		}
	}

	format, req, pok := parseExportRequest(w, r, dataExportMaxRows)
	if !pok {
		return nil
	}

	resp, listErr := s.aiUsageRepo.List(ctx, req)
	if listErr != nil {
		s.log.Errorf(ctx, "export [ai_usage] query failed: %s", listErr.Error())
		http.Error(w, "query failed", http.StatusInternalServerError)
		return nil
	}

	cols := aiUsageExportColumns()
	headers := colHeaders(cols)
	var rows [][]string
	for _, row := range resp.GetItems() {
		rows = append(rows, colValues(cols, row))
	}

	emitExportFile(w, s.log, ctx, "ai_usage", "ai-usage-logs", "ai_usage", headers, rows, format)
	return nil
}

// ServeNotificationDeliveryExport GET /admin/v1/notification-deliveries:export
// —— 通知投递台账导出（平台管理员专属）。
func (s *DataExportService) ServeNotificationDeliveryExport(w http.ResponseWriter, r *http.Request) error {
	ctx, operator, ok := authorizeExportRequest(w, r, s.tokenChecker)
	if !ok {
		return nil
	}
	if !requireExportPlatformAdmin(w, operator) {
		return nil
	}

	format, req, pok := parseExportRequest(w, r, dataExportMaxRows)
	if !pok {
		return nil
	}

	resp, listErr := s.deliveryRepo.List(ctx, req)
	if listErr != nil {
		s.log.Errorf(ctx, "export [notification_delivery] query failed: %s", listErr.Error())
		http.Error(w, "query failed", http.StatusInternalServerError)
		return nil
	}

	cols := notificationDeliveryExportColumns()
	headers := colHeaders(cols)
	var rows [][]string
	for _, row := range resp.GetItems() {
		rows = append(rows, colValues(cols, row))
	}

	emitExportFile(w, s.log, ctx, "notification_delivery", "notification-deliveries", "notification_delivery", headers, rows, format)
	return nil
}

// ==== 列定义（表头用英文字段名：导出文件没有运行时语言上下文）====

// aiUsageExportColumns 与用量页展示列一致（模型/token 计数/耗时/时间）。
// 页面未展示的关联 ID（provider/conversation/user/tenant）不进导出。
func aiUsageExportColumns() []exportColumn[*aiV1.AiUsageLog] {
	return []exportColumn[*aiV1.AiUsageLog]{
		{header: "createdAt", value: func(r *aiV1.AiUsageLog) string { return fmtTime(r.GetCreatedAt()) }},
		{header: "modelName", value: func(r *aiV1.AiUsageLog) string { return r.GetModelName() }},
		{header: "promptTokens", value: func(r *aiV1.AiUsageLog) string { return strconv.FormatInt(int64(r.GetPromptTokens()), 10) }},
		{header: "completionTokens", value: func(r *aiV1.AiUsageLog) string { return strconv.FormatInt(int64(r.GetCompletionTokens()), 10) }},
		{header: "totalTokens", value: func(r *aiV1.AiUsageLog) string { return strconv.FormatInt(int64(r.GetTotalTokens()), 10) }},
		{header: "durationMs", value: func(r *aiV1.AiUsageLog) string { return strconv.FormatInt(int64(r.GetDurationMs()), 10) }},
	}
}

// notificationDeliveryExportColumns 与投递台账页展示列一致。
func notificationDeliveryExportColumns() []exportColumn[*notificationV1.NotificationDelivery] {
	return []exportColumn[*notificationV1.NotificationDelivery]{
		{header: "createdAt", value: func(r *notificationV1.NotificationDelivery) string { return fmtTime(r.GetCreatedAt()) }},
		{header: "sentAt", value: func(r *notificationV1.NotificationDelivery) string { return fmtTime(r.GetSentAt()) }},
		{header: "eventType", value: func(r *notificationV1.NotificationDelivery) string { return r.GetEventType().String() }},
		{header: "channel", value: func(r *notificationV1.NotificationDelivery) string { return r.GetChannel().String() }},
		{header: "status", value: func(r *notificationV1.NotificationDelivery) string { return r.GetStatus().String() }},
		{header: "target", value: func(r *notificationV1.NotificationDelivery) string { return r.GetTarget() }},
		{header: "attempts", value: func(r *notificationV1.NotificationDelivery) string {
			return strconv.FormatInt(int64(r.GetAttempts()), 10)
		}},
		{header: "recipientUserId", value: func(r *notificationV1.NotificationDelivery) string {
			return strconv.FormatInt(r.GetRecipientUserId(), 10)
		}},
		{header: "channelId", value: func(r *notificationV1.NotificationDelivery) string {
			return strconv.FormatInt(r.GetChannelId(), 10)
		}},
		{header: "relatedId", value: func(r *notificationV1.NotificationDelivery) string {
			return strconv.FormatInt(r.GetRelatedId(), 10)
		}},
		{header: "requestId", value: func(r *notificationV1.NotificationDelivery) string { return r.GetRequestId() }},
		{header: "lastError", value: func(r *notificationV1.NotificationDelivery) string { return r.GetLastError() }},
	}
}
