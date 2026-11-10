package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/tx7do/go-crud/entgo/mixin"
)

// AiChunk 知识库切片：embedding 向量列不进 ent schema——ent 对 pgvector 自定义
// 列类型支持受限，该列由启动期手写 SQL 补建（CREATE EXTENSION vector +
// data.EnsureVectorColumnDim 定维补建 vector(1536)），读写走原生 SQL。
type AiChunk struct{ ent.Schema }

func (AiChunk) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "sys_ai_chunks", Charset: "utf8mb4", Collation: "utf8mb4_bin"},
		entsql.WithComments(true),
		schema.Comment("AI 知识库切片"),
	}
}

func (AiChunk) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("doc_id").Comment("所属文档ID（edge 外键）").Optional().Nillable(),
		field.Text("content").Comment("切片文本").Optional().Nillable(),
		field.Uint32("chunk_index").Comment("切片序号").Optional().Nillable(),
	}
}

func (AiChunk) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.CreatedAt{},
		mixin.TenantID[int64]{},
	}
}

func (AiChunk) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("doc", AiDoc.Type).
			Ref("chunks").
			Field("doc_id").
			Unique(),
	}
}
