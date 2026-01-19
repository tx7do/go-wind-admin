package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// Tenant holds the schema definition for the Tenant entity.
type Tenant struct {
	ent.Schema
}

func (Tenant) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_tenants",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("租户表"),
	}
}

// Fields of the Tenant.
func (Tenant) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("租户名称").
			//Unique().
			NotEmpty().
			Optional().
			Nillable(),

		field.String("code").
			Comment("租户编号").
			//Unique().
			NotEmpty().
			Optional().
			Nillable(),

		field.String("logo_url").
			Comment("租户logo地址").
			Optional().
			Nillable(),

		field.String("domain").
			Comment("租户专属域名").
			Optional().
			Nillable(),

		field.String("industry").
			Comment("所属行业").
			Optional().
			Nillable(),

		field.Uint32("admin_user_id").
			Comment("管理员用户ID").
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("租户状态").
			NamedValues(
				"On", "ON",
				"Off", "OFF",
				"Expired", "EXPIRED",
				"Freeze", "FREEZE",
			).
			Default("ON").
			Optional().
			Nillable(),

		field.Enum("type").
			Comment("租户类型").
			NamedValues(
				"Trial", "TRIAL",
				"Paid", "PAID",
				"Internal", "INTERNAL",
				"Partner", "PARTNER",
				"Custom", "CUSTOM",
			).
			Default("PAID").
			Optional().
			Nillable(),

		field.Enum("audit_status").
			Comment("审核状态").
			NamedValues(
				"Pending", "PENDING",
				"Approved", "APPROVED",
				"Rejected", "REJECTED",
			).
			Optional().
			Nillable(),

		field.Time("subscription_at").
			Comment("订阅时间").
			Optional().
			Nillable(),

		field.Time("unsubscribe_at").
			Comment("取消订阅时间").
			Optional().
			Nillable(),

		field.String("subscription_plan").
			Comment("订阅套餐").
			Optional().
			Nillable(),

		field.Time("expired_at").
			Comment("租户有效期").
			Optional().
			Nillable(),
	}
}

// Mixin of the Tenant.
func (Tenant) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.Remark{},
	}
}

// Indexes of the Tenant.
func (Tenant) Indexes() []ent.Index {
	return []ent.Index{
		// 保持 name 唯一
		index.Fields("name").
			Unique().
			StorageKey("idx_sys_tenant_name"),

		// 保持 code 唯一
		index.Fields("code").
			Unique().
			StorageKey("idx_sys_tenant_code"),

		// 按域名快速定位租户
		index.Fields("domain").StorageKey("idx_sys_tenant_domain"),

		// 按管理员查询
		index.Fields("admin_user_id").StorageKey("idx_sys_tenant_admin_user_id"),

		// 状态 + 审核状态联合过滤
		index.Fields("status", "audit_status").StorageKey("idx_sys_tenant_status_audit_status"),

		// 按类型 + 到期时间，用于分类型的到期筛选
		index.Fields("type", "expired_at").StorageKey("idx_sys_tenant_type_expired_at"),

		// 订阅时间（范围查询）
		index.Fields("subscription_at").StorageKey("idx_sys_tenant_subscription_at"),

		// 单列到期时间索引（方便单列范围查询）
		index.Fields("expired_at").StorageKey("idx_sys_tenant_expired_at"),

		// 操作者 + 创建时间，用于审计回溯（时间列放末尾以利于范围扫描）
		index.Fields("created_by", "created_at").StorageKey("idx_sys_tenant_created_by_created_at"),

		// 创建时间索引，用于租户列表的时间区间查询与分页
		index.Fields("created_at").StorageKey("idx_sys_tenant_created_at"),
	}
}
