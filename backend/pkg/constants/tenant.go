package constants

// UserTenantRelationType 用户-租户关系类型，表示 users-tenants 是一对一还是一对多。
type UserTenantRelationType int

const (
	// UserTenantRelationNone 未启用多租户（例如单租户模式）
	UserTenantRelationNone UserTenantRelationType = iota

	// UserTenantRelationOneToOne users 与 tenants 为一对一关系：每个用户只属于一个租户
	UserTenantRelationOneToOne

	// UserTenantRelationOneToMany users 与 tenants 为一对多关系：每个用户可属于多个租户
	UserTenantRelationOneToMany
)

const (
	// DefaultUserTenantRelationType 用户与租户的默认关联关系
	// 如果启用多租户，推荐使用 UserTenantRelationOneToOne；默认一对一。
	DefaultUserTenantRelationType UserTenantRelationType = UserTenantRelationOneToOne

	// IsTenantModeEnabled 是否启用租户模式
	IsTenantModeEnabled = DefaultUserTenantRelationType == UserTenantRelationOneToOne || DefaultUserTenantRelationType == UserTenantRelationOneToMany
)
