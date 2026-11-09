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
		schema.Comment("权限点与前端菜单关联表"),
	}
}

// Fields of the PermissionMenu.
func (PermissionMenu) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("permission_id").
			Comment("权限ID（关联sys_permissions.id）").
			Nillable(),

		// 绑定存菜单全路径快照而非 menu_id：菜单全路径（父链拼接，见 menu_repo
		// joinMenuFullPath）是菜单的身份键——菜单同步 MERGE 按它保号，REPLACE 重建
		// 后同路径新行照常命中，授权不再随菜单 id 重建而漂移（与 sys_permission_apis
		// 的 (path, method) 端点快照同构）。落库时由 menu_id 一次性解析而来。
		field.String("full_path").
			Comment("菜单全路径（菜单身份键绑定快照）").
			Optional().
			Nillable(),
	}
}

// Mixin of the PermissionMenu.
func (PermissionMenu) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TimeAt{},
		mixin.OperatorID64{},
	}
}

// Indexes of the PermissionMenu.
func (PermissionMenu) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("permission_id", "full_path").
			Unique().
			StorageKey("uix_perm_menu_permission_full_path"),

		// 单列索引：按 permission_id 快速查询
		index.Fields("permission_id").
			StorageKey("idx_perm_menu_permission_id"),

		// 全路径反查：菜单行被删除时按 full_path 级联清理绑定
		index.Fields("full_path").
			StorageKey("idx_perm_menu_full_path"),
	}
}
