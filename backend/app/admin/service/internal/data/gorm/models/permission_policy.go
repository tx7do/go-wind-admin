package models

import (
	"github.com/tx7do/go-crud/gorm/mixin"
)

// PermissionPolicy 对应表 sys_permission_policies
type PermissionPolicy struct {
	mixin.AutoIncrementID64

	PermissionId *int64 `gorm:"column:permission_id;type:int unsigned;comment:权限ID"`
	PolicyEngine *string `gorm:"column:policy_engine;type:varchar(128);comment:策略引擎"`
	Definition   *string `gorm:"column:definition;type:text;comment:策略定义"`
	Version      *uint32 `gorm:"column:version;type:int unsigned;comment:版本"`
	EvalOrder    *uint32 `gorm:"column:eval_order;type:int unsigned;comment:评估顺序"`
	CacheTtl     *uint32 `gorm:"column:cache_ttl;type:int unsigned;comment:缓存TTL"`

	mixin.TimeAt
	mixin.OperatorID64
	mixin.SwitchStatus
}

// TableName 指定表名
func (PermissionPolicy) TableName() string {
	return "sys_permission_policies"
}
