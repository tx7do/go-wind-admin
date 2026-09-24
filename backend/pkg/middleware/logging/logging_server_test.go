// logging_server_test.go —— Server() 外层中间件测试。
//
// 覆盖内容：
//  1. 中间件链的 handler 透传语义：无 transport 与非 *http.Transport 传输层时，
//     业务 handler 照常执行、返回值原样透传，五个审计子中间件全部跳过
//     （对应源码中两处 transport 类型断言的 false 分支）；
//  2. 审计分发矩阵：POST/PUT/DELETE + 可解析写操作 → API/操作/权限审计落库，
//     GET 与会话维护端点 → 对应跳过；登录/登出/MFA 端点只进登录审计；
//     SQL 事件 → 数据访问审计（与会话语义正交）；未注册路由无审计；
//  3. 请求体快照的边界分支（空体、nil body、nil 请求）与 replayBody 的
//     Close 透传语义。
//
// 分发矩阵走 harness 的进程内 server（见 harness_test.go），全程无 socket。
package logging

import (
	"context"
	"errors"
	"io"
	nethttp "net/http"
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v2/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
)

// ---------------------------------------------------------------------------
// 非 *http.Transport 的传输层假件（覆盖类型断言 false 分支）
// ---------------------------------------------------------------------------

// fakeHeader 以 map 为底座实现 kratos transport.Header 接口。
type fakeHeader struct{ m map[string][]string }

func (h *fakeHeader) Get(key string) string {
	if v, ok := h.m[key]; ok && len(v) > 0 {
		return v[0]
	}
	return ""
}

func (h *fakeHeader) Set(key, value string) { h.m[key] = []string{value} }
func (h *fakeHeader) Add(key, value string) { h.m[key] = append(h.m[key], value) }
func (h *fakeHeader) Keys() []string {
	out := make([]string, 0, len(h.m))
	for k := range h.m {
		out = append(out, k)
	}
	return out
}
func (h *fakeHeader) Values(key string) []string { return h.m[key] }

// fakeTransporter 满足 transport.Transporter 但不是 *http.Transport：
// Server() 对它必须跳过全部审计（快照与落库两个阶段各有一处类型断言）。
type fakeTransporter struct{ op string }

func (f *fakeTransporter) Kind() transport.Kind { return transport.KindHTTP }
func (f *fakeTransporter) Endpoint() string     { return "fake://endpoint" }
func (f *fakeTransporter) Operation() string    { return f.op }
func (f *fakeTransporter) RequestHeader() transport.Header {
	return &fakeHeader{m: map[string][]string{}}
}
func (f *fakeTransporter) ReplyHeader() transport.Header {
	return &fakeHeader{m: map[string][]string{}}
}

// ---------------------------------------------------------------------------
// handler 透传与审计跳过
// ---------------------------------------------------------------------------

// newDirectCaptureStubOpts 构造五个记录型写入桩，用于不经 server 的直调测试。
func newDirectCaptureStubOpts(t *testing.T, cap *auditCapture) []Option {
	t.Helper()
	return []Option{
		WithWriteApiLogFunc(func(ctx context.Context, d *auditV1.ApiAuditLog) error {
			cap.api = append(cap.api, d)
			cap.apiMeta = append(cap.apiMeta, cap.metaOf(ctx))
			return nil
		}),
		WithWriteLoginLogFunc(func(ctx context.Context, d *auditV1.LoginAuditLog) error {
			cap.login = append(cap.login, d)
			cap.loginMeta = append(cap.loginMeta, cap.metaOf(ctx))
			return nil
		}),
		WithWriteOperationAuditLogFunc(func(ctx context.Context, d *auditV1.OperationAuditLog) error {
			cap.operation = append(cap.operation, d)
			cap.operationMeta = append(cap.operationMeta, cap.metaOf(ctx))
			return nil
		}),
		WithWritePermissionAuditLogFunc(func(ctx context.Context, d *auditV1.PermissionAuditLog) error {
			cap.permission = append(cap.permission, d)
			cap.permissionMeta = append(cap.permissionMeta, cap.metaOf(ctx))
			return nil
		}),
		WithWriteDataAccessAuditLogFunc(func(ctx context.Context, d *auditV1.DataAccessAuditLog) error {
			cap.dataAccess = append(cap.dataAccess, d)
			cap.dataAccessMeta = append(cap.dataAccessMeta, cap.metaOf(ctx))
			return nil
		}),
	}
}

