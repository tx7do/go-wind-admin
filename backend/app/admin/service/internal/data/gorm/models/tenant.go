package models

import (
	"time"

	"github.com/tx7do/go-crud/gorm/mixin"
)

// Tenant 对应表 sys_tenants，租户表
type Tenant struct {
	mixin.AutoIncrementID64

	Name             *string    `gorm:"column:name;type:varchar(255);comment:租户名称;uniqueIndex:idx_sys_tenant_name"`
	Code             *string    `gorm:"column:code;type:varchar(255);comment:租户编号;uniqueIndex:idx_sys_tenant_code"`
	LogoURL          *string    `gorm:"column:logo_url;type:varchar(255);comment:租户logo地址"`
	Domain           *string    `gorm:"column:domain;type:varchar(255);comment:租户域名"`
	Industry         *string    `gorm:"column:industry;type:varchar(255);comment:所属行业"`
	AdminUserID      *uint64    `gorm:"column:admin_user_id;type:int unsigned;comment:管理员用户ID;index:idx_sys_tenant_admin_user_id"`
	Status           *string    `gorm:"column:status;type:varchar(128);comment:租户状态;index:idx_sys_tenant_status_audit_status,priority:1"`
	Type             *string    `gorm:"column:type;type:varchar(128);comment:租户类型"`
	AuditStatus      *string    `gorm:"column:audit_status;type:varchar(128);comment:审核状态;index:idx_sys_tenant_status_audit_status,priority:2"`
	SubscriptionAt   *time.Time `gorm:"column:subscription_at;type:datetime;comment:订阅时间"`
	UnsubscribeAt    *time.Time `gorm:"column:unsubscribe_at;type:datetime;comment:取消订阅时间"`
	SubscriptionPlan *string    `gorm:"column:subscription_plan;type:varchar(255);comment:订阅套餐"`
	ExpiredAt        *time.Time `gorm:"column:expired_at;type:datetime;comment:租户有效期;index:idx_sys_tenant_expired_at"`

	mixin.TimeAt
	mixin.OperatorID64
	mixin.Remark
}

// TableName 指定表名
func (Tenant) TableName() string {
	return "sys_tenants"
}
