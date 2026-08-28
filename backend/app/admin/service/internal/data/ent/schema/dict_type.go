package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
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
			Comment("字典类型唯一编码").
			NotEmpty().
			Immutable().
			Optional().
			Nillable(),

		field.String("type_name").
			Comment("字典类型名称（中文，仅后台用）").
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
		mixin.TenantID[uint32]{},
	}
}

// Edges of the DictType.
func (DictType) Edges() []ent.Edge {
	return []ent.Edge{
		// 一对多边不可 Required()：业务上先建类型后加条目，
		// Required 会让 Create 在运行时校验 "missing required edge" 而 500。
		edge.To("entries", DictEntry.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}).
			StorageKey(edge.Column("type_id")),
	}
}

// Indexes of the DictType.
func (DictType) Indexes() []ent.Index {
	return []ent.Index{
		// 租户级唯一：同一租户下 type_code 唯一
		index.Fields("tenant_id", "type_code").
			Unique().
			StorageKey("uix_sys_dict_types_tenant_type_code"),

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