// TestServerPassesThroughWithoutTransport 验证：ctx 无传输层时，Server() 包装的
// handler 照常执行且返回值原样透传，五个审计桩零写入——审计必须在真实
// *http.Transport 下才有意义，缺传输层不是静默构造脏记录的理由。
func TestServerPassesThroughWithoutTransport(t *testing.T) {
	cap := &auditCapture{}
	mw := Server(newDirectCaptureStubOpts(t, cap)...)

	innerCalled := false
	inner := func(ctx context.Context, req interface{}) (interface{}, error) {
		innerCalled = true
		return "reply-marker", nil
	}
	wrapped := mw(inner)

	out, err := wrapped(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, "reply-marker", out)
	require.True(t, innerCalled, "业务 handler 必须被调用")
	require.Zero(t, cap.total(), "无传输层时不得有任何审计写入")
}

// TestServerSkipsAuditForNonHTTPTransport 验证：传输层不是 kratos *http.Transport
// （如 gRPC 或假件）时，handler 照常执行、返回值透传，但快照与落库两个阶段的
// 类型断言都失败，五个审计桩零写入。
func TestServerSkipsAuditForNonHTTPTransport(t *testing.T) {
	cap := &auditCapture{}
	mw := Server(newDirectCaptureStubOpts(t, cap)...)

	inner := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "reply-marker", nil
	}
	wrapped := mw(inner)

	fakeCtx := transport.NewServerContext(context.Background(),
		&fakeTransporter{op: "/demo.v1.GadgetService/Update"})

	out, err := wrapped(fakeCtx, nil)
	require.NoError(t, err)
	require.Equal(t, "reply-marker", out)
	require.Zero(t, cap.total(), "非 HTTP 传输层不得触发任何审计写入")
}

// ---------------------------------------------------------------------------
// 审计分发矩阵
// ---------------------------------------------------------------------------

// dispatchCase 一条分发场景：指定请求形态与五个审计桩的期望写入数。
type dispatchCase struct {
	name        string
	method      string
	path        string
	op          string            // X-Test-Operation（模拟生成代码的 SetOperation）
	headers     map[string]string // 额外请求头（Content-Type / 数据访问事件开关等）
	expectAPI   int
	expectLogin int
	expectOp    int
	expectPerm  int
	expectDA    int
	expect404   bool
}

