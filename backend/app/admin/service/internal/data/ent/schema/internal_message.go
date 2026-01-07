package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/tx7do/go-crud/entgo/mixin"
)

// InternalMessage holds the schema definition for the InternalMessage entity.
type InternalMessage struct {
	ent.Schema
}

func (InternalMessage) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "internal_messages",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("站内信消息表"),
	}
}

// Fields of the InternalMessage.
func (InternalMessage) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			Comment("消息标题").
			Optional().
			Nillable(),

		field.String("content").
			Comment("消息内容").
			Optional().
			Nillable(),

		field.Uint32("sender_id").
			Comment("发送者用户ID").
			Nillable(),

		field.Uint32("category_id").
			Comment("分类ID").
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("消息状态").
			NamedValues(
				"Draft", "DRAFT",
				"Published", "PUBLISHED",
				"Scheduled", "SCHEDULED",
				"Revoked", "REVOKED",
				"Archived", "ARCHIVED",
				"Deleted", "DELETED",
			).
			Default("DRAFT").
			Optional().
			Nillable(),

		field.Enum("type").
			Comment("消息类型").
			NamedValues(
				"Notification", "NOTIFICATION",
				"Private", "PRIVATE",
				"Group", "GROUP",
			).
			Default("NOTIFICATION").
			Optional().
			Nillable(),
	}
}

// Mixin of the InternalMessage.
func (InternalMessage) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID{},
	}
}

func (InternalMessage) Indexes() []ent.Index {
	return []ent.Index{
		// 按租户 + 创建时间，用于时间区间查询与分页
		index.Fields("tenant_id", "created_at").
			StorageKey("idx_internal_msg_tenant_created_at"),

		// 按租户 + 状态 + 创建时间，用于状态过滤与统计
		index.Fields("tenant_id", "status", "created_at").
			StorageKey("idx_internal_msg_tenant_status_created_at"),

		// 按租户 + 发送者 + 创建时间，用于按发送者检索
		index.Fields("tenant_id", "sender_id", "created_at").
			StorageKey("idx_internal_msg_tenant_sender_created_at"),

		// 按租户 + 分类，用于分类筛选
		index.Fields("tenant_id", "category_id").
			StorageKey("idx_internal_msg_tenant_category"),

		// 按租户 + 操作者 + 创建时间，用于审计回溯
		index.Fields("tenant_id", "created_by", "created_at").
			StorageKey("idx_internal_msg_tenant_created_by_created_at"),
	}
}
