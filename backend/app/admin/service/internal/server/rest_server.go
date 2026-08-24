package server

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	"github.com/go-kratos/kratos/v2/transport/http"

	authz "github.com/tx7do/kratos-authz/middleware"

	swaggerUI "github.com/tx7do/kratos-swagger-ui"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/kratos-bootstrap/rpc"

	"go-wind-admin/app/admin/service/cmd/server/assets"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/service"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"

	"go-wind-admin/pkg/authorizer"
	appViewer "go-wind-admin/pkg/entgo/viewer"
	"go-wind-admin/pkg/middleware/auth"
	applogging "go-wind-admin/pkg/middleware/logging"
)

// NewRestMiddleware 创建中间件
func NewRestMiddleware(
	ctx *bootstrap.Context,
	accessTokenChecker auth.AccessTokenChecker,
	tenantAccessChecker auth.TenantAccessChecker,
	authorizer *authorizer.Authorizer,
	apiAuditLogRepo *data.ApiAuditLogRepo,
	loginLogRepo *data.LoginAuditLogRepo,
) []middleware.Middleware {
	var ms []middleware.Middleware
	// recovery 必须置于链首：任何中间件/handler 的 panic（如审计日志解析畸形 JWT）
	// 兜底为 500，避免崩溃请求 goroutine、被用作未认证 DoS。
	ms = append(ms, recovery.Recovery())
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

	// 输入校验：对所有 RPC（含白名单内的 login/register/refresh/MFA）调用生成代码的
	// Validate()。放 selector 外，否则白名单路由会被跳过——而它们恰是最需要校验入参的。
	// 当前业务 proto 尚未补 (validate.rules)，多数 Validate() 返回 nil；补规则后再生效。
	ms = append(ms, validate.Validator())

	// add white list for authentication.
	rpc.AddWhiteList(
		adminV1.OperationAuthenticationServiceLogin,
		// 注册接口在 proto 中声明免鉴权（security:{}）且登录页注册流程无 token，
		// 此前白名单遗漏导致注册接口 401（前端注册页从未可用）
		adminV1.OperationAuthenticationServiceRegisterUser,
		adminV1.OperationAuthenticationServiceGenerateCaptcha,
		adminV1.OperationAuthenticationServiceVerifyCaptcha,
		// 刷新令牌接口免鉴权：refresh token 现以 HttpOnly Cookie 传输且为自描述 JWT，
		// 可脱离 access token 独立鉴权。页面刷新后 access token 丢失时，前端凭
		// refresh cookie 静默恢复会话，不再强制重新登录。
		adminV1.OperationAuthenticationServiceRefreshToken,
		// MFA 登录挑战验证免鉴权：operation_id 由登录流程签发，见 doGrantTypePassword 的 MFA 闸门。
		// 仅此一个 MFA RPC 免鉴权；管理侧 RPC（GetMFAStatus 等）走正常 auth+authz。
		adminV1.OperationMfaServiceVerifyMFAChallenge,
		//OperationFileTransferServiceDownloadFile,
		//OperationFileTransferServicePostUploadFile,
		//OperationFileTransferServicePutUploadFile,
	)

	ms = append(ms, selector.Server(
			auth.Server(
				auth.WithAccessTokenChecker(accessTokenChecker),
				auth.WithTenantAccessChecker(tenantAccessChecker),
				auth.WithInjectMetadata(false),
				auth.WithInjectEnt(true),
			),
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
	authorizer *authorizer.Authorizer,

	authenticationService *service.AuthenticationService,
	mfaService *service.MfaService,
	loginPolicyService *service.LoginPolicyService,

	portalService *service.AdminPortalService,
	taskService *service.TaskService,

	fileService *service.FileService,
	fileTransferService *service.FileTransferService,

	dictTypeService *service.DictTypeService,
	dictEntryService *service.DictEntryService,
	languageService *service.LanguageService,

	tenantService *service.TenantService,
	planService *service.PlanService,
	planQuotaService *service.PlanQuotaService,
	planModuleService *service.PlanModuleService,
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
	redisCacheMonitorService *service.RedisCacheMonitorService,
	dashboardService *service.DashboardService,

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

	adminV1.RegisterMfaServiceHTTPServer(srv, mfaService)

	adminV1.RegisterUserProfileServiceHTTPServer(srv, userProfileService)

	adminV1.RegisterAdminPortalServiceHTTPServer(srv, portalService)
	adminV1.RegisterTaskServiceHTTPServer(srv, taskService)
	adminV1.RegisterLoginPolicyServiceHTTPServer(srv, loginPolicyService)

	adminV1.RegisterDictTypeServiceHTTPServer(srv, dictTypeService)
	adminV1.RegisterDictEntryServiceHTTPServer(srv, dictEntryService)
	adminV1.RegisterLanguageServiceHTTPServer(srv, languageService)

	adminV1.RegisterApiServiceHTTPServer(srv, apiService)
	adminV1.RegisterMenuServiceHTTPServer(srv, menuService)
	adminV1.RegisterPermissionServiceHTTPServer(srv, permissionService)
	adminV1.RegisterPermissionGroupServiceHTTPServer(srv, permissionGroupService)
	adminV1.RegisterPolicyEvaluationLogServiceHTTPServer(srv, policyEvaluationLogService)
	adminV1.RegisterPermissionAuditLogServiceHTTPServer(srv, permissionAuditLogService)

	adminV1.RegisterUserServiceHTTPServer(srv, adminV1.RedactedUserServiceServer(&userServiceServerAdapter{UserServiceHTTPServer: userService}, nil))
	adminV1.RegisterOrgUnitServiceHTTPServer(srv, orgUnitService)
	adminV1.RegisterRoleServiceHTTPServer(srv, roleService)
	adminV1.RegisterPositionServiceHTTPServer(srv, positionService)
	adminV1.RegisterTenantServiceHTTPServer(srv, tenantService)
	adminV1.RegisterPlanServiceHTTPServer(srv, planService)
	adminV1.RegisterPlanQuotaServiceHTTPServer(srv, planQuotaService)
	adminV1.RegisterPlanModuleServiceHTTPServer(srv, planModuleService)

	adminV1.RegisterLoginAuditLogServiceHTTPServer(srv, loginAuditLogService)
	adminV1.RegisterApiAuditLogServiceHTTPServer(srv, apiAuditLogService)
	adminV1.RegisterOperationAuditLogServiceHTTPServer(srv, operationAuditLogService)
	adminV1.RegisterDataAccessAuditLogServiceHTTPServer(srv, dataAccessAuditLogService)
	adminV1.RegisterRedisCacheMonitorServiceHTTPServer(srv, redisCacheMonitorService)
	adminV1.RegisterDashboardServiceHTTPServer(srv, dashboardService)

	adminV1.RegisterFileServiceHTTPServer(srv, fileService)

	// 注册文件传输服务，用于处理文件上传下载等功能
	// TODO 它不能够使用代码生成器生成的Handler，需要手动注册。代码生成器生成的Handler无法处理文件上传下载的请求。
	// 但，代码生成器生成代码可以提供给OpenAPI使用。
	registerFileTransferServiceHandler(srv, fileTransferService)

	adminV1.RegisterInternalMessageServiceHTTPServer(srv, internalMessageService)
	adminV1.RegisterInternalMessageCategoryServiceHTTPServer(srv, internalMessageCategoryService)
	adminV1.RegisterInternalMessageRecipientServiceHTTPServer(srv, internalMessageRecipientService)

	if cfg.GetServer().GetRest().GetEnableSwagger() {
		swaggerUI.RegisterSwaggerUIServerWithOption(
			srv,
			swaggerUI.WithTitle("GoWind Admin"),
			swaggerUI.WithMemoryData(assets.OpenApiData, "yaml"),
		)
	}

	if authorizer != nil {
		if err = authorizer.ResetPolicies(appViewer.NewSystemViewerContext(ctx.Context())); err != nil {
			log.Errorf("reset policies error: %v", err)
		}
	}

	return srv, nil
}

// userServiceServerAdapter 将 UserServiceHTTPServer 桥接为 UserServiceServer，
// 使其可以被 RedactedUserServiceServer 包装以实现 HTTP 路径的脱敏。
type userServiceServerAdapter struct {
	adminV1.UnsafeUserServiceServer
	adminV1.UserServiceHTTPServer
}
