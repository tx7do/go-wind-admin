package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"github.com/tx7do/go-crud/entgo/mixin"
)

// AdminLoginLog holds the schema definition for the AdminLoginLog entity.
type AdminLoginLog struct {
	ent.Schema
}

func (AdminLoginLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_admin_login_logs",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("后台登录日志表"),
	}
}

// Fields of the AdminLoginLog.
func (AdminLoginLog) Fields() []ent.Field {
	return []ent.Field{
		field.String("login_ip").
			Comment("登录IP地址").
			Optional().
			Nillable(),

		field.String("login_mac").
			Comment("登录MAC地址").
			Optional().
			Nillable(),

		field.Time("login_time").
			Comment("登录时间").
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

		field.Uint32("user_id").
			Comment("操作者用户ID").
			Optional().
			Nillable(),

		field.String("username").
			Comment("操作者账号名").
			Optional().
			Nillable(),

		field.Int32("status_code").
			Comment("状态码").
			Optional().
			Nillable(),

		field.Bool("success").
			Comment("操作成功").
			Optional().
			Nillable(),

		field.String("reason").
			Comment("登录失败原因").
			Optional().
			Nillable(),

		field.String("location").
			Comment("登录地理位置").
			Optional().
			Nillable(),
	}
}

// Mixin of the AdminLoginLog.
func (AdminLoginLog) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.CreatedAt{},
		mixin.TenantID{},
	}
}
