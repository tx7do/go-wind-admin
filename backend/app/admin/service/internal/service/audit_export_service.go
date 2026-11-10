package service

import (
	"net/http"
	"strconv"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/pkg/middleware/auth"
)

// 审计日志服务端导出的单次行数上限。
// 全量导出的意义是突破前端客户端聚合的 1 万行上限，但无上限会把一次点击变成
// 一次无界查询（OOM 风险）；50 万行约等于等保场景下几个月的量，真要更多走归档 JSONL。
const auditExportMaxRows = 500000

// AuditExportService 审计日志服务端导出（XLSX/CSV，流式二进制响应）。
// 五类审计日志共用一个端点，按 type 分派；条件与列表页同源（query JSON 透传），
// 因此"导出的就是当前搜索看到的"。
type AuditExportService struct {
	loginRepo            *data.LoginAuditLogRepo
	apiRepo              *data.ApiAuditLogRepo
	operationRepo        *data.OperationAuditLogRepo
	dataAccessRepo       *data.DataAccessAuditLogRepo
	permissionRepo       *data.PermissionAuditLogRepo
	policyEvaluationRepo *data.PolicyEvaluationLogRepo
	log                  *bLogger.Helper

	// tokenChecker 手动路由的鉴权锚：导出端点未设 Operation，auth 中间件的
	// selector 不会应用（也就不会注入操作人），须自行验 Bearer token。
	tokenChecker auth.AccessTokenChecker
}

func NewAuditExportService(
	ctx *bootstrap.Context,
	loginRepo *data.LoginAuditLogRepo,
	apiRepo *data.ApiAuditLogRepo,
	operationRepo *data.OperationAuditLogRepo,
	dataAccessRepo *data.DataAccessAuditLogRepo,
	permissionRepo *data.PermissionAuditLogRepo,
	policyEvaluationRepo *data.PolicyEvaluationLogRepo,
	tokenChecker auth.AccessTokenChecker,
) *AuditExportService {
	return &AuditExportService{
		loginRepo:            loginRepo,
		apiRepo:              apiRepo,
		operationRepo:        operationRepo,
		dataAccessRepo:       dataAccessRepo,
		permissionRepo:       permissionRepo,
		policyEvaluationRepo: policyEvaluationRepo,
		log:                  ctx.NewLoggerHelper("audit-export/service/admin-service"),
		tokenChecker:         tokenChecker,
	}
}

