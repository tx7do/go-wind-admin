// 本文件补 auth.Server 中间件既有测试（auth_test.go）未覆盖的分支与热路径基准：
//   - Bearer 解析细节：scheme 大小写不敏感（EqualFold）、双空格后的令牌原样传递、
//     checker 收到令牌原文与 skipRedis=false 硬编码；
//   - checker 返回 (true, nil payload) 的现状：无守卫直接放行，claims 为 typed-nil
//     （先钉住现状，若后续加守卫需同步翻转断言）；
//   - 负数 tenantId 跳过租户闸门（>0 条件）；
//   - injectEnt 的 otel TraceID 注入：ctx 带采样 span context 时 viewer 携带 TraceID；
//   - 基准：Server 整链（注入组合分档）、processAuthz、反射回填。
package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-kratos/kratos/v2/transport"

	"github.com/tx7do/go-crud/viewer"
	"github.com/tx7do/go-utils/trans"
	authzMw "github.com/tx7do/kratos-authz/middleware"

	"go.opentelemetry.io/otel/trace"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

// TestServer_BearerSchemeCaseInsensitive Bearer scheme 大小写不敏感（引擎 EqualFold），
// 小写 "bearer" 同样接受。
func TestServer_BearerSchemeCaseInsensitive(t *testing.T) {
	payload := &authenticationV1.UserTokenPayload{UserId: 42}
	var gotToken string
	checker := WithAccessTokenCheckerFromFuncs(
		func(_ context.Context, token string, _ bool) (bool, *authenticationV1.UserTokenPayload) {
			gotToken = token
			return true, payload
		},
		nil,
	)
	reached, _, err := runAuthMiddleware(t, newTransportCtx("bearer lowercase-token"), nil, checker)
	require.NoError(t, err)
	require.True(t, reached)
	require.Equal(t, "lowercase-token", gotToken)
}

// TestServer_BearerTokenVerbatimToChecker 中间件必须把令牌原文（含异常空白）
// 与硬编码 skipRedis=false 传给 checker——checker 侧的 Redis 短路开关
// 由中间件决定，不接受请求侧干预。
func TestServer_BearerTokenVerbatimToChecker(t *testing.T) {
	var gotToken string
	var gotSkipRedis bool
	checker := WithAccessTokenCheckerFromFuncs(
		func(_ context.Context, token string, skipRedis bool) (bool, *authenticationV1.UserTokenPayload) {
			gotToken, gotSkipRedis = token, skipRedis
			return true, &authenticationV1.UserTokenPayload{UserId: 1}
		},
		nil,
	)
	// 双空格：SplitN(2) 后令牌带前导空格原样传递（现状钉住——不 trim）。
	_, _, err := runAuthMiddleware(t, newTransportCtx("Bearer   spaced-token"), nil, checker)
	require.NoError(t, err)
	require.Equal(t, "  spaced-token", gotToken)
	require.False(t, gotSkipRedis, "middleware must hardcode skipRedis=false")
}

// TestServer_ValidTokenNilPayload_PinnedCurrentBehavior checker 返回 (true, nil payload)
// 时中间件无守卫直接放行：claims 为 typed-nil，FromContext 返回 (nil, nil)。
// 这是现状记录——若后续在中间件加 payload 非空守卫，请把本断言改为拒绝路径。
func TestServer_ValidTokenNilPayload_PinnedCurrentBehavior(t *testing.T) {
	checker := WithAccessTokenCheckerFromFuncs(
		func(context.Context, string, bool) (bool, *authenticationV1.UserTokenPayload) {
			return true, nil
		},
		nil,
	)
	reached, handlerCtx, err := runAuthMiddleware(t, newTransportCtx("Bearer nil-payload-token"), nil, checker)
	require.NoError(t, err, "current implementation lets a valid-but-nil payload through")
	require.True(t, reached)

	got, err := FromContext(handlerCtx)
	require.NoError(t, err)
	require.Nil(t, got)
}

