package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// NotificationDelivery holds the schema definition for the NotificationDelivery entity.
// 通知投递台账：一条通知 × 一个渠道 × 一个收件人 = 一行。
//
// 与 sys_notification_channels 同为平台级全局表（无租户隔离）：渠道花名册本就是平台配的，
// 台账记录"平台用了它发没发出去"。收件地址跨租户，因此读侧只开放给平台管理员
// （菜单 authority sys:platform_admin），不挂租户谓词的理由见 docs/notification_domain_design.md §6。
type NotificationDelivery struct {
	ent.Schema
}

func (NotificationDelivery) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_notification_deliveries",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("通知投递台账表"),
	}
}

// Fields of the NotificationDelivery.
func (NotificationDelivery) Fields() []ent.Field {
	return []ent.Field{
		// 三个枚举列一律 Optional().Nillable()：copier 注册的枚举转换是"指针↔指针对"，
		// 值型枚举列（无 Nillable）配可选指针 DTO 字段会失配并让 DTO 落成零值枚举
		// ——同一个坑在 sys_notification_channels.type 上踩过（见 notification_channel_repo.go
		// queryTypeByIDs 的注释），那里只能靠批量回填绕开。
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

		field.Uint32("channel_id").
			Comment("实际选中的渠道配置ID").
			Optional().
			Nillable(),

		field.Uint32("recipient_user_id").
			Comment("收件用户ID（直发模式可为空）").
			Optional().
			Nillable(),

		// 产生本条投递的业务对象主键，含义由 event_type 决定（INTERNAL_MESSAGE → sys_internal_messages.id）。
		// 台账不存正文快照（理由见 docs/notification_domain_design.md §3.3），
		// 这一列是"这条投递发的是什么"的唯一回跳入口。
		field.Uint32("related_id").
			Comment("关联业务对象ID（按事件类型解释）").
			Optional().
			Nillable(),

		field.String("target").
			Comment("投递目标（脱敏后的地址）").
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("投递状态").
			NamedValues(
				"Sending", "SENDING",
				"Sent", "SENT",
				"Failed", "FAILED",
				"Skipped", "SKIPPED",
			).
			Default("SENDING").
			Optional().
			Nillable(),

		field.String("last_error").
			Comment("最近一次失败原因").
			Optional().
			Nillable(),

		field.Time("sent_at").
			Comment("投递完成时间").
			Optional().
			Nillable(),
	}
}

// Mixin of the NotificationDelivery.
func (NotificationDelivery) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
	}
}

// Indexes of the NotificationDelivery.
func (NotificationDelivery) Indexes() []ent.Index {
	return []ent.Index{
		// 台账页默认按时间倒序翻页，并按事件类型/状态过滤
		index.Fields("created_at").
			StorageKey("idx_sys_notification_delivery_created_at"),
		index.Fields("event_type", "created_at").
			StorageKey("idx_sys_notification_delivery_event_created_at"),
		index.Fields("status", "created_at").
			StorageKey("idx_sys_notification_delivery_status_created_at"),
		index.Fields("recipient_user_id").
			StorageKey("idx_sys_notification_delivery_recipient_user"),

		// "这条公告/审批单到底发出去了没"——按业务对象反查投递事实
		index.Fields("event_type", "related_id").
			StorageKey("idx_sys_notification_delivery_event_related"),
	}
}
