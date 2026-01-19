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
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

func (User) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_users",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("用户表"),
	}
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("username").
			Comment("用户名").
			//Unique().
			NotEmpty().
			Immutable().
			Optional().
			Nillable(),

		field.String("nickname").
			Comment("昵称").
			Optional().
			Nillable(),

		field.String("realname").
			Comment("真实名字").
			Optional().
			Nillable(),

		field.String("email").
			Comment("电子邮箱").
			MaxLen(320).
			Optional().
			Nillable(),

		field.String("mobile").
			Comment("手机号码").
			Default("").
			MaxLen(255).
			Optional().
			Nillable(),

		field.String("telephone").
			Comment("座机号码").
			Default("").
			MaxLen(255).
			Optional().
			Nillable(),

		field.String("avatar").
			Comment("头像").
			Optional().
			Nillable(),

		field.String("address").
			Comment("地址").
			Default("").
			Optional().
			Nillable(),

		field.String("region").
			Comment("国家地区").
			Default("").
			Optional().
			Nillable(),

		field.String("description").
			Comment("个人说明").
			MaxLen(1023).
			Optional().
			Nillable(),

		field.Enum("gender").
			Comment("性别").
			NamedValues(
				"Secret", "SECRET",
				"Male", "MALE",
				"Female", "FEMALE",
			).
			Default("SECRET").
			Optional().
			Nillable(),

		field.Time("last_login_at").
			Comment("最后一次登录的时间").
			Optional().
			Nillable(),

		field.String("last_login_ip").
			Comment("最后一次登录的IP").
			Optional().
			Nillable(),

		field.Time("locked_until").
			Comment("锁定截止时间").
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("状态").
			Optional().
			Nillable().
			Default("NORMAL").
			NamedValues(
				"Normal", "NORMAL",
				"Disabled", "DISABLED",
				"Pending", "PENDING",
				"Locked", "LOCKED",
				"Expired", "EXPIRED",
				"Closed", "CLOSED",
			),
	}
}

// Mixin of the User.
func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.OperatorID{},
		mixin.TimeAt{},
		mixin.Remark{},
		mixin.TenantID{},
	}
}

// Policy for all schemas that embed User.
func (User) Policy() ent.Policy {
	return privacy.Policy{
		Query: rule.TenantQueryPolicy(),
	}
}

// Indexes of the User.
func (User) Indexes() []ent.Index {
	return []ent.Index{
		// 在租户范围内保证 username 唯一
		index.Fields("tenant_id", "username").Unique().StorageKey("idx_sys_user_tenant_username"),

		// 在租户范围内保证 email 唯一（email 可为空，DB 上允许多个 NULL）
		index.Fields("tenant_id", "email").Unique().StorageKey("idx_sys_user_tenant_email"),

		// 按租户 + 手机号，用于按手机号查询（非唯一，号码可能为空/默认值）
		index.Fields("tenant_id", "mobile").StorageKey("idx_sys_user_tenant_mobile"),

		// 按租户 + 最近登录时间，用于按时间范围检索最近登录用户
		index.Fields("tenant_id", "last_login_at").StorageKey("idx_sys_user_tenant_last_login_at"),

		// 按租户 + 最后登录 IP，用于来源溯源
		index.Fields("tenant_id", "last_login_ip").StorageKey("idx_sys_user_tenant_last_login_ip"),

		// 按租户 + 操作者，用于审计与变更追溯
		index.Fields("tenant_id", "created_by").StorageKey("idx_sys_user_tenant_created_by"),

		// 按租户 + 创建时间，用于租户范围的时间区间查询与分页
		index.Fields("tenant_id", "created_at").StorageKey("idx_sys_user_tenant_created_at"),
	}
}
