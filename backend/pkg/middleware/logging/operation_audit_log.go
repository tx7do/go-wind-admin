package logging

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/proto"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"

	appViewer "go-wind-admin/pkg/entgo/viewer"
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
// resource_type 取 ServiceName 去除 "Service" 后缀并转小写；action 按 Method 映射 ActionType。
// 非写操作（Create/Update/Delete/BatchCreate 之外）返回 UNSPECIFIED，由调用方决定是否落库。
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
	case "Delete":
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
	// 仅对写操作落库：解析不出 resource_type 或 action 为 UNSPECIFIED 时跳过。
	resourceType, action := parseResourceAndAction(htr.Operation())
	if resourceType == "" || action == auditV1.OperationAuditLog_ACTION_TYPE_UNSPECIFIED {
		return
	}

	operationAuditLog := &auditV1.OperationAuditLog{}

	operationAuditLog.ResourceType = trans.Ptr(resourceType)
	operationAuditLog.Action = trans.Ptr(action)

	clientIp := getClientRealIP(htr.Request())

	operationAuditLog.IpAddress = trans.Ptr(clientIp)
	operationAuditLog.RequestId = trans.Ptr(getRequestId(htr.Request()))

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

	operationAuditLog.LogHash = trans.Ptr(o.hashLog(operationAuditLog))
	operationAuditLog.Signature = o.signature(operationAuditLog)

	if o.op.writeOperationAuditLogFunc != nil {
		ctx = appViewer.NewSystemViewerContext(ctx)
		_ = o.op.writeOperationAuditLogFunc(ctx, operationAuditLog)
	}
}

// hashLog 计算日志的 SHA256 哈希（十六进制小写字符串）
// 规则：排除 log_hash 和 signature 字段，Protobuf 确定性序列化后哈希
func (o *OperationAuditLogMiddleware) hashLog(operationAuditLog *auditV1.OperationAuditLog) string {
	if operationAuditLog == nil {
		return ""
	}

	operationAuditLog.LogHash = nil
	operationAuditLog.Signature = nil

	rawBytes, err := proto.Marshal(operationAuditLog)
	if err != nil {
		fmt.Printf("marshal log failed: %v\n", err)
		return ""
	}

	hash := sha256.Sum256(rawBytes)
	return hex.EncodeToString(hash[:])
}

// signature 生成日志的 ECDSA 数字签名
// 签名内容：tenant_id + user_id + created_at（原始时间戳） + log_hash
// 返回：ECDSA 签名字节数组（r+s 拼接，DER 格式）
func (o *OperationAuditLogMiddleware) signature(operationAuditLog *auditV1.OperationAuditLog) []byte {
	if operationAuditLog == nil || o.op.ecPrivateKey == nil {
		return nil
	}

	tenantID := operationAuditLog.GetTenantId()
	userID := operationAuditLog.GetUserId()
	logHash := operationAuditLog.GetLogHash()
	createdAt := operationAuditLog.GetCreatedAt()

	type signContent struct {
		TenantID uint32 `json:"tenant_id"`
		UserID   uint32 `json:"user_id"`
		Sec      int64  `json:"sec"`
		Nanos    int32  `json:"nanos"`
		LogHash  string `json:"log_hash"`
	}
	sc := signContent{
		TenantID: tenantID,
		UserID:   userID,
		LogHash:  logHash,
	}
	if createdAt != nil {
		sc.Sec = createdAt.Seconds
		sc.Nanos = createdAt.Nanos
	}

	scBytes, err := json.Marshal(sc)
	if err != nil {
		fmt.Printf("marshal sign content failed: %v\n", err)
		return nil
	}

	scHash := sha256.Sum256(scBytes)

	r, s, err := ecdsa.Sign(rand.Reader, o.op.ecPrivateKey, scHash[:])
	if err != nil {
		fmt.Printf("ECDSA sign failed: %v\n", err)
		return nil
	}

	signBytes, err := encodeDER(r, s)
	if err != nil {
		fmt.Printf("encode DER failed: %v\n", err)
		return nil
	}

	return signBytes
}
