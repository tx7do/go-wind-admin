package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
)

// OperationAuditLog holds the schema definition for the OperationAuditLog entity.
type OperationAuditLog struct {
	ent.Schema
}

func (OperationAuditLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_operation_audit_logs",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("操作审计日志表"),
	}
}

// Fields of the OperationAuditLog.
func (OperationAuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("user_id").
			Comment("操作者用户ID").
			Optional().
			Nillable(),

		field.String("username").
			Comment("操作者账号名").
			Optional().
			Nillable(),

		field.String("resource_type").
			Comment("资源类型").
			Optional().
			Nillable(),

		field.String("resource_id").
			Comment("资源ID").
			Optional().
			Nillable(),

		field.Enum("action").
			Comment("动作").
			NamedValues(
				"Create", "CREATE",
				"Update", "UPDATE",
				"Delete", "DELETE",
				"Read", "READ",
				"Assign", "ASSIGN",
				"Unassign", "UNASSIGN",
				"Export", "EXPORT",
				"Import", "IMPORT",
				"Other", "OTHER",
			).
			Optional().
			Nillable(),

		field.String("before_data").
			Comment("操作前数据").
			SchemaType(map[string]string{
				dialect.MySQL:    "json",
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),

		field.String("after_data").
			Comment("操作后数据").
			SchemaType(map[string]string{
				dialect.MySQL:    "json",
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),

		field.Enum("sensitive_level").
			Comment("数据敏感级别").
			NamedValues(
				"Public", "PUBLIC",
				"Internal", "INTERNAL",
				"Confidential", "CONFIDENTIAL",
				"Secret", "SECRET",
			).
			Optional().
			Nillable(),

		field.String("request_id").
			Comment("全局请求ID").
			Optional().
			Nillable(),

		field.String("trace_id").
			Comment("全局链路追踪ID").
			Optional().
			Nillable(),

		field.Bool("success").
			Comment("操作结果").
			Optional().
			Nillable(),

		field.String("failure_reason").
			Comment("失败原因").
			Optional().
			Nillable(),

		field.String("ip_address").
			Comment("IP地址").
			Optional().
			Nillable(),

		field.JSON("geo_location", &auditV1.GeoLocation{}).
			Comment("地理位置(来自IP库)").
			Optional(),

		field.JSON("device_info", &auditV1.DeviceInfo{}).
			Comment("设备信息").
			Optional(),

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

// Mixin of the OperationAuditLog.
func (OperationAuditLog) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.CreatedAt{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes 索引定义
func (OperationAuditLog) Indexes() []ent.Index {
	return []ent.Index{
		// 用户与账号常用查询
		index.Fields("user_id"),
		index.Fields("username"),

		// 请求追踪与会话
		index.Fields("request_id"),
		index.Fields("trace_id"),

		// IP 及按 IP + 时间 的查询
		index.Fields("ip_address"),
		index.Fields("ip_address", "created_at"),

		// 多租户与时间范围查询（常用）
		index.Fields("tenant_id"),
		index.Fields("created_at"),
		index.Fields("tenant_id", "created_at"),
		// 租户 + 用户 + 时间（审计/报表常用）
		index.Fields("tenant_id", "user_id", "created_at"),

		// 资源定位复合索引：资源类型 + 资源ID
		index.Fields("resource_type", "resource_id"),

		// 动作与结果联合过滤（如统计某类操作失败/成功）
		index.Fields("action"),
		index.Fields("action", "success", "created_at"),

		// 敏感级别与是否成功的快速过滤
		index.Fields("sensitive_level"),
		index.Fields("success"),

		// 日志哈希用于快速去重/检索
		index.Fields("log_hash"),
	}
}
