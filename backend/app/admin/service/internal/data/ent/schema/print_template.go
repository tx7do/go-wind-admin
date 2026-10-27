package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// PrintTemplate holds the schema definition for the PrintTemplate entity.
//
// 单据打印模板：payload 存设计器（go-wind-print 内核）的 TemplateData JSON（毫米布局，
// 含 schemaVersion），服务端不解析只透传；paper_size/orientation 冗余成列供列表页直显。
// 模板按租户隔离（TenantID mixin）；is_built_in 行为系统内置（禁删，可复制派生）。
type PrintTemplate struct {
	ent.Schema
}

func (PrintTemplate) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "print_templates",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("单据打印模板表（设计器 TemplateData JSON；租户隔离，内置模板禁删可复制）"),
	}
}

// Fields of the PrintTemplate.
func (PrintTemplate) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("模板名称（展示用）").
			Optional().
			Nillable(),

		field.String("code").
			Comment("模板编码，租户内唯一，业务侧按编码取模板").
			Optional().
			Nillable(),

		field.String("resource_key").
			Comment("绑定资源标识（对应菜单/资源的 generated service 键，字段面板据其拉字段 schema）").
			Optional().
			Nillable(),

		field.Text("payload").
			Comment("模板 JSON（go-wind-print TemplateData：毫米布局 + schemaVersion，服务端不解析只透传）").
			Optional().
			Nillable(),

		field.Bool("is_built_in").
			Comment("是否系统内置模板（内置禁删，可复制派生）").
			Optional().
			Nillable().
			Default(false),

		field.String("paper_size").
			Comment("纸张尺寸冗余列（A4/A5/80mm 等，来自 payload.paperSize，列表页直显免解析 JSON）").
			Optional().
			Nillable(),

		field.String("orientation").
			Comment("纸张方向冗余列（portrait/landscape，来自 payload.orientation）").
			Optional().
			Nillable(),
	}
}

// Mixin of the PrintTemplate.
func (PrintTemplate) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.IsEnabled{},
		mixin.SortOrder{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the PrintTemplate.
func (PrintTemplate) Indexes() []ent.Index {
	return []ent.Index{
		// 模板编码租户内唯一：业务按 (tenant, code) 取模板，跨租户重码不冲突
		index.Fields("code", "tenant_id").
			Unique().
			StorageKey("uidx_print_templates_code_tenant"),

		// 绑定资源索引用于"按资源列模板"（列表行打印按钮按资源取默认模板）
		index.Fields("resource_key").
			StorageKey("idx_print_templates_resource_key"),

		// 创建时间索引用于列表分页与时间区间查询
		index.Fields("created_at").
			StorageKey("idx_print_templates_created_at"),
	}
}
