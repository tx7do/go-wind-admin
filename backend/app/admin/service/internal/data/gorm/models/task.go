package models

import (
	"gorm.io/datatypes"

	"github.com/tx7do/go-crud/gorm/mixin"
)

// Task 对应表 sys_tasks，任务表
type Task struct {
	mixin.AutoIncrementID64

	Type        *string         `gorm:"column:type;type:varchar(128);comment:任务类型;index:idx_sys_tasks_type"`
	TypeName    *string         `gorm:"column:type_name;type:varchar(255);comment:任务执行类型名;uniqueIndex:idx_sys_task_type_name"`
	TaskPayload *datatypes.JSON `gorm:"column:task_payload;type:json;comment:任务数据"`
	CronSpec    *string         `gorm:"column:cron_spec;type:varchar(255);comment:cron表达式"`
	TaskOptions *datatypes.JSON `gorm:"column:task_options;type:json;comment:任务选项"`
	Enable      *bool           `gorm:"column:enable;type:tinyint(1);comment:启用/禁用任务"`

	mixin.TimeAt
	mixin.OperatorID64
	mixin.Remark
	mixin.TenantID64
}

// TableName 指定表名
func (Task) TableName() string {
	return "sys_tasks"
}
