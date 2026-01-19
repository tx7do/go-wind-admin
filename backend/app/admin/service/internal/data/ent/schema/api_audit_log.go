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

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
)

// ApiAuditLog holds the schema definition for the ApiAuditLog entity.
type ApiAuditLog struct {
	ent.Schema
}

func (ApiAuditLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_api_audit_logs",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("API审计日志表"),
	}
}

// Fields of the ApiAuditLog.
func (ApiAuditLog) Fields() []ent.Field {
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

		field.String("referer").
			Comment("请求来源URL").
			Optional().
			Nillable(),

		field.String("app_version").
			Comment("客户端版本号").
			Optional().
			Nillable(),

		field.String("http_method").
			Comment("HTTP请求方法").
			Optional().
			Nillable(),

		field.String("path").
			Comment("请求路径").
			Optional().
			Nillable(),

		field.String("request_uri").
			Comment("完整请求URI").
			Optional().
			Nillable(),

		field.String("api_module").
			Comment("API所属业务模块").
			Optional().
			Nillable(),

		field.String("api_operation").
			Comment("API业务操作").
			Optional().
			Nillable(),

		field.String("api_description").
			Comment("API功能描述").
			Optional().
			Nillable(),

		field.String("request_id").
			Comment("请求ID").
			Optional().
			Nillable(),

		field.String("trace_id").
			Comment("全局链路追踪ID").
			Optional().
			Nillable(),

		field.String("span_id").
			Comment("当前跨度ID").
			Optional().
			Nillable(),

		field.Uint32("latency_ms").
			Comment("操作耗时").
			Optional().
			Nillable(),

		field.Bool("success").
			Comment("操作结果").
			Optional().
			Nillable(),

		field.Uint32("status_code").
			Comment("HTTP状态码").
			Optional().
			Nillable(),

		field.String("reason").
			Comment("操作失败原因").
			Optional().
			Nillable(),

		field.String("request_header").
			Comment("请求头").
			Optional().
			Nillable(),

		field.String("request_body").
			Comment("请求体").
			Optional().
			Nillable(),

		field.String("response").
			Comment("响应信息").
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

// Mixin of the ApiAuditLog.
func (ApiAuditLog) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.CreatedAt{},
		mixin.TenantID{},
	}
}

// Policy for all schemas that embed ApiAuditLog.
func (ApiAuditLog) Policy() ent.Policy {
	return privacy.Policy{
		Query: rule.TenantQueryPolicy(),
	}
}

// Indexes 索引定义
func (ApiAuditLog) Indexes() []ent.Index {
	return []ent.Index{
		// 去重：租户维度下 request_id 唯一，防止重复记录
		index.Fields("tenant_id", "request_id").
			Unique().
			StorageKey("uidx_sys_api_audit_logs_tenant_request_id"),

		// 去重：租户维度下 log_hash 唯一（内容指纹去重）
		index.Fields("tenant_id", "log_hash").
			Unique().
			StorageKey("uidx_sys_api_audit_logs_tenant_log_hash"),

		// 常用：按租户 + 时间范围 查询（范围扫描）
		index.Fields("tenant_id", "created_at").
			StorageKey("idx_sys_api_audit_logs_tenant_created_at"),

		// 全局按时间查询（跨租户聚合/清理）
		index.Fields("created_at").
			StorageKey("idx_sys_api_audit_logs_created_at"),

		// 常用按用户查询：租户 + 用户 + 时间范围
		index.Fields("tenant_id", "user_id", "created_at").
			StorageKey("idx_sys_api_audit_logs_tenant_user_created_at"),

		// 按用户名检索（兼容无 user_id 场景）
		index.Fields("tenant_id", "username", "created_at").
			StorageKey("idx_sys_api_audit_logs_tenant_username_created_at"),

		// IP 相关查询与溯源：租户 + IP + 时间
		index.Fields("tenant_id", "ip_address", "created_at").
			StorageKey("idx_sys_api_audit_logs_tenant_ip_created_at"),

		// 链路追踪与请求追溯：租户 + trace_id / request_id
		index.Fields("tenant_id", "trace_id").
			StorageKey("idx_sys_api_audit_logs_tenant_trace_id"),

		// API 维度筛选：租户 + 模块 + 操作 + 时间
		index.Fields("tenant_id", "api_module", "api_operation", "created_at").
			StorageKey("idx_sys_api_audit_logs_tenant_api_created_at"),

		// 路径与方法检索：租户 + 路径 + 方法 + 时间
		index.Fields("tenant_id", "path", "http_method", "created_at").
			StorageKey("idx_sys_api_audit_logs_tenant_path_method_created_at"),

		// 状态类过滤：租户 + 状态码 + 成功标志 + 时间（便于统计/报警）
		index.Fields("tenant_id", "status_code", "success", "created_at").
			StorageKey("idx_sys_api_audit_logs_tenant_status_created_at"),
	}
}
