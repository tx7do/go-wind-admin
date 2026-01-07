package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/tx7do/go-crud/entgo/mixin"
)

// RoleTemplate holds the schema definition for the RoleTemplate entity.
type RoleTemplate struct {
	ent.Schema
}

func (RoleTemplate) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_role_templates",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("角色模板"),
	}
}

// Fields of the RoleTemplate.
func (RoleTemplate) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("角色名称").
			NotEmpty().
			Nillable(),

		field.String("code").
			Comment("角色编码").
			NotEmpty().
			Nillable(),

		field.String("category").
			Comment("模板分类").
			Optional().
			Nillable(),

		field.Strings("permissions").
			Comment("角色权限列表").
			Optional(),

		field.Bool("is_default").
			Comment("是否是默认模板").
			Default(false).
			Nillable(),

		field.Bool("is_system").
			Comment("是否是系统模板（不可删除）").
			Default(false).
			Nillable(),
	}
}

// Mixin of the RoleTemplate.
func (RoleTemplate) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID{},
		mixin.Description{},
		mixin.SwitchStatus{},
		mixin.SortOrder{},
	}
}

// Indexes of the RoleTemplate.
func (RoleTemplate) Indexes() []ent.Index {
	return []ent.Index{
		// 在租户范围内保证 name 唯一
		index.Fields("tenant_id", "name").Unique().StorageKey("idx_sys_role_tpl_tenant_name"),
		// 在租户范围内保证 code 唯一
		index.Fields("tenant_id", "code").Unique().StorageKey("idx_sys_role_tpl_tenant_code"),
		// 分类查询加速
		index.Fields("category").StorageKey("idx_sys_role_tpl_category"),
		// 系统模板标识查询加速
		index.Fields("is_system").StorageKey("idx_sys_role_tpl_is_system"),
	}
}