// ServeExport 手动注册的 HTTP handler（要写二进制响应头，走不了 proto 生成路由）。
// 鉴权：手动路由不穿 kratos middleware（也就不穿 auth 中间件），token 校验与
// viewer 重建在 export_common 的公共样板里补齐；平台管理员闸与五类审计读接口同一道。
func (s *AuditExportService) ServeExport(w http.ResponseWriter, r *http.Request) error {
	ctx, operator, ok := authorizeExportRequest(w, r, s.tokenChecker)
	if !ok {
		return nil
	}
	if !requireExportPlatformAdmin(w, operator) {
		return nil
	}

	q := r.URL.Query()
	logType := q.Get("type")
	format, req, pok := parseExportRequest(w, r, auditExportMaxRows)
	if !pok {
		return nil
	}

	var (
		fileName string
		sheet    string
		headers  []string
		rows     [][]string
	)

	switch logType {
	case "login":
		resp, listErr := s.loginRepo.List(ctx, req)
		if listErr != nil {
			s.log.Errorf(ctx, "audit export [%s] query failed: %s", logType, listErr.Error())
			http.Error(w, "query failed", http.StatusInternalServerError)
			return nil
		}
		fileName, sheet = "login-audit-logs", "login"
		cols := loginExportColumns()
		headers = colHeaders(cols)
		for _, row := range resp.GetItems() {
			rows = append(rows, colValues(cols, row))
		}
	case "api":
		resp, listErr := s.apiRepo.List(ctx, req)
		if listErr != nil {
			s.log.Errorf(ctx, "audit export [%s] query failed: %s", logType, listErr.Error())
			http.Error(w, "query failed", http.StatusInternalServerError)
			return nil
		}
		fileName, sheet = "api-audit-logs", "api"
		cols := apiExportColumns()
		headers = colHeaders(cols)
		for _, row := range resp.GetItems() {
			rows = append(rows, colValues(cols, row))
		}
	case "operation":
		resp, listErr := s.operationRepo.List(ctx, req)
		if listErr != nil {
			s.log.Errorf(ctx, "audit export [%s] query failed: %s", logType, listErr.Error())
			http.Error(w, "query failed", http.StatusInternalServerError)
			return nil
		}
		fileName, sheet = "operation-audit-logs", "operation"
		cols := operationExportColumns()
		headers = colHeaders(cols)
		for _, row := range resp.GetItems() {
			rows = append(rows, colValues(cols, row))
		}
	case "data_access":
		resp, listErr := s.dataAccessRepo.List(ctx, req)
		if listErr != nil {
			s.log.Errorf(ctx, "audit export [%s] query failed: %s", logType, listErr.Error())
			http.Error(w, "query failed", http.StatusInternalServerError)
			return nil
		}
		fileName, sheet = "data-access-audit-logs", "data_access"
		cols := dataAccessExportColumns()
		headers = colHeaders(cols)
		for _, row := range resp.GetItems() {
			rows = append(rows, colValues(cols, row))
		}
	case "permission":
		resp, listErr := s.permissionRepo.List(ctx, req)
		if listErr != nil {
			s.log.Errorf(ctx, "audit export [%s] query failed: %s", logType, listErr.Error())
			http.Error(w, "query failed", http.StatusInternalServerError)
			return nil
		}
		fileName, sheet = "permission-audit-logs", "permission"
		cols := permissionExportColumns()
		headers = colHeaders(cols)
		for _, row := range resp.GetItems() {
			rows = append(rows, colValues(cols, row))
		}
	case "policy_evaluation":
		resp, listErr := s.policyEvaluationRepo.List(ctx, req)
		if listErr != nil {
			s.log.Errorf(ctx, "audit export [%s] query failed: %s", logType, listErr.Error())
			http.Error(w, "query failed", http.StatusInternalServerError)
			return nil
		}
		fileName, sheet = "policy-evaluation-logs", "policy_evaluation"
		cols := policyEvaluationExportColumns()
		headers = colHeaders(cols)
		for _, row := range resp.GetItems() {
			rows = append(rows, colValues(cols, row))
		}
	default:
		http.Error(w, "type must be one of login/api/operation/data_access/permission/policy_evaluation", http.StatusBadRequest)
		return nil
	}

	emitExportFile(w, s.log, ctx, "audit:"+logType, fileName, sheet, headers, rows, format)
	return nil
}

// ==== 各类型列定义（表头用英文字段名：导出文件没有运行时语言上下文，
// 后端文案不进 i18n 是仓库铁律的边界）====

func loginExportColumns() []exportColumn[*auditV1.LoginAuditLog] {
	return []exportColumn[*auditV1.LoginAuditLog]{
		{header: "id", value: func(r *auditV1.LoginAuditLog) string { return strconv.FormatInt(r.GetId(), 10) }},
		{header: "createdAt", value: func(r *auditV1.LoginAuditLog) string { return fmtTime(r.GetCreatedAt()) }},
		{header: "username", value: func(r *auditV1.LoginAuditLog) string { return r.GetUsername() }},
		{header: "tenantName", value: func(r *auditV1.LoginAuditLog) string { return r.GetTenantName() }},
		{header: "actionType", value: func(r *auditV1.LoginAuditLog) string { return r.GetActionType().String() }},
		{header: "status", value: func(r *auditV1.LoginAuditLog) string { return r.GetStatus().String() }},
		{header: "loginMethod", value: func(r *auditV1.LoginAuditLog) string { return r.GetLoginMethod().String() }},
		{header: "mfaStatus", value: func(r *auditV1.LoginAuditLog) string { return r.GetMfaStatus() }},
		{header: "failureReason", value: func(r *auditV1.LoginAuditLog) string { return r.GetFailureReason() }},
		{header: "ipAddress", value: func(r *auditV1.LoginAuditLog) string { return r.GetIpAddress() }},
		{header: "deviceInfo", value: func(r *auditV1.LoginAuditLog) string { return r.GetDeviceInfo().String() }},
		{header: "requestId", value: func(r *auditV1.LoginAuditLog) string { return r.GetRequestId() }},
		{header: "traceId", value: func(r *auditV1.LoginAuditLog) string { return r.GetTraceId() }},
	}
}

