// permission_audit_log_handle_test.go —— PermissionAuditLogMiddleware
// 的 Handle 层测试（解析纯函数与快照提取的直调表已在
// permission_audit_log_test.go 覆盖，此处不重复）。
//
// 覆盖内容：
//  1. 构造器与 Name()；
//  2. parseTargetAndAction 表驱动：与操作审计的解析差异——无 Export/Import
//     映射（归 OTHER），其余方法映射一致；
//  3. 触发路径（进程内 server）：字段来源映射，重点验证 TargetName ←
//     pre-handler 快照的 JSON 写请求体（{"data":{...}} 包裹下的名称候选）
//     与 TargetId ← REST 路径最后数字段、Operator* ← 令牌、Reason ←
//     方法 + 路径（失败时附错误 reason）；
//  4. 跳过路径：GET、会话维护端点、畸形 operation；
//  5. 快照缺失变体：无名称字段体 / 非 JSON 体 / 空体 → TargetName 不落库；
//  6. 直调空 Transport：Request() 为 nil 的分支（TargetId 与 Reason 的
//     方法/路径部分为空）；
//  7. writePermissionAuditLogFunc 为 nil 时只跳过落库；
//  8. hashLog/signature 的 nil 边界。
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

// TestPermissionAuditLogMiddlewareNameAndConstructor 验证构造器落参与固定名称。
func TestPermissionAuditLogMiddlewareNameAndConstructor(t *testing.T) {
	op := &options{}
	mw := NewPermissionAuditLogMiddleware(op)
	require.NotNil(t, mw)
	assert.Equal(t, op, mw.op)
	assert.Equal(t, "PermissionAuditLogMiddleware", mw.Name())
}

// TestParseTargetAndAction 表驱动验证 operation 解析：目标类型取服务名去
// Service 后缀转小写；注意与操作审计不同，权限审计无 Export/Import 映射，
// 二者归 OTHER；畸形串归 UNSPECIFIED。
func TestParseTargetAndAction(t *testing.T) {
	cases := []struct {
		name       string
		operation  string
		wantTarget string
		wantAction auditV1.PermissionAuditLog_ActionType
	}{
		{"标准更新", "/admin.service.v1.RoleService/Update", "role", auditV1.PermissionAuditLog_UPDATE},
		{"创建", "/x.YService/Create", "y", auditV1.PermissionAuditLog_CREATE},
		{"批量创建", "/x.YService/BatchCreate", "y", auditV1.PermissionAuditLog_CREATE},
		{"删除", "/x.YService/Delete", "y", auditV1.PermissionAuditLog_DELETE},
		{"批量删除", "/x.YService/BatchDelete", "y", auditV1.PermissionAuditLog_DELETE},
		{"分配", "/x.YService/Assign", "y", auditV1.PermissionAuditLog_ASSIGN},
		{"取消分配", "/x.YService/Unassign", "y", auditV1.PermissionAuditLog_UNASSIGN},
		{"导出归OTHER（无导出映射）", "/x.YService/Export", "y", auditV1.PermissionAuditLog_OTHER},
		{"导入归OTHER（无导入映射）", "/x.YService/Import", "y", auditV1.PermissionAuditLog_OTHER},
		{"读方法归OTHER", "/x.YService/Get", "y", auditV1.PermissionAuditLog_OTHER},
		{"空串", "", "", auditV1.PermissionAuditLog_ACTION_TYPE_UNSPECIFIED},
		{"无斜杠", "Update", "", auditV1.PermissionAuditLog_ACTION_TYPE_UNSPECIFIED},
		{"尾斜杠", "/abc/", "", auditV1.PermissionAuditLog_ACTION_TYPE_UNSPECIFIED},
		{"服务段无点", "/abc/Update", "", auditV1.PermissionAuditLog_ACTION_TYPE_UNSPECIFIED},
		{"服务段尾点", "/a.b./Update", "", auditV1.PermissionAuditLog_ACTION_TYPE_UNSPECIFIED},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotTarget, gotAction := parseTargetAndAction(tc.operation)
			assert.Equal(t, tc.wantTarget, gotTarget)
			assert.Equal(t, tc.wantAction, gotAction)
		})
	}
}

