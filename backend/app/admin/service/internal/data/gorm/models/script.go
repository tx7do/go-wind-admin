package models

import (
	"time"

	"github.com/tx7do/go-crud/gorm/mixin"
)

// Script 对应表 sys_scripts - 平台脚本表（脚本引擎插件承载）
type Script struct {
	mixin.AutoIncrementID

	Name        *string    `gorm:"column:name;type:varchar(255);uniqueIndex;comment:脚本唯一名称"`
	Language    *string    `gorm:"column:language;type:varchar(64);default:'LUA';comment:脚本语言：LUA/JAVASCRIPT"`
	HookPoint   *string    `gorm:"column:hook_point;type:varchar(255);comment:挂载的钩子点名称，空表示未挂载"`
	Source      *string    `gorm:"column:source;type:text;comment:脚本源码"`
	Priority    *int32     `gorm:"column:priority;type:int;default:0;comment:执行优先级，越小越先执行"`
	Description *string    `gorm:"column:description;type:varchar(1024);comment:脚本用途说明"`
	Critical    *bool      `gorm:"column:critical;type:TINYINT;default:0;comment:关键脚本：执行失败时中断钩子链"`
	Version     *uint32    `gorm:"column:version;type:int unsigned;default:1;comment:版本号，每次更新自增（热更新指纹）"`
	Enabled     *bool      `gorm:"column:is_enabled;type:TINYINT;default:1;comment:是否启用"`
	CreatedBy   *uint32    `gorm:"column:created_by;type:int unsigned;comment:创建者 ID"`
	UpdatedBy   *uint32    `gorm:"column:updated_by;type:int unsigned;comment:更新者 ID"`
	CreatedAt   *time.Time `gorm:"column:created_at;type:datetime;comment:创建时间"`
	UpdatedAt   *time.Time `gorm:"column:updated_at;type:datetime;comment:更新时间"`
}

// TableName 指定表名
func (Script) TableName() string {
	return "sys_scripts"
}
