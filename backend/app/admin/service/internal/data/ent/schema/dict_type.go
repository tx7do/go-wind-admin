package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"

	"go-wind-admin/app/admin/service/internal/data/ent/privacy"
	"go-wind-admin/app/admin/service/internal/data/ent/rule"
)

// DictType holds the schema definition for the DictType entity.
type DictType struct {
	ent.Schema
}

func (DictType) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_dict_types",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("字典类型表"),
	}
}

// Fields of the DictType.
func (DictType) Fields() []ent.Field {
	return []ent.Field{
		field.String("type_code").
			Comment("字典类型唯一代码").
			NotEmpty().
			Optional().
			Nillable(),

		field.String("type_name").
			Comment("字典类型名称").
			NotEmpty().
			Optional().
			Nillable(),
	}
}

// Mixin of the DictType.
func (DictType) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.IsEnabled{},
		mixin.SortOrder{},
		mixin.Description{},
		mixin.TenantID{},
	}
}

// Edges of the DictType.
func (DictType) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("entries", DictEntry.Type).
			Ref("sys_dict_types"),
	}
}

// Policy for all schemas that embed DictType.
func (DictType) Policy() ent.Policy {
	return privacy.Policy{
		Query: rule.TenantQueryPolicy(),
	}
}

// Indexes of the DictType.
func (DictType) Indexes() []ent.Index {
	return []ent.Index{
		// 租户级唯一：同一租户下 type_code 唯一
		index.Fields("tenant_id", "type_code").
			Unique().
			StorageKey("uix_sys_dict_types_tenant_type_code"),

		// 常用查询：在租户范围内按名称查找
		index.Fields("tenant_id", "type_name").
			StorageKey("idx_sys_dict_types_tenant_type_name"),

		// 单列索引：按名称快速查询/模糊搜索
		index.Fields("type_name").
			StorageKey("idx_sys_dict_types_type_name"),

		// 支持按租户快速筛选
		index.Fields("tenant_id").
			StorageKey("idx_sys_dict_types_tenant_id"),

		// 按启用状态过滤
		index.Fields("is_enabled").
			StorageKey("idx_sys_dict_types_is_enabled"),

		// 按排序值查询/排序优化
		index.Fields("sort_order").
			StorageKey("idx_sys_dict_types_sort_order"),
	}
}
