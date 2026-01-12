package models

import (
	"github.com/tx7do/go-crud/gorm/mixin"
)

// ApiAuditLog 对应表 sys_api_audit_logs
type ApiAuditLog struct {
	mixin.AutoIncrementID

	RequestID      *string  `gorm:"column:request_id;type:varchar(128);comment:请求ID"`
	Method         *string  `gorm:"column:method;type:varchar(16);comment:请求方法"`
	Operation      *string  `gorm:"column:operation;type:varchar(128);comment:操作方法"`
	Path           *string  `gorm:"column:path;type:varchar(255);comment:请求路径"`
	Referer        *string  `gorm:"column:referer;type:varchar(255);comment:请求源"`
	RequestURI     *string  `gorm:"column:request_uri;type:varchar(255);comment:请求URI"`
	RequestBody    *string  `gorm:"column:request_body;type:text;comment:请求体"`
	RequestHeader  *string  `gorm:"column:request_header;type:text;comment:请求头"`
	Response       *string  `gorm:"column:response;type:text;comment:响应信息"`
	CostTime       *float64 `gorm:"column:cost_time;type:double;comment:操作耗时"`
	UserID         *uint32  `gorm:"column:user_id;type:int unsigned;comment:操作者用户ID"`
	Username       *string  `gorm:"column:username;type:varchar(128);comment:操作者账号名"`
	ClientIP       *string  `gorm:"column:client_ip;type:varchar(64);comment:操作者IP"`
	StatusCode     *int32   `gorm:"column:status_code;type:int;comment:状态码"`
	Reason         *string  `gorm:"column:reason;type:varchar(255);comment:操作失败原因"`
	Success        *bool    `gorm:"column:success;type:tinyint(1);comment:操作成功"`
	Location       *string  `gorm:"column:location;type:varchar(255);comment:操作地理位置"`
	UserAgent      *string  `gorm:"column:user_agent;type:text;comment:浏览器的用户代理信息"`
	BrowserName    *string  `gorm:"column:browser_name;type:varchar(128);comment:浏览器名称"`
	BrowserVersion *string  `gorm:"column:browser_version;type:varchar(128);comment:浏览器版本"`
	ClientID       *string  `gorm:"column:client_id;type:varchar(128);comment:客户端ID"`
	ClientName     *string  `gorm:"column:client_name;type:varchar(128);comment:客户端名称"`
	OSName         *string  `gorm:"column:os_name;type:varchar(128);comment:操作系统名称"`
	OSVersion      *string  `gorm:"column:os_version;type:varchar(128);comment:操作系统版本"`

	mixin.CreatedAt
}

// TableName 指定表名
func (ApiAuditLog) TableName() string {
	return "sys_api_audit_logs"
}
