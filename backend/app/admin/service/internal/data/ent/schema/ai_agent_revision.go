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

// AiAgentRevision holds the schema definition for the AiAgentRevision entity.
//
// AI Agent 应用版本快照（docs/ai_agent_orchestration.md「版本化」，镜像 BI 的 revisions 形态）：
// 每次成功更新把**更新前**的 graph 留一行快照（SAVE），恢复 = 把快照写回主表
// （恢复动作本身先快照当前版本，可逆）。只增不改不删（表级无更新入口），主表删除时级联清理。
// 不带 version 列：agent 主表无乐观锁版本号，快照按 created_at 排序即可。
type AiAgentRevision struct {
	ent.Schema
}

func (AiAgentRevision) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_ai_agent_revisions",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("AI Agent 应用版本快照表（每次更新留旧版 graph；只增不更新，随主表级联删除）"),
	}
}

// Fields of the AiAgentRevision.
func (AiAgentRevision) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("app_id").
			Comment("所属应用 ID").
			Optional().
			Nillable(),

		field.Text("graph").
			Comment("更新前的图 DSL 快照").
			Optional().
			Nillable(),

		field.Enum("source").
			Comment("快照来源（SAVE=常规更新触发 RESTORE=恢复前留底）").
			NamedValues("SAVE", "SAVE", "RESTORE", "RESTORE").
			Default("SAVE").
			Optional().
			Nillable(),
	}
}

// Mixin of the AiAgentRevision.
func (AiAgentRevision) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TimeAt{},
		mixin.OperatorID64{},
		mixin.TenantID[int64]{},
	}
}

// Edges of the AiAgentRevision.
func (AiAgentRevision) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("app", AiAgentApp.Type).
			Ref("revisions").
			Field("app_id").
			Unique(),
	}
}

// Indexes of the AiAgentRevision.
func (AiAgentRevision) Indexes() []ent.Index {
	return []ent.Index{
		// 历史列表按应用拉取、新快照在前
		index.Fields("app_id", "created_at").
			StorageKey("idx_ai_revisions_app_created"),
	}
}
