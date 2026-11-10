package models

import "github.com/tx7do/go-crud/gorm/mixin"

// InternalMessage 对应表 internal_messages
type InternalMessage struct {
	mixin.AutoIncrementID64

	Title      *string `gorm:"column:title;type:varchar(255);comment:消息标题"`
	Content    *string `gorm:"column:content;type:text;comment:消息内容"`
	SenderID   *int64 `gorm:"column:sender_id;type:int unsigned;comment:发送者用户ID"`
	CategoryID *int64 `gorm:"column:category_id;type:int unsigned;comment:分类ID"`
	Status     *string `gorm:"column:status;type:varchar(32);default:DRAFT;comment:消息状态"`
	Type       *string `gorm:"column:type;type:varchar(32);default:NOTIFICATION;comment:消息类型"`

	mixin.TimeAt
	mixin.OperatorID64
	mixin.TenantID64
}

// TableName 指定表名
func (InternalMessage) TableName() string {
	return "internal_messages"
}
