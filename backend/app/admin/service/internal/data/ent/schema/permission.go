package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

type Permission struct {
	ent.Schema
}

func (Permission) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_permissions",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("权限点表"),
	}
}

func (Permission) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			Nillable().
			Comment("权限名称（如：删除用户）"),

		field.String("code").
			Nillable().
			Comment("权限编码（如：opm:user:delete、order:export）"),

		field.Uint32("group_id").
			Optional().
			Nillable().
			Comment("关联权限分组 ID"),
	}
}

func (Permission) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.SwitchStatus{},
		mixin.Description{},
	}
}

func (Permission) Indexes() []ent.Index {
	return []ent.Index{
		// 唯一约束：同一租户下 code 唯一
		index.Fields("code").
			Unique().
			StorageKey("uix_perm_code"),

		// 单列索引：按 name 快速查询（全局或模糊搜索场景）
		index.Fields("name").
			StorageKey("idx_perm_name"),

		// 单列索引：按 group_id 快速查询
		index.Fields("group_id").
			StorageKey("idx_perm_group_id"),
	}
}
