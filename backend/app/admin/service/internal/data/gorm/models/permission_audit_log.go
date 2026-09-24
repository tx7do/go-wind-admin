package models

import (
	"github.com/tx7do/go-crud/gorm/mixin"
)

// PermissionAuditLog 对应表 sys_permission_audit_logs
type PermissionAuditLog struct {
	mixin.AutoIncrementID

	OperatorId *uint32 `gorm:"column:operator_id;type:int unsigned;index;comment:操作者用户ID"`
	TargetType *string `gorm:"column:target_type;type:varchar(255);comment:目标类型"`
	TargetId   *string `gorm:"column:target_id;type:varchar(255);comment:目标ID"`
	Action     *string `gorm:"column:action;type:varchar(128);comment:动作"`
	OldValue   *string `gorm:"column:old_value;type:text;comment:旧值"`
	NewValue   *string `gorm:"column:new_value;type:text;comment:新值"`
	IpAddress  *string `gorm:"column:ip_address;type:varchar(255);comment:IP地址"`
	RequestId  *string `gorm:"column:request_id;type:varchar(255);comment:请求ID"`
	Reason     *string `gorm:"column:reason;type:varchar(255);comment:原因"`
	LogHash    *string `gorm:"column:log_hash;type:varchar(255);comment:日志内容哈希（SHA256，十六进制字符串）"`
	Signature  []byte  `gorm:"column:signature;type:longblob;comment:日志数字签名"`

	mixin.CreatedAt
	mixin.TenantID
}

// TableName 指定表名
func (PermissionAuditLog) TableName() string {
	return "sys_permission_audit_logs"
}
