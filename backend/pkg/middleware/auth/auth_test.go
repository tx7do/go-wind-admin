// 本文件对 auth.Server 中间件做白盒测试：
//
//	拒绝路径：无 transport、无/坏 Bearer 头、未配置令牌检查器、无效令牌、租户访问检查拒绝；
//	注入路径：有效令牌后按选项注入 Ent ViewerContext、OperatorMetadata（client 侧元数据）、
//	  authz AuthClaims，以及 OperatorId/TenantId 的请求字段回填；
//	门控路径：租户访问检查仅对 tenantId>0 的租户用户生效，平台管理员直接放行。
//
// transport 用 map 底座的假件（非 *http.Transport，因此租户检查走 tr.Operation() 分支）；
// 令牌检查器用包内提供的 NewAccessTokenCheckerFromFuncs 桩。
package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-kratos/kratos/v2/encoding"
	kmeta "github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/transport"

	"github.com/tx7do/go-crud/viewer"
	"github.com/tx7do/go-utils/trans"
	authzEngine "github.com/tx7do/kratos-authz/engine"
	authzMw "github.com/tx7do/kratos-authz/middleware"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

// ----------------------------------------------------------------------------
// 假件
// ----------------------------------------------------------------------------

// fakeHeader map 底座的 transport.Header 假件。
type fakeHeader map[string]string

func (h fakeHeader) Get(key string) string { return h[key] }
func (h fakeHeader) Set(key, value string) { h[key] = value }
func (h fakeHeader) Add(key, value string) { h[key] = value }
func (h fakeHeader) Keys() []string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	return keys
}
func (h fakeHeader) Values(key string) []string {
	if v, ok := h[key]; ok {
		return []string{v}
	}
	return nil
}

// fakeTransporter kratos transport.Transporter 假件。
type fakeTransporter struct {
	hdr fakeHeader
	op  string
}

func (t *fakeTransporter) Kind() transport.Kind            { return transport.KindHTTP }
func (t *fakeTransporter) Endpoint() string                { return "fake://auth-test" }
func (t *fakeTransporter) Operation() string               { return t.op }
func (t *fakeTransporter) RequestHeader() transport.Header { return t.hdr }
func (t *fakeTransporter) ReplyHeader() transport.Header   { return t.hdr }

// newTransportCtx 构造带假 transporter 与指定 Authorization 头的服务端 transport 上下文。
func newTransportCtx(authzHeader string) context.Context {
	ft := &fakeTransporter{hdr: fakeHeader{}, op: "/fake.Test/Op"}
	if authzHeader != "" {
		ft.hdr.Set("Authorization", authzHeader)
	}
	return transport.NewServerContext(context.Background(), ft)
}

// tenantCheckerStub 记录调用参数的租户访问检查器桩，可注入返回错误。
type tenantCheckerStub struct {
	mu    sync.Mutex
	calls []tenantAccessCall
	err   error
}

type tenantAccessCall struct {
	tenantId uint32
	path     string
	method   string
}

func (s *tenantCheckerStub) CheckTenantAccess(_ context.Context, tenantId uint32, path, method string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, tenantAccessCall{tenantId: tenantId, path: path, method: method})
	return s.err
}

func (s *tenantCheckerStub) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

// validChecker 返回一个恒判有效并回放指定 payload 的令牌检查器。
func validChecker(payload *authenticationV1.UserTokenPayload) Option {
	return WithAccessTokenCheckerFromFuncs(
		func(_ context.Context, _ string, _ bool) (bool, *authenticationV1.UserTokenPayload) {
			return true, payload
		},
		nil,
	)
}

// runAuthMiddleware 以给定选项构造 auth.Server 中间件并执行一次请求。
// handler 探针只记录是否到达与收到的 ctx。
func runAuthMiddleware(t *testing.T, ctx context.Context, req interface{}, opts ...Option) (reached bool, handlerCtx context.Context, err error) {
	t.Helper()
	h := func(c context.Context, r interface{}) (interface{}, error) {
		reached = true
		handlerCtx = c
		return r, nil
	}
	wrapped := Server(opts...)(h)
	_, err = wrapped(ctx, req)
	return reached, handlerCtx, err
}

