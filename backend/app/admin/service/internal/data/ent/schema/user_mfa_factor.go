package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// UserMfaFactor holds the schema definition for the UserMfaFactor entity.
// 存储用户绑定的 MFA 因子。本轮仅 TOTP：secret 字段存 AES-GCM 加密后的 TOTP secret，
// 校验时解密后用 totp 库验证。其余方法（SMS/EMAIL/WEBAUTHN）预留枚举值，本轮不落库。
type UserMfaFactor struct {
	ent.Schema
}

func (UserMfaFactor) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_user_mfa_factors",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("用户 MFA 因子表"),
	}
}

// Fields of the UserMfaFactor.
func (UserMfaFactor) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").
			Comment("关联主表的用户ID").
			Nillable().
			Optional(),

		// MFA 方法：本轮仅 TOTP 会落库；其余枚举值预留，服务层对非 TOTP 直接返回未实现。
		field.Enum("method").
			Comment("MFA 方法").
			NamedValues(
				"Totp", "TOTP",
				"Sms", "SMS",
				"Email", "EMAIL",
				"Webauthn", "WEBAUTHN",
			).
			Default("TOTP").
			Nillable().
			Optional(),

		// TOTP secret 的 AES-GCM 密文（base64）。绝不存明文；校验时解密。
		// 字段名沿用 secret_hash 是历史命名，实际内容为对称加密密文而非单向哈希，
		// 因为 TOTP 校验需要还原明文 secret。
		field.String("secret_hash").
			Comment("MFA secret 密文（AES-GCM 加密，base64 编码；TOTP 校验需还原明文）").
			MaxLen(512).
			NotEmpty().
			Nillable().
			Optional(),

		field.String("display_name").
			Comment("设备/因子展示名（用户自定义）").
			MaxLen(128).
			Nillable().
			Optional(),

		field.Enum("status").
			Comment("因子状态").
			NamedValues(
				"Disabled", "DISABLED",
				"Enabled", "ENABLED",
			).
			Default("ENABLED").
			Nillable().
			Optional(),

		field.Time("last_used_at").
			Comment("最近一次用于验证的时间").
			Nillable().
			Optional(),
	}
}

// Mixin of the UserMfaFactor.
func (UserMfaFactor) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.TimeAt{},
		mixin.AutoIncrementId64{},
		mixin.TenantID[int64]{},
	}
}

// Indexes of the UserMfaFactor.
func (UserMfaFactor) Indexes() []ent.Index {
	return []ent.Index{
		// 按租户 + user_id 列出某用户的全部 MFA 因子
		index.Fields("tenant_id", "user_id").
			StorageKey("idx_sys_user_mfa_tenant_user_id"),

		// 在租户范围内保证 (user_id, method) 唯一：一用户同一方法至多一个因子
		// 注意：若字段可空，Postgres 需 partial unique index
		index.Fields("tenant_id", "user_id", "method").
			Unique().
			StorageKey("idx_sys_user_mfa_tenant_uid_method"),
	}
}

// Policy 追加租户变更防护：go-crud TenantPrivacy 只覆盖 Query/Create，
// 本规则为 Update/UpdateOne/Delete/DeleteOne 注入 tenant_id 过滤，
// 防止知道 ID 的跨租户篡改与删除（平台/系统上下文放行）。
func (UserMfaFactor) Policy() ent.Policy {
	return &TenantMutationGuardPolicy{}
}
