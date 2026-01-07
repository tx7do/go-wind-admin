package auth

import (
	"context"
	"reflect"
	"strconv"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"

	authnEngine "github.com/tx7do/kratos-authn/engine"
	authn "github.com/tx7do/kratos-authn/middleware"

	authzEngine "github.com/tx7do/kratos-authz/engine"
	authz "github.com/tx7do/kratos-authz/middleware"

	pagination "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/stringutil"
	"github.com/tx7do/go-utils/trans"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"

	"go-wind-admin/pkg/entgo/viewer"
	"go-wind-admin/pkg/jwt"
	"go-wind-admin/pkg/metadata"
)

var defaultAction = authzEngine.Action("ANY")

// Server 衔接认证和鉴权
func Server(opts ...Option) middleware.Middleware {
	op := options{
		log: log.NewHelper(log.With(log.DefaultLogger, "module", "auth.middleware")),

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
				op.log.Errorf("auth middleware: missing transport in context")
				return nil, ErrWrongContext
			}

			authnClaims, ok := authn.FromContext(ctx)
			if !ok {
				op.log.Errorf("auth middleware: missing transport in context")
				return nil, ErrWrongContext
			}

			tokenPayload, err := jwt.NewUserTokenPayloadWithClaims(authnClaims)
			if err != nil {
				op.log.Errorf("auth middleware: invalid token payload in context [%s]", err.Error())
				return nil, ErrExtractUserInfoFailed
			}

			// 校验访问令牌是否存在
			if op.accessTokenChecker != nil {
				token := tr.RequestHeader().Get(authnEngine.HeaderAuthorize)
				//op.log.Debug("auth middleware: processing request, method:", tr.Operation(), " token:", token)

				if len(token) > 7 && token[0:6] == authnEngine.BearerWord {
					token = token[7:]
				} else {
					op.log.Errorf("auth middleware: invalid token payload in context [%s]", err.Error())
					return nil, ErrExtractSubjectFailed
				}

				if !op.accessTokenChecker.IsExistAccessToken(ctx, tokenPayload.UserId, token) {
					op.log.Errorf("auth middleware: invalid token payload in context [%s]", err.Error())
					return nil, ErrAccessTokenExpired
				}
			}

			// 检查访问令牌是否被阻止
			if op.accessTokenBlocker != nil {
				token := tr.RequestHeader().Get(authnEngine.HeaderAuthorize)
				if len(token) > 7 && token[0:6] == authnEngine.BearerWord {
					token = token[7:]
				} else {
					op.log.Errorf("auth middleware: invalid token payload in context [%s]", err.Error())
					return nil, ErrExtractSubjectFailed
				}

				if op.accessTokenBlocker.IsBlockedAccessToken(ctx, tokenPayload.UserId, token) {
					op.log.Errorf("auth middleware: invalid token payload in context [%s]", err.Error())
					return nil, ErrAccessTokenExpired
				}
			}

			// 检查访问令牌是否过期
			if op.enableCheckTokenExpiration {
				if jwt.IsTokenExpired(authnClaims) {
					op.log.Errorf("auth middleware: token expired for user id [%d]", tokenPayload.UserId)
					return nil, ErrAccessTokenExpired
				}
			}

			// 检查访问令牌是否生效
			if op.enableCheckTokenValidity {
				if !jwt.IsTokenNotValidYet(authnClaims) {
					op.log.Errorf("auth middleware: token invalid for user id [%d]", tokenPayload.UserId)
					return nil, ErrAccessTokenExpired
				}
			}

			if op.injectOperatorId {
				if err = setRequestOperationId(req, tokenPayload); err != nil {
					op.log.Errorf("auth middleware: invalid token payload in context [%s]", err.Error())
					return nil, err
				}
			}
			if op.injectTenantId {
				if err = setRequestTenantId(req, tokenPayload); err != nil {
					op.log.Errorf("auth middleware: invalid token payload in context [%s]", err.Error())
					return nil, err
				}

				if err = ensurePagingRequestTenantId(req, tokenPayload); err != nil {
					op.log.Errorf("auth middleware: invalid token payload in context [%s]", err.Error())
					return nil, err
				}
			}

			if op.injectEnt {
				userViewer := viewer.NewUserViewer(
					tokenPayload.GetUserId(),
					tokenPayload.GetTenantId(),
					tokenPayload.GetOrgUnitId(),
					tokenPayload.GetDataScope(),
				)
				ctx = viewer.NewContext(ctx, userViewer)
			}

			if op.injectMetadata {
				ctx = metadata.NewOperatorMetadataContext(ctx,
					tokenPayload.GetUserId(),
					tokenPayload.GetTenantId(),
					tokenPayload.GetOrgUnitId(),
					tokenPayload.GetDataScope(),
				)
			}

			if op.enableAuthz {
				ctx, err = processAuthz(ctx, tr, tokenPayload)
				if err != nil {
					op.log.Errorf("auth middleware: invalid token payload in context [%s]", err.Error())
					return nil, err
				}
			}

			return handler(ctx, req)
		}
	}
}