// TestPermissionAuditLogHandleFieldMapping 走 server 验证触发路径的字段来源映射。
// TargetName 必须来自 pre-handler 快照的 JSON 写请求体（CRUD 的
// {"data":{...}} 包裹），其余字段按来源逐项断言。
func TestPermissionAuditLogHandleFieldMapping(t *testing.T) {
	env := newAuditServer(t)
	env.fire(nethttp.MethodPost, "/case/9", map[string]string{
		"X-Test-Operation": "/demo.v1.GadgetService/Update",
		"Content-Type":     "application/json",
		"Authorization":    "Bearer " + mintTestToken(t),
		"X-Request-ID":     "req-pm-1",
	}, `{"data":{"name":"敏感角色","code":"x"}}`, "127.0.0.1:1234")

	require.Len(t, env.capture.permission, 1)
	rec := env.capture.permission[0]

	assert.Equal(t, "gadget", rec.GetTargetType(), "TargetType ← 服务名去 Service 后缀转小写")
	assert.Equal(t, auditV1.PermissionAuditLog_UPDATE, rec.GetAction())
	assert.Equal(t, "9", rec.GetTargetId(), "TargetId ← REST 路径最后的纯数字段")
	assert.Equal(t, "敏感角色", rec.GetTargetName(), "TargetName ← 快照请求体内的名称候选字段")
	assert.Equal(t, uint32(42), rec.GetOperatorId(), "OperatorId ← 令牌 uid")
	assert.Equal(t, "alice", rec.GetOperatorName(), "OperatorName ← 令牌 sub")
	assert.Equal(t, uint32(7), rec.GetTenantId(), "TenantId ← 令牌 tid")
	assert.Equal(t, "127.0.0.1", rec.GetIpAddress())
	assert.Equal(t, "req-pm-1", rec.GetRequestId(), "RequestId ← X-Request-ID 头")
	assert.Equal(t, "POST /case/9", rec.GetReason(), "Reason ← 请求方法 + 路径（无失败时无后缀）")
	requireLogHashHex(t, rec.GetLogHash())
	requireDERSig(t, rec.GetSignature())
}

// TestTargetNameFromBodyFallbackResidualArms 补充 targetNameFromBody
// 兜底路径的残余分支（既有直调表在 permission_audit_log_test.go）：
// 兜底键按字典序、空白值跳过、data 键非对象时回退外层平铺对象。
func TestTargetNameFromBodyFallbackResidualArms(t *testing.T) {
	// 兜底键列表按字典序取首个非空值；空白值跳过。
	assert.Equal(t, "x", targetNameFromBody([]byte(`{"fooName":"   ","barName":"x"}`)),
		"兜底键按字典序（barName 先于 fooName），空白值被跳过")
	// code 结尾兜底键按字典序。
	assert.Equal(t, "c2", targetNameFromBody([]byte(`{"zzCode":"c1","aaCode":"c2"}`)),
		"code 类兜底键按字典序（aaCode 先于 zzCode）")
	// data 键非对象：回退外层平铺对象提取。
	assert.Equal(t, "outer-name", targetNameFromBody([]byte(`{"data":"not-a-map","name":"outer-name"}`)),
		"data 非对象时回退外层平铺对象")
}

// TestPermissionAuditLogHandleReasonFailureSuffix 验证失败路径的 Reason
// 拼接：方法 + 路径 + " failed: " + 错误 reason。
func TestPermissionAuditLogHandleReasonFailureSuffix(t *testing.T) {
	env := newAuditServer(t)
	env.fire(nethttp.MethodPost, "/case/9", map[string]string{
		"X-Test-Operation": "/demo.v1.GadgetService/Update",
		"Content-Type":     "application/json",
		"X-Test-Error":     "kratos",
	}, `{"data":{"name":"x"}}`, "127.0.0.1:1234")
	require.Len(t, env.capture.permission, 1)
	assert.Equal(t, "POST /case/9 failed: TEST_FORBIDDEN", env.capture.permission[0].GetReason())
}

// TestPermissionAuditLogHandleTargetNameAbsentVariants 表驱动验证快照
// 缺失或无名称候选时 TargetName 必须不落库。
func TestPermissionAuditLogHandleTargetNameAbsentVariants(t *testing.T) {
	cases := []struct {
		name        string
		contentType string
		body        string
	}{
		{"无名称字段体", "application/json", `{"data":{"ids":[1,2]}}`},
		{"非JSON体", "text/plain", "hello"},
		{"空体", "application/json", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newAuditServer(t)
			env.fire(nethttp.MethodPost, "/case/9", map[string]string{
				"X-Test-Operation": "/demo.v1.GadgetService/Update",
				"Content-Type":     tc.contentType,
			}, tc.body, "127.0.0.1:1234")
			require.Len(t, env.capture.permission, 1)
			assert.Nil(t, env.capture.permission[0].TargetName, "无快照/无名称候选时 TargetName 不得落库")
		})
	}
}

