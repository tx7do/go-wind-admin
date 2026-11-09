package models

import (
	"time"

	"github.com/tx7do/go-crud/gorm/mixin"
	"gorm.io/datatypes"
)

// OrgUnit 对应表 sys_org_units
type OrgUnit struct {
	mixin.AutoIncrementID64
	mixin.Tree64[OrgUnit]

	Name               *string         `gorm:"column:name;type:varchar(255);comment:名称"`
	Code               *string         `gorm:"column:code;type:varchar(128);comment:唯一编码（可用于导入/识别）"`
	LeaderId           *uint64         `gorm:"column:leader_id;type:int unsigned;comment:负责人用户ID"`
	Type               *string         `gorm:"column:type;type:varchar(128);comment:组织类型"`
	BusinessScopes     *datatypes.JSON `gorm:"column:business_scopes;type:json;comment:组织的业务范围/服务条线"`
	ExternalId         *string         `gorm:"column:external_id;type:varchar(255);comment:外部系统ID"`
	IsLegalEntity      *bool           `gorm:"column:is_legal_entity;type:tinyint(1);comment:是否为法定主体"`
	RegistrationNumber *string         `gorm:"column:registration_number;type:varchar(255);comment:注册号/统一社会信用代码"`
	TaxId              *string         `gorm:"column:tax_id;type:varchar(255);comment:税号"`
	LegalEntityOrgId   *uint64         `gorm:"column:legal_entity_org_id;type:int unsigned;comment:关联的法定主体组织ID"`
	Address            *string         `gorm:"column:address;type:varchar(255);comment:详细地址"`
	Phone              *string         `gorm:"column:phone;type:varchar(255);comment:联系电话"`
	Email              *string         `gorm:"column:email;type:varchar(255);comment:联系邮箱"`
	Timezone           *string         `gorm:"column:timezone;type:varchar(255);comment:时区"`
	Country            *string         `gorm:"column:country;type:varchar(255);comment:国家/地区代码"`
	Latitude           *float64        `gorm:"column:latitude;type:double;comment:纬度"`
	Longitude          *float64        `gorm:"column:longitude;type:double;comment:经度"`
	StartAt            *time.Time      `gorm:"column:start_at;type:datetime;comment:生效时间（UTC）"`
	EndAt              *time.Time      `gorm:"column:end_at;type:datetime;comment:结束有效期（UTC）"`
	ContactUserId      *uint64         `gorm:"column:contact_user_id;type:int unsigned;comment:业务联系人用户ID"`
	PermissionTags     *datatypes.JSON `gorm:"column:permission_tags;type:json;comment:与权限/角色映射的标签"`
	Path               *string         `gorm:"column:path;type:varchar(512);comment:树路径，规范： 根节点: /，非根节点: /1/2/3/（以 / 开头且以 / 结尾）。禁止空字符串（NULL 表示未设置）。"`

	mixin.TimeAt
	mixin.OperatorID64
	mixin.SwitchStatus
	mixin.SortOrder
	mixin.TenantID64
	mixin.Remark
	mixin.Description
}

// TableName 指定表名
func (OrgUnit) TableName() string {
	return "sys_org_units"
}
