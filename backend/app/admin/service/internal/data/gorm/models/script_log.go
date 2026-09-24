package models

import (
	"time"

	"github.com/tx7do/go-crud/gorm/mixin"
)

// ScriptLog 对应表 sys_script_logs - 脚本执行日志：每次脚本/钩子/任务处理器执行一条记录（成败、耗时、错误）
type ScriptLog struct {
	mixin.AutoIncrementID

	ScriptId    *uint32    `gorm:"column:script_id;type:int unsigned;default:0;comment:脚本 ID（试运行草稿为 0）"`
	ScriptName  *string    `gorm:"column:script_name;type:varchar(255);comment:脚本名称"`
	Language    *string    `gorm:"column:language;type:varchar(64);comment:脚本语言：LUA/JAVASCRIPT"`
	TriggerType *string    `gorm:"column:trigger_type;type:varchar(64);comment:触发方式：hook/task/test_run/manual"`
	HookPoint   *string    `gorm:"column:hook_point;type:varchar(255);comment:钩子点或任务类型"`
	Version     *uint32    `gorm:"column:version;type:int unsigned;default:0;comment:执行时的脚本版本（草稿为 0）"`
	Success     *bool      `gorm:"column:success;type:TINYINT;default:0;comment:是否执行成功"`
	DurationMs  *int64     `gorm:"column:duration_ms;type:bigint;default:0;comment:执行耗时（毫秒）"`
	Error       *string    `gorm:"column:error;type:text;comment:失败原因（成功为空）"`
	CreatedAt   *time.Time `gorm:"column:created_at;type:datetime;comment:创建时间"`
	UpdatedAt   *time.Time `gorm:"column:updated_at;type:datetime;comment:更新时间"`
}

// TableName 指定表名
func (ScriptLog) TableName() string {
	return "sys_script_logs"
}
