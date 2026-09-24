package models

import (
	"time"

	"github.com/tx7do/go-crud/gorm/mixin"
)

// SysConfig 对应表 sys_configs - 系统参数表（平台全局动态 KV 运行时配置；区别于字典管理的业务枚举，两者语义不同）
type SysConfig struct {
	mixin.AutoIncrementID

	Name      *string    `gorm:"column:name;type:varchar(255);comment:参数名称（展示用）"`
	Key       *string    `gorm:"column:key;type:varchar(255);uniqueIndex;comment:参数键名，全局唯一，服务侧读取器按键定位"`
	Value     *string    `gorm:"column:value;type:varchar(1024);comment:参数键值"`
	ValueType *string    `gorm:"column:value_type;type:varchar(64);default:'STRING';comment:参数值类型：STRING/BOOL/INT"`
	IsBuiltIn *bool      `gorm:"column:is_built_in;type:TINYINT;default:0;comment:是否系统内置参数（内置参数禁止删除）"`
	CreatedBy *uint32    `gorm:"column:created_by;type:int unsigned;comment:创建者 ID"`
	UpdatedBy *uint32    `gorm:"column:updated_by;type:int unsigned;comment:更新者 ID"`
	CreatedAt *time.Time `gorm:"column:created_at;type:datetime;comment:创建时间"`
	UpdatedAt *time.Time `gorm:"column:updated_at;type:datetime;comment:更新时间"`
}

// TableName 指定表名
func (SysConfig) TableName() string {
	return "sys_configs"
}
