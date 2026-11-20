package server

import (
	"context"
	nethttp "net/http"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"

	authz "github.com/tx7do/kratos-authz/middleware"

	swaggerUI "github.com/tx7do/kratos-swagger-ui"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/tx7do/kratos-bootstrap/rpc"

	"go-wind-admin/app/admin/service/cmd/server/assets"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/service"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"

	"github.com/tx7do/go-crud/viewer"
	"github.com/tx7do/go-utils/authorizer"
	"github.com/tx7do/go-utils/ossutil"
	"go-wind-admin/pkg/middleware/auth"
	applogging "go-wind-admin/pkg/middleware/logging"
	"go-wind-admin/pkg/task"
)

// NewRestMiddleware 创建中间件
func NewRestMiddleware(
	ctx *bootstrap.Context,
	accessTokenChecker auth.AccessTokenChecker,
	tenantAccessChecker auth.TenantAccessChecker,
	authorizer *authorizer.Authorizer,
	apiAuditLogRepo *data.ApiAuditLogRepo,
	loginLogRepo *data.LoginAuditLogRepo,
	operationAuditLogRepo *data.OperationAuditLogRepo,
	permissionAuditLogRepo *data.PermissionAuditLogRepo,
	dataAccessAuditLogRepo *data.DataAccessAuditLogRepo,
	policyEvaluationLogRepo *data.PolicyEvaluationLogRepo,
	taskService *service.TaskService,
) []middleware.Middleware {
	var ms []middleware.Middleware
	// recovery 必须置于链首：任何中间件/handler 的 panic（如审计日志解析畸形 JWT）
	// 兜底为 500，避免崩溃请求 goroutine、被用作未认证 DoS。
	ms = append(ms, recovery.Recovery())
	// 注意：请求体上限不能放在这个中间件链里。kratos 的中间件由 handler 内部
	// 调用（ctx.Middleware），生成代码的 ctx.Bind 与上传 handler 的 FormFile
	// 都在中间件之前执行，挂在中间件里的限制永远晚于 body 读取。
	// 上传路由的硬上限见 i_file_transfer_http.pb.go 的 uploadBodyGuard；
	// 全站护栏见 NewRestServer 里的 requestBodyGuard。
	ms = append(ms, securityHeaders())
	ms = append(ms, logging.Server(bLogger.AsKratosLogger(ctx.GetLogger())))

	// 审计日志异步化（docs/audit-log-producer-design.md）：五个审计回调先经 asynq
	// 入队（at-least-once），入队失败或 asynq 未配置时回退同步落库——宁可慢不可丢。
	var auditEnqueuer auditLogEnqueuer
	var auditProducerLog *bLogger.Helper
	if taskService != nil {
		auditEnqueuer = taskService
		auditProducerLog = ctx.NewLoggerHelper("audit/async-producer")
	}

	ms = append(ms, applogging.Server(
		applogging.WithWriteApiLogFunc(func(ctx context.Context, data *auditV1.ApiAuditLog) error {
			occurredAt := time.Now()
			if err := enqueueAuditLog(ctx, auditEnqueuer, task.AuditLogKindApi, data, occurredAt); err != nil {
				auditLogSyncFallback(ctx, auditProducerLog, task.AuditLogKindApi, err)
				// 回退同步落库（宁可慢不可丢）；Hash/Signature 在 middleware 侧已生成，原样保留
				return apiAuditLogRepo.Create(ctx, &auditV1.CreateApiAuditLogRequest{Data: data})
			}
			return nil
		}),
		applogging.WithWriteLoginLogFunc(func(ctx context.Context, data *auditV1.LoginAuditLog) error {
			occurredAt := time.Now()
			if err := enqueueAuditLog(ctx, auditEnqueuer, task.AuditLogKindLogin, data, occurredAt); err != nil {
				auditLogSyncFallback(ctx, auditProducerLog, task.AuditLogKindLogin, err)
				return loginLogRepo.Create(ctx, &auditV1.CreateLoginAuditLogRequest{Data: data})
			}
			return nil
		}),
		applogging.WithWriteOperationAuditLogFunc(func(ctx context.Context, data *auditV1.OperationAuditLog) error {
			occurredAt := time.Now()
			if err := enqueueAuditLog(ctx, auditEnqueuer, task.AuditLogKindOperation, data, occurredAt); err != nil {
				auditLogSyncFallback(ctx, auditProducerLog, task.AuditLogKindOperation, err)
				return operationAuditLogRepo.Create(ctx, &auditV1.CreateOperationAuditLogRequest{Data: data})
			}
			return nil
		}),
		applogging.WithWritePermissionAuditLogFunc(func(ctx context.Context, data *auditV1.PermissionAuditLog) error {
			occurredAt := time.Now()
			if err := enqueueAuditLog(ctx, auditEnqueuer, task.AuditLogKindPermission, data, occurredAt); err != nil {
				auditLogSyncFallback(ctx, auditProducerLog, task.AuditLogKindPermission, err)
				return permissionAuditLogRepo.Create(ctx, &auditV1.CreatePermissionAuditLogRequest{Data: data})
			}
			return nil
		}),
		applogging.WithWriteDataAccessAuditLogFunc(func(ctx context.Context, data *auditV1.DataAccessAuditLog) error {
			occurredAt := time.Now()
			if err := enqueueAuditLog(ctx, auditEnqueuer, task.AuditLogKindDataAccess, data, occurredAt); err != nil {
				auditLogSyncFallback(ctx, auditProducerLog, task.AuditLogKindDataAccess, err)
				return dataAccessAuditLogRepo.Create(ctx, &auditV1.CreateDataAccessAuditLogRequest{Data: data})
			}
			return nil
		}),
	))

	// 输入校验：对所有 RPC（含白名单内的 login/register/refresh/MFA）调用生成代码的
	// Validate()。放 selector 外，否则白名单路由会被跳过——而它们恰是最需要校验入参的。
	// 当前业务 proto 尚未补 (validate.rules)，多数 Validate() 返回 nil；补规则后再生效。
	ms = append(ms, validate.Validator())

	// add white list for authentication.
	rpc.AddWhiteList(
		adminV1.OperationAuthenticationServiceLogin,
		adminV1.OperationAuthenticationServiceGenerateCaptcha,
		adminV1.OperationAuthenticationServiceVerifyCaptcha,
		// 刷新令牌接口免鉴权：refresh token 现以 HttpOnly Cookie 传输且为自描述 JWT，
		// 可脱离 access token 独立鉴权。页面刷新后 access token 丢失时，前端凭
		// refresh cookie 静默恢复会话，不再强制重新登录。
		adminV1.OperationAuthenticationServiceRefreshToken,
		// MFA 登录挑战验证免鉴权：operation_id 由登录流程签发，见 doGrantTypePassword 的 MFA 闸门。
		// 仅此一个 MFA RPC 免鉴权；管理侧 RPC（GetMFAStatus 等）走正常 auth+authz。
		adminV1.OperationMfaServiceVerifyMFAChallenge,
		// 找回密码两个端点免鉴权：验证码发送与凭码重置，
		// 重置成功后会吊销该用户全部会话。
		// OpenAPI 令牌交换免鉴权：AK/SK 本身即为认证凭据。
		adminV1.OperationAccessKeyServiceIssueToken,
		adminV1.OperationAuthenticationServiceForgotPassword,
		adminV1.OperationAuthenticationServiceResetPasswordByCode,
		// 租户白标查询免鉴权：登录前按租户编号取名称/Logo（只暴露展示字段）
		adminV1.OperationAuthenticationServiceGetTenantBranding,
		// OIDC SSO 三端点免鉴权：它们本身就是认证入口（未登录态使用）
		adminV1.OperationAuthenticationServiceGetSsoLoginInfo,
		adminV1.OperationAuthenticationServiceGetSsoLoginUrl,
		adminV1.OperationAuthenticationServiceSsoLogin,
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
		authz.Server(newEvalLoggingEngine(authorizer.Engine(), policyEvaluationLogRepo, auditEnqueuer)),
	).
		Match(rpc.NewRestWhiteListMatcher()).
		Build(),
	)

	return ms
}

