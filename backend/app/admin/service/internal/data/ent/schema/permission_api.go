package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/tx7do/go-crud/entgo/mixin"
)

// PermissionApi holds the schema definition for the PermissionApi entity.
type PermissionApi struct {
	ent.Schema
}

func (PermissionApi) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_permission_apis",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("权限点与API接口关联表"),
	}
}

// Fields of the PermissionApi.
func (PermissionApi) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("permission_id").
			Comment("权限ID（关联sys_permissions.id）").
			Nillable(),

		// 绑定存 (path, method) 快照而非 api_id：sys_apis 按自然键 MERGE 同步
		// 下 id 虽不再漂移，但"删了再加"场景 id 必变；(path, method) 是授权
		// 执行的最终裁决键（authorizer 策略、租户闸门同键），以它绑定天然免疫
		// 端点的删除重建。落库时由 api_id 一次性解析而来（见 permission_api_repo）。
		field.String("path").
			Comment("接口路径（API身份键绑定快照）").
			Optional().
			Nillable(),

		field.String("method").
			Comment("请求方法（API身份键绑定快照）").
			Optional().
			Nillable(),
	}
}

// Mixin of the PermissionApi.
func (PermissionApi) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
	}
}

// Indexes of the PermissionApi.
func (PermissionApi) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("permission_id", "path", "method").
			Unique().
			StorageKey("uix_perm_api_permission_endpoint"),

		// 单列索引：按 permission_id 快速查询
		index.Fields("permission_id").
			StorageKey("idx_perm_api_permission_id"),

		// 端点反查：API 行被删除/重建时按 (path, method) 级联清理绑定
		index.Fields("path", "method").
			StorageKey("idx_perm_api_path_method"),
	}
}
