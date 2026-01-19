package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"

	"go-wind-admin/app/admin/service/internal/data/ent/privacy"
	"go-wind-admin/app/admin/service/internal/data/ent/rule"
)

type PolicyEvaluationLog struct {
	ent.Schema
}

func (PolicyEvaluationLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_policy_evaluation_logs",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("策略评估日志表"),
	}
}

func (PolicyEvaluationLog) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("user_id").
			Comment("用户ID").
			Nillable(),

		field.Uint32("membership_id").
			Comment("成员身份ID").
			Nillable(),

		field.Uint32("permission_id").
			Comment("权限点ID").
			Nillable(),

		field.Uint32("policy_id").
			Comment("策略ID（可能无策略）").
			Optional().
			Nillable(),

		field.String("request_path").
			Comment("请求API路径").
			Optional().
			Nillable(),

		field.String("request_method").
			Comment("请求HTTP方法").
			Optional().
			Nillable(),

		field.Bool("result").
			Comment("是否通过").
			Default(false).
			Nillable(),

		field.String("effect_details").
			Comment("评估详情/拒绝原因").
			Optional().
			Nillable(),

		field.String("scope_sql").
			Comment("生成的SQL条件").
			Optional().
			Nillable(),

		field.String("ip_address").
			Comment("操作者IP地址").
			Optional().
			Nillable(),

		field.String("trace_id").
			Comment("全局链路追踪ID").
			Optional().
			Nillable(),

		field.String("evaluation_context").
			Comment("决策上下文快照").
			Optional().
			Nillable(),

		field.String("log_hash").
			Comment("日志内容哈希（SHA256，十六进制字符串）").
			Optional().
			Nillable(),

		field.Bytes("signature").
			Comment("日志数字签名").
			Optional().
			Nillable(),
	}
}

func (PolicyEvaluationLog) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.CreatedAt{},
		mixin.TenantID{},
	}
}

// Policy for all schemas that embed PolicyEvaluationLog.
func (PolicyEvaluationLog) Policy() ent.Policy {
	return privacy.Policy{
		Query: rule.TenantQueryPolicy(),
	}
}

func (PolicyEvaluationLog) Indexes() []ent.Index {
	return []ent.Index{
		// 多租户 + 时间（分页/范围检索）
		index.Fields("tenant_id", "created_at").
			StorageKey("idx_policy_eval_tenant_created_at"),

		// 多租户 + 用户 + 权限 + 时间（定位某用户某权限评估记录）
		index.Fields("tenant_id", "user_id", "permission_id", "created_at").
			StorageKey("idx_policy_eval_tenant_user_permission_created_at"),

		// 多租户 + 策略 + 时间（按策略统计/检索）
		index.Fields("tenant_id", "policy_id", "created_at").
			StorageKey("idx_policy_eval_tenant_policy_created_at"),

		// 多租户 + 成员身份 + 时间（按成员身份查询）
		index.Fields("tenant_id", "membership_id", "created_at").
			StorageKey("idx_policy_eval_tenant_membership_created_at"),

		// 多租户 + 权限 + 结果 + 时间（统计通过/未通过）
		index.Fields("tenant_id", "permission_id", "result", "created_at").
			StorageKey("idx_policy_eval_tenant_permission_result_created_at"),

		// 请求路径与方法（注意：如 request_path 很长，考虑在迁移中用前缀索引或全文索引）
		index.Fields("request_path").
			StorageKey("idx_policy_eval_request_path"),
		index.Fields("request_method").
			StorageKey("idx_policy_eval_request_method"),

		// 追踪与回溯字段
		index.Fields("trace_id").
			StorageKey("idx_policy_eval_trace_id"),
		index.Fields("ip_address", "created_at").
			StorageKey("idx_policy_eval_ip_address_created_at"),

		// 日志哈希与签名检索（防篡改/去重）
		index.Fields("log_hash").
			StorageKey("idx_policy_eval_log_hash"),
	}
}