// securityHeaders 全站安全响应头（P3 加固）：
//   - nosniff：防 MIME 嗅探把上传内容当 HTML 执行；
//   - DENY：管理后台不允许被 iframe 嵌套（点击劫持）；
//   - Referrer-Policy：防 URL 中的令牌类参数经 Referer 外泄；
//   - HSTS：TLS 部署下强制 HTTPS（明文 HTTP 下浏览器忽略该头，无害）。
func securityHeaders() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				if htr, ok := tr.(*http.Transport); ok {
					h := htr.ReplyHeader()
					if h != nil {
						h.Set("X-Content-Type-Options", "nosniff")
						h.Set("X-Frame-Options", "DENY")
						h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
						h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
					}
				}
			}
			return handler(ctx, req)
		}
	}
}

const (
	// maxRequestBody 全站请求体上限。需容纳 base64 编码后的 50MiB 上传体
	// （base64 膨胀 4/3）与 JSON 结构开销；比它更小的请求不受影响。
	maxRequestBody = ossutil.MaxUploadSize*4/3 + 4<<20

	// maxUnauthenticatedBody 未认证请求体上限。所有免鉴权端点（登录、验证码、
	// 找回密码、SSO、令牌交换等）的请求体都远小于此值；kratos 生成的 handler
	// 在中间件之前 ctx.Bind（内部 io.ReadAll 整个 body），不设限时未认证请求
	// 即可用超大 JSON 打爆内存。
	maxUnauthenticatedBody = 1 << 20
)

