package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// RoleOrgUnit 角色与组织单元多对多关联表。
// 仅承载 Role.data_scope = SELECTED_UNITS 档位下的自定义授权单元集
// （对应若依 sys_role_dept）。写入侧由 RoleOrgUnitRepo 校验目标单元
// 与角色同租户，拒绝跨租户注入。
type RoleOrgUnit struct {
	ent.Schema
}

func (RoleOrgUnit) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_role_org_units",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("角色与组织单元关联表"),
	}
}

// Fields of the RoleOrgUnit.
func (RoleOrgUnit) Fields() []ent.Field {
	return []ent.Field{

		field.Uint64("role_id").
			Comment("角色ID（关联sys_roles.id）").
			Nillable(),

		field.Uint64("org_unit_id").
			Comment("组织单元ID（关联org_units.id）").
			Nillable(),
	}
}

// Mixin of the RoleOrgUnit.
func (RoleOrgUnit) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TimeAt{},
		mixin.OperatorID64{},
		mixin.TenantID[uint64]{},
	}
}

// Indexes of the RoleOrgUnit.
func (RoleOrgUnit) Indexes() []ent.Index {
	return []ent.Index{
		// 租户维度唯一：同一租户内 role + org_unit 唯一
		index.Fields("tenant_id", "role_id", "org_unit_id").
			Unique().
			StorageKey("uix_rou_tenant_role_org_unit"),

		// 常用查询：按租户+角色取授权单元集
		index.Fields("tenant_id", "role_id").
			StorageKey("idx_rou_tenant_role"),
	}
}

// Policy 追加租户变更防护：go-crud TenantPrivacy 只覆盖 Query/Create，
// 本规则为 Update/UpdateOne/Delete/DeleteOne 注入 tenant_id 过滤，
// 防止知道 ID 的跨租户篡改与删除（平台/系统上下文放行）。
func (RoleOrgUnit) Policy() ent.Policy {
	return &TenantMutationGuardPolicy{}
}
