package models

import (
	"time"

	"gorm.io/datatypes"

	"github.com/tx7do/go-crud/gorm/mixin"
)

// UserCredential 对应表 sys_user_credentials，用户认证信息表
type UserCredential struct {
	mixin.AutoIncrementID64

	UserID                 *int64         `gorm:"column:user_id;type:int unsigned;comment:关联主表的用户ID;index:idx_sys_user_credential_user_id;uniqueIndex:idx_sys_user_credential_uid_identity_identifier,priority:1"`
	IdentityType           *string         `gorm:"column:identity_type;type:varchar(128);comment:认证方式类型;index:idx_sys_user_credential_identity_type;uniqueIndex:idx_sys_user_credential_uid_identity_identifier,priority:2"`
	Identifier             *string         `gorm:"column:identifier;type:varchar(255);comment:身份唯一标识符;index:idx_sys_user_credential_identifier;uniqueIndex:idx_sys_user_credential_uid_identity_identifier,priority:3"`
	Credential             *string         `gorm:"column:credential;type:varchar(1024);comment:凭证"`
	IsPrimary              *bool           `gorm:"column:is_primary;type:tinyint(1);comment:是否主认证方式;index:idx_sys_user_credential_is_primary"`
	Status                 *string         `gorm:"column:status;type:varchar(128);comment:凭证状态;index:idx_sys_user_credential_status"`
	ExtraInfo              *datatypes.JSON `gorm:"column:extra_info;type:json;comment:扩展信息"`
	Provider               *string         `gorm:"column:provider;type:varchar(255);comment:第三方平台标识;index:idx_sys_user_credential_provider;uniqueIndex:idx_sys_user_credential_provider_account,priority:1"`
	ProviderAccountID      *string         `gorm:"column:provider_account_id;type:varchar(255);comment:第三方平台的账号唯一ID;uniqueIndex:idx_sys_user_credential_provider_account,priority:2"`
	ActivateTokenHash      *string         `gorm:"column:activate_token_hash;type:varchar(255);comment:激活令牌哈希（不要存明文）"`
	ActivateTokenExpiresAt *time.Time      `gorm:"column:activate_token_expires_at;type:datetime;comment:激活令牌到期时间"`
	ActivateTokenUsedAt    *time.Time      `gorm:"column:activate_token_used_at;type:datetime;comment:激活令牌使用时间，单次使用时记录"`
	ResetTokenHash         *string         `gorm:"column:reset_token_hash;type:varchar(255);comment:重置密码令牌哈希（不要存明文）"`
	ResetTokenExpiresAt    *time.Time      `gorm:"column:reset_token_expires_at;type:datetime;comment:重置令牌到期时间"`
	ResetTokenUsedAt       *time.Time      `gorm:"column:reset_token_used_at;type:datetime;comment:重置令牌使用时间"`

	mixin.TimeAt
	mixin.TenantID64
}

// TableName 指定表名
func (UserCredential) TableName() string {
	return "sys_user_credentials"
}
