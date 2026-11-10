package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// RoleFieldPermission 角色字段权限配置表。
// 记录角色在指定资源（proto 消息名，如 User）上不可见的字段集（黑名单语义：
// 未配置 = 全字段可见；配置 = 命中字段在读写两侧被剔除）。多角色登录时按并集聚合。
type RoleFieldPermission struct {
	ent.Schema
}

func (RoleFieldPermission) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_role_field_permissions",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("角色字段权限配置表"),
	}
}

// Fields of the RoleFieldPermission.
func (RoleFieldPermission) Fields() []ent.Field {
	return []ent.Field{

		field.Int64("role_id").
			Comment("角色ID（关联sys_roles.id）").
			Nillable(),

		field.String("resource").
			MaxLen(128).
			Comment("资源名（proto 消息名，如 User）").
			Nillable(),

		field.String("field_name").
			MaxLen(128).
			Comment("字段名（proto 字段 json_name，如 email）").
			Nillable(),
	}
}

// Mixin of the RoleFieldPermission.
func (RoleFieldPermission) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TimeAt{},
		mixin.OperatorID64{},
		mixin.TenantID[int64]{},
	}
}

// Indexes of the RoleFieldPermission.
func (RoleFieldPermission) Indexes() []ent.Index {
	return []ent.Index{
		// 租户维度唯一：同一租户内 role + resource + field 唯一
		index.Fields("tenant_id", "role_id", "resource", "field_name").
			Unique().
			StorageKey("uix_rfp_tenant_role_resource_field"),

		// 常用查询：按租户+角色取隐藏字段集
		index.Fields("tenant_id", "role_id").
			StorageKey("idx_rfp_tenant_role"),
	}
}

// Policy 追加租户变更防护：go-crud TenantPrivacy 只覆盖 Query/Create，
// 本规则为 Update/UpdateOne/Delete/DeleteOne 注入 tenant_id 过滤，
// 防止知道 ID 的跨租户篡改与删除（平台/系统上下文放行）。
func (RoleFieldPermission) Policy() ent.Policy {
	return &TenantMutationGuardPolicy{}
}
