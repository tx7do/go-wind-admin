package schema

import (
	"go-wind-admin/app/admin/service/internal/data/ent/privacy"
	"go-wind-admin/app/admin/service/internal/data/ent/rule"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/tx7do/go-crud/entgo/mixin"
)

// tenant_id 为 NULL 或者 0 时代表全局或个人系统成员。
// 建议使用 0 代表全局/平台，避免 NULL 带来的三值逻辑复杂性。

// Membership holds the schema definition for the Membership entity.
type Membership struct {
	ent.Schema
}

func (Membership) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_memberships",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("成员关联表"),
	}
}

// Fields of the Membership.
func (Membership) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("user_id").
			Comment("用户ID").
			Nillable(),

		// 冗余单一场景字段（互斥：与独立关联表二选一）
		field.Uint32("org_unit_id").
			Optional().
			Nillable().
			Comment("组织架构ID（单一，冗余/互斥）"),
		field.Uint32("position_id").
			Optional().
			Nillable().
			Comment("职位ID（单一，冗余/互斥）"),
		field.Uint32("role_id").
			Optional().
			Nillable().
			Comment("角色ID（单一，冗余/互斥）"),

		field.Bool("is_primary").
			Default(false).
			Nillable().
			Comment("是否主身份"),

		field.Time("start_at").
			Comment("生效时间").
			Optional().
			Nillable(),

		field.Time("end_at").
			Comment("失效时间").
			Optional().
			Nillable(),

		field.Time("assigned_at").
			Comment("分配时间（UTC）").
			Optional().
			Nillable(),
		field.Uint32("assigned_by").
			Comment("分配者用户ID").
			Optional().
			Nillable(),

		field.Time("joined_at").
			Comment("加入时间（UTC）").
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("状态").
			Optional().
			Nillable().
			Default("ACTIVE").
			NamedValues(
				"Active", "ACTIVE",
				"Disabled", "DISABLED",
				"Pending", "PENDING",
				"Invited", "INVITED",
				"Expired", "EXPIRED",
				"Rejected", "REJECTED",
			),
	}
}

// Mixin of the Membership.
func (Membership) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID{},
		mixin.Remark{},
	}
}

// Policy for all schemas that embed Membership.
func (Membership) Policy() ent.Policy {
	return privacy.Policy{
		Query: rule.TenantQueryPolicy(),
	}
}

// Indexes of the Membership.
func (Membership) Indexes() []ent.Index {
	return []ent.Index{
		// 在租户范围内保证 (user_id, org_unit_id, role_id, position_id) 组合唯一
		// 若任一列可为 NULL，Postgres 需在迁移中使用 partial unique index
		index.Fields("tenant_id", "user_id").
			Unique().
			StorageKey("uix_sys_membership_tenant_user"),

		// 在租户范围内保证每个用户只有一个主身份（is_primary）
		// 若 is_primary 可为 NULL/false，Postgres 需在迁移中使用 partial unique index（例如 WHERE is_primary = true）
		index.Fields("tenant_id", "user_id", "is_primary").
			Unique().
			StorageKey("uix_sys_membership_tenant_user_primary"),

		// 按租户 + 用户 + 状态 + 生效时间（时间列放末尾便于范围扫描）
		index.Fields("tenant_id", "user_id", "status", "start_at").
			StorageKey("idx_sys_membership_tenant_user_status_start_at"),

		// 单列索引，便于按用户/租户/组织/角色/职位快速过滤
		index.Fields("user_id").
			StorageKey("idx_sys_membership_user_id"),
		index.Fields("tenant_id").
			StorageKey("idx_sys_membership_tenant_id"),
		index.Fields("org_unit_id").
			StorageKey("idx_sys_membership_org_unit_id"),
		index.Fields("role_id").
			StorageKey("idx_sys_membership_role_id"),
		index.Fields("position_id").
			StorageKey("idx_sys_membership_position_id"),

		// 分配者索引
		index.Fields("assigned_by").
			StorageKey("idx_sys_membership_assigned_by"),

		// 状态与时间范围查询索引
		index.Fields("status").
			StorageKey("idx_sys_membership_status"),
		index.Fields("start_at").
			StorageKey("idx_sys_membership_start_at"),
		index.Fields("end_at").
			StorageKey("idx_sys_membership_end_at"),

		// 审计与分页（时间列放末尾以利于范围扫描）
		index.Fields("created_by", "created_at").
			StorageKey("idx_sys_membership_created_by_created_at"),
		index.Fields("created_at").
			StorageKey("idx_sys_membership_created_at"),
	}
}
