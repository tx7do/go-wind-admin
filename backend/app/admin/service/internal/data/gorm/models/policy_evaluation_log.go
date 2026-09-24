package models

import (
	"github.com/tx7do/go-crud/gorm/mixin"
)

// PolicyEvaluationLog 对应表 sys_policy_evaluation_logs
type PolicyEvaluationLog struct {
	mixin.AutoIncrementID

	UserId            *uint32 `gorm:"column:user_id;type:int unsigned;index;comment:操作者用户ID"`
	MembershipId      *uint32 `gorm:"column:membership_id;type:int unsigned;comment:成员关系ID"`
	PermissionId      *uint32 `gorm:"column:permission_id;type:int unsigned;comment:权限ID"`
	PolicyId          *uint32 `gorm:"column:policy_id;type:int unsigned;comment:策略ID"`
	RequestPath       *string `gorm:"column:request_path;type:varchar(255);comment:请求路径"`
	RequestMethod     *string `gorm:"column:request_method;type:varchar(16);comment:请求方法"`
	Result            *bool   `gorm:"column:result;type:boolean;comment:结果"`
	EffectDetails     *string `gorm:"column:effect_details;type:text;comment:效果详情"`
	ScopeSql          *string `gorm:"column:scope_sql;type:text;comment:范围SQL"`
	IpAddress         *string `gorm:"column:ip_address;type:varchar(255);comment:IP地址"`
	TraceId           *string `gorm:"column:trace_id;type:varchar(255);comment:全局链路追踪ID"`
	EvaluationContext *string `gorm:"column:evaluation_context;type:text;comment:评估上下文"`
	LogHash           *string `gorm:"column:log_hash;type:varchar(255);comment:日志内容哈希（SHA256，十六进制字符串）"`
	Signature         []byte  `gorm:"column:signature;type:longblob;comment:日志数字签名"`

	mixin.CreatedAt
	mixin.TenantID
}

// TableName 指定表名
func (PolicyEvaluationLog) TableName() string {
	return "sys_policy_evaluation_logs"
}
