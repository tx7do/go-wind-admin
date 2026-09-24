package models

import (
	"time"

	"github.com/tx7do/go-crud/gorm/mixin"
)

// AccessKey 对应表 sys_access_keys - OpenAPI 访问凭证（AK/SK）：租户级机器凭证，secret 仅存 SHA-256 摘要
type AccessKey struct {
	mixin.AutoIncrementID

	Name       *string    `gorm:"column:name;type:varchar(255);comment:凭证名称（用途说明）"`
	AccessKey  *string    `gorm:"column:access_key;type:varchar(255);uniqueIndex;comment:访问键（AK，公开标识）"`
	SecretHash *string    `gorm:"column:secret_hash;type:char(64);comment:密钥摘要（SHA-256 hex，明文不落库）"`
	ExpiresAt  *time.Time `gorm:"column:expires_at;type:datetime;comment:过期时间（空表示长期有效）"`
	LastUsedAt *time.Time `gorm:"column:last_used_at;type:datetime;comment:最近一次令牌交换时间"`
	Status     *int8      `gorm:"column:status;type:TINYINT;default:1;comment:状态：0=禁用，1=启用"`
	CreatedBy  *uint32    `gorm:"column:created_by;type:int unsigned;comment:创建者 ID"`
	UpdatedBy  *uint32    `gorm:"column:updated_by;type:int unsigned;comment:更新者 ID"`
	CreatedAt  *time.Time `gorm:"column:created_at;type:datetime;comment:创建时间"`
	UpdatedAt  *time.Time `gorm:"column:updated_at;type:datetime;comment:更新时间"`
	TenantId   *uint32    `gorm:"column:tenant_id;type:int unsigned;comment:租户 ID；多租户隔离键"`
	SortOrder  *int       `gorm:"column:sort_order;type:int;default:0;comment:排序顺序；数字越小越靠前"`
	Remark     *string    `gorm:"column:remark;type:varchar(1024);comment:备注"`
}

// TableName 指定表名
func (AccessKey) TableName() string {
	return "sys_access_keys"
}
