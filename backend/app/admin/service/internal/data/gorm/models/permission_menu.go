package models

import (
	"github.com/tx7do/go-crud/gorm/mixin"
)

// PermissionMenu 对应表 sys_permission_menus
type PermissionMenu struct {
	mixin.AutoIncrementID

	PermissionId *uint32 `gorm:"column:permission_id;type:int unsigned;comment:权限ID"`
	// 绑定存菜单全路径快照而非 menu_id：授权执行键（菜单身份键），免疫菜单
	// 同步 REPLACE 重建换 id（ent 栈同构，见 data/permission_menu_repo.go）。
	FullPath *string `gorm:"column:full_path;type:varchar(512);comment:菜单全路径（菜单身份键绑定快照）"`

	mixin.TimeAt
	mixin.OperatorID
}

// TableName 指定表名
func (PermissionMenu) TableName() string {
	return "sys_permission_menus"
}