func apiExportColumns() []exportColumn[*auditV1.ApiAuditLog] {
	return []exportColumn[*auditV1.ApiAuditLog]{
		{header: "id", value: func(r *auditV1.ApiAuditLog) string { return strconv.FormatInt(r.GetId(), 10) }},
		{header: "createdAt", value: func(r *auditV1.ApiAuditLog) string { return fmtTime(r.GetCreatedAt()) }},
		{header: "username", value: func(r *auditV1.ApiAuditLog) string { return r.GetUsername() }},
		{header: "tenantName", value: func(r *auditV1.ApiAuditLog) string { return r.GetTenantName() }},
		{header: "httpMethod", value: func(r *auditV1.ApiAuditLog) string { return r.GetHttpMethod() }},
		{header: "path", value: func(r *auditV1.ApiAuditLog) string { return r.GetPath() }},
		{header: "apiModule", value: func(r *auditV1.ApiAuditLog) string { return r.GetApiModule() }},
		{header: "ipAddress", value: func(r *auditV1.ApiAuditLog) string { return r.GetIpAddress() }},
		{header: "requestId", value: func(r *auditV1.ApiAuditLog) string { return r.GetRequestId() }},
	}
}

func operationExportColumns() []exportColumn[*auditV1.OperationAuditLog] {
	return []exportColumn[*auditV1.OperationAuditLog]{
		{header: "id", value: func(r *auditV1.OperationAuditLog) string { return strconv.FormatInt(r.GetId(), 10) }},
		{header: "createdAt", value: func(r *auditV1.OperationAuditLog) string { return fmtTime(r.GetCreatedAt()) }},
		{header: "username", value: func(r *auditV1.OperationAuditLog) string { return r.GetUsername() }},
		{header: "tenantName", value: func(r *auditV1.OperationAuditLog) string { return r.GetTenantName() }},
		{header: "resourceType", value: func(r *auditV1.OperationAuditLog) string { return r.GetResourceType() }},
		{header: "resourceId", value: func(r *auditV1.OperationAuditLog) string { return r.GetResourceId() }},
		{header: "action", value: func(r *auditV1.OperationAuditLog) string { return r.GetAction().String() }},
		{header: "success", value: func(r *auditV1.OperationAuditLog) string { return boolStr(r.Success) }},
		{header: "sensitiveLevel", value: func(r *auditV1.OperationAuditLog) string { return r.GetSensitiveLevel().String() }},
		{header: "requestId", value: func(r *auditV1.OperationAuditLog) string { return r.GetRequestId() }},
	}
}

func dataAccessExportColumns() []exportColumn[*auditV1.DataAccessAuditLog] {
	return []exportColumn[*auditV1.DataAccessAuditLog]{
		{header: "id", value: func(r *auditV1.DataAccessAuditLog) string { return strconv.FormatInt(r.GetId(), 10) }},
		{header: "createdAt", value: func(r *auditV1.DataAccessAuditLog) string { return fmtTime(r.GetCreatedAt()) }},
		{header: "username", value: func(r *auditV1.DataAccessAuditLog) string { return r.GetUsername() }},
		{header: "tenantName", value: func(r *auditV1.DataAccessAuditLog) string { return r.GetTenantName() }},
		{header: "dataSource", value: func(r *auditV1.DataAccessAuditLog) string { return r.GetDataSource() }},
		{header: "tableName", value: func(r *auditV1.DataAccessAuditLog) string { return r.GetTableName() }},
		{header: "dataId", value: func(r *auditV1.DataAccessAuditLog) string { return r.GetDataId() }},
		{header: "accessType", value: func(r *auditV1.DataAccessAuditLog) string { return r.GetAccessType().String() }},
		{header: "affectedRows", value: func(r *auditV1.DataAccessAuditLog) string { return strconv.FormatInt(int64(r.GetAffectedRows()), 10) }},
		{header: "sqlDigest", value: func(r *auditV1.DataAccessAuditLog) string { return r.GetSqlDigest() }},
		{header: "ipAddress", value: func(r *auditV1.DataAccessAuditLog) string { return r.GetIpAddress() }},
	}
}

