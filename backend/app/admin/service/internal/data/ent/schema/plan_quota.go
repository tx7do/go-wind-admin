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

// PlanQuota holds the schema definition for the PlanQuota entity.
type PlanQuota struct {
	ent.Schema
}

func (PlanQuota) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_plan_quotas",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("套餐配额表"),
	}
}

// Fields of the PlanQuota.
func (PlanQuota) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("quota_type").
			Comment("配额类型").
			NamedValues(
				"UserLimit", "USER_LIMIT",
				"Storage", "STORAGE",
				"ApiCall", "API_CALL",
				"AiTokens", "AI_TOKENS",
			).
			Optional().
			Nillable(),

		field.Uint64("quota_value").
			Comment("配额值").
			Optional().
			Nillable(),
	}
}

// Mixin of the PlanQuota.
func (PlanQuota) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TimeAt{},
		mixin.OperatorID64{},
	}
}

// Edges of the PlanQuota.
func (PlanQuota) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("plan", Plan.Type).
			Ref("quotas").
			Unique(),
	}
}

// Indexes of the PlanQuota.
func (PlanQuota) Indexes() []ent.Index {
	return []ent.Index{
		// 创建时间索引，用于配额列表的时间区间查询与分页
		index.Fields("created_at").StorageKey("idx_sys_plan_quotas_created_at"),
	}
}
