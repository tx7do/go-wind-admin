package models

import (
	"github.com/tx7do/go-crud/gorm/mixin"
)

// DictEntry 对应表 sys_dict_entries
type DictEntry struct {
	mixin.AutoIncrementID64

	EntryValue   *string `gorm:"column:entry_value;type:varchar(255);comment:字典项的实际值"`
	NumericValue *int32  `gorm:"column:numeric_value;type:int;comment:数值型值"`
	TypeID       *uint64 `gorm:"column:type_id;type:int unsigned;comment:所属字典类型ID"`

	DictType *DictType `gorm:"foreignKey:TypeID;references:ID"`

	mixin.TimeAt
	mixin.OperatorID64
	mixin.SortOrder
	mixin.IsEnabled
	mixin.TenantID64
}

// TableName 指定表名
func (DictEntry) TableName() string {
	return "sys_dict_entries"
}
