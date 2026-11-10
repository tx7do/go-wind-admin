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

		// 数据权限范围（若依式角色级数据范围）。枚举值字符串必须与
		// identity.service.v1.DataScope 的枚举值逐字一致，否则
		// mapper.EnumTypeConverter 直传值会在两端口径错位引发 500。
		// 默认 ALL：存量角色迁移回填全量，管理员按角色收紧。
		field.Enum("data_scope").
			Comment("数据权限范围").
			NamedValues(
				"All", "ALL",
				"Self", "SELF",
				"UnitOnly", "UNIT_ONLY",
				"UnitAndChild", "UNIT_AND_CHILD",
				"SelectedUnits", "SELECTED_UNITS",
			).
			Default("ALL").
			Nillable(),

		// 父角色ID，0表示无父级（根角色）。不做自引用 edge（无需 FK 约束），
		// 层级语义（子角色权限 ⊆ 父角色已绑定权限集）在 service 层校验。
		field.Int64("parent_id").
			Comment("父角色ID，0表示无父级").
			Optional().
			Nillable(),
	}
}

// Mixin of the Role.
func (Role) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TimeAt{},
		mixin.OperatorID64{},
		mixin.Remark{},
		mixin.Description{},
		mixin.SortOrder{},
		mixin.TenantID[int64]{},
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

		// 父角色查询（子角色列表/删除父角色时解除引用）
		index.Fields("parent_id").
			StorageKey("idx_sys_roles_parent_id"),

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

// Policy 追加租户变更防护：go-crud TenantPrivacy 只覆盖 Query/Create，
// 本规则为 Update/UpdateOne/Delete/DeleteOne 注入 tenant_id 过滤，
// 防止知道 ID 的跨租户篡改与删除（平台/系统上下文放行）。
func (Role) Policy() ent.Policy {
	return &TenantMutationGuardPolicy{}
}
