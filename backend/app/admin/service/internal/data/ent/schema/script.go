package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/tx7do/go-crud/entgo/mixin"
)

// Script holds the schema definition for the Script entity.
// 平台脚本表：脚本引擎（pkg/scripting）的持久化承载，唯一事实源是数据库。
type Script struct {
	ent.Schema
}

func (Script) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_scripts",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("平台脚本表（脚本引擎插件承载）"),
	}
}

// Fields of the Script.
func (Script) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("脚本唯一名称").
			Optional().
			Nillable(),

		field.Enum("language").
			Comment("脚本语言（LUA=Lua JAVASCRIPT=JavaScript）").
			NamedValues(
				"LUA", "LUA",
				"JAVASCRIPT", "JAVASCRIPT",
			).
			Default("LUA").
			Optional().
			Nillable(),

		field.String("hook_point").
			Comment("挂载的钩子点名称，空表示未挂载").
			Optional().
			Nillable(),

		field.Text("source").
			Comment("脚本源码").
			Optional().
			Nillable(),

		field.Int32("priority").
			Comment("执行优先级，越小越先执行").
			Default(0).
			Optional().
			Nillable(),

		field.String("description").
			Comment("脚本用途说明").
			Optional().
			Nillable(),

		field.Bool("critical").
			Comment("关键脚本：执行失败时中断钩子链").
			Default(false).
			Optional().
			Nillable(),

		field.Uint32("version").
			Comment("版本号，每次更新自增（热更新指纹）").
			Default(1).
			Optional().
			Nillable(),
	}
}

// Mixin of the Script.
func (Script) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TimeAt{},
		mixin.OperatorID64{},
		mixin.IsEnabled{},
	}
}

// Indexes of the Script.
func (Script) Indexes() []ent.Index {
	return []ent.Index{
		// 脚本名全局唯一（DBSource 按名加载的前提）
		index.Fields("name").
			Unique().
			StorageKey("uk_sys_scripts_name"),

		// 按钩子点检索 + 启动时按 hook_point+is_enabled 加载已启用脚本
		index.Fields("hook_point", "is_enabled").
			StorageKey("idx_sys_scripts_hook_point_enabled"),

		// 创建时间索引用于列表分页
		index.Fields("created_at").
			StorageKey("idx_sys_scripts_created_at"),
	}
}
