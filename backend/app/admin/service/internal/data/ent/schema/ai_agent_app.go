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

// AiAgentApp AI Agent 编排应用：可视化画布定义的图（DSL 存 graph 列）。
// 图 DSL 与执行语义见 docs/ai_agent_orchestration.md。
type AiAgentApp struct{ ent.Schema }

func (AiAgentApp) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "sys_ai_agent_apps", Charset: "utf8mb4", Collation: "utf8mb4_bin"},
		entsql.WithComments(true),
		schema.Comment("AI Agent 编排应用"),
	}
}

func (AiAgentApp) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("应用名称（租户内唯一）").
			NotEmpty(),
		field.String("description").
			Comment("应用描述").
			Optional().
			Nillable(),
		field.Text("graph").
			Comment("图 DSL（JSON：nodes/edges，五种节点）").
			Optional().
			Nillable(),
		field.Bool("enabled").
			Comment("是否启用（停用的应用拒绝运行）").
			Default(true),
	}
}

func (AiAgentApp) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TimeAt{},
		mixin.OperatorID64{},
		mixin.TenantID[int64]{},
	}
}

func (AiAgentApp) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "name").Unique().
			StorageKey("uidx_sys_ai_agent_apps_tenant_name"),
	}
}

func (AiAgentApp) Edges() []ent.Edge {
	return []ent.Edge{
		// 版本快照：删除应用时级联清理其全部快照（docs/ai_agent_orchestration.md「版本化」）
		edge.To("revisions", AiAgentRevision.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
	}
}
