package logging

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/proto"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
)

type ApiAuditLogMiddleware struct {
	op *options
}

func NewApiAuditLogMiddleware(op *options) *ApiAuditLogMiddleware {
	return &ApiAuditLogMiddleware{
		op: op,
	}
}

func (a *ApiAuditLogMiddleware) Name() string {
	return "ApiAuditLogMiddleware"
}

func (a *ApiAuditLogMiddleware) Handle(ctx context.Context, htr *http.Transport, middleErr error, latencyMs int64) {
	if htr.Operation() == a.op.loginOperation {
		return
	}

	apiAuditLog := &auditV1.ApiAuditLog{}

	clientIp := getClientRealIP(htr.Request())
	referer, _ := url.QueryUnescape(htr.RequestHeader().Get(HeaderKeyReferer))
	requestUri, _ := url.QueryUnescape(htr.Request().RequestURI)
	bodyBytes, _ := io.ReadAll(htr.Request().Body)

	apiAuditLog.HttpMethod = trans.Ptr(htr.Request().Method)
	apiAuditLog.ApiOperation = trans.Ptr(htr.Operation())
	apiAuditLog.Path = trans.Ptr(htr.PathTemplate())
	apiAuditLog.Referer = trans.Ptr(referer)
	apiAuditLog.IpAddress = trans.Ptr(clientIp)
	apiAuditLog.RequestId = trans.Ptr(getRequestId(htr.Request()))
	apiAuditLog.RequestUri = trans.Ptr(requestUri)
	apiAuditLog.RequestBody = trans.Ptr(string(bodyBytes))

	ut := extractAuthToken(htr)
	if ut != nil {
		apiAuditLog.UserId = trans.Ptr(ut.UserId)
		apiAuditLog.TenantId = ut.TenantId
		apiAuditLog.Username = ut.Username
	}

	// 地理位置
	apiAuditLog.GeoLocation = fillGeoLocation(clientIp)

	// 用户设备信息
	apiAuditLog.DeviceInfo = fillDeviceInfo(htr, ut)

	// 获取错误码和是否成功
	statusCode, reason, success := getStatusCode(middleErr)

	apiAuditLog.CostTimeMs = trans.Ptr(uint64(latencyMs))
	apiAuditLog.StatusCode = trans.Ptr(statusCode)
	apiAuditLog.Reason = trans.Ptr(reason)
	apiAuditLog.Success = trans.Ptr(success)

	// 计算哈希和签名
	apiAuditLog.LogHash = trans.Ptr(a.hashLog(apiAuditLog))
	apiAuditLog.Signature = a.signature(apiAuditLog)

	// 写入日志
	if a.op.writeApiLogFunc != nil {
		_ = a.op.writeApiLogFunc(ctx, apiAuditLog)
	}
}

// hashLog 计算日志的 SHA256 哈希（十六进制小写字符串）
// 规则：排除 log_hash 和 signature 字段，Protobuf 确定性序列化后哈希
func (a *ApiAuditLogMiddleware) hashLog(apiAuditLog *auditV1.ApiAuditLog) string {
	if apiAuditLog == nil {
		return ""
	}

	apiAuditLog.LogHash = nil
	apiAuditLog.Signature = nil

	rawBytes, err := proto.Marshal(apiAuditLog)
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
func (a *ApiAuditLogMiddleware) signature(apiAuditLog *auditV1.ApiAuditLog) []byte {
	if apiAuditLog == nil || a.op.ecPrivateKey == nil {
		return nil
	}

	tenantID := apiAuditLog.GetTenantId()
	userID := apiAuditLog.GetUserId()
	logHash := apiAuditLog.GetLogHash()
	createdAt := apiAuditLog.GetCreatedAt()

	type signContent struct {
		TenantID uint32 `json:"tenant_id"`
		UserID   uint32 `json:"user_id"`
		Sec      int64  `json:"sec"`   // createdAt 秒数
		Nanos    int32  `json:"nanos"` // createdAt 纳秒数
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

	r, s, err := ecdsa.Sign(rand.Reader, a.op.ecPrivateKey, scHash[:])
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
