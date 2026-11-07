package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// BiDashboardRevision holds the schema definition for the BiDashboardRevision entity.
//
// BI 仪表盘版本快照（docs/bi_dashboard.md「已知边界/后续」的 revisions 表）：
// 每次成功保存把**保存前**的 payload/name/version 留一行快照，恢复 = 把快照写回主表
// （恢复动作本身先快照当前版本，可逆）。只增不改不删（表级无更新入口），主表删除时级联清理。
type BiDashboardRevision struct {
	ent.Schema
}

func (BiDashboardRevision) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_bi_dashboard_revisions",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("BI 仪表盘版本快照表（每次保存留旧版快照；只增不更新，随主表级联删除）"),
	}
}

// Fields of the BiDashboardRevision.
func (BiDashboardRevision) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("dashboard_id").
			Comment("所属仪表盘 ID").
			Optional().
			Nillable(),

		field.Uint32("version").
			Comment("快照对应的仪表盘版本号（保存前的旧版本）").
			Default(0).
			Optional().
			Nillable(),

		field.String("name").
			Comment("保存前的名称快照（恢复时一并写回）").
			Optional().
			Nillable(),

		field.String("payload").
			Comment("保存前的仪表盘 JSON 快照").
			SchemaType(map[string]string{
				dialect.MySQL:    "json",
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),

		field.Enum("source").
			Comment("快照来源（SAVE=常规保存触发 RESTORE=恢复前留底）").
			NamedValues("SAVE", "SAVE", "RESTORE", "RESTORE").
			Default("SAVE").
			Optional().
			Nillable(),
	}
}

// Mixin of the BiDashboardRevision.
func (BiDashboardRevision) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

// Edges of the BiDashboardRevision.
func (BiDashboardRevision) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("dashboard", BiDashboard.Type).
			Ref("revisions").
			Field("dashboard_id").
			Unique(),
	}
}

// Indexes of the BiDashboardRevision.
func (BiDashboardRevision) Indexes() []ent.Index {
	return []ent.Index{
		// 一张仪表盘一个版本号只留一份快照（保存/恢复去重）
		index.Fields("dashboard_id", "version").
			Unique().
			StorageKey("uidx_bi_revisions_dash_version"),

		// 历史列表按仪表盘拉取、新快照在前
		index.Fields("dashboard_id", "created_at").
			StorageKey("idx_bi_revisions_dash_created"),
	}
}
