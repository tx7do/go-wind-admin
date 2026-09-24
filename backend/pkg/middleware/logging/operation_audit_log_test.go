// operation_audit_log_test.go —— OperationAuditLogMiddleware 测试。
//
// 覆盖内容：
//  1. 构造器与 Name()；
//  2. parseResourceAndAction 表驱动：operation 串 → 资源类型（服务名去
//     Service 后缀转小写）与动作类型（方法名映射，含 Export/Import 的
//     操作审计特有分支、未知方法的 OTHER、各类畸形串的 UNSPECIFIED）；
//  3. 触发路径（进程内 server）：字段来源映射（资源类型/动作/REST 路径
//     最后数字段→ResourceId、令牌身份、IP、请求 ID、地理、错误映射）；
//  4. 跳过路径：GET 读请求、会话维护端点、畸形 operation；
//  5. 直调空 Transport：Request() 为 nil 的分支（ResourceId/Reason 的
//     空值来源）与错误映射；
//  6. writeOperationAuditLogFunc 为 nil 时只跳过落库；
//  7. hashLog/signature 的 nil 边界。
package logging

import (
	"context"
	nethttp "net/http"
	"testing"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/transport"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
)

// TestOperationAuditLogMiddlewareNameAndConstructor 验证构造器落参与固定名称。
func TestOperationAuditLogMiddlewareNameAndConstructor(t *testing.T) {
	op := &options{}
	mw := NewOperationAuditLogMiddleware(op)
	require.NotNil(t, mw)
	assert.Equal(t, op, mw.op)
	assert.Equal(t, "OperationAuditLogMiddleware", mw.Name())
}

// TestParseResourceAndAction 表驱动验证 operation 解析：资源类型取
// "<包>.<ServiceName>" 的服务名去 Service 后缀转小写，动作按方法名映射；
// 操作审计含 Export/Import 映射，未知方法返回 OTHER；任何畸形串
// （无斜杠/尾斜杠/无点/尾点）返回 UNSPECIFIED。
func TestParseResourceAndAction(t *testing.T) {
	cases := []struct {
		name         string
		operation    string
		wantResource string
		wantAction   auditV1.OperationAuditLog_ActionType
	}{
		{"标准更新", "/admin.service.v1.RoleService/Update", "role", auditV1.OperationAuditLog_UPDATE},
		{"创建", "/x.YService/Create", "y", auditV1.OperationAuditLog_CREATE},
		{"批量创建", "/x.YService/BatchCreate", "y", auditV1.OperationAuditLog_CREATE},
		{"删除", "/x.YService/Delete", "y", auditV1.OperationAuditLog_DELETE},
		{"批量删除", "/x.YService/BatchDelete", "y", auditV1.OperationAuditLog_DELETE},
		{"导出", "/x.YService/Export", "y", auditV1.OperationAuditLog_EXPORT},
		{"导入", "/x.YService/Import", "y", auditV1.OperationAuditLog_IMPORT},
		{"分配", "/x.YService/Assign", "y", auditV1.OperationAuditLog_ASSIGN},
		{"取消分配", "/x.YService/Unassign", "y", auditV1.OperationAuditLog_UNASSIGN},
		{"读方法归OTHER", "/x.YService/Get", "y", auditV1.OperationAuditLog_OTHER},
		{"未知方法归OTHER", "/x.YService/Unknown", "y", auditV1.OperationAuditLog_OTHER},
		{"后缀只裁一次", "/x.RoleServiceService/Delete", "roleservice", auditV1.OperationAuditLog_DELETE},
		{"空串", "", "", auditV1.OperationAuditLog_ACTION_TYPE_UNSPECIFIED},
		{"无斜杠", "Update", "", auditV1.OperationAuditLog_ACTION_TYPE_UNSPECIFIED},
		{"尾斜杠", "/abc/", "", auditV1.OperationAuditLog_ACTION_TYPE_UNSPECIFIED},
		{"服务段无点", "/abc/Update", "", auditV1.OperationAuditLog_ACTION_TYPE_UNSPECIFIED},
		{"服务段尾点", "/a.b./Update", "", auditV1.OperationAuditLog_ACTION_TYPE_UNSPECIFIED},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotResource, gotAction := parseResourceAndAction(tc.operation)
			assert.Equal(t, tc.wantResource, gotResource)
			assert.Equal(t, tc.wantAction, gotAction)
		})
	}
}

