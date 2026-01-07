package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

type MembershipRole struct {
	ent.Schema
}

func (MembershipRole) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_membership_roles",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("成员-角色 关联表（支持多角色及分配属性）"),
	}
}

func (MembershipRole) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("membership_id").
			Comment("membership ID").
			Nillable(),

		// 关联到角色（必填）
		field.Uint32("role_id").
			Comment("角色ID").
			Nillable(),

		// 生效时间（UTC）
		field.Time("start_at").
			Optional().
			Nillable().
			Comment("生效时间（UTC）"),

		// 失效时间（UTC）
		field.Time("end_at").
			Optional().
			Nillable().
			Comment("失效时间（UTC）"),

		// 分配审计：记录分配时刻与分配者（UTC）
		field.Time("assigned_at").
			Comment("分配时间（UTC）").
			Optional().
			Nillable(),
		field.Uint32("assigned_by").
			Comment("分配者用户ID").
			Optional().
			Nillable(),

		// 是否为主角色（用于快速筛选单一主角色场景）
		field.Bool("is_primary").
			Comment("是否为主角色").
			Nillable().
			Default(false),

		field.Enum("status").
			NamedValues(
				"Pending", "PENDING",
				"Active", "ACTIVE",
				"Disabled", "DISABLED",
				"Expired", "EXPIRED",
			).
			Default("ACTIVE").
			Comment("岗位状态"),
	}
}

func (MembershipRole) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID{},
	}
}

func (MembershipRole) Indexes() []ent.Index {
	return []ent.Index{
		// 唯一约束：同一租户下 membership 与 role 的组合唯一
		index.Fields("tenant_id", "membership_id", "role_id").
			Unique().
			StorageKey("uix_mem_role_tenant_memid_role"),

		// 常用查询：在租户范围内按 membership 查所有角色
		index.Fields("tenant_id", "membership_id").
			StorageKey("idx_mem_role_tenant_membership"),

		// 常用查询：在租户范围内按 role 查所有成员
		index.Fields("tenant_id", "role_id").
			StorageKey("idx_mem_role_tenant_role"),

		// 常用查询：快速查找某成员在租户下的主角色
		index.Fields("tenant_id", "membership_id", "is_primary").
			StorageKey("idx_mem_role_tenant_mem_primary"),

		// 按分配者查询（租户范围内或全局）
		index.Fields("tenant_id", "assigned_by").
			StorageKey("idx_mem_role_tenant_assigned_by"),
		index.Fields("assigned_by").
			StorageKey("idx_mem_role_assigned_by"),

		// 保留/补充常用的单列索引以支持简单或模糊查询
		index.Fields("role_id").
			StorageKey("idx_mem_role_role_id"),
		index.Fields("membership_id").
			StorageKey("idx_mem_role_membership_id"),
		index.Fields("tenant_id").
			StorageKey("idx_mem_role_tenant_id"),
		index.Fields("is_primary").
			StorageKey("idx_mem_role_is_primary"),
		index.Fields("status").
			StorageKey("idx_mem_role_status"),
	}
}