// TestPermissionAuditLogHandleSkips 验证三类跳过路径：GET 读请求、
// 会话维护端点、解析不出目标的 operation。
func TestPermissionAuditLogHandleSkips(t *testing.T) {
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
			env.fire(tc.method, "/case/9", map[string]string{
				"X-Test-Operation": tc.op,
			}, "", "127.0.0.1:1234")
			assert.Empty(t, env.capture.permission, "该场景不得写入权限审计")
		})
	}
}

// TestPermissionAuditLogHandleDirectNilRequest 直调空 Transport 覆盖
// Request() 为 nil 的分支：TargetId 不落库、Reason 无方法/路径前缀
// （失败变体仅含 " failed: " 后缀）、IP/请求 ID/操作者字段为空来源。
func TestPermissionAuditLogHandleDirectNilRequest(t *testing.T) {
	newMw := func(t *testing.T) (*PermissionAuditLogMiddleware, *[]*auditV1.PermissionAuditLog) {
		var recs []*auditV1.PermissionAuditLog
		var op options
		WithWritePermissionAuditLogFunc(func(ctx context.Context, d *auditV1.PermissionAuditLog) error {
			recs = append(recs, d)
			return nil
		})(&op)
		key, _, err := generateECDSAKeyPair()
		require.NoError(t, err)
		WithECPrivateKey(key)(&op)
		return NewPermissionAuditLogMiddleware(&op), &recs
	}

	t.Run("无错误", func(t *testing.T) {
		mw, recs := newMw(t)
		tr := &khttp.Transport{}
		ctx := transport.NewServerContext(context.Background(), tr)
		khttp.SetOperation(ctx, "/demo.v1.GadgetService/Assign")
		mw.Handle(ctx, tr, nil, 0)
		require.Len(t, *recs, 1)
		rec := (*recs)[0]
		assert.Equal(t, "gadget", rec.GetTargetType())
		assert.Equal(t, auditV1.PermissionAuditLog_ASSIGN, rec.GetAction())
		assert.Empty(t, rec.GetTargetId(), "nil 请求时 TargetId 不落库")
		assert.Nil(t, rec.TargetName, "无快照上下文时 TargetName 不落库")
		assert.Empty(t, rec.GetIpAddress())
		assert.Empty(t, rec.GetRequestId())
		assert.Zero(t, rec.GetOperatorId())
		assert.Empty(t, rec.GetOperatorName())
		assert.Empty(t, rec.GetReason(), "nil 请求时 Reason 无方法/路径前缀")
	})

	t.Run("kratos错误后缀", func(t *testing.T) {
		mw, recs := newMw(t)
		tr := &khttp.Transport{}
		ctx := transport.NewServerContext(context.Background(), tr)
		khttp.SetOperation(ctx, "/demo.v1.GadgetService/Assign")
		mw.Handle(ctx, tr, kerrors.New(403, "TEST_FORBIDDEN", "forbidden for test"), 0)
		require.Len(t, *recs, 1)
		rec := (*recs)[0]
		assert.Equal(t, " failed: TEST_FORBIDDEN", rec.GetReason(),
			"nil 请求 + 失败：仅含失败后缀（reason ← kratos 错误）")
	})
}

// TestPermissionAuditLogHandleWriteFuncNil 验证 writePermissionAuditLogFunc
// 为 nil 时：记录构造照常（API/操作审计不受影响），权限审计自身不落库。
func TestPermissionAuditLogHandleWriteFuncNil(t *testing.T) {
	env := newAuditServer(t, WithWritePermissionAuditLogFunc(nil))
	env.fire(nethttp.MethodPost, "/case/9", map[string]string{
		"X-Test-Operation": "/demo.v1.GadgetService/Update",
		"Content-Type":     "application/json",
	}, `{"data":{"name":"x"}}`, "127.0.0.1:1234")
	assert.Empty(t, env.capture.permission, "空写入函数时权限审计不得落库")
	assert.Len(t, env.capture.api, 1, "其余审计不受影响")
	assert.Len(t, env.capture.operation, 1)
}

// TestPermissionAuditLogHashLogAndSignatureEdges 直调覆盖 nil 边界。
func TestPermissionAuditLogHashLogAndSignatureEdges(t *testing.T) {
	mwNoKey := NewPermissionAuditLogMiddleware(&options{})
	assert.Equal(t, "", mwNoKey.hashLog(nil))
	assert.Nil(t, mwNoKey.signature(nil))
	assert.Nil(t, mwNoKey.signature(&auditV1.PermissionAuditLog{}), "无私钥时签名必须为 nil")

	key, _, err := generateECDSAKeyPair()
	require.NoError(t, err)
	mwWithKey := NewPermissionAuditLogMiddleware(&options{ecPrivateKey: key})
	assert.Equal(t, "", mwWithKey.hashLog(nil))
	assert.Nil(t, mwWithKey.signature(nil))
}