// TestOperationAuditLogHandleFieldMapping 走 server 验证触发路径的字段来源映射。
func TestOperationAuditLogHandleFieldMapping(t *testing.T) {
	env := newAuditServer(t)
	env.fire(nethttp.MethodPost, "/case/5", map[string]string{
		"X-Test-Operation": "/demo.v1.GadgetService/Update",
		"Authorization":    "Bearer " + mintTestToken(t),
		"X-Request-ID":     "req-op-1",
	}, "", "127.0.0.1:1234")

	require.Len(t, env.capture.operation, 1)
	rec := env.capture.operation[0]

	assert.Equal(t, "gadget", rec.GetResourceType(), "ResourceType ← 服务名去 Service 后缀转小写")
	assert.Equal(t, auditV1.OperationAuditLog_UPDATE, rec.GetAction())
	assert.Equal(t, "5", rec.GetResourceId(), "ResourceId ← REST 路径最后的纯数字段")
	assert.Equal(t, uint32(42), rec.GetUserId(), "UserId ← 令牌 uid")
	assert.Equal(t, uint32(7), rec.GetTenantId(), "TenantId ← 令牌 tid")
	assert.Equal(t, "alice", rec.GetUsername(), "Username ← 令牌 sub")
	assert.Equal(t, "127.0.0.1", rec.GetIpAddress())
	assert.Equal(t, "req-op-1", rec.GetRequestId(), "RequestId ← X-Request-ID 头")
	assert.Equal(t, "局域网", rec.GetGeoLocation().GetCountryCode(), "GeoLocation ← 客户端 IP 的地理解析")
	assert.True(t, rec.GetSuccess())
	assert.Empty(t, rec.GetFailureReason())
	requireLogHashHex(t, rec.GetLogHash())
	requireDERSig(t, rec.GetSignature())
}

// TestOperationAuditLogHandleActionClassification 表驱动验证动作类型在
// 真实请求链路中的映射（含操作审计特有的 Export/Import 与 OTHER）。
func TestOperationAuditLogHandleActionClassification(t *testing.T) {
	cases := []struct {
		method     string
		opMethod   string
		wantAction auditV1.OperationAuditLog_ActionType
	}{
		{nethttp.MethodPost, "Update", auditV1.OperationAuditLog_UPDATE},
		{nethttp.MethodPost, "Create", auditV1.OperationAuditLog_CREATE},
		{nethttp.MethodPost, "Export", auditV1.OperationAuditLog_EXPORT},
		{nethttp.MethodPost, "Import", auditV1.OperationAuditLog_IMPORT},
		{nethttp.MethodPost, "Assign", auditV1.OperationAuditLog_ASSIGN},
		{nethttp.MethodPost, "Unassign", auditV1.OperationAuditLog_UNASSIGN},
		{nethttp.MethodPost, "Get", auditV1.OperationAuditLog_OTHER},
		{nethttp.MethodPut, "Update", auditV1.OperationAuditLog_UPDATE},
		{nethttp.MethodDelete, "Delete", auditV1.OperationAuditLog_DELETE},
	}
	env := newAuditServer(t)
	for _, tc := range cases {
		t.Run(tc.method+"/"+tc.opMethod, func(t *testing.T) {
			base := len(env.capture.operation)
			env.fire(tc.method, "/case/5", map[string]string{
				"X-Test-Operation": "/demo.v1.GadgetService/" + tc.opMethod,
			}, "", "127.0.0.1:1234")
			require.Len(t, env.capture.operation, base+1, "写方法 + 可解析 operation 应各产生一条记录")
			rec := env.capture.operation[base]
			assert.Equal(t, "gadget", rec.GetResourceType())
			assert.Equal(t, tc.wantAction, rec.GetAction())
		})
	}
}

// TestOperationAuditLogHandleResourceIdVariants 验证 ResourceId 的取值边界：
// 非数字段与无数字段的路径不产生 ResourceId。
func TestOperationAuditLogHandleResourceIdVariants(t *testing.T) {
	t.Run("非数字段", func(t *testing.T) {
		env := newAuditServer(t)
		env.fire(nethttp.MethodPost, "/case/abc", map[string]string{
			"X-Test-Operation": "/demo.v1.GadgetService/Update",
		}, "", "127.0.0.1:1234")
		require.Len(t, env.capture.operation, 1)
		assert.Nil(t, env.capture.operation[0].ResourceId, "非数字段不得产生 ResourceId")
	})

	t.Run("无数字段", func(t *testing.T) {
		env := newAuditServer(t)
		env.fire(nethttp.MethodPost, "/case", map[string]string{
			"X-Test-Operation": "/demo.v1.GadgetService/Update",
		}, "", "127.0.0.1:1234")
		require.Len(t, env.capture.operation, 1)
		assert.Nil(t, env.capture.operation[0].ResourceId)
	})
}

// TestOperationAuditLogHandleSkips 验证三类跳过路径：GET 读请求、
// 会话维护端点（登录/刷新无关但含登出与 MFA 验证）、解析不出资源的 operation。
func TestOperationAuditLogHandleSkips(t *testing.T) {
	skipCases := []struct {
		name   string
		method string
		op     string
	}{
		{"GET读请求", nethttp.MethodGet, "/demo.v1.GadgetService/Update"},
		{"登录端点", nethttp.MethodPost, adminV1.OperationAuthenticationServiceLogin},
		{"登出端点", nethttp.MethodPost, adminV1.OperationAuthenticationServiceLogout},
		{"MFA验证端点", nethttp.MethodPost, adminV1.OperationMfaServiceVerifyMFAChallenge},
		{"畸形operation", nethttp.MethodPost, "plain-bad-op"},
		{"仅服务段", nethttp.MethodPost, "/demo.v1.GadgetService/"},
	}
	for _, tc := range skipCases {
		t.Run(tc.name, func(t *testing.T) {
			env := newAuditServer(t)
			env.fire(tc.method, "/case/5", map[string]string{
				"X-Test-Operation": tc.op,
			}, "", "127.0.0.1:1234")
			assert.Empty(t, env.capture.operation, "该场景不得写入操作审计")
		})
	}
}

