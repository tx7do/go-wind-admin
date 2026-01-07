package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/tx7do/go-crud/entgo/mixin"
)

// Position holds the schema definition for the Position entity.
type Position struct {
	ent.Schema
}

func (Position) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_positions",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("职位表"),
	}
}

// Fields of the Position.
func (Position) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("职位名称").
			NotEmpty().
			Nillable(),

		field.String("code").
			Comment("唯一编码").
			NotEmpty().
			Nillable(),

		field.Uint32("org_unit_id").
			Comment("所属组织单元ID").
			Nillable(),

		field.Uint32("reports_to_position_id").
			Comment("汇报关系").
			Optional().
			Nillable(),

		field.String("description").
			Comment("职位描述").
			Optional().
			Nillable(),

		field.String("job_family").
			Comment("职类/序列").
			Optional().
			Nillable(),

		field.String("job_grade").
			Comment("职级").
			Optional().
			Nillable(),

		field.Int32("level").
			Comment("数值化职级").
			Optional().
			Nillable(),

		field.Uint32("headcount").
			Comment("编制人数").
			Default(0).
			Nillable(),

		field.Bool("is_key_position").
			Comment("是否关键岗位").
			Default(false).
			Nillable(),

		field.Enum("type").
			NamedValues(
				"Regular", "REGULAR",
				"Manager", "MANAGER",
				"Lead", "LEAD",
				"Intern", "INTERN",
				"Contract", "CONTRACT",
				"Other", "OTHER",
			).
			Default("REGULAR").
			Comment("岗位类型"),

		field.Time("start_at").
			Comment("生效时间（UTC）").
			Optional().
			Nillable(),

		field.Time("end_at").
			Comment("结束有效期（UTC）").
			Optional().
			Nillable(),
	}
}

// Mixin of the Position.
func (Position) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.SortOrder{},
		mixin.Remark{},
		mixin.TenantID{},
		mixin.SwitchStatus{},
	}
}

// Indexes of the Position.
func (Position) Indexes() []ent.Index {
	return []ent.Index{
		// 租户维度唯一：同一租户下 code 唯一
		index.Fields("tenant_id", "code").
			Unique().
			StorageKey("uix_sys_positions_tenant_code"),

		// 全局快速定位 code（非唯一，跨租户查询）
		index.Fields("code").
			StorageKey("idx_sys_positions_code"),

		// 名称查询：租户范围内按 name 搜索 + 全局 name 索引（模糊/快速搜索）
		index.Fields("tenant_id", "name").
			StorageKey("idx_sys_positions_tenant_name"),
		index.Fields("name").
			StorageKey("idx_sys_positions_name"),

		// 组织单元相关查询
		index.Fields("tenant_id", "org_unit_id").
			StorageKey("idx_sys_positions_tenant_org_unit_id"),
		index.Fields("org_unit_id").
			StorageKey("idx_sys_positions_org_unit_id"),

		// 汇报关系查询（上级岗位）
		index.Fields("tenant_id", "reports_to_position_id").
			StorageKey("idx_sys_positions_tenant_reports_to"),
		index.Fields("reports_to_position_id").
			StorageKey("idx_sys_positions_reports_to"),

		// 按岗位类型查询（租户内与全局）
		index.Fields("tenant_id", "type").
			StorageKey("idx_sys_positions_tenant_type"),
		index.Fields("type").
			StorageKey("idx_sys_positions_type"),

		// 是否关键岗位查询
		index.Fields("tenant_id", "is_key_position").
			StorageKey("idx_sys_positions_tenant_is_key"),
		index.Fields("is_key_position").
			StorageKey("idx_sys_positions_is_key"),

		// 数值/排序/状态/时间相关索引，优化排序与范围查询
		index.Fields("level").
			StorageKey("idx_sys_positions_level"),
		index.Fields("headcount").
			StorageKey("idx_sys_positions_headcount"),
		index.Fields("sort_order").
			StorageKey("idx_sys_positions_sort_order"),
		index.Fields("status").
			StorageKey("idx_sys_positions_status"),
		index.Fields("start_at").
			StorageKey("idx_sys_positions_start_at"),
		index.Fields("end_at").
			StorageKey("idx_sys_positions_end_at"),
		index.Fields("created_at").
			StorageKey("idx_sys_positions_created_at"),

		// 支持按租户快速筛选
		index.Fields("tenant_id").
			StorageKey("idx_sys_positions_tenant_id"),
	}
}
