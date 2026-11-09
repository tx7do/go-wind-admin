package models

import (
	"github.com/tx7do/go-crud/gorm/mixin"
	"gorm.io/datatypes"
)

// OperationAuditLog 对应表 sys_operation_audit_logs
type OperationAuditLog struct {
	mixin.AutoIncrementID64

	UserId         *uint64         `gorm:"column:user_id;type:int unsigned;index;comment:操作者用户ID"`
	Username       *string         `gorm:"column:username;type:varchar(255);comment:操作者账号名"`
	ResourceType   *string         `gorm:"column:resource_type;type:varchar(255);comment:资源类型"`
	ResourceId     *string         `gorm:"column:resource_id;type:varchar(255);comment:资源ID"`
	Action         *string         `gorm:"column:action;type:varchar(128);comment:动作"`
	BeforeData     *string         `gorm:"column:before_data;type:json;comment:操作前数据"`
	AfterData      *string         `gorm:"column:after_data;type:json;comment:操作后数据"`
	SensitiveLevel *string         `gorm:"column:sensitive_level;type:varchar(128);comment:数据敏感级别"`
	RequestId      *string         `gorm:"column:request_id;type:varchar(255);comment:全局请求ID"`
	TraceId        *string         `gorm:"column:trace_id;type:varchar(255);comment:全局链路追踪ID"`
	Success        *bool           `gorm:"column:success;type:boolean;comment:操作结果"`
	FailureReason  *string         `gorm:"column:failure_reason;type:varchar(255);comment:失败原因"`
	IpAddress      *string         `gorm:"column:ip_address;type:varchar(255);comment:IP地址"`
	GeoLocation    *datatypes.JSON `gorm:"column:geo_location;type:json;comment:地理位置(来自IP库)"`
	DeviceInfo     *datatypes.JSON `gorm:"column:device_info;type:json;comment:设备信息"`
	LogHash        *string         `gorm:"column:log_hash;type:varchar(255);comment:日志内容哈希（SHA256，十六进制字符串）"`
	Signature      []byte          `gorm:"column:signature;type:longblob;comment:日志数字签名"`

	mixin.CreatedAt
	mixin.TenantID64
}

// TableName 指定表名
func (OperationAuditLog) TableName() string {
	return "sys_operation_audit_logs"
}
