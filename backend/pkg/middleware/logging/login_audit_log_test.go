// login_audit_log_test.go —— LoginAuditLogMiddleware 测试。
//
// 覆盖内容：
//  1. 构造器与 Name()、Handle 的 nil 守卫与非登录操作的早退分支；
//  2. 触发路径（进程内 server）：登录/登出/MFA 验证三类 operation 的
//     ActionType 与 MfaStatus 语义、用户名的三级来源（请求体 → 令牌 →
//     响应头 X-Audit-Username 兜底）、IP/地理/设备/请求 ID 映射、
//     中间件错误到 Status/FailureReason 的映射，以及端到端的风险评分/
//     等级/风险因素推导；
//  3. 直调空 Transport 的全空来源路径（覆盖 Request()/RequestHeader() 为
//     nil 时各取值分支与 getRequestId(nil) 等空值路径）；
//  4. computeRiskScore / levelFromScore / computeRiskFactors 的表驱动
//     纯函数测试：各启发式因子、负分与上限截断、等级阈值、因素去重排序。
package logging

import (
	"context"
	"fmt"
	nethttp "net/http"
	"testing"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/transport"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
)

// TestLoginAuditLogMiddlewareNameAndConstructor 验证构造器落参与固定名称。
func TestLoginAuditLogMiddlewareNameAndConstructor(t *testing.T) {
	op := &options{}
	mw := NewLoginAuditLogMiddleware(op)
	require.NotNil(t, mw)
	assert.Equal(t, op, mw.op)
	assert.Equal(t, "LoginAuditLogMiddleware", mw.Name())
}

// TestLoginAuditLogHandleNilGuards 直调验证三重早退：options 为 nil、
// 传输层为 nil、operation 既非登录类也非登出——三种情况都必须零写入。
func TestLoginAuditLogHandleNilGuards(t *testing.T) {
	var written int
	stub := func(ctx context.Context, d *auditV1.LoginAuditLog) error {
		written++
		return nil
	}

	// options 为 nil。
	(&LoginAuditLogMiddleware{}).Handle(context.Background(), &khttp.Transport{}, nil)

	// 传输层为 nil。
	var op options
	WithLoginOperation(adminV1.OperationAuthenticationServiceLogin)(&op)
	WithLogoutOperation("unused-logout-op")(&op)
	WithWriteLoginLogFunc(stub)(&op)
	mw := NewLoginAuditLogMiddleware(&op)
	mw.Handle(context.Background(), nil, nil)

	// operation 不匹配（零值 Transport 的 operation 为空串）。
	mw.Handle(context.Background(), &khttp.Transport{}, nil)

	assert.Zero(t, written, "早退分支不得写入登录审计")
}

