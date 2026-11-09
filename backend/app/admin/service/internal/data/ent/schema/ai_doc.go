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

// AiDoc 知识库文档：一份纯文本，切片向量化后落入 AiChunk。
type AiDoc struct{ ent.Schema }

func (AiDoc) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "sys_ai_docs", Charset: "utf8mb4", Collation: "utf8mb4_bin"},
		entsql.WithComments(true),
		schema.Comment("AI 知识库文档"),
	}
}

func (AiDoc) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("base_id").Comment("所属知识库ID（edge 外键）").Optional().Nillable(),
		field.String("name").Comment("文档名称").Optional().Nillable(),
		field.Uint32("chunk_count").Comment("切片数").Optional().Nillable(),
		field.String("status").Comment("状态：READY/FAILED").Optional().Nillable(),
		field.Text("error_message").Comment("失败原因").Optional().Nillable(),
		field.Uint64("user_id").Comment("上传人用户ID").Optional().Nillable(),
	}
}

func (AiDoc) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TimeAt{},
		mixin.OperatorID64{},
		mixin.TenantID[uint64]{},
	}
}

func (AiDoc) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("base", AiKnowledgeBase.Type).
			Ref("docs").
			Field("base_id").
			Unique(),
		// 删除文档时切片随删
		edge.To("chunks", AiChunk.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
	}
}

func (AiDoc) Indexes() []ent.Index {
	// base_id 是 edge 外键列，ent 自动建 FK 索引
	return []ent.Index{
		index.Fields("tenant_id").StorageKey("idx_sys_ai_docs_tenant"),
	}
}
