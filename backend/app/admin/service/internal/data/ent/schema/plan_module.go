package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// PlanModule holds the schema definition for the PlanModule entity.
type PlanModule struct {
	ent.Schema
}

func (PlanModule) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_plan_modules",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("套餐功能模块白名单表"),
	}
}

// Fields of the PlanModule.
func (PlanModule) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("module").
			Comment("功能模块").
			NamedValues(
				"Dashboard", "DASHBOARD",
				"Opm", "OPM",
				"System", "SYSTEM",
				"Dict", "DICT",
				"Tenant", "TENANT",
				"Permission", "PERMISSION",
				"Log", "LOG",
				"InternalMessage", "INTERNAL_MESSAGE",
				"File", "FILE",
				"Task", "TASK",
				"Ai", "AI",
				"Print", "PRINT",
			).
			Optional().
			Nillable(),
	}
}

// Mixin of the PlanModule.
func (PlanModule) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
	}
}

// Edges of the PlanModule.
func (PlanModule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("plan", Plan.Type).
			Ref("modules").
			Unique(),
	}
}

// Indexes of the PlanModule.
func (PlanModule) Indexes() []ent.Index {
	return []ent.Index{
		// 创建时间索引，用于白名单列表的时间区间查询与分页
		index.Fields("created_at").StorageKey("idx_sys_plan_modules_created_at"),
	}
}
