package models

import (
	"github.com/tx7do/go-crud/gorm/mixin"
	"gorm.io/datatypes"
)

// LoginAuditLog 对应表 sys_login_audit_logs
type LoginAuditLog struct {
	mixin.AutoIncrementID

	UserId        *uint32         `gorm:"column:user_id;type:int unsigned;index;comment:操作者用户ID"`
	Username      *string         `gorm:"column:username;type:varchar(255);comment:操作者账号名"`
	IpAddress     *string         `gorm:"column:ip_address;type:varchar(255);comment:IP地址"`
	GeoLocation   *datatypes.JSON `gorm:"column:geo_location;type:json;comment:地理位置(来自IP库)"`
	SessionId     *string         `gorm:"column:session_id;type:varchar(255);comment:会话ID"`
	DeviceInfo    *datatypes.JSON `gorm:"column:device_info;type:json;comment:设备信息"`
	RequestId     *string         `gorm:"column:request_id;type:varchar(255);comment:请求ID"`
	TraceId       *string         `gorm:"column:trace_id;type:varchar(255);comment:全局链路追踪ID"`
	ActionType    *string         `gorm:"column:action_type;type:varchar(128);comment:动作类型"`
	Status        *string         `gorm:"column:status;type:varchar(10);comment:状态"`
	LoginMethod   *string         `gorm:"column:login_method;type:varchar(128);comment:登录方式"`
	FailureReason *string         `gorm:"column:failure_reason;type:varchar(255);comment:失败原因"`
	MfaStatus     *string         `gorm:"column:mfa_status;type:varchar(255);comment:MFA状态"`
	RiskScore     *uint32         `gorm:"column:risk_score;type:int unsigned;comment:风险评分"`
	RiskLevel     *string         `gorm:"column:risk_level;type:varchar(128);comment:风险等级"`
	RiskFactors   *string         `gorm:"column:risk_factors;type:text;comment:风险因子"`
	LogHash       *string         `gorm:"column:log_hash;type:varchar(255);comment:日志内容哈希（SHA256，十六进制字符串）"`
	Signature     []byte          `gorm:"column:signature;type:longblob;comment:日志数字签名"`

	mixin.CreatedAt
	mixin.TenantID
}

// TableName 指定表名
func (LoginAuditLog) TableName() string {
	return "sys_login_audit_logs"
}
