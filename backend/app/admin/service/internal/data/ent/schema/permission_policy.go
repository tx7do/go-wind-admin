package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/tx7do/go-crud/entgo/mixin"
)

// PermissionPolicy 权限点动态策略表（NIST RBAC+ABAC混合标准）
type PermissionPolicy struct {
	ent.Schema
}

func (PermissionPolicy) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_permission_policies",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("权限点动态策略表"),
	}
}

// Fields of the PermissionPolicy.
func (PermissionPolicy) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("permission_id").
			Comment("权限ID（关联sys_permissions.id）").
			Nillable(),

		field.Enum("policy_engine").
			Comment("策略引擎").
			NamedValues(
				"Cel", "CEL",
				"Casbin", "CASBIN",
				"Opa", "OPA",
				"Sql", "SQL",
			).
			Default("CASBIN").
			Nillable(),

		field.String("definition").
			Comment("策略定义（动态结构）").
			SchemaType(map[string]string{
				dialect.MySQL:    "json",
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),

		field.Uint32("version").
			Comment("策略版本（用于灰度/回滚）").
			Default(1).
			Nillable(),

		field.Uint32("eval_order").
			Comment("评估优先级（越小越先执行）").
			Default(0).
			Nillable(),

		field.Uint32("cache_ttl").
			Comment("结果缓存秒数（0=不缓存）").
			Default(300).
			Nillable(),
	}
}

// Mixin of the PermissionPolicy.
func (PermissionPolicy) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID{},
		mixin.SwitchStatus{},
	}
}

// Indexes of the PermissionPolicy.
func (PermissionPolicy) Indexes() []ent.Index {
	return []ent.Index{
		// 唯一约束：同一租户下每个权限只有一条主策略记录（与原实现一致）
		index.Fields("tenant_id", "permission_id").
			Unique().
			StorageKey("uix_perm_policy_tenant_perm"),

		// 常用查询：在租户内按权限+版本查找（用于灰度/回滚场景）
		index.Fields("tenant_id", "permission_id", "version").
			StorageKey("idx_perm_policy_tenant_perm_version"),

		// 常用查询：按租户+策略引擎筛选（不同引擎的策略管理）
		index.Fields("tenant_id", "policy_engine").
			StorageKey("idx_perm_policy_engine_tenant"),

		// 支持按单列快速查询
		index.Fields("permission_id").
			StorageKey("idx_perm_policy_perm"),
		index.Fields("policy_engine").
			StorageKey("idx_perm_policy_engine"),
		index.Fields("version").
			StorageKey("idx_perm_policy_version"),
		index.Fields("tenant_id").
			StorageKey("idx_perm_policy_tenant"),
	}
}
