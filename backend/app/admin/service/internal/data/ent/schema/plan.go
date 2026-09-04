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

// Plan holds the schema definition for the Plan entity.
type Plan struct {
	ent.Schema
}

func (Plan) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_plans",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("套餐目录表"),
	}
}

// Fields of the Plan.
func (Plan) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("套餐名称").
			NotEmpty().
			Optional().
			Nillable(),

		field.Enum("version").
			Comment("套餐版本").
			NamedValues(
				"Free", "FREE",
				"Standard", "STANDARD",
				"Enterprise", "ENTERPRISE",
			).
			Default("FREE").
			Optional().
			Nillable(),

		field.Enum("expiry_policy").
			Comment("到期处置策略").
			NamedValues(
				"Readonly", "READONLY",
				"BlockLogin", "BLOCK_LOGIN",
				"Freeze", "FREEZE",
			).
			Default("READONLY").
			Optional().
			Nillable(),

		field.Uint32("data_retention_days").
			Comment("数据保留周期（天）").
			Optional().
			Nillable(),

		field.String("description").
			Comment("描述").
			Optional().
			Nillable(),
	}
}

// Mixin of the Plan.
func (Plan) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.Remark{},
	}
}

// Edges of the Plan.
	func (Plan) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("tenants", Tenant.Type).
			StorageKey(edge.Column("plan_id")),

		// quotas/modules 的 Required() 会导致 Plan Create 时 ent 报
		// missing required edge——配额与模块白名单都是建套餐后单独维护的，不能必填
		edge.To("quotas", PlanQuota.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}).
			StorageKey(edge.Column("plan_id")),

		edge.To("modules", PlanModule.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}).
			StorageKey(edge.Column("plan_id")),
	}
}

// Indexes of the Plan.
func (Plan) Indexes() []ent.Index {
	return []ent.Index{
		// 创建时间索引，用于套餐列表的时间区间查询与分页
		index.Fields("created_at").StorageKey("idx_sys_plans_created_at"),
	}
}
