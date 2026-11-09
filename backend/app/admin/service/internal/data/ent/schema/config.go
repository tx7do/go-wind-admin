package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// SysConfig holds the schema definition for the SysConfig entity.
//
// proto 侧实体名为 Config（config.service.v1.Config，路由 /admin/v1/configs），
// 但 ent 预声明了小写标识符 config，实体不能叫 Config，故 ent 层用 SysConfig。
// CopierMapper 只按字段名映射，两侧类型名不必一致；表名固定 sys_configs。
type SysConfig struct {
	ent.Schema
}

func (SysConfig) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_configs",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("系统参数表（平台全局动态 KV 运行时配置；区别于字典管理的业务枚举，两者语义不同）"),
	}
}

// Fields of the SysConfig.
func (SysConfig) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("参数名称（展示用）").
			Optional().
			Nillable(),

		field.String("key").
			Comment("参数键名，全局唯一，服务侧读取器按键定位").
			Optional().
			Nillable(),

		field.String("value").
			Comment("参数键值").
			Optional().
			Nillable(),

		field.Enum("value_type").
			Comment("参数值类型（读取器按类型解析，类型不符回退默认值）").
			NamedValues(
				"String", "STRING",
				"Bool", "BOOL",
				"Int", "INT",
			).
			Default("STRING").
			Optional().
			Nillable(),

		field.Bool("is_built_in").
			Comment("是否系统内置参数（内置参数禁止删除）").
			Optional().
			Nillable().
			Default(false),
	}
}

// Mixin of the SysConfig.
func (SysConfig) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TimeAt{},
		mixin.OperatorID64{},
	}
}

// Indexes of the SysConfig.
func (SysConfig) Indexes() []ent.Index {
	return []ent.Index{
		// 键名全局唯一：读取器按 key 定位参数，重复键会让取值语义歧义
		index.Fields("key").
			Unique().
			StorageKey("uidx_sys_configs_key"),

		// 创建时间索引用于列表分页与时间区间查询
		index.Fields("created_at").
			StorageKey("idx_sys_configs_created_at"),
	}
}
