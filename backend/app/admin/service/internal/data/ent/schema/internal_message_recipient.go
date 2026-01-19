package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"

	"go-wind-admin/app/admin/service/internal/data/ent/privacy"
	"go-wind-admin/app/admin/service/internal/data/ent/rule"
)

// InternalMessageRecipient holds the schema definition for the InternalMessageRecipient entity.
type InternalMessageRecipient struct {
	ent.Schema
}

func (InternalMessageRecipient) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "internal_message_recipients",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("站内信消息用户接收信息表"),
	}
}

// Fields of the InternalMessageRecipient.
func (InternalMessageRecipient) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("message_id").
			Comment("站内信内容ID").
			Optional().
			Nillable(),

		field.Uint32("recipient_user_id").
			Comment("接收者用户ID").
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("消息状态").
			NamedValues(
				"Sent", "SENT",
				"Received", "RECEIVED",
				"Read", "READ",
				"Revoked", "REVOKED",
				"Deleted", "DELETED",
			).
			Optional().
			Nillable(),

		field.Time("received_at").
			Comment("消息到达用户收件箱的时间").
			Optional().
			Nillable(),

		field.Time("read_at").
			Comment("用户阅读消息的时间").
			Optional().
			Nillable(),
	}
}

// Mixin of the InternalMessageRecipient.
func (InternalMessageRecipient) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.TenantID{},
	}
}

// Policy for all schemas that embed InternalMessageRecipient.
func (InternalMessageRecipient) Policy() ent.Policy {
	return privacy.Policy{
		Query: rule.TenantQueryPolicy(),
	}
}

func (InternalMessageRecipient) Indexes() []ent.Index {
	return []ent.Index{
		// 按租户 + 创建时间，用于租户范围内的时间区间查询与分页
		index.Fields("tenant_id", "created_at").
			StorageKey("idx_internal_msg_recipient_tenant_created_at"),

		// 按租户 + 消息ID，用于检索某消息的所有接收者（租户隔离）
		index.Fields("tenant_id", "message_id").
			StorageKey("idx_internal_msg_recipient_tenant_message"),

		// 按租户 + 接收者 + 创建时间，用于按用户查看收件记录（按时间范围）
		index.Fields("tenant_id", "recipient_user_id", "created_at").
			StorageKey("idx_internal_msg_recipient_tenant_recipient_created_at"),

		// 按租户 + 状态 + 创建时间，用于状态过滤与统计
		index.Fields("tenant_id", "status", "created_at").
			StorageKey("idx_internal_msg_recipient_tenant_status_created_at"),

		// 按接收者 + 状态 + 创建时间，用于单用户的状态过滤（跨租户场景若需，仍可按前缀添加 tenant_id）
		index.Fields("recipient_user_id", "status", "created_at").
			StorageKey("idx_internal_msg_recipient_recipient_status_created_at"),

		// 按消息 + 接收者，用于快速定位特定消息-接收者的记录
		index.Fields("message_id", "recipient_user_id").
			StorageKey("idx_internal_msg_recipient_message_recipient"),
	}
}