// TestLoginAuditLogHandleLoginOpWithToken 走 server 验证登录 operation 的
// 完整字段映射：用户名取请求体（优先于令牌）、身份字段取令牌、
// IP/地理取 RemoteAddr（私网 → 局域网）、设备 ClientId 取令牌 cid、
// 成功状态、风险评分清零（成功 + 已知用户 + 已知设备 + 内网）。
func TestLoginAuditLogHandleLoginOpWithToken(t *testing.T) {
	env := newAuditServer(t)
	env.fire(nethttp.MethodPost, "/case/5", map[string]string{
		"X-Test-Operation": adminV1.OperationAuthenticationServiceLogin,
		"Content-Type":     "application/json",
		"Authorization":    "Bearer " + mintTestToken(t),
	}, `{"username":"bob","password":"x"}`, "127.0.0.1:1234")

	require.Len(t, env.capture.login, 1)
	rec := env.capture.login[0]

	assert.Equal(t, auditV1.LoginAuditLog_LOGIN, rec.GetActionType(), "登录 operation → LOGIN")
	assert.Empty(t, rec.GetMfaStatus(), "非 MFA 端点不得填充 mfa_status")
	assert.Equal(t, auditV1.LoginAuditLog_SUCCESS, rec.GetStatus())
	assert.Empty(t, rec.GetFailureReason())

	// 用户名来源优先级：请求体优先于令牌。
	assert.Equal(t, "bob", rec.GetUsername(), "Username ← 请求体（优先于令牌 sub）")
	// 令牌身份映射。
	assert.Equal(t, uint32(42), rec.GetUserId())
	assert.Equal(t, uint32(7), rec.GetTenantId())

	// IP 与地理（私网归一）。
	assert.Equal(t, "127.0.0.1", rec.GetIpAddress())
	geo := rec.GetGeoLocation()
	require.NotNil(t, geo)
	assert.Equal(t, "局域网", geo.GetCountryCode())

	// 设备信息：无 UA 头 → 空解析；ClientId ← 令牌 cid。
	di := rec.GetDeviceInfo()
	require.NotNil(t, di)
	assert.Empty(t, di.GetUserAgent())
	assert.Equal(t, auditV1.DeviceInfo_OTHER, di.GetDeviceType())
	assert.Equal(t, "test-client-id", di.GetClientId())

	// 请求 ID：无头时生成 GUID（非空）。
	assert.NotEmpty(t, rec.GetRequestId())

	// 风险评分：成功 + 已知用户 + 已知设备 + 内网 → 0 分 LOW，
	// 因素仅内网与会话缺失。
	assert.Equal(t, uint32(0), rec.GetRiskScore())
	assert.Equal(t, auditV1.LoginAuditLog_LOW, rec.GetRiskLevel())
	assert.Equal(t, []string{"INTERNAL_IP", "NO_SESSION"}, rec.GetRiskFactors())

	requireLogHashHex(t, rec.GetLogHash())
	requireDERSig(t, rec.GetSignature())
	requireTimestampNearNow(t, rec.GetCreatedAt())

	require.Len(t, env.capture.loginMeta, 1)
	assert.True(t, env.capture.loginMeta[0].Sinking)
	assert.True(t, env.capture.loginMeta[0].SystemViewer)
}

// TestLoginAuditLogHandleMFAVerifyStatus 验证 MFA 验证 operation 的
// mfa_status 语义：业务成功 → VERIFIED，业务失败 → FAILED，
// 且失败路径的 Status/FailureReason 取自中间件错误、评分含失败权重。
func TestLoginAuditLogHandleMFAVerifyStatus(t *testing.T) {
	t.Run("验证成功", func(t *testing.T) {
		env := newAuditServer(t)
		env.fire(nethttp.MethodPost, "/case/5", map[string]string{
			"X-Test-Operation": adminV1.OperationMfaServiceVerifyMFAChallenge,
		}, "", "127.0.0.1:1234")
		require.Len(t, env.capture.login, 1)
		rec := env.capture.login[0]
		assert.Equal(t, auditV1.LoginAuditLog_LOGIN, rec.GetActionType(), "MFA 验证属登录流程二阶段")
		assert.Equal(t, "VERIFIED", rec.GetMfaStatus())
		assert.Equal(t, auditV1.LoginAuditLog_SUCCESS, rec.GetStatus())
		// 评分：成功(0) + 匿名(20) + 未知设备(10) + 内网(-10) = 20 → LOW。
		assert.Equal(t, uint32(20), rec.GetRiskScore())
		assert.Equal(t, auditV1.LoginAuditLog_LOW, rec.GetRiskLevel())
		assert.Equal(t, []string{
			"ANONYMOUS_LOGIN", "INTERNAL_IP", "LOW_RISK_SCORE", "NO_SESSION", "UNKNOWN_DEVICE",
		}, rec.GetRiskFactors())
	})

	t.Run("验证失败", func(t *testing.T) {
		env := newAuditServer(t)
		env.fire(nethttp.MethodPost, "/case/5", map[string]string{
			"X-Test-Operation": adminV1.OperationMfaServiceVerifyMFAChallenge,
			"X-Test-Error":     "kratos",
		}, "", "127.0.0.1:1234")
		require.Len(t, env.capture.login, 1)
		rec := env.capture.login[0]
		assert.Equal(t, "FAILED", rec.GetMfaStatus())
		assert.Equal(t, auditV1.LoginAuditLog_FAILED, rec.GetStatus())
		assert.Equal(t, "TEST_FORBIDDEN", rec.GetFailureReason(), "FailureReason ← kratos 错误 reason")
		// 评分：失败(50) + 匿名(20) + 未知设备(10) + 内网(-10) = 70 → MEDIUM。
		assert.Equal(t, uint32(70), rec.GetRiskScore())
		assert.Equal(t, auditV1.LoginAuditLog_MEDIUM, rec.GetRiskLevel())
		assert.Equal(t, []string{
			"ANONYMOUS_LOGIN", "FAILED_LOGIN", "INTERNAL_IP",
			"MEDIUM_RISK_SCORE", "MFA_FAILED", "NO_SESSION", "UNKNOWN_DEVICE",
		}, rec.GetRiskFactors())
	})
}