// policyEvaluationExportColumns 与策略评估日志页展示列一致
// （logHash/signature 为完整性校验字段，页面不展示，不进导出）。
func policyEvaluationExportColumns() []exportColumn[*permissionV1.PolicyEvaluationLog] {
	return []exportColumn[*permissionV1.PolicyEvaluationLog]{
		{header: "id", value: func(r *permissionV1.PolicyEvaluationLog) string { return strconv.FormatInt(r.GetId(), 10) }},
		{header: "createdAt", value: func(r *permissionV1.PolicyEvaluationLog) string { return fmtTime(r.GetCreatedAt()) }},
		{header: "tenantId", value: func(r *permissionV1.PolicyEvaluationLog) string {
			return strconv.FormatInt(r.GetTenantId(), 10)
		}},
		{header: "userId", value: func(r *permissionV1.PolicyEvaluationLog) string { return strconv.FormatInt(r.GetUserId(), 10) }},
		{header: "membershipId", value: func(r *permissionV1.PolicyEvaluationLog) string {
			return strconv.FormatInt(r.GetMembershipId(), 10)
		}},
		{header: "permissionId", value: func(r *permissionV1.PolicyEvaluationLog) string {
			return strconv.FormatInt(r.GetPermissionId(), 10)
		}},
		{header: "policyId", value: func(r *permissionV1.PolicyEvaluationLog) string {
			return strconv.FormatInt(r.GetPolicyId(), 10)
		}},
		{header: "requestPath", value: func(r *permissionV1.PolicyEvaluationLog) string { return r.GetRequestPath() }},
		{header: "requestMethod", value: func(r *permissionV1.PolicyEvaluationLog) string { return r.GetRequestMethod() }},
		{header: "result", value: func(r *permissionV1.PolicyEvaluationLog) string { return boolStr(r.Result) }},
		{header: "effectDetails", value: func(r *permissionV1.PolicyEvaluationLog) string { return r.GetEffectDetails() }},
		{header: "scopeSql", value: func(r *permissionV1.PolicyEvaluationLog) string { return r.GetScopeSql() }},
		{header: "ipAddress", value: func(r *permissionV1.PolicyEvaluationLog) string { return r.GetIpAddress() }},
		{header: "traceId", value: func(r *permissionV1.PolicyEvaluationLog) string { return r.GetTraceId() }},
		{header: "evaluationContext", value: func(r *permissionV1.PolicyEvaluationLog) string { return r.GetEvaluationContext() }},
	}
}

func permissionExportColumns() []exportColumn[*auditV1.PermissionAuditLog] {
	return []exportColumn[*auditV1.PermissionAuditLog]{
		{header: "id", value: func(r *auditV1.PermissionAuditLog) string { return strconv.FormatInt(r.GetId(), 10) }},
		{header: "createdAt", value: func(r *auditV1.PermissionAuditLog) string { return fmtTime(r.GetCreatedAt()) }},
		{header: "operatorName", value: func(r *auditV1.PermissionAuditLog) string { return r.GetOperatorName() }},
		{header: "targetType", value: func(r *auditV1.PermissionAuditLog) string { return r.GetTargetType() }},
		{header: "targetName", value: func(r *auditV1.PermissionAuditLog) string { return r.GetTargetName() }},
		{header: "action", value: func(r *auditV1.PermissionAuditLog) string { return r.GetAction().String() }},
		{header: "reason", value: func(r *auditV1.PermissionAuditLog) string { return r.GetReason() }},
		{header: "ipAddress", value: func(r *auditV1.PermissionAuditLog) string { return r.GetIpAddress() }},
	}
}
