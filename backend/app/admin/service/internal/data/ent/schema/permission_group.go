package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

type PermissionGroup struct {
	ent.Schema
}

func (PermissionGroup) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_permission_groups",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("权限分组表"),
	}
}

func (PermissionGroup) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			Nillable().
			Comment("分组名称（如：用户管理、订单操作）"),

		field.String("path").
			Optional().
			Nillable().
			Comment("树形路径，格式：/1/10/101/（包含自身且首尾带/）"),

		field.String("module").
			Comment("业务模块标识（如：opm、order、pay）").
			Optional().
			Nillable(),
	}
}

func (PermissionGroup) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.SwitchStatus{},
		mixin.TenantID{},
		mixin.SortOrder{},
		mixin.Tree[PermissionGroup]{},
	}
}

func (PermissionGroup) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id").StorageKey("idx_perm_group_tenant_id"),
		index.Fields("parent_id").StorageKey("idx_perm_group_parent_id"),

		// 使用 path 作为树节点的唯一标识（可在租户内唯一）
		index.Fields("tenant_id", "path").
			Unique().
			StorageKey("uix_perm_group_tenant_path"),

		// 常用按租户+名称查询的非唯一索引
		index.Fields("tenant_id", "name").
			StorageKey("idx_perm_group_tenant_name"),

		// 按 module 的查询索引
		index.Fields("module").
			StorageKey("idx_perm_group_module"),
	}
}
