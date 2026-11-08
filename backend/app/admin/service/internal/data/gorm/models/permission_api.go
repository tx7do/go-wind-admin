package models

import (
	"github.com/tx7do/go-crud/gorm/mixin"
)

// PermissionApi 对应表 sys_permission_apis
type PermissionApi struct {
	mixin.AutoIncrementID

	PermissionId *uint32 `gorm:"column:permission_id;type:int unsigned;comment:权限ID"`
	// 绑定存 (path, method) 端点快照而非 api_id：授权执行键，免疫端点删除重建
	//（ent 栈同构，见 data/permission_api_repo.go 的 Endpoint 注释）。
	Path   *string `gorm:"column:path;type:varchar(255);comment:接口路径（API身份键绑定快照）"`
	Method *string `gorm:"column:method;type:varchar(16);comment:请求方法（API身份键绑定快照）"`

	mixin.TimeAt
	mixin.OperatorID
}

// TableName 指定表名
func (PermissionApi) TableName() string {
	return "sys_permission_apis"
}
