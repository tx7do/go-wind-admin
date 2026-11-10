package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/tx7do/go-crud/entgo/mixin"
)

// AccessKey holds the schema definition for the AccessKey entity.
// OpenAPI 访问凭证（AK/SK）：租户级机器凭证，secret 仅存 SHA-256 摘要。
type AccessKey struct {
	ent.Schema
}

func (AccessKey) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_access_keys",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("OpenAPI访问凭证表"),
	}
}

// Fields of the AccessKey.
func (AccessKey) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").Comment("凭证名称（用途说明）").Optional().Nillable(),
		field.String("access_key").Comment("访问键（AK，公开标识）").Optional().Nillable(),
		field.String("secret_hash").Comment("密钥摘要（SHA-256 hex，明文不落库）").Optional().Nillable().Sensitive(),
		field.Time("expires_at").Comment("过期时间（空表示长期有效）").Optional().Nillable(),
		field.Time("last_used_at").Comment("最近一次令牌交换时间").Optional().Nillable(),
	}
}

// Mixin of the AccessKey.
func (AccessKey) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TimeAt{},
		mixin.OperatorID64{},
		mixin.SwitchStatus{},
		mixin.TenantID[int64]{},
	}
}

// Policy 租户写隔离冗余防线：与本仓其余带租户列表一致（库层 TenantPrivacy 为主，
// 本守卫为独立第二道，见 schema/tenant_mutation_guard.go 与 docs/tenant_isolation.md）。
// 此前本表独缺此层（2026-09-12 补挂）。
func (AccessKey) Policy() ent.Policy {
	return &TenantMutationGuardPolicy{}
}

func (AccessKey) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("access_key").Unique().StorageKey("idx_sys_access_keys_access_key"),
		index.Fields("tenant_id").StorageKey("idx_sys_access_keys_tenant_id"),
	}
}
