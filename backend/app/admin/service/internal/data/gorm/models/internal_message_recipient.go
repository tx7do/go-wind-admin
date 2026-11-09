package models

import (
	"time"

	"github.com/tx7do/go-crud/gorm/mixin"
)

// InternalMessageRecipient 对应表 internal_message_recipients
type InternalMessageRecipient struct {
	mixin.AutoIncrementID64

	MessageID       *uint64    `gorm:"column:message_id;type:int unsigned;comment:站内信内容ID"`
	RecipientUserID *uint64    `gorm:"column:recipient_user_id;type:int unsigned;comment:接收者用户ID"`
	Status          *string    `gorm:"column:status;type:varchar(32);comment:消息状态"`
	ReceivedAt      *time.Time `gorm:"column:received_at;type:datetime;comment:消息到达用户收件箱的时间"`
	ReadAt          *time.Time `gorm:"column:read_at;type:datetime;comment:用户阅读消息的时间"`

	mixin.TimeAt
	mixin.TenantID64
}

// TableName 指定表名
func (InternalMessageRecipient) TableName() string {
	return "internal_message_recipients"
}
