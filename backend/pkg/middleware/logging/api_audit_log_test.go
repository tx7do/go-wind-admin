// api_audit_log_test.go —— ApiAuditLogMiddleware 测试。
//
// 覆盖内容：
//  1. 构造器与 Name()；
//  2. 登录类操作（含 MFA 验证）的跳过分支（直调，operation 命中 loginOperations
//     即返回，不触碰请求字段）；
//  3. 触发路径的字段来源映射（进程内 server）：HTTP 方法、operation、路径模板、
//     Referer 反转义、客户端 IP（RemoteAddr / X-Forwarded-For）、请求 ID 头、
//     RequestURI 反转义、请求体重放、令牌身份字段（uid/tid/sub）、地理位置
//     （私网 IP → 局域网）、设备信息（UA 解析 + 令牌 cid）；
//  4. 错误状态映射（kratos 错误 / 普通错误 → status/reason/success）与
//     输入清洗（非法 Referer 转义 → 空串；非 Bearer 令牌 → 无身份）；
//  5. 设备类型分类（桌面 / 移动 / 爬虫 UA）；
//  6. writeApiLogFunc 为 nil 时只跳过落库、记录构造照常；
//  7. hashLog/signature 的 nil 边界，以及注入密钥的签名可回验性
//     （DER 拆解 + ecdsa.Verify，证明 WithECPrivateKey 的密钥真实参与签名）。
package logging

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/json"
	nethttp "net/http"
	"testing"

	"github.com/go-kratos/kratos/v2/transport"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
)

const (
	testDesktopUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	testMobileUA  = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1"
	testBotUA     = "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
	testTabletUA  = "Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1"
)

// TestApiAuditLogMiddlewareNameAndConstructor 验证构造器落参与固定名称。
func TestApiAuditLogMiddlewareNameAndConstructor(t *testing.T) {
	op := &options{}
	mw := NewApiAuditLogMiddleware(op)
	require.NotNil(t, mw)
	assert.Equal(t, op, mw.op, "构造器必须保存 options 引用")
	assert.Equal(t, "ApiAuditLogMiddleware", mw.Name())
}

// TestApiAuditLogHandleSkipsLoginOperationsDirect 直调验证：operation 命中
// loginOperations（登录 / MFA 验证）时 Handle 必须在读请求字段之前返回——
// 零值 Transport 的 Request() 为 nil，若未提前返回会在 RequestURI 处 panic，
// 测试以"不 panic 且零写入"双重断言该跳过语义。
func TestApiAuditLogHandleSkipsLoginOperationsDirect(t *testing.T) {
	var written int
	var op options
	WithLoginOperation(
		adminV1.OperationAuthenticationServiceLogin,
		adminV1.OperationMfaServiceVerifyMFAChallenge,
	)(&op)
	WithWriteApiLogFunc(func(ctx context.Context, d *auditV1.ApiAuditLog) error {
		written++
		return nil
	})(&op)
	mw := NewApiAuditLogMiddleware(&op)

	for _, loginOp := range []string{
		adminV1.OperationAuthenticationServiceLogin,
		adminV1.OperationMfaServiceVerifyMFAChallenge,
	} {
		tr := &khttp.Transport{}
		ctx := transport.NewServerContext(context.Background(), tr)
		khttp.SetOperation(ctx, loginOp)
		mw.Handle(ctx, tr, nil, 0) // 不得 panic：跳过分支在解引用请求前返回
	}
	assert.Zero(t, written, "登录类操作不得写入 API 审计")
}

