package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// NotificationRule holds the schema definition for the NotificationRule entity.
//
// 事件 → 渠道的路由表：`SendDirect` 每次派发拿 `event_type` 来这一张表查"走哪个渠道、
// 要不要异步"。它是 §3.5 那张 Go 静态表（`eventChannels` + `asyncDispatchEvents`）的落地，
// 两列一一对应；播种见 service/notification_rule_service.go 的 init（空表才播，删掉不复活）。
//
// 与 sys_notification_channels / sys_notification_deliveries 同域：都是平台级配置与运行痕迹，
// 不挂租户谓词（理由见 docs/notification_domain_design.md §6 决策点 1、3）。
//
// 一条事件类型一行（event_type 唯一索引）：一张表容不下"同一事件发多个渠道"，
// 真需要时是把唯一索引换成 (event_type, channel) 复合唯一，而不是让调用方猜哪一行生效。
type NotificationRule struct {
	ent.Schema
}

func (NotificationRule) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_notification_rules",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("通知路由规则表（事件类型 → 渠道）"),
	}
}

// Fields of the NotificationRule.
func (NotificationRule) Fields() []ent.Field {
	return []ent.Field{
		// 枚举列一律 Optional().Nillable()：copier 的枚举转换是"指针↔指针对"，
		// 值型枚举列配可选指针 DTO 字段会失配并把 DTO 落成零值枚举（同 sys_notification_deliveries）。
		field.Enum("event_type").
			Comment("业务事件类型").
			NamedValues(
				"PasswordResetCode", "PASSWORD_RESET_CODE",
				"ContactBindCode", "CONTACT_BIND_CODE",
				"ChannelTestEmail", "CHANNEL_TEST_EMAIL",
				"InternalMessage", "INTERNAL_MESSAGE",
			).
			Optional().
			Nillable(),

		field.Enum("channel").
			Comment("投递渠道").
			NamedValues(
				"Email", "EMAIL",
				"Sms", "SMS",
				"Webhook", "WEBHOOK",
				"Internal", "INTERNAL",
			).
			Optional().
			Nillable(),

		field.Bool("is_async").
			Comment("是否异步派发（true = 入队 asynq，请求不等投递结论）").
			Default(false).
			Optional().
			Nillable(),
	}
}

// Mixin of the NotificationRule.
func (NotificationRule) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.IsEnabled{},
		mixin.Remark{},
	}
}

// Indexes of the NotificationRule.
func (NotificationRule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("event_type").Unique().
			StorageKey("uidx_sys_notification_rule_event_type"),
	}
}
