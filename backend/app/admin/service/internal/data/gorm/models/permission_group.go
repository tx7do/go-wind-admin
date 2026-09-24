package models

import (
	"github.com/tx7do/go-crud/gorm/mixin"
)

// PermissionGroup 对应表 sys_permission_groups
type PermissionGroup struct {
	mixin.AutoIncrementID
	mixin.Tree[PermissionGroup]

	Name   *string `gorm:"column:name;type:varchar(255);comment:名称"`
	Module *string `gorm:"column:module;type:varchar(128);comment:模块"`
	Path   *string `gorm:"column:path;type:varchar(512);comment:树路径，规范： 根节点: /，非根节点: /1/2/3/（以 / 开头且以 / 结尾）。禁止空字符串（NULL 表示未设置）。"`

	mixin.TimeAt
	mixin.OperatorID
	mixin.Description
	mixin.SwitchStatus
	mixin.SortOrder
}

// TableName 指定表名
func (PermissionGroup) TableName() string {
	return "sys_permission_groups"
}
