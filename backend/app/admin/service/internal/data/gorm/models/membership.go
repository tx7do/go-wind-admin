package models

import (
	"time"

	"github.com/tx7do/go-crud/gorm/mixin"
)

// Membership 对应表 sys_memberships
type Membership struct {
	mixin.AutoIncrementID

	UserId     *uint32    `gorm:"column:user_id;type:int unsigned;comment:用户ID"`
	OrgUnitId  *uint32    `gorm:"column:org_unit_id;type:int unsigned;comment:组织单元ID"`
	PositionId *uint32    `gorm:"column:position_id;type:int unsigned;comment:职位ID"`
	RoleId     *uint32    `gorm:"column:role_id;type:int unsigned;comment:角色ID"`
	IsPrimary  *bool      `gorm:"column:is_primary;type:boolean;comment:是否主成员关系"`
	StartAt    *time.Time `gorm:"column:start_at;type:datetime;comment:生效时间（UTC）"`
	EndAt      *time.Time `gorm:"column:end_at;type:datetime;comment:结束有效期（UTC）"`
	AssignedAt *time.Time `gorm:"column:assigned_at;type:datetime;comment:分配时间"`
	AssignedBy *uint32    `gorm:"column:assigned_by;type:int unsigned;comment:分配人ID"`
	JoinedAt   *time.Time `gorm:"column:joined_at;type:datetime;comment:加入时间"`
	Status     *string    `gorm:"column:status;type:varchar(10);comment:状态"`

	mixin.TimeAt
	mixin.OperatorID
	mixin.TenantID
	mixin.Remark
}

// TableName 指定表名
func (Membership) TableName() string {
	return "sys_memberships"
}
