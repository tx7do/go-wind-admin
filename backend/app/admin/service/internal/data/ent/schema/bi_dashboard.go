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

// BiDashboard holds the schema definition for the BiDashboard entity.
//
// BI 仪表盘：payload 存设计器（@go-wind/bi-core schema）产出的 JSON 原文，
// 服务端不解析只透传——与 print_templates 同口径，避免前后端版本漂移时后端拒存。
// chart_count / theme_key 冗余成列供列表页直显（免逐行解析 JSON）。
// version 为乐观锁：设计器整份覆盖保存，多标签/多人编辑靠它挡 stale 写。
type BiDashboard struct {
	ent.Schema
}

func (BiDashboard) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "bi_dashboards",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("BI 仪表盘表（设计器 schema JSON；租户隔离，内置禁删可复制，version 乐观锁）"),
	}
}

// Fields of the BiDashboard.
func (BiDashboard) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("名称（展示用）").
			Optional().
			Nillable(),

		field.String("code").
			Comment("编码，租户内唯一，业务侧按编码取仪表盘").
			Optional().
			Nillable(),

		field.Enum("kind").
			Comment("种类（DASHBOARD=仪表盘 SCREEN=大屏 TEMPLATE=可复用模板）").
			NamedValues(
				"DASHBOARD", "DASHBOARD",
				"SCREEN", "SCREEN",
				"TEMPLATE", "TEMPLATE",
			).
			Default("DASHBOARD").
			Optional().
			Nillable(),

		field.String("remark").
			Comment("备注").
			Optional().
			Nillable(),

		field.String("payload").
			Comment("仪表盘 JSON（设计器 schema：数据集 + 图表布局 + 主题；服务端不解析只透传）").
			SchemaType(map[string]string{
				dialect.MySQL:    "json",
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),

		field.Uint32("chart_count").
			Comment("图表数量冗余列（来自 payload.charts，列表页直显免解析 JSON）").
			Default(0).
			Optional().
			Nillable(),

		field.String("theme_key").
			Comment("主题预设冗余列（来自 payload.theme.key）").
			Optional().
			Nillable(),

		field.String("thumbnail").
			Comment("缩略图 URL（设计器导出 PNG 后上传，列表页预览用）").
			Optional().
			Nillable(),

		field.Bool("is_built_in").
			Comment("是否系统内置（内置禁删，可复制派生）").
			Optional().
			Nillable().
			Default(false),

		field.Bool("is_public").
			Comment("是否租户内共享（false 仅创建者可见）").
			Optional().
			Nillable().
			Default(false),

		field.Uint32("version").
			Comment("版本号（乐观锁：更新需回传读到的版本，成功后 +1）").
			Default(1).
			Optional().
			Nillable(),
	}
}

// Mixin of the BiDashboard.
func (BiDashboard) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.IsEnabled{},
		mixin.SortOrder{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the BiDashboard.
func (BiDashboard) Indexes() []ent.Index {
	return []ent.Index{
		// 编码租户内唯一：业务按 (tenant, code) 取仪表盘，跨租户重码不冲突
		index.Fields("code", "tenant_id").
			Unique().
			StorageKey("uidx_bi_dashboards_code_tenant"),

		// 列表页按种类分组（模板库/大屏筛选）
		index.Fields("kind").
			StorageKey("idx_bi_dashboards_kind"),

		// 创建时间索引用于列表分页与时间区间查询
		index.Fields("created_at").
			StorageKey("idx_bi_dashboards_created_at"),
	}
}
