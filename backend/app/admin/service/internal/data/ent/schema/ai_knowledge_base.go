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

// AiKnowledgeBase AI 知识库：聚合文档，检索注入对话上下文（RAG）。
type AiKnowledgeBase struct{ ent.Schema }

func (AiKnowledgeBase) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "sys_ai_knowledge_bases", Charset: "utf8mb4", Collation: "utf8mb4_bin"},
		entsql.WithComments(true),
		schema.Comment("AI 知识库"),
	}
}

func (AiKnowledgeBase) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").Comment("知识库名称").Optional().Nillable(),
		field.String("description").Comment("描述").Optional().Nillable(),
		field.Uint64("provider_id").Comment("向量化使用的模型提供商ID").Optional().Nillable(),
		field.String("embedding_model").Comment("embedding 模型名").Optional().Nillable(),
		field.Uint64("user_id").Comment("创建人用户ID").Optional().Nillable(),
	}
}

func (AiKnowledgeBase) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TimeAt{},
		mixin.OperatorID64{},
		mixin.TenantID[uint64]{},
	}
}

func (AiKnowledgeBase) Edges() []ent.Edge {
	return []ent.Edge{
		// 外键列 base_id 由 AiDoc 侧 Fields+edge.Field 声明（本仓 O2M 同型写法）
		edge.To("docs", AiDoc.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
	}
}

func (AiKnowledgeBase) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id").StorageKey("idx_sys_ai_knowledge_bases_tenant"),
	}
}
