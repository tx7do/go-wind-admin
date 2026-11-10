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

// DictEntryI18n holds the schema definition for the DictEntryI18n entity.
type DictEntryI18n struct {
	ent.Schema
}

func (DictEntryI18n) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_dict_entry_i18n",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("字典项翻译表"),
	}
}

// Fields of the DictEntryI18n.
func (DictEntryI18n) Fields() []ent.Field {
	return []ent.Field{
		field.String("language_code").
			Comment("语言代码").
			NotEmpty().
			Immutable().
			Optional().
			Nillable(),

		field.String("entry_label").
			Comment("字典项的显示标签").
			NotEmpty().
			Optional().
			Nillable(),
	}
}

// Mixin of the DictEntryI18n.
func (DictEntryI18n) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TimeAt{},
		mixin.OperatorID64{},
		mixin.Description{},
		mixin.SortOrder{},
		mixin.TenantID[int64]{},
	}
}

// Edges of the DictEntryI18n.
func (DictEntryI18n) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("dict_entry", DictEntry.Type).
			Ref("i18ns").
			Unique(),
	}
}

// Indexes of the DictEntryI18n.
func (DictEntryI18n) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("language_code").
			StorageKey("idx_sys_dict_entry_i18n_language_code"),
	}
}

// Policy 追加租户变更防护：go-crud TenantPrivacy 只覆盖 Query/Create，
// 本规则为 Update/UpdateOne/Delete/DeleteOne 注入 tenant_id 过滤，
// 防止知道 ID 的跨租户篡改与删除（平台/系统上下文放行）。
func (DictEntryI18n) Policy() ent.Policy {
	return &TenantMutationGuardPolicy{}
}
