package auth

import (
	"context"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	http "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/tx7do/go-crud/viewer"
	"go.opentelemetry.io/otel/trace"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"

	authnEngine "github.com/tx7do/kratos-authn/engine"
	authzEngine "github.com/tx7do/kratos-authz/engine"

	appViewer "go-wind-admin/pkg/entgo/viewer"
	"go-wind-admin/pkg/metadata"
)

var defaultAction = authzEngine.Action("ANY")

// Server 衔接认证和鉴权
func Server(opts ...Option) middleware.Middleware {
	op := options{
		log: bLogger.NewHelper(bLogger.GetLogger().With("module", "auth/middleware")),

		injectOperatorId: false,
		injectTenantId:   false,
		enableAuthz:      true,
		injectEnt:        true,
		injectMetadata:   true,
	}
	for _, o := range opts {
		o(&op)
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				op.log.Errorf(ctx, "auth middleware: missing transport in context")
				return nil, ErrWrongContext
			}

			token, err := authnEngine.AuthFromMD(ctx, authnEngine.BearerWord, authnEngine.ContextTypeKratosMetaData)
			if err != nil {
				return nil, ErrMissingBearerToken
			}

			if op.accessTokenChecker == nil {
				op.log.Errorf(ctx, "auth middleware: access token checker is not configured")
				return nil, ErrAccessTokenCheckerNotConfigured
			}

			var tokenPayload *authenticationV1.UserTokenPayload
			var valid bool
			if valid, tokenPayload = op.accessTokenChecker.IsValidAccessToken(ctx, token, false); !valid {
				op.log.Errorf(ctx, "auth middleware: invalid access token")
				return nil, ErrAccessTokenExpired
			}

			ctx = NewContext(ctx, tokenPayload)

			if op.injectOperatorId {
				if err = setRequestOperationId(req, tokenPayload); err != nil {
					op.log.Errorf(ctx, "auth middleware: invalid token payload in context [%s]", err.Error())
					return nil, err
				}
			}
			if op.injectTenantId {
				if err = setRequestTenantId(req, tokenPayload); err != nil {
					op.log.Errorf(ctx, "auth middleware: invalid token payload in context [%s]", err.Error())
					return nil, err
				}
			}

			if op.injectEnt {
				var traceID string
				spanContext := trace.SpanContextFromContext(ctx)
				if spanContext.HasTraceID() {
					traceID = spanContext.TraceID().String()
				}

				userViewer := appViewer.NewUserViewer(
					uint64(tokenPayload.GetUserId()),
					uint64(tokenPayload.GetTenantId()),
					uint64(tokenPayload.GetOrgUnitId()),
					traceID,
					tokenPayload.GetDataScope(),
				)
				ctx = viewer.WithContext(ctx, userViewer)
			}

	if op.injectMetadata {
		ctx, err = metadata.NewContext(ctx,
			&authenticationV1.OperatorMetadata{
				UserId:    uint64(tokenPayload.GetUserId()),
				TenantId:  uint64(tokenPayload.GetTenantId()),
				OrgUnitId: uint64(tokenPayload.GetOrgUnitId()),
				DataScope: tokenPayload.GetDataScope(),
			},
		)
		if err != nil {
			op.log.Errorf(ctx, "auth middleware: invalid token payload in context [%s]", err.Error())
			return nil, err
		}
	}

	// 租户级访问检查：仅对租户用户（tenantId>0）生效，平台管理员（tenantId=0）直接放行。
	// 检查项：租户状态（OFF/EXPIRED/FREEZE 拒绝）、到期只读策略（仅放行 GET/HEAD/OPTIONS）、
	// 套餐模块白名单（请求所属业务模块不在白名单则拒绝）。
	if op.tenantAccessChecker != nil && tokenPayload.GetTenantId() > 0 {
		var path, method string
		if htr, ok := tr.(*http.Transport); ok {
			path = htr.PathTemplate()
			method = htr.Request().Method
		} else {
			path = tr.Operation()
			method = ""
		}
		if err := op.tenantAccessChecker.CheckTenantAccess(ctx, tokenPayload.GetTenantId(), path, method); err != nil {
			return nil, err
		}
	}

	if op.enableAuthz {
				ctx, err = processAuthz(ctx, tr, tokenPayload)
				if err != nil {
					op.log.Errorf(ctx, "auth middleware: invalid token payload in context [%s]", err.Error())
					return nil, err
				}
			}

			return handler(ctx, req)
		}
	}
}
