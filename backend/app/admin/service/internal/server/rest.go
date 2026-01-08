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

	"go-wind-admin/pkg/middleware/auth"
	applogging "go-wind-admin/pkg/middleware/logging"
)

// NewMiddleware 创建中间件
func newRestMiddleware(
	logger log.Logger,
	authenticator authnEngine.Authenticator,
	authorizer *data.Authorizer,
	operationLogRepo *data.AdminOperationLogRepo,
	loginLogRepo *data.AdminLoginLogRepo,
) []middleware.Middleware {
	var ms []middleware.Middleware
	ms = append(ms, logging.Server(logger))

	ms = append(ms, applogging.Server(
		applogging.WithWriteOperationLogFunc(func(ctx context.Context, data *adminV1.AdminOperationLog) error {
			// TODO 如果系统的负载比较小，可以同步写入数据库，否则，建议使用异步方式，即投递进队列。
			return operationLogRepo.Create(ctx, &adminV1.CreateAdminOperationLogRequest{Data: data})
		}),
		applogging.WithWriteLoginLogFunc(func(ctx context.Context, data *adminV1.AdminLoginLog) error {
			// TODO 如果系统的负载比较小，可以同步写入数据库，否则，建议使用异步方式，即投递进队列。
			return loginLogRepo.Create(ctx, &adminV1.CreateAdminLoginLogRequest{Data: data})
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
	).Match(rpc.NewRestWhiteListMatcher()).Build())

	return ms
}

// NewRestServer new an HTTP server.
func NewRestServer(
	ctx *bootstrap.Context,

	authenticator authnEngine.Authenticator, authorizer *data.Authorizer,

	operationLogRepo *data.AdminOperationLogRepo,
	loginLogRepo *data.AdminLoginLogRepo,

	authenticationService *service.AuthenticationService,
	userService *service.UserService,
	menuService *service.MenuService,
	routerService *service.RouterService,
	orgUnitService *service.OrgUnitService,
	roleService *service.RoleService,
	positionService *service.PositionService,
	dictService *service.DictService,
	adminLoginLogService *service.AdminLoginLogService,
	adminOperationLogService *service.AdminOperationLogService,
	ossService *service.OssService,
	uEditorService *service.UEditorService,
	fileService *service.FileService,
	tenantService *service.TenantService,
	taskService *service.TaskService,
	internalMessageService *service.InternalMessageService,
	internalMessageCategoryService *service.InternalMessageCategoryService,
	internalMessageRecipientService *service.InternalMessageRecipientService,
	adminLoginRestrictionService *service.AdminLoginRestrictionService,
	userProfileService *service.UserProfileService,
	apiService *service.ApiService,
	permissionService *service.PermissionService,
	permissionGroupService *service.PermissionGroupService,
	permissionAuditLogService *service.PermissionAuditLogService,
	policyEvaluationLogService *service.PolicyEvaluationLogService,
) (*http.Server, error) {
	cfg := ctx.GetConfig()

	if cfg == nil || cfg.Server == nil || cfg.Server.Rest == nil {
		return nil, nil
	}

	srv, err := rpc.CreateRestServer(cfg,
		newRestMiddleware(ctx.GetLogger(), authenticator, authorizer, operationLogRepo, loginLogRepo)...,
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
	adminV1.RegisterAdminLoginRestrictionServiceHTTPServer(srv, adminLoginRestrictionService)

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

	adminV1.RegisterAdminLoginLogServiceHTTPServer(srv, adminLoginLogService)
	adminV1.RegisterAdminOperationLogServiceHTTPServer(srv, adminOperationLogService)

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
