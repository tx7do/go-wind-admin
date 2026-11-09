package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/tx7do/go-crud/entgo/mixin"
)

// AiProvider AI 模型提供商：一行 = 一个可调用的模型端点配置。
// "怎么调"在 service 层的客户端工厂，本表是"用什么账号/端点调"。
type AiProvider struct{ ent.Schema }

func (AiProvider) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "sys_ai_providers", Charset: "utf8mb4", Collation: "utf8mb4_bin"},
		entsql.WithComments(true),
		schema.Comment("AI 模型提供商"),
	}
}

func (AiProvider) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").Comment("提供商显示名称").Optional().Nillable(),
		field.Enum("model_type").
			Comment("模型部署形态").
			NamedValues(
				"LOCAL", "LOCAL",
				"CLOUD", "CLOUD",
			).
			Optional().
			Nillable(),
		field.String("model_name").Comment("模型名称").Optional().Nillable(),
		field.String("base_url").Comment("云端 OpenAI 兼容 API 地址").Optional().Nillable(),
		field.String("organization").Comment("OpenAI Organization").Optional().Nillable(),
		field.String("api_key").Comment("API Key（AES-GCM 加密落库，enc: 前缀）").Optional().Nillable().Sensitive(),
		field.String("api_key_hint").Comment("API Key 脱敏提示（如 sk-***abcd）").Optional().Nillable(),
		field.String("local_host").Comment("本地模型主机").Optional().Nillable(),
		field.Int("local_port").Comment("本地模型端口").Optional().Nillable(),
		field.Int("timeout_seconds").Comment("连接超时秒数").Optional().Nillable(),
		field.Text("system_prompt").Comment("系统提示词").Optional().Nillable(),
		field.Bool("is_default").Comment("是否默认提供商").Default(false).Optional().Nillable(),
		field.String("remark").Comment("备注").Optional().Nillable(),
	}
}

func (AiProvider) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TimeAt{},
		mixin.OperatorID64{},
		mixin.IsEnabled{},
	}
}

func (AiProvider) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("is_enabled", "is_default").StorageKey("idx_sys_ai_providers_enabled_default"),
		index.Fields("created_at").StorageKey("idx_sys_ai_providers_created_at"),
	}
}
