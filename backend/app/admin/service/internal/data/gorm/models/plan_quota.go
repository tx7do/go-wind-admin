package models

import (
	"github.com/tx7do/go-crud/gorm/mixin"
)

// PlanQuota 对应表 sys_plan_quotas
type PlanQuota struct {
	mixin.AutoIncrementID64

	QuotaType  *string `gorm:"column:quota_type;type:varchar(128);comment:配额类型"`
	QuotaValue *uint64 `gorm:"column:quota_value;type:bigint unsigned;comment:配额值"`

	mixin.TimeAt
	mixin.OperatorID64
}

// TableName 指定表名
func (PlanQuota) TableName() string {
	return "sys_plan_quotas"
}
