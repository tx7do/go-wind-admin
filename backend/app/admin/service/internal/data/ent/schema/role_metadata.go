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

	userV1 "go-wind-admin/api/gen/go/user/service/v1"
)

// RoleMetadata 角色元数据（模板标记/覆盖项/版本控制）
type RoleMetadata struct {
	ent.Schema
}

func (RoleMetadata) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_role_metadata",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("角色元数据"),
	}
}

// Fields of the RoleMetadata.
func (RoleMetadata) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("role_id").
			Comment("角色ID").
			Optional().
			Nillable(),

		field.Bool("is_template").
			Comment("是否是模版").
			Optional().
			Default(false).
			Nillable(),

		field.String("template_for").
			Comment("模板适用对象").
			Optional().
			Nillable(),

		field.Int32("template_version").
			Comment("模板版本号").
			Default(1).
			Optional().
			Nillable(),

		field.Int32("last_synced_version").
			Comment("上次同步的版本号").
			Optional().
			Nillable(),

		field.Time("last_synced_at").
			Comment("最后同步时间").
			Optional().
			Nillable(),

		field.Enum("sync_policy").
			Comment("同步策略").
			NamedValues(
				"Auto", "AUTO",
				"Manual", "MANUAL",
				"Blocked", "BLOCKED",
			).
			Default("AUTO").
			Optional().
			Nillable(),

		field.Enum("scope").
			Comment("作用域").
			NamedValues(
				"Platform", "PLATFORM",
				"Tenant", "TENANT",
			).
			Default("TENANT").
			Optional().
			Nillable(),

		field.JSON("custom_overrides", &userV1.RoleOverride{}).
			Comment("租户自定义覆盖项").
			Default(&userV1.RoleOverride{}),
	}
}

// Mixin of the RoleMetadata.
func (RoleMetadata) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID{},
	}
}

// Policy for all schemas that embed RoleMetadata.
func (RoleMetadata) Policy() ent.Policy {
	return privacy.Policy{
		Query: rule.TenantQueryPolicy(),
	}
}

// Indexes of the RoleMetadata.
func (RoleMetadata) Indexes() []ent.Index {
	return []ent.Index{
		// 每个租户内 role_id 唯一（支持多租户隔离）
		index.Fields("tenant_id", "role_id").
			Unique().
			StorageKey("idx_role_metadata_tenant_role"),

		// 防止同一租户中同一适用对象在同一版本/作用域下有重复的模版记录
		index.Fields("tenant_id", "is_template", "template_for", "template_version", "scope").
			Unique().
			StorageKey("idx_role_metadata_unique_template"),

		// 常用于查询某个适用对象的所有模版（按租户 + 是否为模版 + 适用对象）
		index.Fields("tenant_id", "is_template", "template_for").
			StorageKey("idx_role_metadata_template_lookup"),

		// 常用于按作用域筛选并定位到对应角色的元数据（按租户隔离）
		index.Fields("tenant_id", "scope", "role_id").
			StorageKey("idx_role_metadata_scope_role"),

		// 便于按同步版本快速查询或筛选（例如增量同步，按租户）
		index.Fields("tenant_id", "last_synced_version").
			StorageKey("idx_role_metadata_last_synced_version"),

		// 便于按最近同步时间排序/查询（按租户）
		index.Fields("tenant_id", "last_synced_at").
			StorageKey("idx_role_metadata_last_synced_at"),
	}
}
