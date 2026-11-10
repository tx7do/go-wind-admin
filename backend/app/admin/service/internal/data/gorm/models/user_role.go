package models

import "github.com/tx7do/go-crud/gorm/mixin"

// UserRole 对应表 sys_user_role，用户 - 角色 关联表
type UserRole struct {
	mixin.AutoIncrementID64

	UserID *int64 `gorm:"column:user_id;type:int unsigned;comment:用户ID;index:idx_sys_user_role_user_id;uniqueIndex:idx_sys_user_role_user_id_role_id,priority:1"`
	RoleID *int64 `gorm:"column:role_id;type:int unsigned;comment:角色ID;index:idx_sys_user_role_role_id;uniqueIndex:idx_sys_user_role_user_id_role_id,priority:2"`

	mixin.TimeAt
	mixin.OperatorID64
}

// TableName 指定表名
func (UserRole) TableName() string {
	return "sys_user_roles"
}
