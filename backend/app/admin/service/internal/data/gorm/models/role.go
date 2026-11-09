package models

import (
	"github.com/tx7do/go-crud/gorm/mixin"
)

// Role 对应表 sys_roles
type Role struct {
	mixin.AutoIncrementID64

	Name        *string `gorm:"column:name;type:varchar(255);comment:角色名称"`
	Code        *string `gorm:"column:code;type:varchar(128);comment:角色标识"`
	IsProtected *bool   `gorm:"column:is_protected;type:boolean;comment:是否受保护"`
	Type        *string `gorm:"column:type;type:varchar(128);comment:类型"`

	mixin.TimeAt
	mixin.OperatorID64
	mixin.Remark
	mixin.Description
	mixin.SortOrder
	mixin.TenantID64
	mixin.SwitchStatus
}

// TableName 指定表名
func (Role) TableName() string {
	return "sys_roles"
}