// TestServerDispatchMatrix 验证 Server() 把请求分发给五个审计子中间件的完整矩阵：
//   - POST/PUT/DELETE + 可解析写操作 → API + 操作 + 权限审计各一条（JSON 体含
//     name 时权限审计提取目标名）；
//   - GET → 仅 API 审计（读请求不构成操作/权限变更）；
//   - 登录/MFA 验证端点 → 仅登录审计（API 审计按 loginOperations 跳过、操作/权限
//     按 sessionOnlyOperations 跳过）；登出端点 → API + 登录审计（登出不在 API
//     审计的跳过名单里，但属会话维护，操作/权限审计跳过）；
//   - 解析不出 resource 的 operation → 仅 API 审计；
//   - SQL 事件注入 → 数据访问审计，条数与事件数一致，且与会话/操作语义正交；
//   - 未注册的路由方法（PATCH）→ mux 404，中间件链根本不运行，零写入。
func TestServerDispatchMatrix(t *testing.T) {
	jsonHdr := map[string]string{"Content-Type": "application/json"}
	cases := []dispatchCase{
		{
			name: "POST可解析写操作带JSON体", method: nethttp.MethodPost, path: "/case/5",
			op: "/demo.v1.GadgetService/Update", headers: jsonHdr,
			expectAPI: 1, expectOp: 1, expectPerm: 1,
		},
		{
			name: "GET读请求只进API审计", method: nethttp.MethodGet, path: "/case/5",
			op:        "/demo.v1.GadgetService/Update",
			expectAPI: 1,
		},
		{
			name: "登录端点仅登录审计", method: nethttp.MethodPost, path: "/case/5",
			op:          adminV1.OperationAuthenticationServiceLogin,
			expectLogin: 1,
		},
		{
			name: "MFA验证端点仅登录审计", method: nethttp.MethodPost, path: "/case/5",
			op:          adminV1.OperationMfaServiceVerifyMFAChallenge,
			expectLogin: 1,
		},
		{
			name: "登出端点进API与登录审计", method: nethttp.MethodPost, path: "/case/5",
			op:        adminV1.OperationAuthenticationServiceLogout,
			expectAPI: 1, expectLogin: 1,
		},
		{
			name: "SQL事件进数据访问审计且与写操作并存", method: nethttp.MethodPost, path: "/case/5",
			op:        "/demo.v1.GadgetService/Update",
			headers:   map[string]string{"X-Test-Data-Access": "all", "Content-Type": "application/json"},
			expectAPI: 1, expectOp: 1, expectPerm: 1, expectDA: 4,
		},
		{
			name: "解析不出资源的operation仅API审计", method: nethttp.MethodPost, path: "/case/5",
			op:        "plain-bad-op",
			expectAPI: 1,
		},
		{
			name: "未注册方法PATCH零审计", method: nethttp.MethodPatch, path: "/case/5",
			op:        "/demo.v1.GadgetService/Update",
			expect404: true,
		},
		{
			name: "PUT写方法同样进三个写审计", method: nethttp.MethodPut, path: "/case/5",
			op: "/demo.v1.GadgetService/Create", headers: jsonHdr,
			expectAPI: 1, expectOp: 1, expectPerm: 1,
		},
		{
			name: "DELETE写方法同样进三个写审计", method: nethttp.MethodDelete, path: "/case/5",
			op: "/demo.v1.GadgetService/Delete", headers: jsonHdr,
			expectAPI: 1, expectOp: 1, expectPerm: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// 每个场景独立挂具，避免写入计数互相叠加。
			env := newAuditServer(t)
			hdrs := map[string]string{}
			for k, v := range tc.headers {
				hdrs[k] = v
			}
			hdrs["X-Test-Operation"] = tc.op
			rec := env.fire(tc.method, tc.path, hdrs, "", "127.0.0.1:1234")

			if tc.expect404 {
				require.Equal(t, 404, rec.Code, "未注册路由必须 404")
			} else {
				require.Equal(t, 200, rec.Code)
			}
			require.Len(t, env.capture.api, tc.expectAPI, "API 审计写入数")
			require.Len(t, env.capture.login, tc.expectLogin, "登录审计写入数")
			require.Len(t, env.capture.operation, tc.expectOp, "操作审计写入数")
			require.Len(t, env.capture.permission, tc.expectPerm, "权限审计写入数")
			require.Len(t, env.capture.dataAccess, tc.expectDA, "数据访问审计写入数")

			// 落库阶段的上下文不变量：sink 标记（防递归采集）+ 系统 viewer
			// （审计写入必须绕过租户隔离）。该不变量由 Server() 在落库前植入。
			for _, m := range env.capture.apiMeta {
				require.True(t, m.Sinking, "API 审计写入必须带 sink 标记")
				require.True(t, m.SystemViewer, "API 审计写入必须以系统 viewer 执行")
			}
			for _, m := range env.capture.operationMeta {
				require.True(t, m.Sinking, "操作审计写入必须带 sink 标记")
				require.True(t, m.SystemViewer, "操作审计写入必须以系统 viewer 执行")
			}
			for _, m := range env.capture.permissionMeta {
				require.True(t, m.Sinking, "权限审计写入必须带 sink 标记")
				require.True(t, m.SystemViewer, "权限审计写入必须以系统 viewer 执行")
			}
			for _, m := range env.capture.loginMeta {
				require.True(t, m.Sinking, "登录审计写入必须带 sink 标记")
				require.True(t, m.SystemViewer, "登录审计写入必须以系统 viewer 执行")
			}
			for _, m := range env.capture.dataAccessMeta {
				require.True(t, m.Sinking, "数据访问审计写入必须带 sink 标记")
				require.True(t, m.SystemViewer, "数据访问审计写入必须以系统 viewer 执行")
			}

			// 注入过 SQL 事件的场景：accumulator 必须被清空，防止后续复用 ctx 重复落库。
			if tc.expectDA > 0 {
				require.NotNil(t, env.acc, "业务闭包应能取到 accumulator")
				require.Len(t, *env.acc, 0, "落库后 accumulator 必须清空")
			}
		})
	}
}

