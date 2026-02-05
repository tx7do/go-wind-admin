package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
)

// DataAccessAuditLog holds the schema definition for the DataAccessAuditLog entity.
type DataAccessAuditLog struct {
	ent.Schema
}

func (DataAccessAuditLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_data_access_audit_logs",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("数据访问审计日志表"),
	}
}

// Fields of the DataAccessAuditLog.
func (DataAccessAuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("user_id").
			Comment("操作者用户ID").
			Optional().
			Nillable(),

		field.String("username").
			Comment("操作者账号名").
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

		field.String("request_id").
			Comment("全局请求ID").
			Optional().
			Nillable(),

		field.String("trace_id").
			Comment("全局链路追踪ID").
			Optional().
			Nillable(),

		field.String("data_source").
			Comment("数据源类型").
			Optional().
			Nillable(),

		field.String("table_name").
			Comment("数据表名").
			Optional().
			Nillable(),

		field.String("data_id").
			Comment("数据主键ID").
			Optional().
			Nillable(),

		field.Enum("access_type").
			Comment("数据访问类型").
			NamedValues(
				"Select", "SELECT",
				"Insert", "INSERT",
				"Update", "UPDATE",
				"Delete", "DELETE",
				"View", "VIEW",
				"BulkRead", "BULK_READ",
				"Export", "EXPORT",
				"import", "IMPORT",
				"DDLCreate", "DDL_CREATE",
				"DDLAlter", "DDL_ALTER",
				"DDLDrop", "DDL_DROP",
				"MetadataRead", "METADATA_READ",
				"Scan", "SCAN",
				"AdminOperation", "ADMIN_OPERATION",
				"Other", "OTHER",
			).
			Optional().
			Nillable(),

		field.String("sql_digest").
			Comment("执行的SQL语句摘要").
			Optional().
			Nillable(),

		field.String("sql_text").
			Comment("执行的SQL语句").
			Optional().
			Nillable(),

		field.Uint32("affected_rows").
			Comment("影响行数").
			Optional().
			Nillable(),

		field.Uint32("latency_ms").
			Comment("延迟时间（毫秒）").
			Optional().
			Nillable(),

		field.Bool("success").
			Comment("操作结果").
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

		field.Bool("data_masked").
			Comment("是否已脱敏").
			Optional().
			Nillable(),

		field.String("masking_rules").
			Comment("脱敏规则").
			Optional().
			Nillable(),

		field.String("business_purpose").
			Comment("业务处理目的").
			Optional().
			Nillable(),

		field.String("data_category").
			Comment("数据分类标签").
			Optional().
			Nillable(),

		field.String("db_user").
			Comment("数据库用户").
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

// Mixin of the DataAccessAuditLog.
func (DataAccessAuditLog) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.CreatedAt{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes 索引定义
func (DataAccessAuditLog) Indexes() []ent.Index {
	return []ent.Index{
		// 单列索引：快速定位用户/追踪/会话
		index.Fields("user_id"),
		index.Fields("username"),
		index.Fields("request_id"),
		index.Fields("trace_id"),

		// IP 以及按 IP+时间 的查询
		index.Fields("ip_address"),
		index.Fields("ip_address", "created_at"),

		// 多租户 + 时间范围查询（常用）
		index.Fields("tenant_id"),
		index.Fields("created_at"),
		index.Fields("tenant_id", "created_at"),
		// 按租户+用户+时间（常用于审计/报表）
		index.Fields("tenant_id", "user_id", "created_at"),

		// 资源定位复合索引：数据源、表/集合、具体对象ID
		index.Fields("data_source", "table_name", "data_id"),

		// 访问类型与结果联合过滤（例如统计某类访问失败/成功）
		index.Fields("access_type"),
		index.Fields("access_type", "success", "created_at"),

		// SQL 相关：按 SQL 摘要检索重复/相似请求
		index.Fields("sql_digest"),

		// 用于快速过滤是否已脱敏
		index.Fields("data_masked"),
	}
}