// ----------------------------------------------------------------------------
// 拒绝路径
// ----------------------------------------------------------------------------

// TestServer_NoTransport_Rejected 无 transport 的上下文必须被 ErrWrongContext 拒绝，
// 且请求不得到达业务 handler。
func TestServer_NoTransport_Rejected(t *testing.T) {
	reached, _, err := runAuthMiddleware(t, context.Background(), nil)
	require.Error(t, err)
	require.Equal(t, ErrWrongContext, err)
	require.False(t, reached)
}

// TestServer_BearerToken_MissingOrMalformed Bearer 头缺失、scheme 错误、无令牌值
// 三种情形都必须以 ErrMissingBearerToken 拒绝。
func TestServer_BearerToken_MissingOrMalformed(t *testing.T) {
	for name, header := range map[string]string{
		"无 Authorization 头": "",
		"非 Bearer scheme":   "Basic sometoken",
		"Bearer 后无令牌":       "Bearer",
	} {
		t.Run(name, func(t *testing.T) {
			reached, _, err := runAuthMiddleware(t, newTransportCtx(header), nil)
			require.Error(t, err)
			require.Equal(t, ErrMissingBearerToken, err)
			require.False(t, reached)
		})
	}
}

// TestServer_CheckerNotConfigured 配置了 Bearer 头但未注入令牌检查器时，
// 必须以 ErrAccessTokenCheckerNotConfigured 拒绝（fail-closed，而非放行）。
func TestServer_CheckerNotConfigured(t *testing.T) {
	reached, _, err := runAuthMiddleware(t, newTransportCtx("Bearer some-token"), nil)
	require.Error(t, err)
	require.Equal(t, ErrAccessTokenCheckerNotConfigured, err)
	require.False(t, reached)
}

// TestServer_InvalidToken_Rejected 检查器判无效的令牌必须以 ErrAccessTokenExpired 拒绝。
func TestServer_InvalidToken_Rejected(t *testing.T) {
	invalid := WithAccessTokenCheckerFromFuncs(
		func(_ context.Context, _ string, _ bool) (bool, *authenticationV1.UserTokenPayload) {
			return false, nil
		},
		nil,
	)
	reached, _, err := runAuthMiddleware(t, newTransportCtx("Bearer stale-token"), nil, invalid)
	require.Error(t, err)
	require.Equal(t, ErrAccessTokenExpired, err)
	require.False(t, reached)
}

// ----------------------------------------------------------------------------
// 注入路径
// ----------------------------------------------------------------------------

// TestServer_ValidToken_ContextInjection 有效令牌 + 默认注入选项下，
// Ent ViewerContext / OperatorMetadata / authz AuthClaims 必须按令牌 payload 注入。
func TestServer_ValidToken_ContextInjection(t *testing.T) {
	payload := &authenticationV1.UserTokenPayload{
		UserId:     42,
		OrgUnitId:  trans.Ptr(uint32(9)),
		Roles:      []string{"ROLE_ADMIN"},
		DataScopes: []identityV1.DataScope{identityV1.DataScope_ALL},
	}
	reached, handlerCtx, err := runAuthMiddleware(t, newTransportCtx("Bearer good-token"), nil, validChecker(payload))
	require.NoError(t, err)
	require.True(t, reached)

	// viewer 注入断言：UserViewer 携带 payload 的身份三元组（TenantId 未设 → 0）。
	vc, ok := viewer.FromContext(handlerCtx)
	require.True(t, ok)
	require.Equal(t, uint64(42), vc.UserID())
	require.Equal(t, uint64(0), vc.TenantID())
	require.Equal(t, uint64(9), vc.OrgUnitID())
	require.Len(t, vc.DataScope(), 1)

	// metadata 注入断言：client 侧元数据携带 base64(proto) 编码的 OperatorMetadata，
	// 字段与 payload 一致。
	md, ok := kmeta.FromClientContext(handlerCtx)
	require.True(t, ok)
	raw := md.Get("x-md-global-operator")
	require.NotEmpty(t, raw)
	b, derr := base64.RawStdEncoding.DecodeString(raw)
	require.NoError(t, derr)
	info := &authenticationV1.OperatorMetadata{}
	require.NoError(t, encoding.GetCodec("proto").Unmarshal(b, info))
	require.Equal(t, uint64(42), info.UserId)
	require.Equal(t, uint64(9), info.OrgUnitId)

	// authz 注入断言：AuthClaims 以 payload 角色与假 transport 的操作名构造，
	// 非_http.Transport 假件下 Resource 取 Operation()，Action 恒为 "ANY"。
	claims, ok := authzMw.FromContext(handlerCtx)
	require.True(t, ok)
	require.Equal(t, []string{"ROLE_ADMIN"}, *claims.Subjects)
	require.Equal(t, authzEngine.Action("ANY"), *claims.Action)
	require.Equal(t, authzEngine.Resource("/fake.Test/Op"), *claims.Resource)
}