// TestServer_NegativeTenantIdSkipsTenantGate 负数 tenantId 同样跳过租户闸门
// （仅 tenantId>0 才检查）——负值不在合法租户域，钉住该放行语义。
func TestServer_NegativeTenantIdSkipsTenantGate(t *testing.T) {
	stub := &tenantCheckerStub{}
	reached, _, err := runAuthMiddleware(
		t, newTransportCtx("Bearer good-token"), nil,
		validChecker(&authenticationV1.UserTokenPayload{UserId: 1, TenantId: trans.Ptr(int64(-5))}),
		WithTenantAccessChecker(stub),
	)
	require.NoError(t, err)
	require.True(t, reached)
	require.Zero(t, stub.callCount(), "negative tenantId must not trigger the tenant gate")
}

// TestServer_TenantGateRunsAfterInjections 租户闸在三类注入之后执行：
// 拒绝时上下文已注入但到不了 handler（调用方拿到的是闸门错误而非注入错误）。
func TestServer_TenantGateRunsAfterInjections(t *testing.T) {
	deny := &tenantCheckerStub{err: context.Canceled}
	reached, handlerCtx, err := runAuthMiddleware(
		t, newTransportCtx("Bearer good-token"), nil,
		validChecker(&authenticationV1.UserTokenPayload{UserId: 2, TenantId: trans.Ptr(int64(7))}),
		WithTenantAccessChecker(deny),
	)
	require.Error(t, err)
	require.False(t, reached)
	// handlerCtx 为 nil 证明到不了 handler；注入本身不报错（错误来自闸门）。
	require.Nil(t, handlerCtx)
	require.ErrorIs(t, err, context.Canceled)
}

// TestServer_TraceIDInjectedToViewer injectEnt 时 otel span context 的 TraceID
// 必须透传进 viewer；无 span 时空串。
func TestServer_TraceIDInjectedToViewer(t *testing.T) {
	payload := &authenticationV1.UserTokenPayload{UserId: 9}

	// 带合法采样 span context 的请求。
	tid, terr := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	require.NoError(t, terr)
	sid, serr := trace.SpanIDFromHex("0102030405060708")
	require.NoError(t, serr)
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: trace.FlagsSampled,
	})

	ctx := transport.NewServerContext(
		trace.ContextWithSpanContext(context.Background(), sc),
		&fakeTransporter{hdr: fakeHeader{"Authorization": "Bearer good-token"}, op: "/fake.Test/Op"},
	)
	reached, handlerCtx, err := runAuthMiddleware(t, ctx, nil, validChecker(payload))
	require.NoError(t, err)
	require.True(t, reached)
	vc, ok := viewer.FromContext(handlerCtx)
	require.True(t, ok)
	require.Equal(t, "0102030405060708090a0b0c0d0e0f10", vc.TraceID(), "trace id must be carried into the viewer")

	// 无 span：TraceID 为空串。
	reached, handlerCtx, err = runAuthMiddleware(t, newTransportCtx("Bearer good-token"), nil, validChecker(payload))
	require.NoError(t, err)
	require.True(t, reached)
	vc, ok = viewer.FromContext(handlerCtx)
	require.True(t, ok)
	require.Empty(t, vc.TraceID())
}

// TestServer_DataScopesCarriedToViewer payload 的数据范围三元组必须完整进 viewer。
func TestServer_DataScopesCarriedToViewer(t *testing.T) {
	payload := &authenticationV1.UserTokenPayload{
		UserId:            11,
		TenantId:          trans.Ptr(int64(3)),
		OrgUnitId:         trans.Ptr(int64(6)),
		DataScope:         trans.Ptr(identityV1.DataScope_UNIT_AND_CHILD),
		DataScopes:        []identityV1.DataScope{identityV1.DataScope_UNIT_AND_CHILD, identityV1.DataScope_SELF},
		DataScopeUnitIds:  []int64{21, 22},
	}
	reached, handlerCtx, err := runAuthMiddleware(t, newTransportCtx("Bearer good-token"), nil, validChecker(payload))
	require.NoError(t, err)
	require.True(t, reached)
	vc, ok := viewer.FromContext(handlerCtx)
	require.True(t, ok)
	require.Equal(t, int64(11), vc.UserID())
	require.Equal(t, int64(3), vc.TenantID())
	require.Equal(t, int64(6), vc.OrgUnitID())
	require.Len(t, vc.DataScope(), 2)
}

