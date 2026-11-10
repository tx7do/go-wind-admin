package models

import (
	"github.com/tx7do/go-crud/gorm/mixin"
)

// Permission 对应表 sys_permissions
type Permission struct {
	mixin.AutoIncrementID64

	Name    *string `gorm:"column:name;type:varchar(255);comment:名称"`
	Code    *string `gorm:"column:code;type:varchar(255);comment:编码"`
	GroupId *int64 `gorm:"column:group_id;type:int unsigned;comment:权限组ID"`

	mixin.TimeAt
	mixin.OperatorID64
	mixin.SwitchStatus
	mixin.Description
}

// TableName 指定表名
func (Permission) TableName() string {
	return "sys_permissions"
}
