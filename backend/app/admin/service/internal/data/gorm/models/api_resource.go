package models

import "github.com/tx7do/go-crud/gorm/mixin"

// Api 对应表 sys_apis
type Api struct {
	mixin.AutoIncrementID

	Description       *string `gorm:"column:description;type:varchar(255);comment:描述"`
	Module            *string `gorm:"column:module;type:varchar(128);comment:所属业务模块"`
	ModuleDescription *string `gorm:"column:module_description;type:varchar(255);comment:业务模块描述"`
	Operation         *string `gorm:"column:operation;type:varchar(128);comment:接口操作名"`
	Path              *string `gorm:"column:path;type:varchar(255);comment:接口路径"`
	Method            *string `gorm:"column:method;type:varchar(16);comment:请求方法"`
	Scope             *string `gorm:"column:scope;type:varchar(32);default:ADMIN;comment:作用域"`

	mixin.TimeAt
	mixin.OperatorID
}

// TableName 指定表名
func (Api) TableName() string {
	return "sys_apis"
}
