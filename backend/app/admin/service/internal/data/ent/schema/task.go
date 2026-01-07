package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/tx7do/go-crud/entgo/mixin"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
)

// Task holds the schema definition for the Task entity.
type Task struct {
	ent.Schema
}

func (Task) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_tasks",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("任务表"),
	}
}

// Fields of the Task.
func (Task) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("type").
			Comment("任务类型").
			NamedValues(
				"Periodic", "PERIODIC",
				"Delay", "DELAY",
				"WaitResult", "WAIT_RESULT",
			).
			Default("PERIODIC").
			Optional().
			Nillable(),

		field.String("type_name").
			Comment("任务执行类型名").
			//Unique().
			Optional().
			Nillable(),

		field.String("task_payload").
			Comment("任务数据").
			SchemaType(map[string]string{
				dialect.MySQL:    "json",
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),

		field.String("cron_spec").
			Comment("cron表达式").
			Optional().
			Nillable(),

		field.JSON("task_options", &adminV1.TaskOption{}).
			Comment("任务选项").
			Optional(),

		field.Bool("enable").
			Comment("启用/禁用任务").
			Default(false).
			Optional().
			Nillable(),
	}
}

// Mixin of the Task.
func (Task) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID{},
	}
}

// Indexes of the Task.
func (Task) Indexes() []ent.Index {
	return []ent.Index{
		// 在租户范围内保证 type_name 唯一
		index.Fields("tenant_id", "type_name").
			Unique().
			StorageKey("idx_sys_task_tenant_type_name"),

		// 按租户 + type，用于按任务类型检索
		index.Fields("tenant_id", "type").
			StorageKey("idx_sys_task_tenant_type"),

		// 按租户 + enable + created_at，用于按启用状态过滤并按时间范围查询/分页（时间列放末尾）
		index.Fields("tenant_id", "enable", "created_at").
			StorageKey("idx_sys_task_tenant_enable_created_at"),

		// 按租户 + 操作者 + 创建时间，用于审计回溯（时间列放末尾）
		index.Fields("tenant_id", "created_by", "created_at").
			StorageKey("idx_sys_task_tenant_created_by_created_at"),

		// 按租户 + 创建时间，用于租户范围的时间区间查询与分页
		index.Fields("tenant_id", "created_at").
			StorageKey("idx_sys_task_tenant_created_at"),
	}
}
