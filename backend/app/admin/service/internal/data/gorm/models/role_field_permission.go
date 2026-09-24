package models

import (
	"time"

	"github.com/tx7do/go-crud/gorm/mixin"
)

// RoleFieldPermission 对应表 sys_role_field_permissions - 角色字段权限配置表
// 记录角色在指定资源（proto 消息名，如 User）上不可见的字段集（黑名单语义：
// 未配置 = 全字段可见；配置 = 命中字段在读写两侧被剔除）。多角色登录时按并集聚合。
type RoleFieldPermission struct {
	mixin.AutoIncrementID

	RoleId    *uint32    `gorm:"column:role_id;type:int unsigned;comment:角色 ID（关联 sys_roles.id）"`
	Resource  *string    `gorm:"column:resource;type:varchar(128);comment:资源名（proto 消息名，如 User）"`
	FieldName *string    `gorm:"column:field_name;type:varchar(128);comment:字段名（proto 字段 json_name，如 email）"`
	CreatedBy *uint32    `gorm:"column:created_by;type:int unsigned;comment:创建者 ID"`
	UpdatedBy *uint32    `gorm:"column:updated_by;type:int unsigned;comment:更新者 ID"`
	CreatedAt *time.Time `gorm:"column:created_at;type:datetime;comment:创建时间"`
	UpdatedAt *time.Time `gorm:"column:updated_at;type:datetime;comment:更新时间"`
	TenantId  *uint32    `gorm:"column:tenant_id;type:int unsigned;comment:租户 ID；多租户隔离键"`
}

// TableName 指定表名
func (RoleFieldPermission) TableName() string {
	return "sys_role_field_permissions"
}
