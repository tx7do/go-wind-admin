package server

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/transport"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/tx7do/go-utils/trans"
	authzEngine "github.com/tx7do/kratos-authz/engine"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	appViewer "go-wind-admin/pkg/entgo/viewer"
	"go-wind-admin/pkg/middleware/auth"
)

// evalLoggingEngine 包装 authz 引擎，把每次 IsAuthorized 评估判定落
// PolicyEvaluationLog（sys_policy_evaluation_logs）。判定信息与日志 schema 对应：
// subject=角色码、action=HTTP 方法、resource=路径模板（由 auth 中间件构建 claims）。
// 落库失败只吞错，不影响鉴权主流程。多角色请求每个角色评估各落一行。
type evalLoggingEngine struct {
	authzEngine.Authorizer
	repo *data.PolicyEvaluationLogRepo
}

// newEvalLoggingEngine 包装引擎；inner 或 repo 为 nil 时原样返回（不埋点）。
func newEvalLoggingEngine(inner authzEngine.Authorizer, repo *data.PolicyEvaluationLogRepo) authzEngine.Authorizer {
	if inner == nil || repo == nil {
		return inner
	}
	return &evalLoggingEngine{Authorizer: inner, repo: repo}
}

func (e *evalLoggingEngine) IsAuthorized(ctx context.Context, subject authzEngine.Subject, action authzEngine.Action, resource authzEngine.Resource, project authzEngine.Project) (bool, error) {
	start := time.Now()
	allowed, err := e.Authorizer.IsAuthorized(ctx, subject, action, resource, project)

	rec := &permissionV1.PolicyEvaluationLog{
		RequestPath:   trans.Ptr(string(resource)),
		RequestMethod: trans.Ptr(string(action)),
		Result:        trans.Ptr(allowed && err == nil),
		EffectDetails: trans.Ptr(e.describe(start, subject, project, err, allowed)),
	}

	if ut, utErr := auth.FromContext(ctx); utErr == nil && ut != nil {
		rec.UserId = trans.Ptr(ut.UserId)
		rec.TenantId = ut.TenantId
	}
	if ip := clientIPFromContext(ctx); ip != "" {
		rec.IpAddress = trans.Ptr(ip)
	}

	// SystemViewer：评估发生在 authz 阶段，无租户 viewer 上下文，落库需系统视角。
	_ = e.repo.Create(appViewer.NewSystemViewerContext(ctx), &permissionV1.CreatePolicyEvaluationLogRequest{Data: rec})

	return allowed, err
}

// describe 生成 effect_details：引擎、角色、项目、耗时，err/拒绝时附原因。
func (e *evalLoggingEngine) describe(start time.Time, subject authzEngine.Subject, project authzEngine.Project, err error, allowed bool) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "engine=%s subject=%s", e.Authorizer.Name(), subject)
	if project != "" {
		fmt.Fprintf(&sb, " project=%s", project)
	}
	fmt.Fprintf(&sb, " latency=%dms", time.Since(start).Milliseconds())
	switch {
	case err != nil:
		fmt.Fprintf(&sb, "; error: %v", err)
	case !allowed:
		sb.WriteString("; denied")
	default:
		sb.WriteString("; allowed")
	}
	return sb.String()
}

// clientIPFromContext 从 transport 尽力取客户端 IP（代理头优先）。
func clientIPFromContext(ctx context.Context) string {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return ""
	}
	htr, ok := tr.(*khttp.Transport)
	if !ok || htr == nil || htr.Request() == nil {
		return ""
	}
	req := htr.Request()
	if ip := strings.TrimSpace(req.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if host, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		return host
	}
	return req.RemoteAddr
}
