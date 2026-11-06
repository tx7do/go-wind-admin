package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/tx7do/go-utils/auditutil"

	"go-wind-admin/pkg/netutil"
	"github.com/tx7do/go-utils/trans"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"

	"github.com/tx7do/go-crud/viewer"
)

type PermissionAuditLogMiddleware struct {
	op *options
}

func NewPermissionAuditLogMiddleware(op *options) *PermissionAuditLogMiddleware {
	return &PermissionAuditLogMiddleware{
		op: op,
	}
}

func (p *PermissionAuditLogMiddleware) Name() string {
	return "PermissionAuditLogMiddleware"
}

// parseTargetAndAction 从 kratos operation 字符串解析目标类型与动作。
// operation 格式统一为 "/<package>.<ServiceName>/<Method>"（如 "/admin.service.v1.PermissionService/Update"）。
// target_type 取 ServiceName 去除 "Service" 后缀转小写；action 按 Method 映射
// PermissionAuditLog_ActionType。未识别的写方法返回 OTHER（读请求由调用方按
// HTTP 方法先行拦截，不会走到这里）。
func parseTargetAndAction(operation string) (string, auditV1.PermissionAuditLog_ActionType) {
	slash := strings.LastIndex(operation, "/")
	if slash < 0 || slash == len(operation)-1 {
		return "", auditV1.PermissionAuditLog_ACTION_TYPE_UNSPECIFIED
	}
	servicePart := operation[:slash]
	method := operation[slash+1:]

	dot := strings.LastIndex(servicePart, ".")
	if dot < 0 || dot == len(servicePart)-1 {
		return "", auditV1.PermissionAuditLog_ACTION_TYPE_UNSPECIFIED
	}
	svcName := servicePart[dot+1:]
	svcName = strings.TrimSuffix(svcName, "Service")
	targetType := strings.ToLower(svcName)

	var action auditV1.PermissionAuditLog_ActionType
	switch method {
	case "Create", "BatchCreate":
		action = auditV1.PermissionAuditLog_CREATE
	case "Update":
		action = auditV1.PermissionAuditLog_UPDATE
	case "Delete", "BatchDelete":
		action = auditV1.PermissionAuditLog_DELETE
	case "Assign":
		action = auditV1.PermissionAuditLog_ASSIGN
	case "Unassign":
		action = auditV1.PermissionAuditLog_UNASSIGN
	default:
		return targetType, auditV1.PermissionAuditLog_OTHER
	}
	return targetType, action
}

func (p *PermissionAuditLogMiddleware) Handle(ctx context.Context, htr *http.Transport, middleErr error, latencyMs int64) {
	// 仅对写操作落库。按 HTTP 方法判读请求（比方法名前缀猜测可靠）：
	// GET/HEAD/OPTIONS 属读请求，不构成权限变更——此前 default 分支返回 OTHER
	// 使每次页面浏览都产生一条 GET 噪音行（且无请求体，目标名称恒空）。
	// 读请求的审计归 API日志（全量请求）与 策略评估日志（鉴权评估）。
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
	// 解析不出 target_type 或 action 为 UNSPECIFIED 时跳过。
	targetType, action := parseTargetAndAction(htr.Operation())
	if targetType == "" || action == auditV1.PermissionAuditLog_ACTION_TYPE_UNSPECIFIED {
		return
	}

	permissionAuditLog := &auditV1.PermissionAuditLog{}

	permissionAuditLog.TargetType = trans.Ptr(targetType)
	permissionAuditLog.Action = trans.Ptr(action)

	// 目标ID：REST 路径最后一个纯数字段（如 /admin/v1/roles/5 → "5"）。
	if req := htr.Request(); req != nil {
		if tid := lastNumericPathSegment(req.URL.Path); tid != "" {
			permissionAuditLog.TargetId = trans.Ptr(tid)
		}
	}
	// 目标名称：JSON 写请求体（CRUD 包 {"data":{...}}）里第一个非空名称类字段。
	if name := targetNameFromBody(bodySnapshotFromContext(ctx)); name != "" {
		permissionAuditLog.TargetName = trans.Ptr(name)
	}

	clientIp := netutil.ClientRealIPFromRequest(htr.Request())

	permissionAuditLog.IpAddress = trans.Ptr(clientIp)
	permissionAuditLog.RequestId = trans.Ptr(auditutil.RequestID(htr.Request()))

	ut := extractAuthToken(htr)
	if ut != nil {
		permissionAuditLog.OperatorId = trans.Ptr(ut.UserId)
		permissionAuditLog.OperatorName = ut.Username
		permissionAuditLog.TenantId = ut.TenantId
	}

	// 变更原因：记录操作上下文（方法 + 路径）；失败时附错误原因码。
	reason := ""
	if req := htr.Request(); req != nil {
		reason = req.Method + " " + req.URL.Path
	}
	if middleErr != nil {
		_, errReason, _ := getStatusCode(middleErr)
		reason = reason + " failed: " + errReason
	}
	permissionAuditLog.Reason = trans.Ptr(reason)

	permissionAuditLog.LogHash = trans.Ptr(auditutil.HashLog(permissionAuditLog))
	signature, signErr := auditutil.SignLogContent(p.op.ecPrivateKey, permissionAuditLog.GetTenantId(), permissionAuditLog.GetOperatorId(), permissionAuditLog.GetCreatedAt(), permissionAuditLog.GetLogHash())
	if signErr != nil {
		fmt.Printf("sign log content failed: %v\n", signErr)
	}
	permissionAuditLog.Signature = signature

	if p.op.writePermissionAuditLogFunc != nil {
		ctx = viewer.WithSystemContext(ctx)
		_ = p.op.writePermissionAuditLogFunc(ctx, permissionAuditLog)
	}
}

// lastNumericPathSegment 返回路径中最后一个纯数字段（如 /admin/v1/roles/5 → "5"），
// 无数字段返回空串。嵌套资源（/users/1/roles）取最深一层的属主 ID。
func lastNumericPathSegment(path string) string {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for i := len(segments) - 1; i >= 0; i-- {
		s := segments[i]
		if s == "" {
			continue
		}
		allDigits := true
		for j := 0; j < len(s); j++ {
			if s[j] < '0' || s[j] > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return s
		}
	}
	return ""
}

// targetNameFromNameCandidates 按优先级尝试的名称类字段（protojson 驼峰）。
var targetNameFromNameCandidates = []string{"name", "username", "typeName", "code", "typeCode", "title"}

// targetNameFromBody 从 JSON 写请求体提取目标名称：兼容 CRUD 的 {"data":{...}}
// 包装与平铺对象，先按候选字段精确匹配，再兜底任何以 name/code 结尾的键
// （覆盖 roleName/menuName 等未列举字段）。
func targetNameFromBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return ""
	}
	if d, ok := m["data"].(map[string]any); ok {
		m = d
	}
	for _, key := range targetNameFromNameCandidates {
		if s, ok := m[key].(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	// 兜底：按 key 排序保证确定性，name 结尾优先于 code 结尾
	var nameKeys, codeKeys []string
	for key := range m {
		lk := strings.ToLower(key)
		switch {
		case strings.HasSuffix(lk, "name"):
			nameKeys = append(nameKeys, key)
		case strings.HasSuffix(lk, "code"):
			codeKeys = append(codeKeys, key)
		}
	}
	sort.Strings(nameKeys)
	sort.Strings(codeKeys)
	for _, keys := range [][]string{nameKeys, codeKeys} {
		for _, key := range keys {
			if s, ok := m[key].(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
	}
	return ""
}
