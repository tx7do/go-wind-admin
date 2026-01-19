package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"

	"go-wind-admin/app/admin/service/internal/data/ent/privacy"
	"go-wind-admin/app/admin/service/internal/data/ent/rule"
)

// UserCredential holds the schema definition for the UserCredential entity.
type UserCredential struct {
	ent.Schema
}

func (UserCredential) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_user_credentials",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("用户认证信息表"),
	}
}

// Fields of the UserCredential.
func (UserCredential) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("user_id").
			Comment("关联主表的用户ID").
			Nillable().
			Optional(),

		field.Enum("identity_type").
			Comment("认证方式类型").
			NamedValues(
				"Username", "USERNAME",
				"UserId", "USERID",
				"Email", "EMAIL",
				"Phone", "PHONE",

				"SocialOauth", "SOCIAL_OAUTH",
				"EnterpriseSso", "ENTERPRISE_SSO",
				"IdentityApiKey", "IDENTITY_API_KEY",
				"DeviceId", "DEVICE_ID",
				"Custom", "CUSTOM",
			).
			Default("USERNAME").
			Nillable().
			Optional(),

		field.String("identifier").
			Comment("身份唯一标识符").
			NotEmpty().
			Nillable().
			Optional(),

		field.Enum("credential_type").
			Comment("凭证类型").
			NamedValues(
				"PasswordHash", "PASSWORD_HASH",

				"ApiKey", "API_KEY",
				"ApiSecret", "API_SECRET",

				"AccessToken", "ACCESS_TOKEN",
				"RefreshToken", "REFRESH_TOKEN",
				"JWT", "JWT",

				"OauthToken", "OAUTH_TOKEN",
				"OauthAuthorizationCode", "OAUTH_AUTHORIZATION_CODE",
				"OauthClientCredentials", "OAUTH_CLIENT_CREDENTIALS",

				"OTP", "OTP",
				"TOTP", "TOTP",
				"SmsOtp", "SMS_OTP",
				"EmailOtp", "EMAIL_OTP",

				"HardwareToken", "HARDWARE_TOKEN",
				"SoftwareToken", "SOFTWARE_TOKEN",
				"SecurityQuestion", "SECURITY_QUESTION",

				"Biometric", "BIOMETRIC",
				"BiometricToken", "BIOMETRIC_TOKEN",

				"SsoToken", "SSO_TOKEN",
				"SamlAssertion", "SAML_ASSERTION",
				"OpenidConnectIdToken", "OPENID_CONNECT_ID_TOKEN",

				"SessionCookie", "SESSION_COOKIE",
				"TemporaryCredential", "TEMPORARY_CREDENTIAL",

				"Custom", "CUSTOM",
				"ReservedForFuture", "RESERVED_FOR_FUTURE",
			).
			Default("PASSWORD_HASH").
			Nillable().
			Optional(),

		field.String("credential").
			Comment("凭证").
			NotEmpty().
			Nillable().
			Optional(),

		field.Bool("is_primary").
			Comment("是否主认证方式").
			Default(false).
			Nillable().
			Optional(),

		field.Enum("status").
			Comment("凭证状态").
			NamedValues(
				"Disabled", "DISABLED",
				"Enabled", "ENABLED",
				"Expired", "EXPIRED",
				"Unverified", "UNVERIFIED",
				"Removed", "REMOVED",
				"Blocked", "BLOCKED",
				"Temporary", "TEMPORARY",
			).
			Default("ENABLED").
			Nillable().
			Optional(),

		field.String("extra_info").
			Comment("扩展信息").
			SchemaType(map[string]string{
				dialect.MySQL:    "json",
				dialect.Postgres: "jsonb",
			}).
			Nillable().
			Optional(),

		field.String("provider").
			Comment("第三方平台标识").
			Nillable().
			Optional(),

		field.String("provider_account_id").
			Comment("第三方平台的账号唯一ID").
			Nillable().
			Optional(),

		field.String("activate_token_hash").
			Comment("激活令牌哈希（不要存明文）").
			MaxLen(255).
			Nillable().
			Optional(),

		field.Time("activate_token_expires_at").
			Comment("激活令牌到期时间").
			Nillable().
			Optional(),

		field.Time("activate_token_used_at").
			Comment("激活令牌使用时间，单次使用时记录").
			Nillable().
			Optional(),

		field.String("reset_token_hash").
			Comment("重置密码令牌哈希（不要存明文）").
			MaxLen(255).
			Nillable().
			Optional(),

		field.Time("reset_token_expires_at").
			Comment("重置令牌到期时间").
			Nillable().
			Optional(),

		field.Time("reset_token_used_at").
			Comment("重置令牌使用时间").
			Nillable().
			Optional(),
	}
}

// Mixin of the UserCredential.
func (UserCredential) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.TimeAt{},
		mixin.AutoIncrementId{},
		mixin.TenantID{},
	}
}

// Policy for all schemas that embed UserCredential.
func (UserCredential) Policy() ent.Policy {
	return privacy.Policy{
		Query: rule.TenantQueryPolicy(),
	}
}

// Indexes of the UserCredential.
func (UserCredential) Indexes() []ent.Index {
	return []ent.Index{
		// 在租户范围内保证 (user_id, identity_type, identifier) 唯一
		// 注意：若 identifier 允许 NULL，Postgres 需要在迁移中使用 partial unique index 来严格约束
		index.Fields("tenant_id", "user_id", "identity_type", "identifier").
			Unique().
			StorageKey("idx_sys_user_cred_tenant_uid_identity_identifier"),

		// 按租户 + identifier 快速查找（例如登录时按 identifier 查询）
		index.Fields("tenant_id", "identifier").
			StorageKey("idx_sys_user_cred_tenant_identifier"),

		// 按租户 + user_id，用于查找某用户的所有凭证
		index.Fields("tenant_id", "user_id").
			StorageKey("idx_sys_user_cred_tenant_user_id"),

		// 在租户范围内保证第三方平台账号不重复
		// 若 provider/provider_account_id 可为 NULL，请在迁移脚本中为 Postgres 创建 partial unique index
		index.Fields("tenant_id", "provider", "provider_account_id").
			Unique().
			StorageKey("idx_sys_user_cred_tenant_provider_account"),

		// 按租户 + provider，用于按平台聚合或过滤
		index.Fields("tenant_id", "provider").
			StorageKey("idx_sys_user_cred_tenant_provider"),

		// 按租户 + 是否为主认证，用于快速定位主认证方式
		index.Fields("tenant_id", "is_primary").
			StorageKey("idx_sys_user_cred_tenant_is_primary"),

		// 按租户 + 状态 + 创建时间，用于状态过滤与时间范围查询
		index.Fields("tenant_id", "status", "created_at").
			StorageKey("idx_sys_user_cred_tenant_status_created_at"),

		// 按租户 + 激活/重置令牌到期时间，用于按令牌过期查询
		index.Fields("tenant_id", "activate_token_expires_at").
			StorageKey("idx_sys_user_cred_tenant_activate_expires_at"),
	}
}
