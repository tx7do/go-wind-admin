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

	// SystemPlatformAdminPermissionCode 系统平台管理员权限代码
	SystemPlatformAdminPermissionCode = SystemPermissionCodePrefix + "platform_admin"
	// SystemTenantManagerPermissionCode 系统租户管理员权限代码
	SystemTenantManagerPermissionCode = SystemPermissionCodePrefix + "tenant_manager"

	// SystemPermissionModule 系统权限模块标识
	SystemPermissionModule = "sys"

	// DefaultBizPermissionModule 业务权限模块标识
	DefaultBizPermissionModule = "biz"

	// UncategorizedPermissionGroup 未分类权限组标识
	UncategorizedPermissionGroup = "uncategorized"
)

// ProtectedPermissionCodes 受保护的权限代码列表，禁止删除
var ProtectedPermissionCodes = []string{
	SystemAccessBackendPermissionCode,
	SystemManageTenantsPermissionCode,
	SystemAuditLogsPermissionCode,
	SystemPlatformAdminPermissionCode,
	SystemTenantManagerPermissionCode,
}

// IsReservedPermissionCode 判断权限码是否为系统保留，不可通过权限 CRUD API 创建/修改。
// sys: 命名空间在服务端承载判定语义（sys:access_backend 是登录后台的闸门码，
// 登录链路按码值精确匹配），用户自建权限一旦占用即可伪造登录资格等权限语义。
func IsReservedPermissionCode(code string) bool {
	return len(code) >= len(SystemPermissionCodePrefix) &&
		code[:len(SystemPermissionCodePrefix)] == SystemPermissionCodePrefix
}