// TestApiAuditLogHandleFieldMapping 走进程内 server 验证触发路径的完整字段
// 来源映射：每个字段都断言其从请求的哪个部位取值，防止映射漂移。
func TestApiAuditLogHandleFieldMapping(t *testing.T) {
	env := newAuditServer(t)
	body := `{"data":{"name":"gadget"}}`
	env.fire(nethttp.MethodPost, "/case/5?redir=%2Fadmin", map[string]string{
		"X-Test-Operation": "/demo.v1.GadgetService/Update",
		"Content-Type":     "application/json",
		"User-Agent":       testDesktopUA,
		"X-Request-ID":     "req-api-1",
		"Referer":          "https%3A%2F%2Fevil.example%2Fpath",
		"Authorization":    "Bearer " + mintTestToken(t),
	}, body, "127.0.0.1:1234")

	require.Len(t, env.capture.api, 1, "API 审计应写入一条记录")
	rec := env.capture.api[0]

	// 请求原语映射。
	assert.Equal(t, nethttp.MethodPost, rec.GetHttpMethod(), "HttpMethod ← 请求方法")
	assert.Equal(t, "/demo.v1.GadgetService/Update", rec.GetApiOperation(), "ApiOperation ← SetOperation 值")
	assert.Equal(t, "/case/{id}", rec.GetPath(), "Path ← 路由路径模板而非具体路径")
	assert.Equal(t, "/case/5?redir=/admin", rec.GetRequestUri(), "RequestUri ← RequestURI 且经 URL 反转义")
	assert.Equal(t, "https://evil.example/path", rec.GetReferer(), "Referer ← 请求头且经 URL 反转义")
	assert.Equal(t, "127.0.0.1", rec.GetIpAddress(), "IpAddress ← RemoteAddr（无代理头时）")
	assert.Equal(t, "req-api-1", rec.GetRequestId(), "RequestId ← X-Request-ID 头")
	assert.Equal(t, body, rec.GetRequestBody(), "RequestBody ← 快照重放后的完整 JSON 体")

	// 令牌身份映射（uid/tid/sub → UserId/TenantId/Username）。
	assert.Equal(t, uint32(42), rec.GetUserId())
	assert.Equal(t, uint32(7), rec.GetTenantId())
	assert.Equal(t, "alice", rec.GetUsername())

	// 地理位置映射（私网 IP → 局域网，GeoLite 库内建分支）。
	geo := rec.GetGeoLocation()
	require.NotNil(t, geo)
	assert.Equal(t, "局域网", geo.GetCountryCode())
	assert.Equal(t, "局域网", geo.GetProvince())
	assert.Equal(t, "局域网", geo.GetCity())

	// 设备信息映射（UA 解析 + 令牌 cid）。
	di := rec.GetDeviceInfo()
	require.NotNil(t, di)
	assert.Equal(t, testDesktopUA, di.GetUserAgent(), "UserAgent ← 原始 UA 头")
	assert.Equal(t, auditV1.DeviceInfo_DESKTOP, di.GetDeviceType(), "桌面 UA → DESKTOP")
	assert.Equal(t, "PC", di.GetClientName(), "桌面 UA 无设备名时回落 PC")
	assert.Equal(t, "Chrome", di.GetBrowserName())
	assert.Equal(t, "120.0.0.0", di.GetBrowserVersion())
	assert.Equal(t, "Windows", di.GetOsName())
	assert.Equal(t, "10.0", di.GetOsVersion())
	assert.Equal(t, PlatformWeb, di.GetPlatform(), "Mozilla+Windows 桌面浏览器 → Web")
	assert.Equal(t, "test-client-id", di.GetClientId(), "ClientId ← 令牌 cid（无 X-Client-ID 头时）")

	// 错误映射：无中间件错误 → 200 / 成功。
	assert.Equal(t, uint32(200), rec.GetStatusCode())
	assert.Empty(t, rec.GetReason())
	assert.True(t, rec.GetSuccess())

	// 耗时与完整性字段。
	require.NotNil(t, rec.LatencyMs)
	assert.Less(t, rec.GetLatencyMs(), uint32(60000), "耗时应为测量值而非溢出垃圾")
	requireLogHashHex(t, rec.GetLogHash())
	requireDERSig(t, rec.GetSignature()) // 结构合法的 DER 签名

	// 落库不变量。
	require.Len(t, env.capture.apiMeta, 1)
	assert.True(t, env.capture.apiMeta[0].Sinking, "落库阶段必须带 sink 标记")
	assert.True(t, env.capture.apiMeta[0].SystemViewer, "落库必须以系统 viewer 执行")
}

// TestApiAuditLogHandleErrorStatusMapping 验证中间件错误到
// status/reason/success 的映射：kratos 错误透传 code/reason，普通错误按
// 500/空 reason 处理，二者 success 均为 false。
func TestApiAuditLogHandleErrorStatusMapping(t *testing.T) {
	cases := []struct {
		name        string
		errFlag     string
		wantStatus  uint32
		wantReason  string
		wantSuccess bool
	}{
		{"kratos错误透传code与reason", "kratos", 403, "TEST_FORBIDDEN", false},
		{"普通错误映射为500空reason", "plain", 500, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newAuditServer(t)
			env.fire(nethttp.MethodPost, "/case/5", map[string]string{
				"X-Test-Operation": "/demo.v1.GadgetService/Update",
				"X-Test-Error":     tc.errFlag,
			}, "", "127.0.0.1:1234")
			require.Len(t, env.capture.api, 1)
			rec := env.capture.api[0]
			assert.Equal(t, tc.wantStatus, rec.GetStatusCode())
			assert.Equal(t, tc.wantReason, rec.GetReason())
			assert.Equal(t, tc.wantSuccess, rec.GetSuccess())
		})
	}
}