// TestLoginAuditLogHandleLogoutAction 验证登出 operation → LOGOUT 动作类型，
// 其余字段按登录审计同规则映射（无体无令牌时全空来源）。
func TestLoginAuditLogHandleLogoutAction(t *testing.T) {
	env := newAuditServer(t)
	env.fire(nethttp.MethodPost, "/case/5", map[string]string{
		"X-Test-Operation": adminV1.OperationAuthenticationServiceLogout,
	}, "", "127.0.0.1:1234")

	require.Len(t, env.capture.login, 1)
	rec := env.capture.login[0]
	assert.Equal(t, auditV1.LoginAuditLog_LOGOUT, rec.GetActionType())
	assert.Equal(t, uint32(0), rec.GetUserId())
	assert.Empty(t, rec.GetUsername())
}

// TestLoginAuditLogHandleReplyHeaderFallbackUsername 验证免鉴权端点的
// 用户名兜底：请求体与令牌都取不到时，取 handler 经响应头
// X-Audit-Username 回传的挑战上下文用户名；公网 IP 计入外网因素。
func TestLoginAuditLogHandleReplyHeaderFallbackUsername(t *testing.T) {
	env := newAuditServer(t)
	env.fire(nethttp.MethodPost, "/case/5", map[string]string{
		"X-Test-Operation":      adminV1.OperationAuthenticationServiceLogin,
		"X-Test-Reply-Username": "audit-user",
	}, "", "8.8.8.8:1234")

	require.Len(t, env.capture.login, 1)
	rec := env.capture.login[0]
	assert.Equal(t, "audit-user", rec.GetUsername(), "Username ← 响应头 X-Audit-Username 兜底")
	assert.Equal(t, "8.8.8.8", rec.GetIpAddress())
	assert.Equal(t, "美国", rec.GetGeoLocation().GetCountryCode(), "公网 IP 走地理库解析")
	// 评分：成功(0) + 未知用户(10) + 未知设备(10) + 公网(0) = 20 → LOW。
	assert.Equal(t, uint32(20), rec.GetRiskScore())
	assert.Equal(t, auditV1.LoginAuditLog_LOW, rec.GetRiskLevel())
	assert.Equal(t, []string{
		"EXTERNAL_IP", "LOW_RISK_SCORE", "NO_SESSION", "UNKNOWN_DEVICE", "UNKNOWN_USER",
	}, rec.GetRiskFactors())
}

// TestLoginAuditLogHandleTokenUsernameFallback 验证请求体无用户名时
// 用户名回退到令牌 sub（用户名三级来源的第二级）。
func TestLoginAuditLogHandleTokenUsernameFallback(t *testing.T) {
	env := newAuditServer(t)
	env.fire(nethttp.MethodPost, "/case/5", map[string]string{
		"X-Test-Operation": adminV1.OperationAuthenticationServiceLogin,
		"Authorization":    "Bearer " + mintTestToken(t),
	}, "", "127.0.0.1:1234")

	require.Len(t, env.capture.login, 1)
	rec := env.capture.login[0]
	assert.Equal(t, "alice", rec.GetUsername(), "请求体无用户名时回退令牌 sub")
	assert.Equal(t, uint32(42), rec.GetUserId())
	assert.Equal(t, uint32(7), rec.GetTenantId())
}

