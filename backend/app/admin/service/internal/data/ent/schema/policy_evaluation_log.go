package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/tx7do/go-crud/entgo/mixin"
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

		field.Bool("result").
			Comment("是否通过").
			Default(false).
			Nillable(),

		field.String("scope_sql").
			Comment("生成的SQL条件").
			Optional().
			Nillable(),

		field.String("request_path").
			Comment("请求API路径").
			Optional().
			Nillable(),

		field.String("ip_address").
			Comment("操作者IP地址").
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

func (PolicyEvaluationLog) Indexes() []ent.Index {
	return []ent.Index{
		// 按租户 + 时间，用于时间区间查询（分页/历史检索）
		index.Fields("tenant_id", "created_at").
			StorageKey("idx_policy_eval_tenant_created_at"),

		// 按租户 + 用户 + 权限 + 时间，用于定位某用户某权限的评估记录
		index.Fields("tenant_id", "user_id", "permission_id", "created_at").
			StorageKey("idx_policy_eval_tenant_user_permission_created_at"),

		// 按租户 + 权限 + 结果 + 时间，用于统计 / 过滤通过/未通过的记录
		index.Fields("tenant_id", "permission_id", "result", "created_at").
			StorageKey("idx_policy_eval_tenant_permission_result_created_at"),

		// 请求路径检索（注意：如为长文本，考虑仅索引前缀或使用全文索引）
		index.Fields("request_path").
			StorageKey("idx_policy_eval_request_path"),

		// 按 IP 检索（追溯来源）
		index.Fields("ip_address").
			StorageKey("idx_policy_eval_ip_address"),
	}
}
