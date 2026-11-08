package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/tx7do/go-utils/auditutil"

	"github.com/tx7do/go-utils/netutil"
	"github.com/tx7do/go-utils/trans"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"

	"github.com/tx7do/go-crud/viewer"
)

// sensitiveRequestBodyKeys 请求体脱敏的键名（小写包含匹配）。
// API 审计此前全量落请求体且无长度上限：CreateAiProvider 的明文 api_key、
// 通知渠道的 SMTP 密码/webhook secret、用户创建/改密的密码字段、知识库上传的
// ≤10MB contentBase64 全部进了 sys_api_audit_logs——任何能读该表的入口
// （列表/详情查询、库备份泄露）都能拿到这些明文凭据，超长体还会灌爆审计表。
var sensitiveRequestBodyKeys = []string{
	"password", "passwd", "secret", "token", "apikey", "api_key",
	"authorization", "credential", "contentbase64", "private_key",
}

const redactedMarker = "[REDACTED]"

// sanitizeAuditRequestBody 对落审计的请求体做脱敏 + 64KB 限长
// （与操作/权限审计的 maxBodySnapshot 快照口径对齐）。
// JSON 体按键名递归遮蔽；非 JSON 体原样截断。截断处补标记。
func sanitizeAuditRequestBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	truncated := false
	snap := body
	if len(body) > maxBodySnapshot {
		snap = body[:maxBodySnapshot]
		truncated = true
	}

	var out any
	if err := json.Unmarshal(snap, &out); err == nil {
		maskSensitiveJSON(out)
		if b, jerr := json.Marshal(out); jerr == nil {
			s := string(b)
			if truncated {
				s += "...[truncated]"
			}
			return s
		}
	}

	s := string(snap)
	if truncated {
		s += "...[truncated]"
	}
	return s
}

func isSensitiveRequestBodyKey(key string) bool {
	lk := strings.ToLower(key)
	for _, s := range sensitiveRequestBodyKeys {
		if strings.Contains(lk, s) {
			return true
		}
	}
	return false
}

func maskSensitiveJSON(node any) {
	switch v := node.(type) {
	case map[string]any:
		for k, val := range v {
			if isSensitiveRequestBodyKey(k) {
				v[k] = redactedMarker
				continue
			}
			maskSensitiveJSON(val)
		}
	case []any:
		for _, item := range v {
			maskSensitiveJSON(item)
		}
	}
}

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
	// 登录类操作（含 MFA 二次验证）由 LoginAuditLog 负责审计，这里跳过避免重复记录。
	for _, op := range a.op.loginOperations {
		if htr.Operation() == op {
			return
		}
	}

	apiAuditLog := &auditV1.ApiAuditLog{}

	clientIp := netutil.ClientIPFromRequest(htr.Request())
	referer, _ := url.QueryUnescape(htr.RequestHeader().Get(HeaderKeyReferer))
	requestUri, _ := url.QueryUnescape(htr.Request().RequestURI)
	bodyBytes, _ := io.ReadAll(htr.Request().Body)

	apiAuditLog.HttpMethod = trans.Ptr(htr.Request().Method)
	apiAuditLog.ApiOperation = trans.Ptr(htr.Operation())
	apiAuditLog.Path = trans.Ptr(htr.PathTemplate())
	apiAuditLog.Referer = trans.Ptr(referer)
	apiAuditLog.IpAddress = trans.Ptr(clientIp)
	apiAuditLog.RequestId = trans.Ptr(auditutil.RequestID(htr.Request()))
	apiAuditLog.RequestUri = trans.Ptr(requestUri)
	apiAuditLog.RequestBody = trans.Ptr(sanitizeAuditRequestBody(bodyBytes))

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

	apiAuditLog.LatencyMs = trans.Ptr(uint32(latencyMs))
	apiAuditLog.StatusCode = trans.Ptr(statusCode)
	apiAuditLog.Reason = trans.Ptr(reason)
	apiAuditLog.Success = trans.Ptr(success)

	// 计算哈希和签名
	apiAuditLog.LogHash = trans.Ptr(auditutil.HashLog(apiAuditLog))
	signature, signErr := auditutil.SignLogContent(a.op.ecPrivateKey, apiAuditLog.GetTenantId(), apiAuditLog.GetUserId(), apiAuditLog.GetCreatedAt(), apiAuditLog.GetLogHash())
	if signErr != nil {
		fmt.Printf("sign log content failed: %v\n", signErr)
	}
	apiAuditLog.Signature = signature

	// 写入日志
	if a.op.writeApiLogFunc != nil {
		ctx = viewer.WithSystemContext(ctx)
		_ = a.op.writeApiLogFunc(ctx, apiAuditLog)
	}
}