// TestProcessAuthzContext 从 processAuthz 产出的 ctx 中取 AuthClaims 的一致性
// （非 http 假件：Resource=Operation()，Action=ANY）。
func TestProcessAuthzContext(t *testing.T) {
	payload := &authenticationV1.UserTokenPayload{UserId: 1, Roles: []string{"r1", "r2"}}
	ctx, err := processAuthz(newTransportCtx("Bearer x"), &fakeTransporter{op: "/op.Name/Method"}, payload)
	require.NoError(t, err)
	claims, ok := authzMw.FromContext(ctx)
	require.True(t, ok)
	require.Equal(t, []string{"r1", "r2"}, *claims.Subjects)
	require.Equal(t, "ANY", string(*claims.Action))
	require.Equal(t, "/op.Name/Method", string(*claims.Resource))
}

// --- 基准：认证链路热路径 ---

// benchPayload 构造带全量字段的令牌 payload（贴近生产注入成本）。
func benchPayload() *authenticationV1.UserTokenPayload {
	return &authenticationV1.UserTokenPayload{
		UserId:           42,
		TenantId:         trans.Ptr(int64(3)),
		OrgUnitId:        trans.Ptr(int64(9)),
		Roles:            []string{"ROLE_ADMIN", "ROLE_OPERATOR"},
		DataScope:        trans.Ptr(identityV1.DataScope_UNIT_AND_CHILD),
		DataScopes:       []identityV1.DataScope{identityV1.DataScope_UNIT_AND_CHILD, identityV1.DataScope_SELF},
		DataScopeUnitIds: []int64{21, 22, 23},
	}
}

func benchCtx() context.Context {
	return newTransportCtx("Bearer bench-token")
}

// BenchmarkServer_FullChain 认证中间件整链一次请求，按注入组合分档：
// 全开（生产默认）最重——每请求一次 proto marshal + base64 + viewer 构造。
func BenchmarkServer_FullChain(b *testing.B) {
	cases := []struct {
		name string
		opts []Option
	}{
		{"all-injections-on", nil},
		{"metadata-off", []Option{WithInjectMetadata(false)}},
		{"ent-off", []Option{WithInjectEnt(false)}},
		{"all-injections-off", []Option{
			WithInjectMetadata(false), WithInjectEnt(false), WithEnableAuthority(false),
		}},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			noop := func(c context.Context, r interface{}) (interface{}, error) { return r, nil }
			mw := Server(append(tc.opts, validChecker(benchPayload()))...)(noop)
			ctx := benchCtx()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := mw(ctx, nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkServer_WithTenantGate 带租户闸门的整链（tenantId>0 生产路径）。
func BenchmarkServer_WithTenantGate(b *testing.B) {
	noopChecker := &tenantCheckerStub{}
	noop := func(c context.Context, r interface{}) (interface{}, error) { return r, nil }
	mw := Server(
		validChecker(benchPayload()),
		WithTenantAccessChecker(noopChecker),
	)(noop)
	ctx := benchCtx()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := mw(ctx, nil); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkProcessAuthz 每请求的 AuthClaims 构造。
func BenchmarkProcessAuthz(b *testing.B) {
	tr := &fakeTransporter{op: "/admin.v1.UserService/GetUser"}
	payload := benchPayload()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := processAuthz(context.Background(), tr, payload); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSetRequestIds 反射回填（开启注入时每请求两次 FieldByName）。
func BenchmarkSetRequestIds(b *testing.B) {
	payload := benchPayload()
	req := &tenantCarrier{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := setRequestTenantId(req, payload); err != nil {
			b.Fatal(err)
		}
		if err := setRequestOperationId(req, payload); err != nil {
			b.Fatal(err)
		}
	}
}