func processAuthz(
	ctx context.Context,
	tr transport.Transporter,
	tokenPayload *authenticationV1.UserTokenPayload,
) (context.Context, error) {
	path := authzEngine.Resource(tr.Operation())
	action := defaultAction

	var htr *http.Transport
	var ok bool
	if htr, ok = tr.(*http.Transport); ok {
		path = authzEngine.Resource(htr.PathTemplate())
		action = authzEngine.Action(htr.Request().Method)
	}

	//log.Infof("Coming API Request: PATH[%s] ACTION[%s] USER ROLES[%v] USER ID[%d]",
	//	path, action, tokenPayload.GetRoles(), tokenPayload.UserId,
	//)

	authzClaims := authzEngine.AuthClaims{
		Subjects: trans.Ptr(tokenPayload.GetRoles()),
		Action:   trans.Ptr(action),
		Resource: trans.Ptr(path),
	}

	ctx = authz.NewContext(ctx, &authzClaims)

	return ctx, nil
}

func FromContext(ctx context.Context) (*authenticationV1.UserTokenPayload, error) {
	claims, ok := authnEngine.AuthClaimsFromContext(ctx)
	if !ok {
		return nil, ErrMissingJwtToken
	}

	return jwt.NewUserTokenPayloadWithClaims(claims)
}

func setRequestOperationId(req interface{}, payload *authenticationV1.UserTokenPayload) error {
	if req == nil {
		return ErrInvalidRequest
	}

	v := reflect.ValueOf(req).Elem()
	field := v.FieldByName("OperatorId")
	if field.IsValid() && field.Kind() == reflect.Ptr {
		field.Set(reflect.ValueOf(&payload.UserId))
	}

	return nil
}

func setRequestTenantId(req interface{}, payload *authenticationV1.UserTokenPayload) error {
	if req == nil {
		return ErrInvalidRequest
	}

	v := reflect.ValueOf(req).Elem()
	field := v.FieldByName("tenantId")
	if field.IsValid() && field.Kind() == reflect.Ptr {
		field.Set(reflect.ValueOf(&payload.TenantId))
	}

	return nil
}

func ensurePagingRequestTenantId(req interface{}, payload *authenticationV1.UserTokenPayload) error {
	if paging, ok := req.(*pagination.PagingRequest); ok && payload.GetTenantId() > 0 {
		if paging.Query != nil {
			newStr := stringutil.ReplaceJSONField("tenantId|tenant_id", strconv.Itoa(int(payload.GetTenantId())), paging.GetQuery())
			paging.Query = trans.Ptr(newStr)
		}
		if paging.OrQuery != nil {
			newStr := stringutil.ReplaceJSONField("tenantId|tenant_id", strconv.Itoa(int(payload.GetTenantId())), paging.GetOrQuery())
			paging.OrQuery = trans.Ptr(newStr)
		}
	}
	return nil
}