// TestServerEmptyBodySnapshotBranch 验证空 JSON 写体的快照分支：
// ReadFull 读到 0 字节时 body 被重置为空流、ctx 不携带快照值，
// API 审计读到空请求体、权限审计提取不到目标名称。
func TestServerEmptyBodySnapshotBranch(t *testing.T) {
	env := newAuditServer(t)
	rec := env.fire(nethttp.MethodPost, "/case/5", map[string]string{
		"X-Test-Operation": "/demo.v1.GadgetService/Update",
		"Content-Type":     "application/json",
	}, "", "127.0.0.1:1234")

	require.Equal(t, 200, rec.Code)
	require.Len(t, env.capture.api, 1)
	require.Len(t, env.capture.permission, 1)
	assert.Empty(t, env.capture.api[0].GetRequestBody(), "空体快照后 API 审计只应读到空串")
	assert.Nil(t, env.capture.permission[0].TargetName, "无快照值时权限审计不得提取目标名称")
}

// ---------------------------------------------------------------------------
// 登录/登出操作名单的选项覆盖语义
// ---------------------------------------------------------------------------

// TestServerEmptyLoginOperationListOption 验证 WithLoginOperation() 空名单
// 的覆盖语义：默认登录端点既不进登录审计（名单被清空），也不再被 API
// 审计跳过（跳过名单同源）——空名单是全量替换而非追加。
func TestServerEmptyLoginOperationListOption(t *testing.T) {
	env := newAuditServer(t, WithLoginOperation())
	env.fire(nethttp.MethodPost, "/case/5", map[string]string{
		"X-Test-Operation": adminV1.OperationAuthenticationServiceLogin,
	}, "", "127.0.0.1:1234")
	assert.Empty(t, env.capture.login, "空名单下登录端点不进登录审计")
	assert.Len(t, env.capture.api, 1, "空名单下 API 审计不再跳过登录端点")
	assert.Empty(t, env.capture.operation, "会话维护端点仍被操作审计跳过（硬编码名单）")
	assert.Empty(t, env.capture.permission, "会话维护端点仍被权限审计跳过（硬编码名单）")
}

