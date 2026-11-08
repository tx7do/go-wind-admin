package logging

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/tx7do/go-utils/auditutil"

	"github.com/tx7do/go-utils/netutil"
	"github.com/tx7do/go-utils/trans"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"

	"github.com/tx7do/go-crud/viewer"
)

type OperationAuditLogMiddleware struct {
	op *options
}

func NewOperationAuditLogMiddleware(op *options) *OperationAuditLogMiddleware {
	return &OperationAuditLogMiddleware{
		op: op,
	}
}

func (o *OperationAuditLogMiddleware) Name() string {
	return "OperationAuditLogMiddleware"
}

// parseResourceAndAction 从 kratos operation 字符串解析资源类型与动作。
// operation 格式统一为 "/<package>.<ServiceName>/<Method>"（如 "/admin.service.v1.RoleService/Update"）。
// resource_type 取 ServiceName 去除 "Service" 后缀转小写；action 按 Method 映射 ActionType。
// 未识别的写方法返回 OTHER（读请求由调用方按 HTTP 方法先行拦截，不会走到这里）。
func parseResourceAndAction(operation string) (string, auditV1.OperationAuditLog_ActionType) {
	slash := strings.LastIndex(operation, "/")
	if slash < 0 || slash == len(operation)-1 {
		return "", auditV1.OperationAuditLog_ACTION_TYPE_UNSPECIFIED
	}
	servicePart := operation[:slash]
	method := operation[slash+1:]

	dot := strings.LastIndex(servicePart, ".")
	if dot < 0 || dot == len(servicePart)-1 {
		return "", auditV1.OperationAuditLog_ACTION_TYPE_UNSPECIFIED
	}
	svcName := servicePart[dot+1:]
	svcName = strings.TrimSuffix(svcName, "Service")
	resourceType := strings.ToLower(svcName)

	var action auditV1.OperationAuditLog_ActionType
	switch method {
	case "Create", "BatchCreate":
		action = auditV1.OperationAuditLog_CREATE
	case "Update":
		action = auditV1.OperationAuditLog_UPDATE
	case "Delete", "BatchDelete":
		action = auditV1.OperationAuditLog_DELETE
	case "Export":
		action = auditV1.OperationAuditLog_EXPORT
	case "Import":
		action = auditV1.OperationAuditLog_IMPORT
	case "Assign":
		action = auditV1.OperationAuditLog_ASSIGN
	case "Unassign":
		action = auditV1.OperationAuditLog_UNASSIGN
	default:
		return resourceType, auditV1.OperationAuditLog_OTHER
	}
	return resourceType, action
}

func (o *OperationAuditLogMiddleware) Handle(ctx context.Context, htr *http.Transport, middleErr error, latencyMs int64) {
	// 仅对写操作落库。按 HTTP 方法判读请求（比方法名前缀猜测可靠）：
	// GET/HEAD/OPTIONS 属读请求，不构成变更——此前 default 分支返回 OTHER
	// 使每次页面浏览都产生一条 GET 噪音行。读请求的审计归 API日志（全量请求）。
	if req := htr.Request(); req != nil {
		switch req.Method {
		case "POST", "PUT", "PATCH", "DELETE":
		default:
			return
		}
	}
	// 会话维护端点（登录/刷新/登出/MFA 验证）不是变更，交给登录审计。
	if sessionOnlyOperations[htr.Operation()] {
		return
	}
	// 解析不出 resource_type 或 action 为 UNSPECIFIED 时跳过。
	resourceType, action := parseResourceAndAction(htr.Operation())
	if resourceType == "" || action == auditV1.OperationAuditLog_ACTION_TYPE_UNSPECIFIED {
		return
	}

	operationAuditLog := &auditV1.OperationAuditLog{}

	operationAuditLog.ResourceType = trans.Ptr(resourceType)
	operationAuditLog.Action = trans.Ptr(action)

	// 资源ID：REST 路径最后一个纯数字段（如 /admin/v1/roles/5 → "5"）。
	// Create 无路径 ID（资源 id 在响应体中，post-handler 不可见），留空。
	if req := htr.Request(); req != nil {
		if rid := lastNumericPathSegment(req.URL.Path); rid != "" {
			operationAuditLog.ResourceId = trans.Ptr(rid)
		}
	}

	clientIp := netutil.ClientIPFromRequest(htr.Request())

	operationAuditLog.IpAddress = trans.Ptr(clientIp)
	operationAuditLog.RequestId = trans.Ptr(auditutil.RequestID(htr.Request()))

	ut := extractAuthToken(htr)
	if ut != nil {
		operationAuditLog.UserId = trans.Ptr(ut.UserId)
		operationAuditLog.TenantId = ut.TenantId
		operationAuditLog.Username = ut.Username
	}

	operationAuditLog.GeoLocation = fillGeoLocation(clientIp)

	statusCode, reason, success := getStatusCode(middleErr)

	operationAuditLog.Success = trans.Ptr(success)
	operationAuditLog.FailureReason = trans.Ptr(reason)

	_ = statusCode

	operationAuditLog.LogHash = trans.Ptr(auditutil.HashLog(operationAuditLog))
	signature, signErr := auditutil.SignLogContent(o.op.ecPrivateKey, operationAuditLog.GetTenantId(), operationAuditLog.GetUserId(), operationAuditLog.GetCreatedAt(), operationAuditLog.GetLogHash())
	if signErr != nil {
		fmt.Printf("sign log content failed: %v\n", signErr)
	}
	operationAuditLog.Signature = signature

	if o.op.writeOperationAuditLogFunc != nil {
		ctx = viewer.WithSystemContext(ctx)
		_ = o.op.writeOperationAuditLogFunc(ctx, operationAuditLog)
	}
}
