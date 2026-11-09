package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// NotificationTemplate 通知模板：可复用的标题/正文占位模板，平台全局（不挂租户），
// 与渠道/规则同域。code 是 SendDirect 与测试渲染的引用锚，全局唯一。
type NotificationTemplate struct {
	ent.Schema
}

func (NotificationTemplate) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "notification_templates",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("通知模板表"),
	}
}

// Fields of the NotificationTemplate.
func (NotificationTemplate) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("模板名称").
			NotEmpty().
			Optional().
			Nillable(),

		field.String("code").
			Comment("模板编码（全局唯一，发送方以 template_code 引用）").
			NotEmpty().
			Optional().
			Nillable(),

		field.String("title_template").
			Comment("标题模板，支持 {{var}} 占位符").
			NotEmpty().
			Optional().
			Nillable(),

		field.String("content_template").
			Comment("正文模板，支持 {{var}} 占位符").
			NotEmpty().
			Optional().
			Nillable(),
	}
}

// Mixin of the NotificationTemplate.
func (NotificationTemplate) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TimeAt{},
		mixin.OperatorID64{},
		mixin.IsEnabled{},
		mixin.Remark{},
	}
}

// Indexes of the NotificationTemplate.
func (NotificationTemplate) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("code").Unique().StorageKey("uk_notification_tpl_code"),
		index.Fields("is_enabled").StorageKey("idx_notification_tpl_enabled"),
	}
}
