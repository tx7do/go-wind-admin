package models

import (
	"github.com/tx7do/go-crud/gorm/mixin"
)

// Plan 对应表 sys_plans
type Plan struct {
	mixin.AutoIncrementID

	Name              *string `gorm:"column:name;type:varchar(255);comment:名称"`
	Version           *string `gorm:"column:version;type:varchar(128);comment:版本"`
	ExpiryPolicy      *string `gorm:"column:expiry_policy;type:varchar(128);comment:到期策略"`
	DataRetentionDays *uint32 `gorm:"column:data_retention_days;type:int unsigned;comment:数据保留天数"`
	Description       *string `gorm:"column:description;type:text;comment:描述"`

	mixin.TimeAt
	mixin.OperatorID
	mixin.Remark
}

// TableName 指定表名
func (Plan) TableName() string {
	return "sys_plans"
}
