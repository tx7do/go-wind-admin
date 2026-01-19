package models

import (
	"github.com/tx7do/go-crud/gorm/mixin"
)

// LoginPolicy 对应表 sys_login_policies
type LoginPolicy struct {
	mixin.AutoIncrementID
	mixin.TimeAt
	mixin.OperatorID

	TargetID *uint32 `gorm:"column:target_id;type:int unsigned;comment:目标用户ID"`
	Value    *string `gorm:"column:value;type:varchar(255);comment:限制值（如IP地址、MAC地址或地区代码）"`
	Reason   *string `gorm:"column:reason;type:varchar(255);comment:限制原因"`

	// 使用字符串表示枚举，保留可空语义
	Type   *string `gorm:"column:type;type:varchar(32);default:BLACKLIST;comment:限制类型"`
	Method *string `gorm:"column:method;type:varchar(32);default:IP;comment:限制方式"`
}

func (LoginPolicy) TableName() string {
	return "sys_login_policies"
}
