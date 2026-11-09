package models

import (
	"github.com/tx7do/go-crud/gorm/mixin"
)

// DictEntryI18n 对应表 sys_dict_entry_i18n
type DictEntryI18n struct {
	mixin.AutoIncrementID64

	LanguageCode *string `gorm:"column:language_code;type:varchar(128);comment:标准语言代码"`
	EntryLabel   *string `gorm:"column:entry_label;type:varchar(255);comment:字典项标签"`

	mixin.TimeAt
	mixin.OperatorID64
	mixin.Description
	mixin.SortOrder
	mixin.TenantID64
}

// TableName 指定表名
func (DictEntryI18n) TableName() string {
	return "sys_dict_entry_i18n"
}
