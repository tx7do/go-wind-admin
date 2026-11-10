package models

import (
	"github.com/tx7do/go-crud/gorm/mixin"
	"gorm.io/datatypes"
)

// DataAccessAuditLog 对应表 sys_data_access_audit_logs
type DataAccessAuditLog struct {
	mixin.AutoIncrementID64

	UserId          *int64         `gorm:"column:user_id;type:int unsigned;index;comment:操作者用户ID"`
	Username        *string         `gorm:"column:username;type:varchar(255);comment:操作者账号名"`
	IpAddress       *string         `gorm:"column:ip_address;type:varchar(255);comment:IP地址"`
	GeoLocation     *datatypes.JSON `gorm:"column:geo_location;type:json;comment:地理位置(来自IP库)"`
	DeviceInfo      *datatypes.JSON `gorm:"column:device_info;type:json;comment:设备信息"`
	RequestId       *string         `gorm:"column:request_id;type:varchar(255);comment:请求ID"`
	TraceId         *string         `gorm:"column:trace_id;type:varchar(255);comment:全局链路追踪ID"`
	DataSource      *string         `gorm:"column:data_source;type:varchar(255);comment:数据源"`
	TableNameCol    *string         `gorm:"column:table_name;type:varchar(255);comment:表名"`
	DataId          *string         `gorm:"column:data_id;type:varchar(255);comment:数据ID"`
	AccessType      *string         `gorm:"column:access_type;type:varchar(128);comment:访问类型"`
	SqlDigest       *string         `gorm:"column:sql_digest;type:varchar(255);comment:SQL摘要"`
	SqlText         *string         `gorm:"column:sql_text;type:text;comment:SQL文本"`
	AffectedRows    *uint32         `gorm:"column:affected_rows;type:int unsigned;comment:受影响行数"`
	LatencyMs       *uint32         `gorm:"column:latency_ms;type:int unsigned;comment:操作耗时"`
	Success         *bool           `gorm:"column:success;type:boolean;comment:操作结果"`
	SensitiveLevel  *string         `gorm:"column:sensitive_level;type:varchar(128);comment:数据敏感级别"`
	DataMasked      *bool           `gorm:"column:data_masked;type:boolean;comment:是否脱敏"`
	MaskingRules    *string         `gorm:"column:masking_rules;type:text;comment:脱敏规则"`
	BusinessPurpose *string         `gorm:"column:business_purpose;type:varchar(255);comment:业务目的"`
	DataCategory    *string         `gorm:"column:data_category;type:varchar(255);comment:数据分类"`
	DbUser          *string         `gorm:"column:db_user;type:varchar(255);comment:数据库用户"`
	LogHash         *string         `gorm:"column:log_hash;type:varchar(255);comment:日志内容哈希（SHA256，十六进制字符串）"`
	Signature       []byte          `gorm:"column:signature;type:longblob;comment:日志数字签名"`

	mixin.CreatedAt
	mixin.TenantID64
}

// TableName 指定表名
func (DataAccessAuditLog) TableName() string {
	return "sys_data_access_audit_logs"
}
