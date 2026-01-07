package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/tx7do/go-crud/entgo/mixin"
)

// Language holds the schema definition for the Language entity.
type Language struct {
	ent.Schema
}

func (Language) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_languages",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("语言表"),
	}
}

// Fields of the Language.
func (Language) Fields() []ent.Field {
	return []ent.Field{
		field.String("language_code").
			Comment("标准语言代码").
			NotEmpty().
			Optional().
			Nillable(),

		field.String("language_name").
			Comment("语言名称").
			NotEmpty().
			Optional().
			Nillable(),

		field.String("native_name").
			Comment("本地语言名称").
			NotEmpty().
			Optional().
			Nillable(),

		field.Bool("is_default").
			Comment("是否为默认语言").
			Optional().
			Nillable().
			Default(false),
	}
}

// Mixin of the Language.
func (Language) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.SortOrder{},
		mixin.IsEnabled{},
	}
}

// Indexes of the Language.
func (Language) Indexes() []ent.Index {
	return []ent.Index{
		// 全局唯一：language_code 唯一
		index.Fields("language_code").
			Unique().
			StorageKey("uix_sys_languages_language_code"),

		// 单列索引：按 code/name/native_name 快速查询/模糊搜索
		index.Fields("language_code").
			StorageKey("idx_sys_languages_language_code"),
		index.Fields("language_name").
			StorageKey("idx_sys_languages_language_name"),
		index.Fields("native_name").
			StorageKey("idx_sys_languages_native_name"),

		// 常用过滤字段索引
		index.Fields("is_default").
			StorageKey("idx_sys_languages_is_default"),
		index.Fields("is_enabled").
			StorageKey("idx_sys_languages_is_enabled"),

		// 排序/范围查询索引
		index.Fields("sort_order").
			StorageKey("idx_sys_languages_sort_order"),
		index.Fields("created_at").
			StorageKey("idx_sys_languages_created_at"),
	}
}
