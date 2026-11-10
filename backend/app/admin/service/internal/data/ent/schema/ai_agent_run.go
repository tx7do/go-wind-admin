package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/tx7do/go-crud/entgo/mixin"
)

// AiAgentRun Agent 应用的运行记录：同步执行一次图 = 一行。
// trace 存逐节点执行轨迹（JSON），用量与 sys_ai_usage_logs 同源累计。
type AiAgentRun struct{ ent.Schema }

func (AiAgentRun) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "sys_ai_agent_runs", Charset: "utf8mb4", Collation: "utf8mb4_bin"},
		entsql.WithComments(true),
		schema.Comment("AI Agent 运行记录"),
	}
}

func (AiAgentRun) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("app_id").
			Comment("应用ID").
			Optional().
			Nillable(),
		field.Enum("status").
			Comment("运行状态").
			NamedValues(
				"Running", "RUNNING",
				"Succeeded", "SUCCEEDED",
				"Failed", "FAILED",
			).
			Default("RUNNING"),
		field.Text("input").
			Comment("运行入参 JSON（start 节点声明的变量）").
			Optional().
			Nillable(),
		field.Text("output").
			Comment("产出（到达的 end 节点渲染值）").
			Optional().
			Nillable(),
		field.Text("error").
			Comment("失败原因").
			Optional().
			Nillable(),
		field.Text("trace").
			Comment("逐节点执行轨迹 JSON"),
		field.Uint32("prompt_tokens").
			Comment("输入 token 合计").
			Default(0),
		field.Uint32("completion_tokens").
			Comment("输出 token 合计").
			Default(0),
		field.Uint32("total_tokens").
			Comment("总 token 合计").
			Default(0),
		field.Uint32("duration_ms").
			Comment("总耗时（毫秒）").
			Default(0),
	}
}

func (AiAgentRun) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TimeAt{},
		mixin.OperatorID64{},
		mixin.TenantID[int64]{},
	}
}

func (AiAgentRun) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "app_id").
			StorageKey("idx_sys_ai_agent_runs_tenant_app"),
	}
}
