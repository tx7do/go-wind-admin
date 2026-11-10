package models

import (
	"time"

	"github.com/tx7do/go-crud/gorm/mixin"
)

// MembershipOrgUnit 对应表 sys_membership_org_units
type MembershipOrgUnit struct {
	mixin.AutoIncrementID64

	MembershipId *int64    `gorm:"column:membership_id;type:int unsigned;comment:成员关系ID"`
	OrgUnitId    *int64    `gorm:"column:org_unit_id;type:int unsigned;comment:组织单元ID"`
	PositionId   *int64    `gorm:"column:position_id;type:int unsigned;comment:职位ID"`
	RoleId       *int64    `gorm:"column:role_id;type:int unsigned;comment:角色ID"`
	StartAt      *time.Time `gorm:"column:start_at;type:datetime;comment:生效时间（UTC）"`
	EndAt        *time.Time `gorm:"column:end_at;type:datetime;comment:结束有效期（UTC）"`
	AssignedAt   *time.Time `gorm:"column:assigned_at;type:datetime;comment:分配时间"`
	AssignedBy   *int64    `gorm:"column:assigned_by;type:int unsigned;comment:分配人ID"`
	IsPrimary    *bool      `gorm:"column:is_primary;type:boolean;comment:是否主成员关系"`
	Status       *string    `gorm:"column:status;type:varchar(10);comment:状态"`

	mixin.TimeAt
	mixin.OperatorID64
	mixin.TenantID64
	mixin.Remark
}

// TableName 指定表名
func (MembershipOrgUnit) TableName() string {
	return "sys_membership_org_units"
}
