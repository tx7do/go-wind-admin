package schema

import (
	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/tx7do/go-crud/entgo/mixin"
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

// Indexes 索引定义
func (ApiAuditLog) Indexes() []ent.Index {
	return []ent.Index{
		// 按租户与时间范围查询
		index.Fields("tenant_id"),
		index.Fields("created_at"),
		index.Fields("tenant_id", "created_at"),

		// 常用按用户查询
		index.Fields("user_id"),
		index.Fields("username"),
		index.Fields("tenant_id", "user_id", "created_at"), // 多租户下按用户的时间范围查询

		// 按 IP 查询与追踪
		index.Fields("ip_address"),
		index.Fields("ip_address", "created_at"),

		// 请求追踪与去重
		index.Fields("request_id"),
		index.Fields("log_hash"),

		// API 维度的筛选
		index.Fields("api_module"),
		index.Fields("api_operation"),
		index.Fields("api_module", "api_operation"),

		// 路径与方法
		index.Fields("path"),
		index.Fields("http_method"),
		index.Fields("path", "http_method"),

		// 状态相关
		index.Fields("status_code"),
		index.Fields("success"),
	}
}