// TestApiAuditLogHandleInputSanitization 验证异常输入的清洗行为：
// 非法 Referer 转义序列 → 空串；非 Bearer 令牌 → 解析失败、身份字段为零值；
// 无请求 ID 头 → 生成 GUID 而非空串（保证可追踪性）；
// X-Forwarded-For 跳过不可解析项取首个合法 IP（且优先于 RemoteAddr）。
func TestApiAuditLogHandleInputSanitization(t *testing.T) {
	t.Run("非法Referer转义", func(t *testing.T) {
		env := newAuditServer(t)
		env.fire(nethttp.MethodPost, "/case/5", map[string]string{
			"X-Test-Operation": "/demo.v1.GadgetService/Update",
			"Referer":          "%zz-bad-escape",
		}, "", "127.0.0.1:1234")
		require.Len(t, env.capture.api, 1)
		assert.Empty(t, env.capture.api[0].GetReferer(), "非法转义必须归一为空串")
	})

	t.Run("非Bearer令牌", func(t *testing.T) {
		env := newAuditServer(t)
		env.fire(nethttp.MethodPost, "/case/5", map[string]string{
			"X-Test-Operation": "/demo.v1.GadgetService/Update",
			"Authorization":    "Basic xyz",
		}, "", "127.0.0.1:1234")
		require.Len(t, env.capture.api, 1)
		rec := env.capture.api[0]
		assert.Zero(t, rec.GetUserId(), "不可解析的令牌不得带出用户身份")
		assert.Zero(t, rec.GetTenantId())
		assert.Empty(t, rec.GetUsername())
		assert.Empty(t, rec.GetDeviceInfo().GetClientId(), "无令牌时设备 ClientId 必须为空")
	})

	t.Run("无请求ID头生成GUID", func(t *testing.T) {
		env := newAuditServer(t)
		env.fire(nethttp.MethodPost, "/case/5", map[string]string{
			"X-Test-Operation": "/demo.v1.GadgetService/Update",
		}, "", "127.0.0.1:1234")
		require.Len(t, env.capture.api, 1)
		requestID := env.capture.api[0].GetRequestId()
		assert.NotEmpty(t, requestID, "无请求 ID 头时必须生成 GUID 保证可追踪")
		assert.NotEqual(t, "req-api-1", requestID)
	})

	t.Run("无sub声明令牌不落用户名", func(t *testing.T) {
		env := newAuditServer(t)
		env.fire(nethttp.MethodPost, "/case/5", map[string]string{
			"X-Test-Operation": "/demo.v1.GadgetService/Update",
			"Authorization":    "Bearer " + mintTestTokenWithoutSubject(t),
		}, "", "127.0.0.1:1234")
		require.Len(t, env.capture.api, 1)
		rec := env.capture.api[0]
		// 载荷无 sub：GetSubject 报错走日志分支，数字身份照常映射、用户名不落。
		assert.Empty(t, rec.GetUsername())
		assert.Equal(t, uint32(42), rec.GetUserId())
		assert.Equal(t, uint32(7), rec.GetTenantId())
	})

	t.Run("角色声明类型非法令牌", func(t *testing.T) {
		env := newAuditServer(t)
		env.fire(nethttp.MethodPost, "/case/5", map[string]string{
			"X-Test-Operation": "/demo.v1.GadgetService/Update",
			"Authorization":    "Bearer " + mintTestTokenWithBadRoleClaim(t),
		}, "", "127.0.0.1:1234")
		require.Len(t, env.capture.api, 1)
		rec := env.capture.api[0]
		// 载荷构造器对非法角色声明类型报错：身份提取失败，全部身份字段归零。
		assert.Zero(t, rec.GetUserId())
		assert.Zero(t, rec.GetTenantId())
		assert.Empty(t, rec.GetUsername())
	})

	t.Run("XFF跳过非法项取首个合法IP", func(t *testing.T) {
		env := newAuditServer(t)
		env.fire(nethttp.MethodPost, "/case/5", map[string]string{
			"X-Test-Operation": "/demo.v1.GadgetService/Update",
			"X-Forwarded-For":  "garbage, 10.0.0.5",
		}, "", "8.8.8.8:1234")
		require.Len(t, env.capture.api, 1)
		rec := env.capture.api[0]
		assert.Equal(t, "10.0.0.5", rec.GetIpAddress(), "XFF 中首个可解析 IP 优先于 RemoteAddr")
		assert.Equal(t, "局域网", rec.GetGeoLocation().GetCountryCode(), "私网 IP 的地理归一")
	})
}

