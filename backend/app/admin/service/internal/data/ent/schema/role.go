package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// Role holds the schema definition for the Role entity.
type Role struct {
	ent.Schema
}

func (Role) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_roles",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("角色表"),
	}
}

// Fields of the Role.
func (Role) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("角色名称").
			NotEmpty().
			Optional().
			Nillable(),

		field.String("code").
			Comment("角色标识").
			NotEmpty().
			Optional().
			Nillable(),

		field.Bool("is_protected").
			Comment("是否受保护的角色").
			Default(false).
			Nillable(),

		field.Enum("type").
			Comment("角色类型").
			NamedValues(
				"System", "SYSTEM",
				"Template", "TEMPLATE",
				"Tenant", "TENANT",
			).
			Default("TENANT").
			Nillable(),
	}
}

// Mixin of the Role.
func (Role) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.Description{},
		mixin.SortOrder{},
		mixin.TenantID[uint32]{},
		mixin.SwitchStatus{},
	}
}

// Indexes of the Role.
func (Role) Indexes() []ent.Index {
	return []ent.Index{
		// 租户维度唯一：同一租户下 code 唯一
		index.Fields("tenant_id", "code").
			Unique().
			StorageKey("uix_sys_roles_tenant_code"),

		// 租户范围内按 name 快速查询
		index.Fields("tenant_id", "name").
			StorageKey("idx_sys_roles_tenant_name"),
		// 全局 name 索引（模糊/快速搜索）
		index.Fields("name").
			StorageKey("idx_sys_roles_name"),

		// 全局 code 快速定位（跨租户查询）
		index.Fields("code").
			StorageKey("idx_sys_roles_code"),

		// 支持按租户快速筛选
		index.Fields("tenant_id").
			StorageKey("idx_sys_roles_tenant_id"),

		// 保护/启用状态查询
		index.Fields("is_protected").
			StorageKey("idx_sys_roles_is_protected"),
		index.Fields("status").
			StorageKey("idx_sys_roles_status"),

		// 排序/范围查询优化
		index.Fields("sort_order").
			StorageKey("idx_sys_roles_sort_order"),
		index.Fields("created_at").
			StorageKey("idx_sys_roles_created_at"),

		// 按操作人查询（audit）
		index.Fields("created_by").
			StorageKey("idx_sys_roles_created_by"),
	}
}
