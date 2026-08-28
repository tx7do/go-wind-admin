package server

import (
	nethttp "net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tx7do/go-utils/trans"

	authzEngine "github.com/tx7do/kratos-authz/engine"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

// newRecWithUser 构造带操作者归属的日志记录，供快照测试。
func newRecWithUser(uid, tenant uint32) *permissionV1.PolicyEvaluationLog {
	return &permissionV1.PolicyEvaluationLog{
		UserId:   trans.Ptr(uid),
		TenantId: trans.Ptr(tenant),
	}
}

func TestTraceIDFromHeader(t *testing.T) {
	// W3C traceparent：取 32 位 trace-id 段
	h := nethttp.Header{}
	h.Set("traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
	assert.Equal(t, "0af7651916cd43dd8448eb211c80319c", traceIDFromHeader(h))

	// traceparent 畸形时回退 X-Request-Id
	h2 := nethttp.Header{}
	h2.Set("traceparent", "garbage")
	h2.Set("X-Request-Id", "req-abc-123")
	assert.Equal(t, "req-abc-123", traceIDFromHeader(h2))

	// 无 traceparent 直接用 X-Request-Id（与 api/operation 审计 request_id 同源）
	h3 := nethttp.Header{}
	h3.Set("X-Request-Id", "req-xyz")
	assert.Equal(t, "req-xyz", traceIDFromHeader(h3))

	// 两者皆无
	assert.Equal(t, "", traceIDFromHeader(nethttp.Header{}))
}

func TestEvaluationContextJSON(t *testing.T) {
	got := evaluationContextJSON("noop",
		authzEngine.Subject("platform:admin"),
		authzEngine.Action("GET"),
		authzEngine.Resource("/admin/v1/roles"),
		authzEngine.Project(""),
		nil)
	assert.JSONEq(t, `{
		"engine": "noop",
		"subject": "platform:admin",
		"action": "GET",
		"resource": "/admin/v1/roles"
	}`, got)

	// project 与 user/tenant 归属一并入快照
	got2 := evaluationContextJSON("casbin",
		authzEngine.Subject("ops"),
		authzEngine.Action("PUT"),
		authzEngine.Resource("/admin/v1/roles/3"),
		authzEngine.Project("main"),
		newRecWithUser(7, 2))
	assert.JSONEq(t, `{
		"engine": "casbin",
		"subject": "ops",
		"action": "PUT",
		"resource": "/admin/v1/roles/3",
		"project": "main",
		"userId": 7,
		"tenantId": 2
	}`, got2)
}
