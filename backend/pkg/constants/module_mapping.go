package constants

import (
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

// ServiceTagToBusinessModule 将 OpenAPI tag 名（gRPC 服务名）映射到业务功能模块。
// 用于 sys_apis.business_module 的回填，供套餐模块白名单过滤使用。
// 新增服务时必须在此登记，否则其 API 的 business_module 将为 UNSPECIFIED，
// 租户白名单过滤会把 UNSPECIFIED 视为"不在任何白名单内"而拒绝访问。
//
// 反过来说：登记成哪个模块，就等于把该服务的全部端点交给那个模块的白名单。
// 通知域三个服务登记为 SYSTEM（与套餐白名单联动），而它们管的是**平台全局**配置与台账，
// 因此租户侧的拒绝由 service 层的 requirePlatformAdmin 显式承担
// （app/admin/service/internal/service/notification_platform_guard.go），不在这里做。
var ServiceTagToBusinessModule = map[string]identityV1.Module{
	"AdminPortalService":             identityV1.Module_DASHBOARD,
	"DashboardService":               identityV1.Module_DASHBOARD,

	"UserService":                    identityV1.Module_OPM,
	"OrgUnitService":                 identityV1.Module_OPM,
	"PositionService":                identityV1.Module_OPM,
	"UserProfileService":             identityV1.Module_OPM,
	"RoleService":                    identityV1.Module_OPM,

	"MenuService":                    identityV1.Module_PERMISSION,
	"ApiService":                     identityV1.Module_PERMISSION,
	"PermissionService":              identityV1.Module_PERMISSION,
	"PermissionGroupService":         identityV1.Module_PERMISSION,

	"DictTypeService":                identityV1.Module_DICT,
	"DictEntryService":               identityV1.Module_DICT,
	"LanguageService":                identityV1.Module_SYSTEM,
	"FileService":                    identityV1.Module_FILE,
	"FileTransferService":            identityV1.Module_FILE,
	"TaskService":                    identityV1.Module_TASK,
	"LoginPolicyService":             identityV1.Module_SYSTEM,
	"ConfigService":                  identityV1.Module_SYSTEM,
	"AccessKeyService":               identityV1.Module_SYSTEM,

	"TenantService":                  identityV1.Module_TENANT,
	"PlanService":                    identityV1.Module_TENANT,
	"PlanQuotaService":               identityV1.Module_TENANT,

	"ApiAuditLogService":             identityV1.Module_LOG,
	"LoginAuditLogService":           identityV1.Module_LOG,
	"OperationAuditLogService":       identityV1.Module_LOG,
	"DataAccessAuditLogService":      identityV1.Module_LOG,
	"PermissionAuditLogService":      identityV1.Module_LOG,
	"PolicyEvaluationLogService":     identityV1.Module_LOG,
	"RedisCacheMonitorService":       identityV1.Module_LOG,
	"ServerMonitorService":           identityV1.Module_SYSTEM,
	"NotificationChannelService":     identityV1.Module_SYSTEM,
	"NotificationService":            identityV1.Module_SYSTEM,
	"NotificationRuleService":        identityV1.Module_SYSTEM,
	"OnlineSessionService":           identityV1.Module_SYSTEM,

	"InternalMessageService":         identityV1.Module_INTERNAL_MESSAGE,
	"InternalMessageCategoryService": identityV1.Module_INTERNAL_MESSAGE,
	"InternalMessageRecipientService": identityV1.Module_INTERNAL_MESSAGE,

	"AuthenticationService":          identityV1.Module_DASHBOARD,
}
