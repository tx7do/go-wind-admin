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

// AiMessage AI 对话消息：USER/ASSISTANT/SYSTEM 三种角色，ASSISTANT 行带 token 用量快照。
type AiMessage struct{ ent.Schema }

func (AiMessage) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "sys_ai_messages", Charset: "utf8mb4", Collation: "utf8mb4_bin"},
		entsql.WithComments(true),
		schema.Comment("AI 对话消息"),
	}
}

func (AiMessage) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("role").
			Comment("消息角色").
			NamedValues(
				"USER", "USER",
				"ASSISTANT", "ASSISTANT",
				"SYSTEM", "SYSTEM",
			).
			Optional().
			Nillable(),
		field.Text("content").Comment("消息正文（Markdown）").Optional().Nillable(),
		field.String("model_name").Comment("模型名称快照").Optional().Nillable(),
		field.Uint32("prompt_tokens").Comment("输入 token 数").Optional().Nillable(),
		field.Uint32("completion_tokens").Comment("输出 token 数").Optional().Nillable(),
		field.Uint32("duration_ms").Comment("生成耗时（毫秒）").Optional().Nillable(),
		field.Text("error_message").Comment("生成失败原因").Optional().Nillable(),
		field.Int64("user_id").Comment("归属用户ID").Optional().Nillable(),
		field.Int64("conversation_id").Comment("所属会话ID（edge 外键）").Optional().Nillable(),
	}
}

func (AiMessage) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TimeAt{},
		mixin.OperatorID64{},
		mixin.TenantID[int64]{},
	}
}

func (AiMessage) Edges() []ent.Edge {
	return []ent.Edge{
		// 外键列 conversation_id 在上方 Fields 显式声明（dict_entry 的 type_id 同型）；
		// 不写 Required()——M2O 边 Required 会让 Create 运行时校验 missing required edge 而 500。
		edge.From("conversation", AiConversation.Type).
			Ref("messages").
			Field("conversation_id").
			Unique(),
	}
}

func (AiMessage) Indexes() []ent.Index {
	// conversation_id 是 edge 外键列，ent 会自动建 FK 索引；此处不再显式索引
	// （ent edge 外键列不能在从表 Indexes() 引用）。
	return []ent.Index{
		index.Fields("tenant_id", "user_id").StorageKey("idx_sys_ai_messages_tenant_user"),
	}
}