// TestServer_InjectionsDisabled 显式关闭 Ent / metadata / authz 注入时，
// 请求照常放行，但三类上下文均不得出现。
func TestServer_InjectionsDisabled(t *testing.T) {
	payload := &authenticationV1.UserTokenPayload{UserId: 1}
	reached, handlerCtx, err := runAuthMiddleware(
		t, newTransportCtx("Bearer good-token"), nil,
		validChecker(payload),
		WithInjectEnt(false),
		WithInjectMetadata(false),
		WithEnableAuthority(false),
	)
	require.NoError(t, err)
	require.True(t, reached)

	_, ok := viewer.FromContext(handlerCtx)
	require.False(t, ok, "viewer 不应被注入")
	_, ok = kmeta.FromClientContext(handlerCtx)
	require.False(t, ok, "operator metadata 不应被注入")
	_, ok = authzMw.FromContext(handlerCtx)
	require.False(t, ok, "authz claims 不应被注入")
}

// ----------------------------------------------------------------------------
// 租户访问检查门控
// ----------------------------------------------------------------------------

// TestServer_TenantCheckerGating 租户访问检查的门控语义：
// 平台管理员（tenantId=0）不触发检查；租户用户触发检查且路径取自 Operation()
// （非 http transport 假件下 method 为空）；检查器拒绝则请求被终止。
func TestServer_TenantCheckerGating(t *testing.T) {
	// 平台管理员：检查器不被调用。
	platform := &tenantCheckerStub{}
	reached, _, err := runAuthMiddleware(
		t, newTransportCtx("Bearer good-token"), nil,
		validChecker(&authenticationV1.UserTokenPayload{UserId: 1}),
		WithTenantAccessChecker(platform),
	)
	require.NoError(t, err)
	require.True(t, reached)
	require.Zero(t, platform.callCount(), "平台管理员不得触发租户访问检查")

	// 租户用户：检查器被调用，参数为（租户ID, Operation(), 空方法）。
	tenant := &tenantCheckerStub{}
	reached, _, err = runAuthMiddleware(
		t, newTransportCtx("Bearer good-token"), nil,
		validChecker(&authenticationV1.UserTokenPayload{
			UserId:   2,
			TenantId: trans.Ptr(uint32(7)),
		}),
		WithTenantAccessChecker(tenant),
	)
	require.NoError(t, err)
	require.True(t, reached)
	require.Equal(t, 1, tenant.callCount())
	tenant.mu.Lock()
	call := tenant.calls[0]
	tenant.mu.Unlock()
	require.Equal(t, uint32(7), call.tenantId)
	require.Equal(t, "/fake.Test/Op", call.path)
	require.Equal(t, "", call.method)

	// 检查器拒绝：请求被终止，不到达 handler。
	deny := &tenantCheckerStub{err: errors.New("tenant frozen")}
	reached, _, err = runAuthMiddleware(
		t, newTransportCtx("Bearer good-token"), nil,
		validChecker(&authenticationV1.UserTokenPayload{
			UserId:   2,
			TenantId: trans.Ptr(uint32(7)),
		}),
		WithTenantAccessChecker(deny),
	)
	require.EqualError(t, err, "tenant frozen")
	require.False(t, reached)
}

// ----------------------------------------------------------------------------
// 请求字段回填（OperatorId / TenantId）
// ----------------------------------------------------------------------------

