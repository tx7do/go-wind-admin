package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"

	"go-wind-admin/app/admin/service/internal/data/ent/privacy"
	"go-wind-admin/app/admin/service/internal/data/ent/rule"
)

// MembershipOrgUnit 成员与组织单元关联表
type MembershipOrgUnit struct {
	ent.Schema
}

func (MembershipOrgUnit) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_membership_org_units",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("成员与组织单元关联表"),
	}
}

func (MembershipOrgUnit) Fields() []ent.Field {
	return []ent.Field{
		// 关联到 membership（必填）
		field.Uint32("membership_id").
			Comment("membership ID").
			Nillable(),

		// 关联到组织单元（必填）
		field.Uint32("org_unit_id").
			Comment("组织单元 ID").
			Nillable(),

		// 可选：在该组织内的岗位/角色引用（冗余）
		field.Uint32("position_id").
			Optional().
			Nillable().
			Comment("岗位 ID（可选，冗余）"),
		field.Uint32("role_id").
			Optional().
			Nillable().
			Comment("角色 ID（可选，冗余）"),

		// 可选：该关联的生效/结束时间
		field.Time("start_at").
			Optional().
			Nillable().
			Comment("生效时间（UTC）"),
		field.Time("end_at").
			Optional().
			Nillable().
			Comment("结束时间（UTC）"),

		// 分配审计字段（记录分配时刻与分配者）
		field.Time("assigned_at").
			Optional().
			Nillable().
			Comment("分配时间（UTC）"),
		field.Uint32("assigned_by").
			Optional().
			Nillable().
			Comment("分配者用户 ID"),

		// 标记当前是否为主要所属（用于单一场景）
		field.Bool("is_primary").
			Default(false).
			Nillable().
			Comment("是否为主所属"),

		field.Enum("status").
			Comment("关联状态").
			Optional().
			Nillable().
			Default("ACTIVE").
			NamedValues(
				"Active", "ACTIVE",
				"Pending", "PENDING",
				"Inactive", "INACTIVE",
				"Suspended", "SUSPENDED",
				"Expired", "EXPIRED",
			),
	}
}

func (MembershipOrgUnit) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID{},
		mixin.Remark{},
	}
}

// Policy for all schemas that embed MembershipOrgUnit.
func (MembershipOrgUnit) Policy() ent.Policy {
	return privacy.Policy{
		Query: rule.TenantQueryPolicy(),
	}
}

func (MembershipOrgUnit) Indexes() []ent.Index {
	return []ent.Index{
		// 在租户范围内保证 (membership_id, org_unit_id, position_id, role_id) 组合唯一，避免重复分配记录
		index.Fields("tenant_id", "membership_id", "org_unit_id", "position_id", "role_id").
			Unique().
			StorageKey("uix_mou_tenant_mem_org_pos_role"),

		// 在租户范围内保证每个 membership 只有一个主所属（is_primary）
		// 注意：如果希望只允许 is_primary = true 唯一（Postgres），需在迁移脚本使用 partial unique index（例如 WHERE is_primary = true）
		index.Fields("tenant_id", "membership_id", "is_primary").
			Unique().
			StorageKey("uix_mou_tenant_membership_primary"),

		// 常用复合索引：按租户 + membership 快速定位该会员所有关联
		index.Fields("tenant_id", "membership_id").
			StorageKey("idx_mou_tenant_membership"),

		// 常用复合索引：按租户 + org_unit 快速定位某组织下的成员关联
		index.Fields("tenant_id", "org_unit_id").
			StorageKey("idx_mou_tenant_org_unit"),

		// 单列索引，便于按 membership/org_unit/position/role 快速过滤
		index.Fields("membership_id").
			StorageKey("idx_mou_membership_id"),
		index.Fields("org_unit_id").
			StorageKey("idx_mou_org_unit_id"),
		index.Fields("position_id").
			StorageKey("idx_mou_position_id"),
		index.Fields("role_id").
			StorageKey("idx_mou_role_id"),

		// 分配者索引，用于查询由谁分配
		index.Fields("assigned_by").
			StorageKey("idx_mou_assigned_by"),

		// 主标志与状态索引
		index.Fields("is_primary").
			StorageKey("idx_mou_is_primary"),
		index.Fields("status").
			StorageKey("idx_mou_status"),

		// 时间范围查询索引（生效/结束/分配时间）
		index.Fields("start_at").
			StorageKey("idx_mou_start_at"),
		index.Fields("end_at").
			StorageKey("idx_mou_end_at"),
		index.Fields("assigned_at").
			StorageKey("idx_mou_assigned_at"),

		// 审计与分页索引（时间列放末尾便于范围扫描）
		index.Fields("created_by", "created_at").
			StorageKey("idx_mou_created_by_created_at"),
		index.Fields("tenant_id", "created_at").
			StorageKey("idx_mou_tenant_created_at"),
		index.Fields("created_at").
			StorageKey("idx_mou_created_at"),
	}
}
