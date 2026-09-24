package models

import (
	"time"

	"github.com/tx7do/go-crud/gorm/mixin"
)

// Position 对应表 sys_positions
type Position struct {
	mixin.AutoIncrementID

	Name                *string    `gorm:"column:name;type:varchar(255);comment:职位名称"`
	Code                *string    `gorm:"column:code;type:varchar(128);comment:唯一编码"`
	OrgUnitId           *uint32    `gorm:"column:org_unit_id;type:int unsigned;comment:所属组织单元ID"`
	ReportsToPositionId *uint32    `gorm:"column:reports_to_position_id;type:int unsigned;comment:汇报对象职位ID"`
	Description         *string    `gorm:"column:description;type:varchar(1024);comment:职能描述"`
	JobFamily           *string    `gorm:"column:job_family;type:varchar(255);comment:职位族"`
	JobGrade            *string    `gorm:"column:job_grade;type:varchar(255);comment:职级"`
	Level               *int32     `gorm:"column:level;type:int;comment:层级"`
	Headcount           *uint32    `gorm:"column:headcount;type:int unsigned;comment:编制人数"`
	IsKeyPosition       *bool      `gorm:"column:is_key_position;type:boolean;comment:是否关键职位"`
	Type                *string    `gorm:"column:type;type:varchar(128);comment:类型"`
	StartAt             *time.Time `gorm:"column:start_at;type:datetime;comment:生效时间（UTC）"`
	EndAt               *time.Time `gorm:"column:end_at;type:datetime;comment:结束有效期（UTC）"`

	mixin.TimeAt
	mixin.OperatorID
	mixin.SortOrder
	mixin.Remark
	mixin.TenantID
	mixin.SwitchStatus
}

// TableName 指定表名
func (Position) TableName() string {
	return "sys_positions"
}
