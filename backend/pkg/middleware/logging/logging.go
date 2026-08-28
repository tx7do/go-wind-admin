package logging

import (
	"bytes"
	"context"
	"io"
	nethttp "net/http"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	"go-wind-admin/pkg/audit"
)

// reqBodyKey 携带 pre-handler 快照的 JSON 写请求体，供 post-handler 审计解析。
type reqBodyKey struct{}

// sessionOnlyOperations 是会话维护类端点：虽为 POST 但不构成业务/权限变更。
// refresh-token 会被前端定时刷新器周期触发、login 已有专门的登录审计，
// 记入操作/权限审计只是持续噪音。
var sessionOnlyOperations = map[string]bool{
	adminV1.OperationAuthenticationServiceLogin:             true,
	adminV1.OperationAuthenticationServiceRegisterUser:      true,
	adminV1.OperationAuthenticationServiceRefreshToken:      true,
	adminV1.OperationAuthenticationServiceLogout:            true,
	adminV1.OperationMfaServiceVerifyMFAChallenge:           true,
}

// maxBodySnapshot 快照上限：CRUD JSON 体远小于此；超长时剩余部分透传原流。
const maxBodySnapshot = 64 << 10

// bodySnapshotFromContext 取 pre-handler 快照的请求体（无则 nil）。
func bodySnapshotFromContext(ctx context.Context) []byte {
	b, _ := ctx.Value(reqBodyKey{}).([]byte)
	return b
}

// replayBody 把快照与剩余原始流拼回可重放 body，Close 透传原 body。
type replayBody struct {
	io.Reader
	closer io.Closer
}

func (b *replayBody) Close() error { return b.closer.Close() }

// snapshotWriteBody 对 JSON 写请求体做限长快照并重置为可重放流。
// 仅 POST/PUT/PATCH/DELETE 且 Content-Type 含 application/json（避开 multipart
// 文件上传等大/非结构体）；post-handler 审计（权限日志取目标名称）从 ctx 读取。
func snapshotWriteBody(ctx context.Context, req *nethttp.Request) context.Context {
	if req == nil || req.Body == nil {
		return ctx
	}
	switch req.Method {
	case "POST", "PUT", "PATCH", "DELETE":
	default:
		return ctx
	}
	if ct := req.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		return ctx
	}

	buf := make([]byte, maxBodySnapshot)
	n, _ := io.ReadFull(req.Body, buf)
	if n == 0 {
		req.Body = io.NopCloser(bytes.NewReader(nil))
		return ctx
	}
	snap := buf[:n]
	var rdr io.Reader = bytes.NewReader(snap)
	if n == maxBodySnapshot {
		rdr = io.MultiReader(bytes.NewReader(snap), req.Body)
	}
	req.Body = &replayBody{Reader: rdr, closer: req.Body}
	return context.WithValue(ctx, reqBodyKey{}, snap)
}

// Server is an server logging middleware.
func Server(opts ...Option) middleware.Middleware {
	op := options{
		loginOperations: []string{
			adminV1.OperationAuthenticationServiceLogin,
			// MFA 登录挑战验证也按登录事件审计：它是登录流程的二次验证阶段，
			// 审计 schema 已预埋 mfa_status / Status.PARTIAL 等字段支持此语义。
			adminV1.OperationMfaServiceVerifyMFAChallenge,
		},
		logoutOperation: adminV1.OperationAuthenticationServiceLogout,
	}
	for _, o := range opts {
		o(&op)
	}

	if op.ecPrivateKey == nil || op.ecPublicKey == nil {
		op.ecPrivateKey, op.ecPublicKey, _ = generateECDSAKeyPair()
	}

	loginAuditLogMiddleware := NewLoginAuditLogMiddleware(&op)
	apiAuditLogMiddleware := NewApiAuditLogMiddleware(&op)
	operationAuditLogMiddleware := NewOperationAuditLogMiddleware(&op)
	permissionAuditLogMiddleware := NewPermissionAuditLogMiddleware(&op)
	dataAccessAuditLogMiddleware := NewDataAccessAuditLogMiddleware(&op)

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			startTime := time.Now()

			// DataAccessAuditLog: pre-handler 植入 accumulator，driver wrapper
			// 在 handler 内执行 SQL 时向其 append 事件。post-handler 取出落库。
			var dataAccessEvents []audit.AuditEvent
			ctx = context.WithValue(ctx, audit.AccumulatorKey(), &dataAccessEvents)

			// pre-handler 快照 JSON 写请求体（重置为可重放流），post-handler
			// 权限审计从中提取目标名称。
			if tr, ok := transport.FromServerContext(ctx); ok {
				if htr, ok := tr.(*http.Transport); ok {
					ctx = snapshotWriteBody(ctx, htr.Request())
				}
			}

			reply, err = handler(ctx, req)

			// 统计耗时
			latencyMs := time.Since(startTime).Milliseconds()

			if tr, ok := transport.FromServerContext(ctx); ok {
				var htr *http.Transport
				if htr, ok = tr.(*http.Transport); ok {
				// 审计落库阶段标记：本阶段各审计表自身的 INSERT 不再被 driver
				// wrapper 采集，防递归、防审计表写入噪音进入 data_access 审计。
				ctx = context.WithValue(ctx, audit.SinkKey(), true)
				loginAuditLogMiddleware.Handle(ctx, htr, err)
				apiAuditLogMiddleware.Handle(ctx, htr, err, latencyMs)
				operationAuditLogMiddleware.Handle(ctx, htr, err, latencyMs)
				permissionAuditLogMiddleware.Handle(ctx, htr, err, latencyMs)
				dataAccessAuditLogMiddleware.Handle(ctx, htr, err, latencyMs)
				}
			}

			return
		}
	}
}