// TestLoginAuditLogHandleNonLoginOperation 验证非登录类 operation 不进登录审计。
func TestLoginAuditLogHandleNonLoginOperation(t *testing.T) {
	env := newAuditServer(t)
	env.fire(nethttp.MethodPost, "/case/5", map[string]string{
		"X-Test-Operation": "/demo.v1.GadgetService/Update",
	}, "", "127.0.0.1:1234")
	assert.Empty(t, env.capture.login, "普通业务操作不得写入登录审计")
}

// TestLoginAuditLogHandleWriteFuncNil 验证 writeLoginLogFunc 为 nil 时
// 登录端点只跳过登录审计落库（本场景其余审计也各有跳过理由），零写入不 panic。
func TestLoginAuditLogHandleWriteFuncNil(t *testing.T) {
	env := newAuditServer(t, WithWriteLoginLogFunc(nil))
	env.fire(nethttp.MethodPost, "/case/5", map[string]string{
		"X-Test-Operation": adminV1.OperationAuthenticationServiceLogin,
	}, "", "127.0.0.1:1234")
	assert.Zero(t, env.capture.total(), "登录端点在空登录写入函数下不得有任何落库")
}

// TestLoginAuditLogHandleDirectEmptySources 直调空 Transport 覆盖
// Request()/RequestHeader() 为 nil 的全空来源路径：IP/请求 ID/用户名/设备
// 全空，评分取全空来源组合（35 → MEDIUM），并覆盖 getRequestId(nil) 分支
// 产生的空请求 ID（NO_REQUEST_ID 因素）。
func TestLoginAuditLogHandleDirectEmptySources(t *testing.T) {
	t.Run("成功", func(t *testing.T) {
		var rec *auditV1.LoginAuditLog
		var meta auditCallMeta
		var op options
		WithLoginOperation(adminV1.OperationAuthenticationServiceLogin)(&op)
		WithWriteLoginLogFunc(func(ctx context.Context, d *auditV1.LoginAuditLog) error {
			rec = d
			meta = (&auditCapture{}).metaOf(ctx)
			return nil
		})(&op)
		key, _, err := generateECDSAKeyPair()
		require.NoError(t, err)
		WithECPrivateKey(key)(&op)
		mw := NewLoginAuditLogMiddleware(&op)

		tr := &khttp.Transport{}
		ctx := transport.NewServerContext(context.Background(), tr)
		khttp.SetOperation(ctx, adminV1.OperationAuthenticationServiceLogin)
		mw.Handle(ctx, tr, nil)

		require.NotNil(t, rec, "空来源路径仍应构造并落库记录")
		assert.Equal(t, auditV1.LoginAuditLog_LOGIN, rec.GetActionType())
		assert.Empty(t, rec.GetIpAddress())
		assert.Empty(t, rec.GetRequestId(), "nil 请求的请求 ID 归一为空串")
		assert.Empty(t, rec.GetUsername())
		assert.Empty(t, rec.GetGeoLocation().GetCountryCode())
		di := rec.GetDeviceInfo()
		require.NotNil(t, di)
		assert.Empty(t, di.GetUserAgent())
		assert.Equal(t, auditV1.DeviceInfo_OTHER, di.GetDeviceType())
		assert.Equal(t, PlatformOther, di.GetPlatform())
		assert.Empty(t, di.GetClientId())
		// 评分：成功(0) + 匿名(20) + 未知设备(10) + IP 缺失(5) = 35 → MEDIUM。
		assert.Equal(t, uint32(35), rec.GetRiskScore())
		assert.Equal(t, auditV1.LoginAuditLog_MEDIUM, rec.GetRiskLevel())
		assert.Equal(t, []string{
			"ANONYMOUS_LOGIN", "IP_MISSING", "MEDIUM_RISK_SCORE",
			"NO_REQUEST_ID", "NO_SESSION", "UNKNOWN_DEVICE",
		}, rec.GetRiskFactors())
		requireLogHashHex(t, rec.GetLogHash())
		requireDERSig(t, rec.GetSignature())
		requireTimestampNearNow(t, rec.GetCreatedAt())
		assert.True(t, meta.SystemViewer, "直调落库同样必须切系统 viewer")
		assert.False(t, meta.Sinking, "直调路径无 Server 包装时无 sink 标记（由 Server 落库阶段统一植入）")
	})

	t.Run("失败错误填充", func(t *testing.T) {
		var rec *auditV1.LoginAuditLog
		var op options
		WithLoginOperation(adminV1.OperationAuthenticationServiceLogin)(&op)
		WithWriteLoginLogFunc(func(ctx context.Context, d *auditV1.LoginAuditLog) error {
			rec = d
			return nil
		})(&op)
		mw := NewLoginAuditLogMiddleware(&op)

		tr := &khttp.Transport{}
		ctx := transport.NewServerContext(context.Background(), tr)
		khttp.SetOperation(ctx, adminV1.OperationAuthenticationServiceLogin)
		mw.Handle(ctx, tr, kerrors.New(403, "TEST_FORBIDDEN", "forbidden for test"))

		require.NotNil(t, rec)
		assert.Equal(t, auditV1.LoginAuditLog_FAILED, rec.GetStatus())
		assert.Equal(t, "TEST_FORBIDDEN", rec.GetFailureReason())
		// 评分：失败(50) + 匿名(20) + 未知设备(10) + IP 缺失(5) = 85 → HIGH。
		assert.Equal(t, uint32(85), rec.GetRiskScore())
		assert.Equal(t, auditV1.LoginAuditLog_HIGH, rec.GetRiskLevel())
	})
}