// TestServerCustomLoginLogoutOperationOptions 验证自定义名单的替换语义：
// 自定义登录端点进登录审计且被 API 审计跳过；默认登出端点因名单被替换
// 不再进登录审计；自定义登出端点经 WithLogoutOperation 生效进登录审计
// （LOGOUT 动作），且不在 API 审计跳过名单内。
func TestServerCustomLoginLogoutOperationOptions(t *testing.T) {
	env := newAuditServer(t,
		WithLoginOperation("custom.login"),
		WithLogoutOperation("custom.logout"),
	)

	type expect struct {
		name       string
		op         string
		apiDelta   int
		loginDelta int
		loginType  auditV1.LoginAuditLog_ActionType
	}
	cases := []expect{
		{"自定义登录端点进登录审计且API审计跳过", "custom.login", 0, 1, auditV1.LoginAuditLog_LOGIN},
		{"默认登出端点被名单替换跳过登录审计", adminV1.OperationAuthenticationServiceLogout, 1, 0, 0},
		{"自定义登出端点生效", "custom.logout", 1, 1, auditV1.LoginAuditLog_LOGOUT},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			apiBase := len(env.capture.api)
			loginBase := len(env.capture.login)
			env.fire(nethttp.MethodPost, "/case/5", map[string]string{
				"X-Test-Operation": tc.op,
			}, "", "127.0.0.1:1234")
			assert.Len(t, env.capture.api, apiBase+tc.apiDelta, "API 审计写入增量")
			assert.Len(t, env.capture.login, loginBase+tc.loginDelta, "登录审计写入增量")
			if tc.loginDelta == 1 {
				assert.Equal(t, tc.loginType, env.capture.login[loginBase].GetActionType())
			}
			// 硬编码会话名单不受选项影响。
			assert.Empty(t, env.capture.operation, "操作审计对会话端点恒跳过")
			assert.Empty(t, env.capture.permission, "权限审计对会话端点恒跳过")
		})
	}
}

// TestSnapshotWriteBodyNilAndEmptyBranches 直调覆盖 snapshotWriteBody 的
// 早退分支：nil 请求与无 body 请求原样返回 ctx；空 JSON 体走 0 字节分支，
// body 被重置为空流且不产生快照值。
func TestSnapshotWriteBodyNilAndEmptyBranches(t *testing.T) {
	ctx := context.Background()

	// nil 请求。
	assert.Equal(t, ctx, snapshotWriteBody(ctx, nil))

	// POST + JSON 头但 body 为 nil。
	reqNilBody := &nethttp.Request{
		Method: nethttp.MethodPost,
		Header: nethttp.Header{"Content-Type": []string{"application/json"}},
	}
	assert.Equal(t, ctx, snapshotWriteBody(ctx, reqNilBody))
	assert.Nil(t, reqNilBody.Body)

	// 空 JSON body：ReadFull 计 0 字节 → 重置空流、无快照。
	reqEmpty := &nethttp.Request{
		Method: nethttp.MethodPost,
		Body:   io.NopCloser(strings.NewReader("")),
		Header: nethttp.Header{"Content-Type": []string{"application/json"}},
	}
	ctxEmpty := snapshotWriteBody(ctx, reqEmpty)
	assert.Nil(t, bodySnapshotFromContext(ctxEmpty), "0 字节体不得产生快照")
	b, err := io.ReadAll(reqEmpty.Body)
	assert.NoError(t, err)
	assert.Empty(t, b, "重置后的 body 应为空流")
}

// sentinelCloseBody 可读但 Close 返回哨兵错误，用于验证 replayBody.Close
// 对原始 body Close 的透传。
type sentinelCloseBody struct {
	r   io.Reader
	err error
}

func (b *sentinelCloseBody) Read(p []byte) (int, error) { return b.r.Read(p) }
func (b *sentinelCloseBody) Close() error               { return b.err }

// TestReplayBodyClosePassthrough 验证：快照后的 replayBody.Close 必须把
// Close 调用透传给原始 body（含其返回的错误），否则原流的资源会泄漏。
func TestReplayBodyClosePassthrough(t *testing.T) {
	sentinel := errors.New("sentinel-close-error")
	orig := &sentinelCloseBody{r: strings.NewReader(`{"x":1}`), err: sentinel}
	req := &nethttp.Request{
		Method: nethttp.MethodPost,
		Body:   orig,
		Header: nethttp.Header{"Content-Type": []string{"application/json"}},
	}
	snapCtx := snapshotWriteBody(context.Background(), req)
	require.NotNil(t, bodySnapshotFromContext(snapCtx), "小 JSON 体应被完整快照")
	require.IsType(t, &replayBody{}, req.Body, "body 应被替换为可重放流")

	err := req.Body.Close()
	assert.ErrorIs(t, err, sentinel, "replayBody.Close 必须透传原始 body 的 Close 错误")
}
