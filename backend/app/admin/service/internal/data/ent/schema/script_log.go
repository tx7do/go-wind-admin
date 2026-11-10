package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/tx7do/go-crud/entgo/mixin"
)

// ScriptLog holds the schema definition for the ScriptLog entity.
// 脚本执行日志：每次脚本/钩子/任务处理器执行一条记录（成败、耗时、错误）。
// 高频写入，按时间滚动清理由运维侧处理（可复用审计归档思路）。
type ScriptLog struct {
	ent.Schema
}

func (ScriptLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_script_logs",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("脚本执行日志"),
	}
}

// Fields of the ScriptLog.
func (ScriptLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("script_id").
			Comment("脚本ID（试运行草稿为 0）").
			Default(0).
			Optional().
			Nillable(),

		field.String("script_name").
			Comment("脚本名称").
			Optional().
			Nillable(),

		field.String("language").
			Comment("脚本语言（LUA/JAVASCRIPT）").
			Optional().
			Nillable(),

		field.String("trigger_type").
			Comment("触发方式（hook/task/test_run/manual）").
			Optional().
			Nillable(),

		field.String("hook_point").
			Comment("钩子点或任务类型").
			Optional().
			Nillable(),

		field.Uint32("version").
			Comment("执行时的脚本版本（草稿为 0）").
			Default(0).
			Optional().
			Nillable(),

		field.Bool("success").
			Comment("是否执行成功").
			Default(false).
			Optional().
			Nillable(),

		field.Int64("duration_ms").
			Comment("执行耗时（毫秒）").
			Default(0).
			Optional().
			Nillable(),

		field.Text("error").
			Comment("失败原因（成功为空）").
			Optional().
			Nillable(),
	}
}

// Mixin of the ScriptLog.
func (ScriptLog) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TimeAt{},
	}
}

// Indexes of the ScriptLog.
func (ScriptLog) Indexes() []ent.Index {
	return []ent.Index{
		// 按脚本检索 + 时间倒序浏览
		index.Fields("script_id", "created_at").
			StorageKey("idx_sys_script_logs_script_created"),

		// 失败排查
		index.Fields("success", "created_at").
			StorageKey("idx_sys_script_logs_success_created"),

		index.Fields("created_at").
			StorageKey("idx_sys_script_logs_created_at"),
	}
}
