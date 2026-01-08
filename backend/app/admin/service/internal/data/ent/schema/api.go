package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/tx7do/go-crud/entgo/mixin"
)

// Api holds the schema definition for the Api entity.
type Api struct {
	ent.Schema
}

func (Api) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_apis",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("API资源表"),
	}
}

// Fields of the Api.
func (Api) Fields() []ent.Field {
	return []ent.Field{
		field.String("description").
			Comment("描述").
			Optional().
			Nillable(),

		field.String("module").
			Comment("所属业务模块").
			Optional().
			Nillable(),

		field.String("module_description").
			Comment("业务模块描述").
			Optional().
			Nillable(),

		field.String("operation").
			Comment("接口操作名").
			Optional().
			Nillable(),

		field.String("path").
			Comment("接口路径").
			Optional().
			Nillable(),

		field.String("method").
			Comment("请求方法").
			Optional().
			Nillable(),

		field.Enum("scope").
			Comment("作用域").
			NamedValues(
				"Admin", "ADMIN",
				"App", "APP",
			).
			Default("ADMIN").
			Optional().
			Nillable(),
	}
}

// Mixin of the Api.
func (Api) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.SwitchStatus{},
	}
}

// Indexes of the Api.
func (Api) Indexes() []ent.Index {
	return []ent.Index{
		// 在模块范围内保持同一路径/方法/作用域唯一，防止重复注册
		index.Fields("module", "path", "method", "scope").
			Unique().
			StorageKey("idx_sys_api_res_module_path_method_scope"),

		// 按模块快速检索
		index.Fields("module").
			StorageKey("idx_sys_api_res_module"),

		// 按作用域（Admin/App）过滤
		index.Fields("scope").
			StorageKey("idx_sys_api_res_scope"),

		// 按路径 + 方法 快速定位接口
		index.Fields("path", "method").
			StorageKey("idx_sys_api_res_path_method"),

		// 操作者 + 创建时间，用于审计回溯（时间列放末尾以利于范围扫描）
		index.Fields("created_by", "created_at").
			StorageKey("idx_sys_api_res_created_by_created_at"),

		// 创建时间索引用于列表分页与时间区间查询
		index.Fields("created_at").
			StorageKey("idx_sys_api_res_created_at"),
	}
}
