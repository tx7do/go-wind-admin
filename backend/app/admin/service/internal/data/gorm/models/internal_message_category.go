package models

import "github.com/tx7do/go-crud/gorm/mixin"

// InternalMessageCategory 对应表 internal_message_categories
type InternalMessageCategory struct {
	mixin.AutoIncrementID64

	Name    *string `gorm:"column:name;type:varchar(255);comment:名称"`
	Code    *string `gorm:"column:code;type:varchar(128);comment:编码"`
	IconURL *string `gorm:"column:icon_url;type:varchar(1024);comment:图标URL"`

	// 简单树结构字段（原 Ent 的 Tree 在 GORM mixin 中可能不同，这里保留常用字段）
	ParentID *int64 `gorm:"column:parent_id;type:int unsigned;comment:父级ID"`
	Path     *string `gorm:"column:path;type:varchar(1024);comment:路径"`

	mixin.TimeAt
	mixin.OperatorID64
	mixin.IsEnabled
	mixin.SortOrder
	mixin.Remark
	mixin.TenantID64
}

// TableName 指定表名
func (InternalMessageCategory) TableName() string {
	return "internal_message_categories"
}