// TestApiAuditLogHandleDeviceClassification 表驱动验证设备分类：
// 移动 UA → MOBILE/iOSApp/iPhone，平板 UA → TABLET/iOSApp/iPad，
// 爬虫 UA → BOT/Other/空设备名。
func TestApiAuditLogHandleDeviceClassification(t *testing.T) {
	cases := []struct {
		name           string
		ua             string
		wantDeviceType auditV1.DeviceInfo_DeviceType
		wantPlatform   string
		wantClientName string
	}{
		{"移动Safari", testMobileUA, auditV1.DeviceInfo_MOBILE, PlatformiOSApp, "iPhone"},
		{"平板Safari", testTabletUA, auditV1.DeviceInfo_TABLET, PlatformiOSApp, "iPad"},
		{"搜索引擎爬虫", testBotUA, auditV1.DeviceInfo_BOT, PlatformOther, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newAuditServer(t)
			env.fire(nethttp.MethodPost, "/case/5", map[string]string{
				"X-Test-Operation": "/demo.v1.GadgetService/Update",
				"User-Agent":       tc.ua,
			}, "", "127.0.0.1:1234")
			require.Len(t, env.capture.api, 1)
			di := env.capture.api[0].GetDeviceInfo()
			require.NotNil(t, di)
			assert.Equal(t, tc.wantDeviceType, di.GetDeviceType())
			assert.Equal(t, tc.wantPlatform, di.GetPlatform())
			assert.Equal(t, tc.wantClientName, di.GetClientName())
		})
	}
}

// TestApiAuditLogHandleWriteFuncNil 验证 writeApiLogFunc 为 nil 时：
// 记录构造照常执行（其余审计不受影响），仅 API 审计自身跳过落库且不 panic。
func TestApiAuditLogHandleWriteFuncNil(t *testing.T) {
	env := newAuditServer(t, WithWriteApiLogFunc(nil))
	env.fire(nethttp.MethodPost, "/case/5", map[string]string{
		"X-Test-Operation": "/demo.v1.GadgetService/Update",
		"Content-Type":     "application/json",
	}, `{"data":{"name":"x"}}`, "127.0.0.1:1234")

	assert.Empty(t, env.capture.api, "空写入函数时 API 审计不得落库")
	assert.Len(t, env.capture.operation, 1, "其余审计不受影响")
	assert.Len(t, env.capture.permission, 1)
}

// TestApiAuditLogHashLogAndSignatureEdges 直调覆盖 hashLog/signature 的
// nil 边界：nil 记录返回空哈希/nil 签名；私钥缺失时签名必须为 nil 而非半成品。
func TestApiAuditLogHashLogAndSignatureEdges(t *testing.T) {
	mwNoKey := NewApiAuditLogMiddleware(&options{})
	assert.Equal(t, "", mwNoKey.hashLog(nil))
	assert.Nil(t, mwNoKey.signature(nil))
	assert.Nil(t, mwNoKey.signature(&auditV1.ApiAuditLog{}), "无私钥时签名必须为 nil")

	key, _, err := generateECDSAKeyPair()
	require.NoError(t, err)
	mwWithKey := NewApiAuditLogMiddleware(&options{ecPrivateKey: key})
	assert.Equal(t, "", mwWithKey.hashLog(nil))
	assert.Nil(t, mwWithKey.signature(nil))
}

// apiSignContentReplica 复刻生产签名的载荷结构（字段名与顺序须与
// api_audit_log.go 内的 signContent 完全一致），用于回验签名内容。
type apiSignContentReplica struct {
	TenantID uint32 `json:"tenant_id"`
	UserID   uint32 `json:"user_id"`
	Sec      int64  `json:"sec"`
	Nanos    int32  `json:"nanos"`
	LogHash  string `json:"log_hash"`
}

// TestApiAuditLogSignatureVerifiesWithInjectedKey 验证 WithECPrivateKey
// 注入的密钥真实参与签名：记录的签名可由注入公钥 + 复刻的签名载荷
// （tenant_id/user_id/零时间戳/log_hash 的 JSON）经 ecdsa.Verify 验证，
// 同时证明 DER 编码可正确往返。
func TestApiAuditLogSignatureVerifiesWithInjectedKey(t *testing.T) {
	key, _, err := generateECDSAKeyPair()
	require.NoError(t, err)
	env := newAuditServer(t,
		WithECPrivateKey(key),
		WithECPublicKey(&key.PublicKey),
	)
	env.fire(nethttp.MethodPost, "/case/5", map[string]string{
		"X-Test-Operation": "/demo.v1.GadgetService/Update",
	}, "", "127.0.0.1:1234")

	require.Len(t, env.capture.api, 1)
	rec := env.capture.api[0]
	r, s := parseDERSig(t, rec.GetSignature())

	// API 审计不设 created_at，载荷中时间戳恒为零。
	payload, err := json.Marshal(apiSignContentReplica{
		TenantID: rec.GetTenantId(),
		UserID:   rec.GetUserId(),
		LogHash:  rec.GetLogHash(),
	})
	require.NoError(t, err)
	digest := sha256.Sum256(payload)

	require.True(t, ecdsa.Verify(&key.PublicKey, digest[:], r, s),
		"注入私钥签出的签名必须能被注入公钥验证")
}
