package models

import "github.com/tx7do/go-crud/gorm/mixin"

// Language 对应表 sys_languages
type Language struct {
	mixin.AutoIncrementID64

	LanguageCode *string `gorm:"column:language_code;type:varchar(128);comment:标准语言代码"`
	LanguageName *string `gorm:"column:language_name;type:varchar(255);comment:语言名称"`
	NativeName   *string `gorm:"column:native_name;type:varchar(255);comment:本地语言名称"`
	IsDefault    *bool   `gorm:"column:is_default;type:tinyint(1);default:0;comment:是否为默认语言"`

	mixin.TimeAt
	mixin.OperatorID64
	mixin.SortOrder
	mixin.IsEnabled
}

// TableName 指定表名
func (Language) TableName() string {
	return "sys_languages"
}
