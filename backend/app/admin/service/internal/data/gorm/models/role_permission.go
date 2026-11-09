package models

import (
	"github.com/tx7do/go-crud/gorm/mixin"
)

// RolePermission 对应表 sys_role_permissions
type RolePermission struct {
	mixin.AutoIncrementID64

	RoleId       *uint64 `gorm:"column:role_id;type:int unsigned;comment:角色ID"`
	PermissionId *uint64 `gorm:"column:permission_id;type:int unsigned;comment:权限ID"`
	Effect       *string `gorm:"column:effect;type:varchar(128);comment:效果"`
	Priority     *int32  `gorm:"column:priority;type:int;comment:优先级"`

	mixin.TimeAt
	mixin.OperatorID64
	mixin.TenantID64
	mixin.SwitchStatus
}

// TableName 指定表名
func (RolePermission) TableName() string {
	return "sys_role_permissions"
}
