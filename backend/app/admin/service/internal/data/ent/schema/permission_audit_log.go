package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

type PermissionAuditLog struct {
	ent.Schema
}

func (PermissionAuditLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_permission_audit_logs",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("权限变更审计日志表"),
	}
}

func (PermissionAuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("operator_id").
			Comment("操作者 用户ID").
			Optional().
			Nillable(),

		field.String("target_type").
			Comment("目标类型").
			Optional().
			Nillable(),

		field.String("target_id").
			Comment("目标ID").
			Optional().
			Nillable(),

		field.Enum("action").
			Comment("动作").
			NamedValues(
				"Grant", "GRANT",
				"Revoke", "REVOKE",
				"Update", "UPDATE",
				"Reset", "RESET",
				"Create", "CREATE",
				"Delete", "DELETE",
				"Assign", "ASSIGN",
				"Unassign", "UNASSIGN",
				"BulkGrant", "BULK_GRANT",
				"BulkRevoke", "BULK_REVOKE",
				"Expire", "EXPIRE",
				"Suspend", "SUSPEND",
				"Resume", "RESUME",
				"Rollback", "ROLLBACK",
				"Other", "OTHER",
			).
			Optional().
			Nillable(),

		field.String("old_value").
			Comment("旧值（JSON）").
			SchemaType(map[string]string{
				dialect.MySQL:    "json",
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),

		field.String("new_value").
			Comment("新值（JSON）").
			SchemaType(map[string]string{
				dialect.MySQL:    "json",
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),

		field.String("ip_address").
			Comment("操作者IP地址").
			Nillable(),

		field.String("request_id").
			Comment("关联全局请求ID").
			Nillable(),

		field.String("reason").
			Comment("变更原因").
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

func (PermissionAuditLog) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.CreatedAt{},
		mixin.TenantID[uint32]{},
	}
}

func (PermissionAuditLog) Indexes() []ent.Index {
	return []ent.Index{
		// 按租户 + 时间，用于时间区间查询（分页/历史检索）
		index.Fields("tenant_id", "created_at").
			StorageKey("idx_permission_audit_tenant_created_at"),

		// 按租户 + 操作者 + 时间，用于定位某操作者的变更记录
		index.Fields("tenant_id", "operator_id", "created_at").
			StorageKey("idx_permission_audit_tenant_operator_created_at"),

		// 按租户 + 目标类型 + 目标ID + 时间，用于按目标检索（例如某资源的变更历史）
		index.Fields("tenant_id", "target_type", "target_id", "created_at").
			StorageKey("idx_permission_audit_tenant_target_created_at"),

		// 单独按目标（跨租户场景或没有租户列时使用）
		index.Fields("target_type", "target_id").
			StorageKey("idx_permission_audit_target"),

		// 按租户 + 动作 + 时间，用于动作维度的过滤/统计
		index.Fields("tenant_id", "action", "created_at").
			StorageKey("idx_permission_audit_tenant_action_created_at"),

		// 按租户 + IP，用于来源溯源；如 IP 长度问题，可在 DB 上改为前缀索引
		index.Fields("tenant_id", "ip_address").
			StorageKey("idx_permission_audit_tenant_ip"),
	}
}
