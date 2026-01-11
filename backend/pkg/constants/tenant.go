package constants

// UserTenantRelationType 用户-租户关系类型，表示 users-tenants 是一对一还是一对多。
type UserTenantRelationType int

const (
	// UserTenantRelationOneToOne users 与 tenants 为一对一关系：每个用户只属于一个租户
	UserTenantRelationOneToOne UserTenantRelationType = iota + 1

	// UserTenantRelationOneToMany users 与 tenants 为一对多关系：每个用户可属于多个租户
	UserTenantRelationOneToMany
)

const (
	// EnableMultiTenancy 是否启用多租户功能
	EnableMultiTenancy = true

	// DefaultUserTenantRelation 用户与租户的默认关联关系
	// 如果启用多租户，推荐使用 UserTenantRelationOneToMany；否则默认一对一。
	DefaultUserTenantRelation = UserTenantRelationOneToOne
)