// TestOperationAuditLogHandleDirectNilRequest 直调空 Transport 覆盖
// Request() 为 nil 的分支：ResourceId 与 Reason 的方法/路径部分为空、
// IP/请求 ID 归一空串、令牌与地理字段为空；错误变体覆盖 Reason 的
// 失败后缀拼接与 Success=false。
func TestOperationAuditLogHandleDirectNilRequest(t *testing.T) {
	newMw := func(t *testing.T) (*OperationAuditLogMiddleware, *[]*auditV1.OperationAuditLog) {
		var recs []*auditV1.OperationAuditLog
		var op options
		WithWriteOperationAuditLogFunc(func(ctx context.Context, d *auditV1.OperationAuditLog) error {
			recs = append(recs, d)
			return nil
		})(&op)
		key, _, err := generateECDSAKeyPair()
		require.NoError(t, err)
		WithECPrivateKey(key)(&op)
		return NewOperationAuditLogMiddleware(&op), &recs
	}

	t.Run("无错误", func(t *testing.T) {
		mw, recs := newMw(t)
		tr := &khttp.Transport{}
		ctx := transport.NewServerContext(context.Background(), tr)
		khttp.SetOperation(ctx, "/demo.v1.GadgetService/Update")
		mw.Handle(ctx, tr, nil, 0)
		require.Len(t, *recs, 1)
		rec := (*recs)[0]
		assert.Equal(t, "gadget", rec.GetResourceType())
		assert.Equal(t, auditV1.OperationAuditLog_UPDATE, rec.GetAction())
		assert.Empty(t, rec.GetResourceId(), "nil 请求时 ResourceId 不落库")
		assert.Empty(t, rec.GetIpAddress())
		assert.Empty(t, rec.GetRequestId())
		assert.Zero(t, rec.GetUserId())
		assert.Empty(t, rec.GetUsername())
		assert.Empty(t, rec.GetGeoLocation().GetCountryCode())
		assert.Empty(t, rec.GetFailureReason(), "nil 请求时 Reason 无方法/路径前缀")
		assert.True(t, rec.GetSuccess())
	})

	t.Run("kratos错误", func(t *testing.T) {
		mw, recs := newMw(t)
		tr := &khttp.Transport{}
		ctx := transport.NewServerContext(context.Background(), tr)
		khttp.SetOperation(ctx, "/demo.v1.GadgetService/Update")
		mw.Handle(ctx, tr, kerrors.New(403, "TEST_FORBIDDEN", "forbidden for test"), 0)
		require.Len(t, *recs, 1)
		rec := (*recs)[0]
		assert.False(t, rec.GetSuccess())
		assert.Equal(t, "TEST_FORBIDDEN", rec.GetFailureReason(), "FailureReason ← kratos 错误 reason")
	})
}

// TestOperationAuditLogHandleWriteFuncNil 验证 writeOperationAuditLogFunc
// 为 nil 时：记录构造照常（API/权限审计不受影响），操作审计自身不落库。
func TestOperationAuditLogHandleWriteFuncNil(t *testing.T) {
	env := newAuditServer(t, WithWriteOperationAuditLogFunc(nil))
	env.fire(nethttp.MethodPost, "/case/5", map[string]string{
		"X-Test-Operation": "/demo.v1.GadgetService/Update",
		"Content-Type":     "application/json",
	}, `{"data":{"name":"x"}}`, "127.0.0.1:1234")
	assert.Empty(t, env.capture.operation, "空写入函数时操作审计不得落库")
	assert.Len(t, env.capture.api, 1, "其余审计不受影响")
	assert.Len(t, env.capture.permission, 1)
}

// TestOperationAuditLogHashLogAndSignatureEdges 直调覆盖 nil 边界。
func TestOperationAuditLogHashLogAndSignatureEdges(t *testing.T) {
	mwNoKey := NewOperationAuditLogMiddleware(&options{})
	assert.Equal(t, "", mwNoKey.hashLog(nil))
	assert.Nil(t, mwNoKey.signature(nil))
	assert.Nil(t, mwNoKey.signature(&auditV1.OperationAuditLog{}), "无私钥时签名必须为 nil")

	key, _, err := generateECDSAKeyPair()
	require.NoError(t, err)
	mwWithKey := NewOperationAuditLogMiddleware(&options{ecPrivateKey: key})
	assert.Equal(t, "", mwWithKey.hashLog(nil))
	assert.Nil(t, mwWithKey.signature(nil))
}
