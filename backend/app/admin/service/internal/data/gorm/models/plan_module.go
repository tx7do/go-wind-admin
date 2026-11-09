package models

import (
	"github.com/tx7do/go-crud/gorm/mixin"
)

// PlanModule 对应表 sys_plan_modules
type PlanModule struct {
	mixin.AutoIncrementID64

	Module *string `gorm:"column:module;type:varchar(128);comment:模块"`

	mixin.TimeAt
	mixin.OperatorID64
}

// TableName 指定表名
func (PlanModule) TableName() string {
	return "sys_plan_modules"
}
