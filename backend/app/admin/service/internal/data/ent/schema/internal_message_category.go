package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/tx7do/go-crud/entgo/mixin"
)

// InternalMessageCategory holds the schema definition for the InternalMessageCategory entity.
type InternalMessageCategory struct {
	ent.Schema
}

func (InternalMessageCategory) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "internal_message_categories",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("站内信消息分类表"),
	}
}

// Fields of the InternalMessageCategory.
func (InternalMessageCategory) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("名称").
			NotEmpty().
			Optional().
			Nillable(),

		field.String("code").
			Comment("编码").
			NotEmpty().
			Optional().
			Nillable(),

		field.String("icon_url").
			Comment("图标URL").
			Optional().
			Nillable(),
	}
}

// Mixin of the InternalMessageCategory.
func (InternalMessageCategory) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.IsEnabled{},
		mixin.SortOrder{},
		mixin.Remark{},
		mixin.TenantID{},
	}
}

// Indexes of the InternalMessageCategory.
func (InternalMessageCategory) Indexes() []ent.Index {
	return []ent.Index{
		// 在租户范围内保证 code 唯一
		index.Fields("tenant_id", "code").Unique().StorageKey("idx_internal_msg_cat_tenant_code"),

		// 按租户 + 名称，用于按名称检索
		index.Fields("tenant_id", "name").StorageKey("idx_internal_msg_cat_tenant_name"),

		// 按租户 + 启用状态，用于过滤启用/禁用项
		index.Fields("tenant_id", "is_enabled").StorageKey("idx_internal_msg_cat_tenant_enabled"),

		// 按租户 + 创建时间，用于时间范围查询与分页
		index.Fields("tenant_id", "created_at").StorageKey("idx_internal_msg_cat_tenant_created_at"),

		// 按租户 + 操作者，用于审计与变更追溯
		index.Fields("tenant_id", "created_by").StorageKey("idx_internal_msg_cat_tenant_created_by"),
	}
}
