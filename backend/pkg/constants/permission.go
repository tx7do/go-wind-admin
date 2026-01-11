package constants

const (
	// SystemPermissionCodePrefix 系统权限代码前缀
	SystemPermissionCodePrefix = "sys:"

	// SystemAccessBackendPermissionCode 系统访问后台权限代码
	SystemAccessBackendPermissionCode = SystemPermissionCodePrefix + "access_backend"

	// SystemManageTenantsPermissionCode 系统管理租户权限代码
	SystemManageTenantsPermissionCode = SystemPermissionCodePrefix + "manage_tenants"

	// SystemAuditLogsPermissionCode 系统审计日志权限代码
	SystemAuditLogsPermissionCode = SystemPermissionCodePrefix + "audit_logs"

	// SystemPermissionModule 系统权限模块标识
	SystemPermissionModule = "sys"

	// DefaultBizPermissionModule 业务权限模块标识
	DefaultBizPermissionModule = "biz"

	// RoleCodeTemplatePrefix 角色代码模板前缀
	RoleCodeTemplatePrefix = "template:"

	// PlatformAdminRoleCode 平台管理员角色代码
	PlatformAdminRoleCode = "platform_admin"

	// TenantAdminRoleCode 租户管理员角色代码
	TenantAdminRoleCode = "tenant_admin"
)
