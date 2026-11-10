package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/tx7do/go-crud/entgo/mixin"
)

// AiUsageLog AI 调用用量流水：每次成功的模型调用一行，配额按 total_tokens 聚合。
// 只增不改不删（留档对账），无 operator/软删除字段。
type AiUsageLog struct{ ent.Schema }

func (AiUsageLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "sys_ai_usage_logs", Charset: "utf8mb4", Collation: "utf8mb4_bin"},
		entsql.WithComments(true),
		schema.Comment("AI 调用用量流水"),
	}
}

func (AiUsageLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("provider_id").Comment("提供商ID").Optional().Nillable(),
		field.Int64("conversation_id").Comment("会话ID（0=非对话调用）").Optional().Nillable(),
		field.Int64("user_id").Comment("发起用户ID").Optional().Nillable(),
		field.String("model_name").Comment("模型名称").Optional().Nillable(),
		field.Uint32("prompt_tokens").Comment("输入 token 数").Optional().Nillable(),
		field.Uint32("completion_tokens").Comment("输出 token 数").Optional().Nillable(),
		field.Uint32("total_tokens").Comment("总 token 数").Optional().Nillable(),
		field.Uint32("duration_ms").Comment("调用耗时（毫秒）").Optional().Nillable(),
	}
}

func (AiUsageLog) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.CreatedAt{},
		mixin.TenantID[int64]{},
	}
}

func (AiUsageLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "created_at").StorageKey("idx_sys_ai_usage_logs_tenant_created_at"),
		index.Fields("user_id").StorageKey("idx_sys_ai_usage_logs_user"),
	}
}
