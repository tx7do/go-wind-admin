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

// LoginAuditLog holds the schema definition for the LoginAuditLog entity.
type LoginAuditLog struct {
	ent.Schema
}

func (LoginAuditLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_login_audit_logs",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("登录审计日志表"),
	}
}

// Fields of the LoginAuditLog.
func (LoginAuditLog) Fields() []ent.Field {
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

		field.String("session_id").
			Comment("会话ID").
			Optional().
			Nillable(),

		field.JSON("device_info", &auditV1.DeviceInfo{}).
			Comment("设备信息").
			Optional(),

		field.String("request_id").
			Comment("全局请求ID").
			Optional().
			Nillable(),

		field.Enum("action_type").
			Comment("事件动作类型").
			NamedValues(
				"Login", "LOGIN",
				"Logout", "LOGOUT",
				"SessionExpired", "SESSION_EXPIRED",
			).
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("操作结果状态").
			NamedValues(
				"Success", "SUCCESS",
				"Failed", "FAILED",
				"Partial", "PARTIAL",
			).
			Optional().
			Nillable(),

		field.String("failure_reason").
			Comment("失败原因").
			Optional().
			Nillable(),

		field.String("mfa_status").
			Comment("MFA状态").
			Optional().
			Nillable(),

		field.Uint32("risk_score").
			Comment("风险评分（0-100，分值越高风险越大）").
			Optional().
			Nillable(),

		field.Enum("risk_level").
			Comment("风险等级（高风险需实时告警）").
			NamedValues(
				"Low", "LOW",
				"Medium", "MEDIUM",
				"High", "HIGH",
			).
			Optional().
			Nillable(),

		field.Strings("risk_factors").
			Comment("风险因素（ISO 27001标准，如：异地登录/新设备/密码尝试次数过多）").
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

// Mixin of the LoginAuditLog.
func (LoginAuditLog) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.CreatedAt{},
		mixin.TenantID{},
	}
}

// Indexes 索引定义
func (LoginAuditLog) Indexes() []ent.Index {
	return []ent.Index{
		// 常用于按用户查询
		index.Fields("user_id"),
		index.Fields("username"),

		// 按 IP 查询与追踪
		index.Fields("ip_address"),
		index.Fields("ip_address", "created_at"),

		// 会话与请求追踪
		index.Fields("session_id"),
		index.Fields("request_id"),

		// 事件类型与状态常用于过滤
		index.Fields("action_type"),
		index.Fields("status"),

		// 多租户与时间范围查询
		index.Fields("tenant_id"),
		index.Fields("created_at"),
		index.Fields("tenant_id", "created_at"),
	}
}