// TestLoginAuditLogHashLogAndSignatureEdges 直调覆盖 nil 边界：
// nil 记录返回空哈希/nil 签名；私钥缺失时签名必须为 nil 而非半成品。
func TestLoginAuditLogHashLogAndSignatureEdges(t *testing.T) {
	mwNoKey := NewLoginAuditLogMiddleware(&options{})
	assert.Equal(t, "", mwNoKey.hashLog(nil))
	assert.Nil(t, mwNoKey.signature(nil))
	assert.Nil(t, mwNoKey.signature(&auditV1.LoginAuditLog{}), "无私钥时签名必须为 nil")

	key, _, err := generateECDSAKeyPair()
	require.NoError(t, err)
	mwWithKey := NewLoginAuditLogMiddleware(&options{ecPrivateKey: key})
	assert.Equal(t, "", mwWithKey.hashLog(nil))
	assert.Nil(t, mwWithKey.signature(nil))
}

// ---------------------------------------------------------------------------
// 风险评分 / 等级 / 风险因素：纯函数表驱动
// ---------------------------------------------------------------------------

// mkScoreLog 构造评分输入：设备信息按 clientId 生成（nil 表示缺省设备）。
func mkScoreLog(
	status auditV1.LoginAuditLog_Status,
	uid uint32,
	username string,
	di *auditV1.DeviceInfo,
	ip string,
) *auditV1.LoginAuditLog {
	return &auditV1.LoginAuditLog{
		Status:     trans.Ptr(status),
		UserId:     trans.Ptr(uid),
		Username:   trans.Ptr(username),
		DeviceInfo: di,
		IpAddress:  trans.Ptr(ip),
	}
}