// TestServer_OperatorIdInjection 开启 injectOperatorId 时，
// 带可选 OperatorId 字段的请求必须被回填为令牌 payload 的 UserId。
func TestServer_OperatorIdInjection(t *testing.T) {
	payload := &authenticationV1.UserTokenPayload{UserId: 77}
	req := &permissionV1.DeleteMenuRequest{}
	reached, _, err := runAuthMiddleware(
		t, newTransportCtx("Bearer good-token"), req,
		validChecker(payload),
		WithInjectOperatorId(true),
	)
	require.NoError(t, err)
	require.True(t, reached)
	require.NotNil(t, req.OperatorId, "OperatorId 应回填为令牌用户ID")
	require.Equal(t, uint32(77), *req.OperatorId)
}

// tenantCarrier 带 TenantId 指针字段的本地载体，
// 用于验证 setRequestTenantId 的字段回填（请求壳无此字段类型，用反射语义等价的本地结构体）。
type tenantCarrier struct {
	TenantId *uint32
}

// TestServer_TenantIdInjection 开启 injectTenantId 时，
// 带 TenantId 指针字段的请求必须被回填为令牌 payload 的 TenantId。
func TestServer_TenantIdInjection(t *testing.T) {
	payload := &authenticationV1.UserTokenPayload{
		UserId:   2,
		TenantId: trans.Ptr(uint32(3)),
	}
	req := &tenantCarrier{}
	reached, _, err := runAuthMiddleware(
		t, newTransportCtx("Bearer good-token"), req,
		validChecker(payload),
		WithInjectTenantId(true),
	)
	require.NoError(t, err)
	require.True(t, reached)
	require.NotNil(t, req.TenantId, "TenantId 应回填为令牌租户ID")
	require.Equal(t, uint32(3), *req.TenantId)
}

// TestSetRequestIds_NilAndNoMatch 直接单元测试回填函数：
// req 为 nil 返回 ErrInvalidRequest；无匹配字段时静默跳过返回 nil。
func TestSetRequestIds_NilAndNoMatch(t *testing.T) {
	payload := &authenticationV1.UserTokenPayload{UserId: 1}

	require.Equal(t, ErrInvalidRequest, setRequestOperationId(nil, payload))
	require.Equal(t, ErrInvalidRequest, setRequestTenantId(nil, payload))

	require.NoError(t, setRequestOperationId(&struct{}{}, payload))
	require.NoError(t, setRequestTenantId(&struct{}{}, payload))
}

// ----------------------------------------------------------------------------
// 令牌 payload 上下文与组合检查器
// ----------------------------------------------------------------------------

// TestAuthClaimsContext_RoundTrip NewContext/FromContext 往返与缺失语义。
func TestAuthClaimsContext_RoundTrip(t *testing.T) {
	payload := &authenticationV1.UserTokenPayload{UserId: 5}
	ctx := NewContext(context.Background(), payload)
	got, err := FromContext(ctx)
	require.NoError(t, err)
	require.Same(t, payload, got)

	got, err = FromContext(context.Background())
	require.Nil(t, got)
	require.Equal(t, ErrMissingJwtToken, err)
}

// TestComposedChecker_NilFuncDefaults 组合检查器在未提供函数时的默认值：
// 令牌默认有效、默认不阻止。
func TestComposedChecker_NilFuncDefaults(t *testing.T) {
	c := NewAccessTokenCheckerFromFuncs(nil, nil)
	valid, payload := c.IsValidAccessToken(context.Background(), "x", false)
	require.True(t, valid)
	require.Nil(t, payload)
	require.False(t, c.IsBlockedAccessToken(context.Background(), "x"))
}

// TestOptionSetters 全部选项构造器可无 panic 地构造中间件。
func TestOptionSetters(t *testing.T) {
	require.NotPanics(t, func() {
		_ = Server(
			WithLogger(bLogger.NopLogger()),
			WithAccessTokenChecker(NewAccessTokenCheckerFromFuncs(nil, nil)),
			WithTenantAccessChecker(nil),
			WithEnableCheckRefreshTokenExpiration(true),
			WithEnableCheckScopes(true),
			WithInjectOperatorId(false),
			WithInjectTenantId(false),
			WithInjectEnt(true),
			WithInjectMetadata(true),
			WithEnableAuthority(true),
		)
	})
}
