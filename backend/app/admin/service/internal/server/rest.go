package server

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport/http"

	authnEngine "github.com/tx7do/kratos-authn/engine"
	authn "github.com/tx7do/kratos-authn/middleware"

	authz "github.com/tx7do/kratos-authz/middleware"

	swaggerUI "github.com/tx7do/kratos-swagger-ui"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/kratos-bootstrap/rpc"

	"go-wind-admin/app/admin/service/cmd/server/assets"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/service"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"

	"go-wind-admin/pkg/middleware/auth"
	applogging "go-wind-admin/pkg/middleware/logging"
)

// NewRestMiddleware 创建中间件
func NewRestMiddleware(
	ctx *bootstrap.Context,
	authenticator authnEngine.Authenticator,
	authorizer *data.Authorizer,
	apiAuditLogRepo *data.ApiAuditLogRepo,
	loginLogRepo *data.LoginAuditLogRepo,
) []middleware.Middleware {
	var ms []middleware.Middleware
	ms = append(ms, logging.Server(ctx.GetLogger()))

	ms = append(ms, applogging.Server(
		applogging.WithWriteApiLogFunc(func(ctx context.Context, data *auditV1.ApiAuditLog) error {
			// TODO 如果系统的负载比较小，可以同步写入数据库，否则，建议使用异步方式，即投递进队列。
			return apiAuditLogRepo.Create(ctx, &auditV1.CreateApiAuditLogRequest{Data: data})
		}),
		applogging.WithWriteLoginLogFunc(func(ctx context.Context, data *auditV1.LoginAuditLog) error {
			// TODO 如果系统的负载比较小，可以同步写入数据库，否则，建议使用异步方式，即投递进队列。
			return loginLogRepo.Create(ctx, &auditV1.CreateLoginAuditLogRequest{Data: data})
		}),
	))

	// add white list for authentication.
	rpc.AddWhiteList(
		adminV1.OperationAuthenticationServiceLogin,
	)

	ms = append(ms, selector.Server(
		authn.Server(authenticator),
		auth.Server(),
		authz.Server(authorizer.Engine()),
	).
		Match(rpc.NewRestWhiteListMatcher()).
		Build(),
	)

	return ms
}

// NewRestServer new an REST server.
func NewRestServer(
	ctx *bootstrap.Context,

	middlewares []middleware.Middleware,
	authorizer *data.Authorizer,

	authenticationService *service.AuthenticationService,
	routerService *service.RouterService,
	dictService *service.DictService,
	ossService *service.OssService,
	uEditorService *service.UEditorService,
	fileService *service.FileService,
	taskService *service.TaskService,
	loginPolicyService *service.LoginPolicyService,

	tenantService *service.TenantService,
	userService *service.UserService,
	userProfileService *service.UserProfileService,
	roleService *service.RoleService,
	positionService *service.PositionService,
	orgUnitService *service.OrgUnitService,

	menuService *service.MenuService,
	apiService *service.ApiService,
	permissionService *service.PermissionService,
	permissionGroupService *service.PermissionGroupService,
	permissionAuditLogService *service.PermissionAuditLogService,
	policyEvaluationLogService *service.PolicyEvaluationLogService,

	loginAuditLogService *service.LoginAuditLogService,
	apiAuditLogService *service.ApiAuditLogService,
	operationAuditLogService *service.OperationAuditLogService,
	dataAccessAuditLogService *service.DataAccessAuditLogService,

	internalMessageService *service.InternalMessageService,
	internalMessageCategoryService *service.InternalMessageCategoryService,
	internalMessageRecipientService *service.InternalMessageRecipientService,

) (*http.Server, error) {
	cfg := ctx.GetConfig()

	if cfg == nil || cfg.Server == nil || cfg.Server.Rest == nil {
		return nil, nil
	}

	srv, err := rpc.CreateRestServer(cfg,
		middlewares...,
	)
	if err != nil {
		return nil, err
	}

	apiService.RegisterRouteWalker(srv)

	adminV1.RegisterAuthenticationServiceHTTPServer(srv, authenticationService)

	adminV1.RegisterUserProfileServiceHTTPServer(srv, userProfileService)

	adminV1.RegisterRouterServiceHTTPServer(srv, routerService)
	adminV1.RegisterDictServiceHTTPServer(srv, dictService)
	adminV1.RegisterTaskServiceHTTPServer(srv, taskService)
	adminV1.RegisterLoginPolicyServiceHTTPServer(srv, loginPolicyService)

	adminV1.RegisterApiServiceHTTPServer(srv, apiService)
	adminV1.RegisterMenuServiceHTTPServer(srv, menuService)
	adminV1.RegisterPermissionServiceHTTPServer(srv, permissionService)
	adminV1.RegisterPermissionGroupServiceHTTPServer(srv, permissionGroupService)
	adminV1.RegisterPolicyEvaluationLogServiceHTTPServer(srv, policyEvaluationLogService)
	adminV1.RegisterPermissionAuditLogServiceHTTPServer(srv, permissionAuditLogService)

	adminV1.RegisterUserServiceHTTPServer(srv, userService)
	adminV1.RegisterOrgUnitServiceHTTPServer(srv, orgUnitService)
	adminV1.RegisterRoleServiceHTTPServer(srv, roleService)
	adminV1.RegisterPositionServiceHTTPServer(srv, positionService)
	adminV1.RegisterTenantServiceHTTPServer(srv, tenantService)

	adminV1.RegisterLoginAuditLogServiceHTTPServer(srv, loginAuditLogService)
	adminV1.RegisterApiAuditLogServiceHTTPServer(srv, apiAuditLogService)
	adminV1.RegisterOperationAuditLogServiceHTTPServer(srv, operationAuditLogService)
	adminV1.RegisterDataAccessAuditLogServiceHTTPServer(srv, dataAccessAuditLogService)

	adminV1.RegisterOssServiceHTTPServer(srv, ossService)
	adminV1.RegisterFileServiceHTTPServer(srv, fileService)

	adminV1.RegisterUEditorServiceHTTPServer(srv, uEditorService)

	adminV1.RegisterInternalMessageServiceHTTPServer(srv, internalMessageService)
	adminV1.RegisterInternalMessageCategoryServiceHTTPServer(srv, internalMessageCategoryService)
	adminV1.RegisterInternalMessageRecipientServiceHTTPServer(srv, internalMessageRecipientService)

	registerFileUploadHandler(srv, ossService)
	registerUEditorUploadHandler(srv, uEditorService)

	if cfg.GetServer().GetRest().GetEnableSwagger() {
		swaggerUI.RegisterSwaggerUIServerWithOption(
			srv,
			swaggerUI.WithTitle("GoWind Admin"),
			swaggerUI.WithMemoryData(assets.OpenApiData, "yaml"),
		)
	}

	if authorizer != nil {
		if err = authorizer.ResetPolicies(ctx.Context()); err != nil {
			log.Errorf("reset policies error: %v", err)
		}
	}

	return srv, nil
}
