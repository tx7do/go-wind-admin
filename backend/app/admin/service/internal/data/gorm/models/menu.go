package models

import (
	"github.com/tx7do/go-crud/gorm/mixin"
	"gorm.io/datatypes"
)

// Menu 对应表 sys_menus
type Menu struct {
	mixin.AutoIncrementID64
	mixin.Tree64[Menu]

	Type      *string        `gorm:"column:type;type:varchar(32);default:MENU;comment:菜单类型 CATALOG: 目录 MENU: 菜单 BUTTON: 按钮 EMBEDDED: 内嵌 LINK: 外链"`
	Path      *string        `gorm:"column:path;type:varchar(512);comment:树路径，规范： 根节点: /，非根节点: /1/2/3/（以 / 开头且以 / 结尾）。禁止空字符串（NULL 表示未设置）。"`
	Redirect  *string        `gorm:"column:redirect;type:varchar(255);comment:重定向地址"`
	Alias     *string        `gorm:"column:alias;type:varchar(255);comment:路由别名"`
	Name      *string        `gorm:"column:name;type:varchar(255);comment:路由命名"`
	Component *string        `gorm:"column:component;type:varchar(255);comment:前端页面组件"`
	Meta      datatypes.JSON `gorm:"column:meta;type:json;comment:前端页面组件元数据"`
	Module    *string        `gorm:"column:module;type:varchar(128);comment:所属业务功能模块（用于套餐白名单过滤）"`

	mixin.TimeAt
	mixin.OperatorID64
	mixin.Remark
	mixin.SwitchStatus
}

// TableName 指定表名
func (Menu) TableName() string {
	return "sys_menus"
}