// requestBodyGuard 挂在整个 REST 服务最外层（mux 之前），给所有请求封顶：
//   - 声明长度超过 maxRequestBody：直接 413，不读 body；
//   - 声明长度超过 maxUnauthenticatedBody（或长度未知）：必须携带有效访问令牌，
//     否则 401，不读 body——大 body 的合法场景（上传/头像/知识库文档）都是
//     已登录操作，免鉴权端点没有大 body 的需求；
//   - 其余请求：只套 MaxBytesReader 兜底。
//
// 之所以放在 HTTP Filter 而不是 kratos 中间件：后者的执行点在 handler 内部，
// 晚于 ctx.Bind / FormFile 的 body 读取。
func requestBodyGuard(checker auth.AccessTokenChecker) http.FilterFunc {
	return func(next nethttp.Handler) nethttp.Handler {
		return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			if r.ContentLength > maxRequestBody {
				nethttp.Error(w, "request entity too large", nethttp.StatusRequestEntityTooLarge)
				return
			}

			limit := int64(maxUnauthenticatedBody)

			switch {
			case r.ContentLength > maxUnauthenticatedBody:
				// 大 body：先认证，再放行读取。
				if !hasValidAccessToken(r, checker) {
					nethttp.Error(w, "missing or invalid bearer token", nethttp.StatusUnauthorized)
					return
				}
				limit = maxRequestBody
			case r.ContentLength < 0:
				// 长度未知（chunked）：有有效令牌才允许超过未认证上限。
				if hasValidAccessToken(r, checker) {
					limit = maxRequestBody
				}
			}

			r.Body = nethttp.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

// hasValidAccessToken 在读取 body 之前校验 Bearer 令牌。
func hasValidAccessToken(r *nethttp.Request, checker auth.AccessTokenChecker) bool {
	if checker == nil {
		return false
	}
	token := bearerToken(r)
	if token == "" {
		return false
	}
	valid, _ := checker.IsValidAccessToken(r.Context(), token, false)
	return valid
}

// bearerToken 从 Authorization 头提取 Bearer 令牌，与 kratos-authn 的
// AuthFromMD 同口径（scheme 大小写不敏感）。
func bearerToken(r *nethttp.Request) string {
	scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}

// NewRestServer new an REST server.
func NewRestServer(
	ctx *bootstrap.Context,

	middlewares []middleware.Middleware,
	authorizer *authorizer.Authorizer,
	accessTokenChecker auth.AccessTokenChecker,

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
	serverMonitorService *service.ServerMonitorService,
	notificationChannelService *service.NotificationChannelService,
	notificationService *service.NotificationService,
	onlineSessionService *service.OnlineSessionService,
	dashboardService *service.DashboardService,

	internalMessageService *service.InternalMessageService,
	internalMessageCategoryService *service.InternalMessageCategoryService,
	internalMessageRecipientService *service.InternalMessageRecipientService,

	scriptService *service.ScriptService,
	scriptLogService *service.ScriptLogService,

	// register:param ── 新模块服务形参在此行后注册(make register 工具锚点,勿删)
	biDashboardService *service.BiDashboardService,
	printTemplateService *service.PrintTemplateService,
	notificationRuleService *service.NotificationRuleService,
	notificationPreferenceService *service.NotificationPreferenceService,
	notificationTemplateService *service.NotificationTemplateService,
	monitorAlertService *service.MonitorAlertService,
	myTenantUsageService *service.MyTenantUsageService,
	auditExportService *service.AuditExportService,
	dataExportService *service.DataExportService,
	taskMonitorService *service.TaskMonitorService,
	accessKeyService *service.AccessKeyService,
	configService *service.ConfigService,
	aiProviderService *service.AiProviderService,
	aiConversationService *service.AiConversationService,
	aiMessageService *service.AiMessageService,
	aiUsageLogService *service.AiUsageLogService,
	aiKnowledgeService *service.AiKnowledgeService,
	aiChatService *service.AiChatService,
	aiQueryService *service.AiQueryService,
	aiContentService *service.AiContentService,
	aiAgentService *service.AiAgentService,
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

	// 全站请求体护栏包在最外层（CORS/pprof 过滤器与路由都在其内层），
	// 使 401/413 的判定发生在任何 body 读取之前。
	srv.Handler = requestBodyGuard(accessTokenChecker)(srv.Handler)

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

	// 字段权限装饰器在最内层（紧贴业务实现），静态脱敏 redact 在外层兜底；
	// 两者都按"清值"裁剪，protojson 默认不输出未填充字段，效果等价于移除字段。
	adminV1.RegisterUserServiceHTTPServer(srv, adminV1.RedactedUserServiceServer(
		service.NewFieldPermissionUserServiceServer(&userServiceServerAdapter{UserServiceHTTPServer: userService}),
		nil))
	// Role 资源同款装饰器（字段权限铺开第二个资源）：可管控字段 permissions
	// （角色权限集，权限体系的元权限）。
	adminV1.RegisterRoleServiceHTTPServer(srv, service.NewFieldPermissionRoleServiceServer(roleService))
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
	adminV1.RegisterServerMonitorServiceHTTPServer(srv, serverMonitorService)
	adminV1.RegisterNotificationChannelServiceHTTPServer(srv, notificationChannelService)
	adminV1.RegisterNotificationServiceHTTPServer(srv, notificationService)
	adminV1.RegisterOnlineSessionServiceHTTPServer(srv, onlineSessionService)
	adminV1.RegisterDashboardServiceHTTPServer(srv, dashboardService)

	adminV1.RegisterFileServiceHTTPServer(srv, fileService)

	// 注册文件传输服务，用于处理文件上传下载等功能
	// TODO 它不能够使用代码生成器生成的Handler，需要手动注册。代码生成器生成的Handler无法处理文件上传下载的请求。
	// 但，代码生成器生成代码可以提供给OpenAPI使用。
	registerFileTransferServiceHandler(srv, fileTransferService)

	// 审计日志服务端导出：手动注册（二进制文件响应，proto 生成路由写不了响应头）。
	// 手动路由无 Operation → auth 中间件不应用 → 鉴权在 handler 内显式做（平台管理员）。
	srv.Route("/").GET("admin/v1/audit-logs:export", func(ctx http.Context) error {
		return auditExportService.ServeExport(ctx.Response(), ctx.Request())
	})

	// AI 用量流水服务端导出（手动注册，理由同上；租户管理员按 viewer 语义导本租户，
	// 与 /admin/v1/ai/usage-logs 列表口径一致）。
	srv.Route("/").GET("admin/v1/ai/usage-logs:export", func(ctx http.Context) error {
		return dataExportService.ServeAiUsageExport(ctx.Response(), ctx.Request())
	})

	// 通知投递台账服务端导出（手动注册；平台管理员专属，同台账读接口的闸）。
	srv.Route("/").GET("admin/v1/notification-deliveries:export", func(ctx http.Context) error {
		return dataExportService.ServeNotificationDeliveryExport(ctx.Response(), ctx.Request())
	})

	adminV1.RegisterInternalMessageServiceHTTPServer(srv, internalMessageService)
	adminV1.RegisterInternalMessageCategoryServiceHTTPServer(srv, internalMessageCategoryService)
	adminV1.RegisterInternalMessageRecipientServiceHTTPServer(srv, internalMessageRecipientService)

	adminV1.RegisterScriptServiceHTTPServer(srv, scriptService)
	adminV1.RegisterScriptLogServiceHTTPServer(srv, scriptLogService)

	// register:route ── 新模块路由在此行后注册(make register 工具锚点,勿删)
	adminV1.RegisterBiDashboardServiceHTTPServer(srv, biDashboardService)
	adminV1.RegisterPrintTemplateServiceHTTPServer(srv, printTemplateService)
	adminV1.RegisterNotificationRuleServiceHTTPServer(srv, notificationRuleService)
	adminV1.RegisterNotificationPreferenceServiceHTTPServer(srv, notificationPreferenceService)
	adminV1.RegisterNotificationTemplateServiceHTTPServer(srv, notificationTemplateService)
	adminV1.RegisterMonitorAlertServiceHTTPServer(srv, monitorAlertService)
	adminV1.RegisterMyTenantUsageServiceHTTPServer(srv, myTenantUsageService)
	adminV1.RegisterTaskMonitorServiceHTTPServer(srv, taskMonitorService)
	adminV1.RegisterAccessKeyServiceHTTPServer(srv, accessKeyService)
	adminV1.RegisterConfigServiceHTTPServer(srv, configService)
	adminV1.RegisterAiProviderServiceHTTPServer(srv, aiProviderService)
	adminV1.RegisterAiConversationServiceHTTPServer(srv, aiConversationService)
	adminV1.RegisterAiMessageServiceHTTPServer(srv, aiMessageService)
	adminV1.RegisterAiUsageLogServiceHTTPServer(srv, aiUsageLogService)
	adminV1.RegisterAiKnowledgeBaseServiceHTTPServer(srv, aiKnowledgeService)
	adminV1.RegisterAiQueryServiceHTTPServer(srv, aiQueryService)
	adminV1.RegisterAiContentServiceHTTPServer(srv, aiContentService)
	adminV1.RegisterAiChatServiceHTTPServer(srv, aiChatService)
	adminV1.RegisterAiAgentServiceHTTPServer(srv, aiAgentService)

	if cfg.GetServer().GetRest().GetEnableSwagger() {
		swaggerUI.RegisterSwaggerUIServerWithOption(
			srv,
			swaggerUI.WithTitle("GoWind Admin"),
			swaggerUI.WithMemoryData(assets.OpenApiData, "yaml"),
		)
	}

	if authorizer != nil {
		if err = authorizer.ResetPolicies(viewer.WithSystemContext(ctx.Context())); err != nil {
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
