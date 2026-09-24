package models

import (
	"time"

	"github.com/tx7do/go-crud/gorm/mixin"
)

// UserMfaFactor 对应表 sys_user_mfa_factors，用户 MFA 因子表
type UserMfaFactor struct {
	mixin.AutoIncrementID

	UserID      *uint32    `gorm:"column:user_id;type:int unsigned;comment:关联主表的用户ID;index:idx_sys_user_mfa_factor_user_id;uniqueIndex:idx_sys_user_mfa_factor_uid_method,priority:1"`
	Method      *string    `gorm:"column:method;type:varchar(128);comment:MFA 方法;uniqueIndex:idx_sys_user_mfa_factor_uid_method,priority:2"`
	SecretHash  *string    `gorm:"column:secret_hash;type:varchar(512);comment:MFA secret 密文（AES-GCM 加密，base64 编码；TOTP 校验需还原明文）"`
	DisplayName *string    `gorm:"column:display_name;type:varchar(128);comment:设备/因子展示名（用户自定义）"`
	Status      *string    `gorm:"column:status;type:varchar(128);comment:因子状态"`
	LastUsedAt  *time.Time `gorm:"column:last_used_at;type:datetime;comment:最近一次用于验证的时间"`

	mixin.TimeAt
	mixin.TenantID
}

// TableName 指定表名
func (UserMfaFactor) TableName() string {
	return "sys_user_mfa_factors"
}
