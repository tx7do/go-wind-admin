package auth

import (
	"context"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	http "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/tx7do/go-crud/viewer"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"

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

			// 代租户操作（X-Tenant-Id）：平台管理员可携带该头把本请求的生效
			// 租户切到所选租户——后续的租户级访问检查、viewer 数据面、上下文
			// 注入、租户配额全部按生效租户行事；token 未动（payload 克隆后
			// 覆写，避免污染 checker 缓存的共享体），审计里的操作者仍是真实
			// 平台用户。头缺失即平台全域视图（现状）；租户成员与非平台管理
			// 员的该头一律忽略（其租户由 token 决定）。
			if tokenPayload.GetTenantId() == 0 && isPlatformAdmin(tokenPayload) {
				if eff, ok := impersonatedTenantID(ctx, tr); ok {
					tokenPayload = cloneTokenPayload(tokenPayload)
					tokenPayload.TenantId = trans.Ptr(eff)
				}
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

				userViewer := viewer.NewUserContext(
					int64(tokenPayload.GetUserId()),
					int64(tokenPayload.GetTenantId()),
					int64(tokenPayload.GetOrgUnitId()),
					traceID,
					appViewer.BuildDataScopes(
						tokenPayload.GetDataScopes(),
						tokenPayload.GetDataScopeUnitIds(),
						tokenPayload.GetDataScope(),
					),
				)
				ctx = viewer.WithContext(ctx, userViewer)
			}

			if op.injectMetadata {
				ctx, err = metadata.NewContext(ctx,
					&authenticationV1.OperatorMetadata{
						UserId:           int64(tokenPayload.GetUserId()),
						TenantId:         int64(tokenPayload.GetTenantId()),
						OrgUnitId:        int64(tokenPayload.GetOrgUnitId()),
						DataScope:        tokenPayload.GetDataScope(),
						DataScopes:       tokenPayload.GetDataScopes(),
						DataScopeUnitIds: tokenPayload.GetDataScopeUnitIds(),
					},
				)
				if err != nil {
					op.log.Errorf(ctx, "auth middleware: invalid token payload in context [%s]", err.Error())
					return nil, err
				}
			}

			// 租户级访问检查：仅对租户用户（tenantId>0）生效，平台管理员（tenantId=0）直接放行。
			// 检查项：租户状态（OFF/EXPIRED/FREEZE 拒绝）、到期只读策略（仅放行 GET/HEAD/OPTIONS）、
			// 套餐模块白名单（请求所属业务模块不在白名单则拒绝）、
			// 套餐配额（API_CALL 调用量达套餐上限则拒绝）。
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

// isPlatformAdmin 判断载荷是否平台管理员：平台管理员标志按角色码值承载
// （platform:admin），与登录链路的判定同口径。
func isPlatformAdmin(payload *authenticationV1.UserTokenPayload) bool {
	for _, r := range payload.GetRoles() {
		if r == "platform:admin" {
			return true
		}
	}
	return false
}

// impersonatedTenantID 从请求头取 X-Tenant-Id 并解析为正整数租户 ID；
// 缺失/空白/非数字/零/负数一律不构成代租户——宽松失败，垃圾值不得变成
// 另一个身份。
func impersonatedTenantID(ctx context.Context, tr transport.Transporter) (int64, bool) {
	var raw string
	if htr, ok := tr.(*http.Transport); ok && htr.Request() != nil {
		raw = htr.Request().Header.Get("X-Tenant-Id")
	} else if h := tr.RequestHeader(); h != nil {
		raw = h.Get("X-Tenant-Id")
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	eff, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || eff <= 0 {
		return 0, false
	}
	return eff, true
}

// cloneTokenPayload 深拷贝载荷：覆写生效租户前隔离 checker 缓存的共享体。
func cloneTokenPayload(payload *authenticationV1.UserTokenPayload) *authenticationV1.UserTokenPayload {
	return proto.Clone(payload).(*authenticationV1.UserTokenPayload)
}
