package constants

const (
	// RoleCodeSpilt 角色代码分隔符
	RoleCodeSpilt = ":"

	// TemplateRoleCodePrefix 模板角色代码前缀
	TemplateRoleCodePrefix = "template" + RoleCodeSpilt
	// PlatformRoleCodePrefix 平台角色代码前缀
	PlatformRoleCodePrefix = "platform" + RoleCodeSpilt
	// TenantRoleCodePrefix 租户角色代码前缀
	TenantRoleCodePrefix = "tenant" + RoleCodeSpilt

	// PlatformAdminRoleCode 平台管理员角色代码
	PlatformAdminRoleCode = PlatformRoleCodePrefix + "admin"
	// TenantAdminRoleCode 租户管理员角色代码
	TenantAdminRoleCode = TenantRoleCodePrefix + "manager"
	// TenantAdminTemplateRoleCode 租户管理员模板角色代码
	TenantAdminTemplateRoleCode = TemplateRoleCodePrefix + TenantAdminRoleCode

	// DefaultPlatformAdminRoleName 平台管理员角色默认名称
	DefaultPlatformAdminRoleName = "平台管理员"
	// DefaultTenantManagerRoleName 租户管理员角色默认名称
	DefaultTenantManagerRoleName = "租户管理员"
)

func HasRoleCodePrefix(roleCode string, prefix string) bool {
	return len(roleCode) >= len(prefix) && roleCode[0:len(prefix)] == prefix
}

func IsPlatformRoleCode(roleCode string) bool {
	return HasRoleCodePrefix(roleCode, PlatformRoleCodePrefix)
}

func IsTenantRoleCode(roleCode string) bool {
	return HasRoleCodePrefix(roleCode, TenantRoleCodePrefix)
}

func IsTemplateRoleCode(roleCode string) bool {
	return HasRoleCodePrefix(roleCode, TemplateRoleCodePrefix)
}

func ExtractRoleCodeFromTemplate(roleCode string) string {
	if IsTemplateRoleCode(roleCode) {
		return roleCode[len(TemplateRoleCodePrefix):]
	}
	return roleCode
}

// IsReservedRoleCode 判断角色码是否为系统保留，不可通过角色 CRUD API 创建/修改。
// platform: / template: 命名空间与内置租户管理员码在程序里承载身份判定语义
// （token 管理员标志按码值精确匹配、模板实例化按码值查询、鉴权策略按码值做主体），
// 用户自建角色一旦占用即可伪装成内置身份越权。系统内的合法保留码角色只能由
// 种子数据与租户开通等内部链路（绕过 service 层校验）产生。
func IsReservedRoleCode(roleCode string) bool {
	return roleCode == TenantAdminRoleCode ||
		IsPlatformRoleCode(roleCode) ||
		IsTemplateRoleCode(roleCode)
}