// TestComputeRiskScore 表驱动验证各启发式因子的加权与 0 下限截断：
// 失败 +50；无用户 ID 时（有用户名 +10 / 匿名 +20）；设备缺失或无
// ClientId +10；IP 缺失 +5 / 内网 -10；总分截断到 [0,100]。
// 注：全部因子叠加的理论上限为 85，>100 截断分支不可达（由输入域决定）。
func TestComputeRiskScore(t *testing.T) {
	l := &LoginAuditLogMiddleware{}
	cases := []struct {
		name string
		log  *auditV1.LoginAuditLog
		want uint32
	}{
		{"nil日志", nil, 0},
		{"成功且身份设备齐备且内网", mkScoreLog(auditV1.LoginAuditLog_SUCCESS, 42, "alice", &auditV1.DeviceInfo{ClientId: trans.Ptr("c")}, "127.0.0.1"), 0},
		{"失败且匿名且无设备且缺IP", mkScoreLog(auditV1.LoginAuditLog_FAILED, 0, "", nil, ""), 85},
		{"失败且未知用户名且无ClientId且公网", mkScoreLog(auditV1.LoginAuditLog_FAILED, 0, "bob", &auditV1.DeviceInfo{}, "8.8.8.8"), 70},
		{"成功但设备缺失且公网", mkScoreLog(auditV1.LoginAuditLog_SUCCESS, 42, "alice", nil, "192.0.2.9"), 10},
		{"失败但身份设备齐备且内网", mkScoreLog(auditV1.LoginAuditLog_FAILED, 42, "alice", &auditV1.DeviceInfo{ClientId: trans.Ptr("c")}, "10.0.0.1"), 40},
		{"成功但匿名且缺IP但设备齐备", mkScoreLog(auditV1.LoginAuditLog_SUCCESS, 0, "", &auditV1.DeviceInfo{ClientId: trans.Ptr("c")}, ""), 25},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, l.computeRiskScore(tc.log))
		})
	}
}

// TestLevelFromScore 表驱动验证评分到等级的阈值映射：
// 0–30 LOW / 31–70 MEDIUM / 71–100 HIGH。
func TestLevelFromScore(t *testing.T) {
	l := &LoginAuditLogMiddleware{}
	cases := []struct {
		score uint32
		want  auditV1.LoginAuditLog_RiskLevel
	}{
		{0, auditV1.LoginAuditLog_LOW},
		{1, auditV1.LoginAuditLog_LOW},
		{30, auditV1.LoginAuditLog_LOW},
		{31, auditV1.LoginAuditLog_MEDIUM},
		{70, auditV1.LoginAuditLog_MEDIUM},
		{71, auditV1.LoginAuditLog_HIGH},
		{100, auditV1.LoginAuditLog_HIGH},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("score=%d", tc.score), func(t *testing.T) {
			assert.Equal(t, tc.want, l.levelFromScore(tc.score))
		})
	}
}

// mkFactorLog 构造风险因素输入：显式指定会话/请求 ID、mfa 状态、
// 失败原因与风险分（因素推导读记录上的既有字段，不重算评分）。
func mkFactorLog(
	status auditV1.LoginAuditLog_Status,
	uid uint32,
	username, clientId, ip, mfa, failure, session, requestID string,
	score uint32,
) *auditV1.LoginAuditLog {
	return &auditV1.LoginAuditLog{
		Status:        trans.Ptr(status),
		UserId:        trans.Ptr(uid),
		Username:      trans.Ptr(username),
		DeviceInfo:    &auditV1.DeviceInfo{ClientId: trans.Ptr(clientId)},
		IpAddress:     trans.Ptr(ip),
		MfaStatus:     trans.Ptr(mfa),
		FailureReason: trans.Ptr(failure),
		SessionId:     trans.Ptr(session),
		RequestId:     trans.Ptr(requestID),
		RiskScore:     trans.Ptr(score),
	}
}

