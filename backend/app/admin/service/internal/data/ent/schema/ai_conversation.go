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

// AiConversation AI 对话会话：归属用户，聚合消息。
type AiConversation struct{ ent.Schema }

func (AiConversation) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "sys_ai_conversations", Charset: "utf8mb4", Collation: "utf8mb4_bin"},
		entsql.WithComments(true),
		schema.Comment("AI 对话会话"),
	}
}

func (AiConversation) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").Comment("会话标题").Optional().Nillable(),
		field.Uint64("provider_id").Comment("会话使用的提供商ID（0=未指定）").Optional().Nillable(),
		field.Uint64("user_id").Comment("归属用户ID").Optional().Nillable(),
		field.Time("last_message_at").Comment("最近一条消息时间").Optional().Nillable(),
	}
}

func (AiConversation) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TimeAt{},
		mixin.OperatorID64{},
		mixin.TenantID[uint64]{},
	}
}

func (AiConversation) Edges() []ent.Edge {
	return []ent.Edge{
		// 外键列 conversation_id 由 AiMessage 侧 Fields+edge.Field 声明，此处不再 StorageKey
		// （重复声明会报 "should be replaced with Field()"）；不用 Required()——
		// O2M 边写 Required() 会造成 Create 运行时 500（missing required edge）。
		edge.To("messages", AiMessage.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
	}
}

func (AiConversation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "user_id").StorageKey("idx_sys_ai_conversations_tenant_user"),
		index.Fields("last_message_at").StorageKey("idx_sys_ai_conversations_last_message_at"),
	}
}
