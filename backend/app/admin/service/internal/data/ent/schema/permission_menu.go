package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/tx7do/go-crud/entgo/mixin"
)

// PermissionMenu holds the schema definition for the PermissionMenu entity.
type PermissionMenu struct {
	ent.Schema
}

func (PermissionMenu) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_permission_menus",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("权限点-前端菜单关联表"),
	}
}

// Fields of the PermissionMenu.
func (PermissionMenu) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("permission_id").
			Comment("权限ID（关联sys_permissions.id）").
			Nillable(),

		field.Uint32("menu_id").
			Comment("菜单ID（关联sys_menus.id）").
			Nillable(),
	}
}

// Mixin of the PermissionMenu.
func (PermissionMenu) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID{},
	}
}

// Indexes of the PermissionMenu.
func (PermissionMenu) Indexes() []ent.Index {
	return []ent.Index{
		// 唯一约束：同一租户下权限与菜单的组合唯一
		index.Fields("tenant_id", "permission_id", "menu_id").
			Unique().
			StorageKey("uix_perm_menu_tenant"),

		// 常用查询：根据租户+权限查该权限下的所有菜单
		index.Fields("tenant_id", "permission_id").
			StorageKey("idx_perm_menu_tenant_perm"),

		// 常用查询：根据租户+菜单查关联的权限
		index.Fields("tenant_id", "menu_id").
			StorageKey("idx_perm_menu_tenant_menu"),

		// 单列索引：按 permission_id 快速查询
		index.Fields("permission_id").
			StorageKey("idx_perm_menu_permission_id"),

		// 单列索引：按 menu_id 快速查询
		index.Fields("menu_id").
			StorageKey("idx_perm_menu_menu_id"),
	}
}
