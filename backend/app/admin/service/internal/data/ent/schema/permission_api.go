package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/tx7do/go-crud/entgo/mixin"
)

// PermissionApi holds the schema definition for the PermissionApi entity.
type PermissionApi struct {
	ent.Schema
}

func (PermissionApi) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_permission_apis",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("权限点与API接口关联表"),
	}
}

// Fields of the PermissionApi.
func (PermissionApi) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("permission_id").
			Comment("权限ID（关联sys_permissions.id）").
			Nillable(),

		field.Uint32("api_id").
			Comment("API资源ID（关联sys_apis.id）").
			Nillable(),
	}
}

// Mixin of the PermissionApi.
func (PermissionApi) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID{},
	}
}

// Indexes of the PermissionApi.
func (PermissionApi) Indexes() []ent.Index {
	return []ent.Index{
		// 唯一约束：同一租户下权限与 API 资源的组合唯一
		index.Fields("tenant_id", "permission_id", "api_id").
			Unique().
			StorageKey("uix_perm_api_tenant_permission_api_id"),

		// 常用查询：根据租户+权限查该权限下的所有 API 资源
		index.Fields("tenant_id", "permission_id").
			StorageKey("idx_perm_api_tenant_permission_api"),

		// 常用查询：根据租户+API 资源查关联的权限
		index.Fields("tenant_id", "api_id").
			StorageKey("idx_perm_api_tenant_api"),

		// 单列索引：按 permission_id 快速查询
		index.Fields("permission_id").
			StorageKey("idx_perm_api_permission_id"),

		// 单列索引：按 api_id 快速查询
		index.Fields("api_id").
			StorageKey("idx_perm_api_api_id"),
	}
}
