package models

import (
	"github.com/tx7do/go-crud/gorm/mixin"
	"gorm.io/datatypes"
)

// ApiAuditLog 对应表 sys_api_audit_logs
type ApiAuditLog struct {
	mixin.AutoIncrementID

	UserId         *uint32         `gorm:"column:user_id;type:int unsigned;index;comment:操作者用户ID"`
	Username       *string         `gorm:"column:username;type:varchar(255);comment:操作者账号名"`
	IpAddress      *string         `gorm:"column:ip_address;type:varchar(255);comment:IP地址"`
	GeoLocation    *datatypes.JSON `gorm:"column:geo_location;type:json;comment:地理位置(来自IP库)"`
	DeviceInfo     *datatypes.JSON `gorm:"column:device_info;type:json;comment:设备信息"`
	Referer        *string         `gorm:"column:referer;type:varchar(255);comment:请求来源URL"`
	AppVersion     *string         `gorm:"column:app_version;type:varchar(255);comment:客户端版本号"`
	HttpMethod     *string         `gorm:"column:http_method;type:varchar(16);comment:HTTP请求方法"`
	Path           *string         `gorm:"column:path;type:varchar(255);comment:请求路径"`
	RequestUri     *string         `gorm:"column:request_uri;type:varchar(255);comment:完整请求URI"`
	ApiModule      *string         `gorm:"column:api_module;type:varchar(128);comment:API所属业务模块"`
	ApiOperation   *string         `gorm:"column:api_operation;type:varchar(128);comment:API业务操作"`
	ApiDescription *string         `gorm:"column:api_description;type:varchar(255);comment:API功能描述"`
	RequestId      *string         `gorm:"column:request_id;type:varchar(255);comment:请求ID"`
	TraceId        *string         `gorm:"column:trace_id;type:varchar(255);comment:全局链路追踪ID"`
	SpanId         *string         `gorm:"column:span_id;type:varchar(255);comment:当前跨度ID"`
	LatencyMs      *uint32         `gorm:"column:latency_ms;type:int unsigned;comment:操作耗时"`
	Success        *bool           `gorm:"column:success;type:boolean;comment:操作结果"`
	StatusCode     *uint32         `gorm:"column:status_code;type:int unsigned;comment:HTTP状态码"`
	Reason         *string         `gorm:"column:reason;type:varchar(255);comment:操作失败原因"`
	RequestHeader  *string         `gorm:"column:request_header;type:text;comment:请求头"`
	RequestBody    *string         `gorm:"column:request_body;type:text;comment:请求体"`
	Response       *string         `gorm:"column:response;type:text;comment:响应信息"`
	LogHash        *string         `gorm:"column:log_hash;type:varchar(255);comment:日志内容哈希（SHA256，十六进制字符串）"`
	Signature      []byte          `gorm:"column:signature;type:longblob;comment:日志数字签名"`

	mixin.CreatedAt
	mixin.TenantID
}

// TableName 指定表名
func (ApiAuditLog) TableName() string {
	return "sys_api_audit_logs"
}