// TestComputeRiskFactors 表驱动验证风险因素的启发式推导：
// 各因素独立触发、按字典序排序输出、无匹配时输出空列表。
// 该列表是下游风险告警的唯一输入，顺序与内容都必须稳定。
func TestComputeRiskFactors(t *testing.T) {
	l := &LoginAuditLogMiddleware{}
	clean := func(mutate func(la *auditV1.LoginAuditLog)) *auditV1.LoginAuditLog {
		// 基线：成功 + 已知用户 + 已知设备 + 内网 + 会话/请求 ID 齐备 + 0 分。
		la := mkFactorLog(auditV1.LoginAuditLog_SUCCESS, 42, "alice", "c",
			"127.0.0.1", "", "", "sess-1", "req-1", 0)
		if mutate != nil {
			mutate(la)
		}
		return la
	}
	cases := []struct {
		name string
		log  *auditV1.LoginAuditLog
		want []string
	}{
		{"nil日志", nil, nil},
		{"基线仅内网因素", clean(nil), []string{"INTERNAL_IP"}},
		{"失败叠加全缺失", mkFactorLog(auditV1.LoginAuditLog_FAILED, 0, "", "", "", "", "", "", "", 85),
			[]string{"ANONYMOUS_LOGIN", "FAILED_LOGIN", "HIGH_RISK_SCORE", "IP_MISSING", "NO_REQUEST_ID", "NO_SESSION", "UNKNOWN_DEVICE"}},
		{"mfa失败与未验证双因素", func() *auditV1.LoginAuditLog {
			return clean(func(la *auditV1.LoginAuditLog) {
				la.IpAddress = trans.Ptr("8.8.8.8")
				la.MfaStatus = trans.Ptr("failed unverified")
			})
		}(), []string{"EXTERNAL_IP", "MFA_FAILED", "MFA_UNVERIFIED"}},
		{"mfa已验证不产生因素", func() *auditV1.LoginAuditLog {
			return clean(func(la *auditV1.LoginAuditLog) {
				la.MfaStatus = trans.Ptr("VERIFIED")
			})
		}(), []string{"INTERNAL_IP"}},
		{"密码类失败原因", func() *auditV1.LoginAuditLog {
			return clean(func(la *auditV1.LoginAuditLog) {
				la.IpAddress = trans.Ptr("8.8.8.8")
				la.FailureReason = trans.Ptr("wrong password entered")
			})
		}(), []string{"EXTERNAL_IP", "PASSWORD_FAILURE"}},
		{"mfa类失败原因", func() *auditV1.LoginAuditLog {
			return clean(func(la *auditV1.LoginAuditLog) {
				la.IpAddress = trans.Ptr("8.8.8.8")
				la.FailureReason = trans.Ptr("mfa code incorrect")
			})
		}(), []string{"EXTERNAL_IP", "MFA_FAILURE_REASON", "PASSWORD_FAILURE"}},
		{"无关失败原因不产生因素", func() *auditV1.LoginAuditLog {
			return clean(func(la *auditV1.LoginAuditLog) {
				la.IpAddress = trans.Ptr("8.8.8.8")
				la.FailureReason = trans.Ptr("account banned")
			})
		}(), []string{"EXTERNAL_IP"}},
		{"评分30低风险因素", clean(func(la *auditV1.LoginAuditLog) { la.RiskScore = trans.Ptr(uint32(30)) }),
			[]string{"INTERNAL_IP", "LOW_RISK_SCORE"}},
		{"评分31中风险因素", clean(func(la *auditV1.LoginAuditLog) { la.RiskScore = trans.Ptr(uint32(31)) }),
			[]string{"INTERNAL_IP", "MEDIUM_RISK_SCORE"}},
		{"评分71高风险因素", clean(func(la *auditV1.LoginAuditLog) { la.RiskScore = trans.Ptr(uint32(71)) }),
			[]string{"HIGH_RISK_SCORE", "INTERNAL_IP"}},
		{"未知用户因素", func() *auditV1.LoginAuditLog {
			return clean(func(la *auditV1.LoginAuditLog) {
				la.UserId = trans.Ptr(uint32(0))
				la.Username = trans.Ptr("bob")
				la.IpAddress = trans.Ptr("8.8.8.8")
				la.RiskScore = trans.Ptr(uint32(30))
			})
		}(), []string{"EXTERNAL_IP", "LOW_RISK_SCORE", "UNKNOWN_USER"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, l.computeRiskFactors(tc.log))
		})
	}
}
