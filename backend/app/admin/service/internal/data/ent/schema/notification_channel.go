package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// NotificationChannel holds the schema definition for the NotificationChannel entity.
// 通知渠道为平台级全局配置（无租户隔离）：由平台管理员维护，供找回密码/
// 联系方式验证码/站内信外发等通知能力使用。
type NotificationChannel struct {
	ent.Schema
}

func (NotificationChannel) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_notification_channels",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("通知渠道表"),
	}
}

// Fields of the NotificationChannel.
func (NotificationChannel) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("渠道名称").
			NotEmpty(),
		field.Enum("type").
			Comment("渠道类型（一期仅实现 EMAIL）").
			NamedValues(
				"Email", "EMAIL",
				"Webhook", "WEBHOOK",
			).
			Default("EMAIL"),
		field.String("smtp_host").
			Comment("SMTP 服务器地址").
			Optional().
			Nillable(),
		field.Uint32("smtp_port").
			Comment("SMTP 端口（25/465/587）").
			Optional().
			Nillable(),
		field.String("smtp_username").
			Comment("SMTP 用户名").
			Optional().
			Nillable(),
		field.String("smtp_password").
			Comment("SMTP 密码/授权码（EncryptIfNeeded 加密存储）").
			Sensitive().
			Optional().
			Nillable(),
		field.String("smtp_from").
			Comment("发件人地址").
			Optional().
			Nillable(),
		field.Enum("smtp_tls").
			Comment("加密方式").
			NamedValues(
				"None", "NONE",
				"StartTls", "START_TLS",
				"Ssl", "SSL",
			).
			Default("START_TLS").
			Optional().
			Nillable(),

		// WEBHOOK 渠道的两列：SMTP 那几列对它没有意义，见 docs §6 决策点 2（选 A：按渠道加列，
		// 不上 settings JSON——SMTP 会因此变成两种真相源）。
		field.String("webhook_url").
			Comment("Webhook 回调地址（仅 WEBHOOK 渠道）").
			Optional().
			Nillable(),
		field.String("webhook_secret").
			Comment("Webhook 签名密钥（EncryptIfNeeded 加密存储，仅 WEBHOOK 渠道）").
			Sensitive().
			Optional().
			Nillable(),
		// 出站风格与载荷模板：同属"这一条 webhook 到底发成什么形状"，所以跟着渠道行走，
		// 不进规则表（规则表答的是"这个事件走哪个渠道"）。见 docs §4 N。
		field.Enum("webhook_sign_style").
			Comment("Webhook 出站风格（签名位置/算法与应答判据，仅 WEBHOOK 渠道）").
			NamedValues(
				"Custom", "CUSTOM",
				"None", "NONE",
				"Dingtalk", "DINGTALK",
				"Feishu", "FEISHU",
				"Wecom", "WECOM",
			).
			Default("CUSTOM").
			Optional().
			Nillable(),
		field.String("webhook_payload_template").
			Comment("Webhook 载荷模板（{{占位符}} 渲染；留空则用该风格的内置默认 JSON）").
			Optional().
			Nillable(),
	}
}

// Mixin of the NotificationChannel.
func (NotificationChannel) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.SwitchStatus{},
	}
}

// Indexes of the NotificationChannel.
func (NotificationChannel) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name").Unique().
			StorageKey("uidx_sys_notification_channel_name"),
	}
}
