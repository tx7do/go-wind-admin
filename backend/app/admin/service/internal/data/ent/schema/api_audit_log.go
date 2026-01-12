package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
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
		field.String("request_id").
			Comment("请求ID").
			Optional().
			Nillable(),

		field.String("method").
			Comment("请求方法").
			Optional().
			Nillable(),

		field.String("operation").
			Comment("操作方法").
			Optional().
			Nillable(),

		field.String("path").
			Comment("请求路径").
			Optional().
			Nillable(),

		field.String("referer").
			Comment("请求源").
			Optional().
			Nillable(),

		field.String("request_uri").
			Comment("请求URI").
			Optional().
			Nillable(),

		field.String("request_body").
			Comment("请求体").
			Optional().
			Nillable(),

		field.String("request_header").
			Comment("请求头").
			Optional().
			Nillable(),

		field.String("response").
			Comment("响应信息").
			Optional().
			Nillable(),

		field.Float("cost_time").
			Comment("操作耗时").
			Optional().
			Nillable(),

		field.Uint32("user_id").
			Comment("操作者用户ID").
			Optional().
			Nillable(),

		field.String("username").
			Comment("操作者账号名").
			Optional().
			Nillable(),

		field.String("client_ip").
			Comment("操作者IP").
			Optional().
			Nillable(),

		field.Int32("status_code").
			Comment("状态码").
			Optional().
			Nillable(),

		field.String("reason").
			Comment("操作失败原因").
			Optional().
			Nillable(),

		field.Bool("success").
			Comment("操作成功").
			Optional().
			Nillable(),

		field.String("location").
			Comment("操作地理位置").
			Optional().
			Nillable(),

		field.String("user_agent").
			Comment("浏览器的用户代理信息").
			Optional().
			Nillable(),

		field.String("browser_name").
			Comment("浏览器名称").
			Optional().
			Nillable(),

		field.String("browser_version").
			Comment("浏览器版本").
			Optional().
			Nillable(),

		field.String("client_id").
			Comment("客户端ID").
			Optional().
			Nillable(),

		field.String("client_name").
			Comment("客户端名称").
			Optional().
			Nillable(),

		field.String("os_name").
			Comment("操作系统名称").
			Optional().
			Nillable(),

		field.String("os_version").
			Comment("操作系统版本").
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
