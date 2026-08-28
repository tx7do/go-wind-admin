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

// DictEntry holds the schema definition for the DictEntry entity.
type DictEntry struct {
	ent.Schema
}

func (DictEntry) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_dict_entries",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("字典项表"),
	}
}

// Fields of the DictEntry.
func (DictEntry) Fields() []ent.Field {
	return []ent.Field{
		field.String("entry_value").
			Comment("字典项的实际值").
			NotEmpty().
			Nillable(),

		field.Int32("numeric_value").
			Comment("数值型值").
			Optional().
			Nillable(),
	}
}

// Mixin of the DictEntry.
func (DictEntry) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.SortOrder{},
		mixin.IsEnabled{},
		mixin.TenantID[uint32]{},
	}
}

// Edges of the DictEntry.
func (DictEntry) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("dict_type", DictType.Type).
			Ref("entries").
			Unique(),

		// 一对多边不可 Required()：i18n 在条目落库后单独建，
		// Required 会让 Create 在运行时校验 "missing required edge" 而 500。
		edge.To("i18ns", DictEntryI18n.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}).
			StorageKey(edge.Column("entry_id")),
	}
}

// Indexes of the DictEntry.
func (DictEntry) Indexes() []ent.Index {
	return []ent.Index{
		// 唯一约束：同一租户下同一类型的 entry_value 唯一
		index.Fields("tenant_id", "id", "entry_value").
			Unique().
			StorageKey("uix_sys_dict_entries_tenant_type_value"),

		// 常用查询：按租户+类型 查询该类型下所有条目
		index.Fields("tenant_id", "id").
			StorageKey("idx_sys_dict_entries_tenant_type"),

		// 常用查询：按租户+值 查询条目（租户范围内的快速定位）
		index.Fields("tenant_id", "entry_value").
			StorageKey("idx_sys_dict_entries_tenant_entry_value"),

		// 单列索引：按 entry_value 快速查询（全局或模糊搜索）
		index.Fields("entry_value").
			StorageKey("idx_sys_dict_entries_entry_value"),

		// 单列索引：按数值字段快速查询/排序
		index.Fields("numeric_value").
			StorageKey("idx_sys_dict_entries_numeric_value"),

		// 支持按租户快速筛选
		index.Fields("tenant_id").
			StorageKey("idx_sys_dict_entries_tenant_id"),
	}
}
