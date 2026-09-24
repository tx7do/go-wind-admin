package models

import (
	"time"

	"github.com/tx7do/go-crud/gorm/mixin"
)

// User 对应表 sys_users
type User struct {
	mixin.AutoIncrementID

	Username    *string    `gorm:"column:username;type:varchar(255);comment:用户名;uniqueIndex:idx_sys_user_username"`
	Nickname    *string    `gorm:"column:nickname;type:varchar(255);comment:昵称"`
	Realname    *string    `gorm:"column:realname;type:varchar(255);comment:真实名字"`
	Email       *string    `gorm:"column:email;type:varchar(320);comment:电子邮箱"`
	Mobile      *string    `gorm:"column:mobile;type:varchar(255);comment:手机号码"`
	Telephone   *string    `gorm:"column:telephone;type:varchar(255);comment:座机号码"`
	Avatar      *string    `gorm:"column:avatar;type:varchar(255);comment:头像"`
	Address     *string    `gorm:"column:address;type:varchar(255);comment:地址"`
	Region      *string    `gorm:"column:region;type:varchar(255);comment:国家地区"`
	Description *string    `gorm:"column:description;type:varchar(1023);comment:个人说明"`
	Gender      *string    `gorm:"column:gender;type:varchar(128);comment:性别;index:idx_sys_user_gender"`
	LastLoginAt *time.Time `gorm:"column:last_login_at;type:datetime;comment:最后一次登录的时间"`
	LastLoginIP *string    `gorm:"column:last_login_ip;type:varchar(45);comment:最后一次登录的IP"`
	LockedUntil *time.Time `gorm:"column:locked_until;type:datetime;comment:锁定截止时间"`

	mixin.OperatorID
	mixin.TimeAt
	mixin.Remark
	mixin.TenantID
}

// TableName 指定表名
func (User) TableName() string {
	return "sys_users"
}
