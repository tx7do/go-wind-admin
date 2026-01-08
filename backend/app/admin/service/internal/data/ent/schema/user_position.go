package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/tx7do/go-crud/entgo/mixin"
)

// UserPosition 用户与岗位关联表
type UserPosition struct {
	ent.Schema
}

func (UserPosition) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_user_positions",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("用户与岗位关联表"),
	}
}

// Fields of the UserPosition.
func (UserPosition) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("user_id").
			Comment("用户ID").
			Nillable(),

		// 关联到岗位（必填）
		field.Uint32("position_id").
			Comment("岗位ID").
			Nillable(),

		field.Bool("is_primary").
			Comment("是否主岗位").
			Optional().
			Nillable().
			Default(false),

		field.Time("start_at").
			Comment("生效时间").
			Optional().
			Nillable(),

		field.Time("end_at").
			Comment("失效时间").
			Optional().
			Nillable(),

		field.Time("assigned_at").
			Comment("岗位分配时间").
			Optional().
			Nillable(),

		field.Uint32("assigned_by").
			Comment("分配者用户 ID").
			Optional().
			Nillable(),

		field.Enum("status").
			NamedValues(
				"Probation", "PROBATION",
				"Active", "ACTIVE",
				"Leave", "LEAVE",
				"Terminated", "TERMINATED",
				"Expired", "EXPIRED",
			).
			Default("ACTIVE").
			Comment("岗位状态"),
	}
}

// Mixin of the UserPosition.
func (UserPosition) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID{},
		mixin.Remark{},
	}
}

// Indexes of the UserPosition.
func (UserPosition) Indexes() []ent.Index {
	return []ent.Index{
		// 在 tenant 维度上保证唯一性，避免全局记录冲突
		index.Fields("tenant_id", "user_id", "position_id").
			Unique().
			StorageKey("uix_up_tenant_user_pos"),

		// 常用查询：在租户范围内按 user 查所有岗位
		index.Fields("tenant_id", "user_id").
			StorageKey("idx_up_tenant_user"),

		// 常用查询：在租户范围内按 position 查所有成员
		index.Fields("tenant_id", "position_id").
			StorageKey("idx_up_tenant_position"),

		// 快速查找某成员在租户下的主岗位
		index.Fields("tenant_id", "user_id", "is_primary").
			StorageKey("idx_up_tenant_user_primary"),

		// 按分配者查询（租户范围内或全局）
		index.Fields("tenant_id", "assigned_by").
			StorageKey("idx_up_tenant_assigned_by"),
		index.Fields("assigned_by").
			StorageKey("idx_up_assigned_by"),

		// 保留/补充常用的单列索引以支持简单或模糊查询
		index.Fields("user_id").
			StorageKey("idx_up_user_id"),
		index.Fields("position_id").
			StorageKey("idx_up_position_id"),
		index.Fields("tenant_id").
			StorageKey("idx_up_tenant_id"),
		index.Fields("is_primary").
			StorageKey("idx_up_is_primary"),
		index.Fields("status").
			StorageKey("idx_up_status"),
	}
}
