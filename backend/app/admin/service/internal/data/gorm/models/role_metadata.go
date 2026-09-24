package models

import (
	"time"

	"github.com/tx7do/go-crud/gorm/mixin"
	"gorm.io/datatypes"
)

// RoleMetadata 对应表 sys_role_metadata
type RoleMetadata struct {
	mixin.AutoIncrementID

	RoleId            *uint32         `gorm:"column:role_id;type:int unsigned;comment:角色ID"`
	IsTemplate        *bool           `gorm:"column:is_template;type:boolean;comment:是否模板"`
	TemplateFor       *string         `gorm:"column:template_for;type:varchar(255);comment:模板用于"`
	TemplateVersion   *int32          `gorm:"column:template_version;type:int;comment:模板版本"`
	LastSyncedVersion *int32          `gorm:"column:last_synced_version;type:int;comment:最后同步版本"`
	LastSyncedAt      *time.Time      `gorm:"column:last_synced_at;type:datetime;comment:最后同步时间"`
	SyncPolicy        *string         `gorm:"column:sync_policy;type:varchar(128);comment:同步策略"`
	Scope             *string         `gorm:"column:scope;type:varchar(128);comment:范围"`
	CustomOverrides   *datatypes.JSON `gorm:"column:custom_overrides;type:json;comment:自定义覆盖"`

	mixin.TimeAt
	mixin.OperatorID
	mixin.TenantID
}

// TableName 指定表名
func (RoleMetadata) TableName() string {
	return "sys_role_metadata"
}
