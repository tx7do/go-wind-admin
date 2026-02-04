package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/tx7do/go-crud/entgo/mixin"

	resourceV1 "go-wind-admin/api/gen/go/resource/service/v1"
)

// Menu holds the schema definition for the Menu entity.
type Menu struct {
	ent.Schema
}

func (Menu) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_menus",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("菜单资源表"),
	}
}

// Fields of the Menu.
func (Menu) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("type").
			Comment("菜单类型 CATALOG: 目录 MENU: 菜单 BUTTON: 按钮 EMBEDDED: 内嵌 LINK: 外链").
			NamedValues(
				"Catalog", "CATALOG",
				"Menu", "MENU",
				"Button", "BUTTON",
				"Embedded", "EMBEDDED",
				"Link", "LINK",
			).
			Default("MENU").
			Optional().
			Nillable(),

		field.String("path").
			Comment("路径,当其类型为'按钮'的时候对应的数据操作名,例如:/identity.service.v1.UserService/Login").
			Default("").
			Optional().
			Nillable(),

		field.String("redirect").
			Comment("重定向地址").
			Optional().
			Nillable(),

		field.String("alias").
			Comment("路由别名").
			Optional().
			Nillable(),

		field.String("name").
			Comment("路由命名，然后我们可以使用 name 而不是 path 来传递 to 属性给 <router-link>。").
			Optional().
			Nillable(),

		field.String("component").
			Comment("前端页面组件").
			Default("").
			Optional().
			Nillable(),

		field.JSON("meta", &resourceV1.MenuMeta{}).
			Comment("路由元信息").
			Optional().
			Annotations(
				entsql.Annotation{ /* 选填 */ },
			),
	}
}

func (Menu) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.Tree[Menu]{},
		//mixin.TreePath{},
		mixin.Remark{},
		mixin.SwitchStatus{},
	}
}

// Indexes of the Menu.
func (Menu) Indexes() []ent.Index {
	return []ent.Index{
		// 在同一父节点下保证 name 唯一（避免同级重复名称）
		index.Fields("parent_id", "name").
			Unique().
			StorageKey("idx_sys_menu_parent_name"),

		// 在同一父节点下保证 path 唯一（避免同级重复路由）
		index.Fields("parent_id", "path").
			Unique().
			StorageKey("idx_sys_menu_parent_path"),

		// 按路径快速查找（全局）
		index.Fields("path").
			StorageKey("idx_sys_menu_path"),

		// 别名索引，用于按 alias 定位
		index.Fields("alias").
			StorageKey("idx_sys_menu_alias"),

		// 前端组件索引（便于按组件聚合或过滤）
		index.Fields("component").
			StorageKey("idx_sys_menu_component"),

		// 菜单类型索引（目录/菜单/按钮等）
		index.Fields("type").
			StorageKey("idx_sys_menu_type"),

		// 菜单状态索引（启用/禁用）
		index.Fields("status").
			StorageKey("idx_sys_menu_status"),

		// 父节点索引，用于树结构查询及子节点检索
		index.Fields("parent_id").
			StorageKey("idx_sys_menu_parent"),

		// 操作者 + 创建时间，用于审计回溯（时间列放末尾利于范围扫描）
		index.Fields("created_by", "created_at").
			StorageKey("idx_sys_menu_created_by_created_at"),

		// 创建时间索引用于列表分页与时间区间查询
		index.Fields("created_at").
			StorageKey("idx_sys_menu_created_at"),
	}
}
