package task

// AuditLogWriteTaskType 是审计日志异步写任务的类型常量。
// rest 中间件的审计回调不再同步落库：先把 DTO 以 protojson 形态装进本载荷入队
// （入队失败或 asynq 未配置时回退同步写，宁可慢不可丢），asynq 消费侧按 kind
// 解回 DTO 落对应审计表。at-least-once：消费侧落库成功但 ack 失败的极小概率
// 重复被接受（审计场景宁可重复不可丢失）。
const AuditLogWriteTaskType = "audit_log_write"

// 审计日志种类（payload.kind 取值）。
const (
	AuditLogKindApi              = "api"
	AuditLogKindLogin            = "login"
	AuditLogKindOperation        = "operation"
	AuditLogKindPermission       = "permission"
	AuditLogKindDataAccess       = "data_access"
	AuditLogKindPolicyEvaluation = "policy_evaluation"
)

// AuditLogWriteTaskData 审计日志异步写任务的载荷。
type AuditLogWriteTaskData struct {
	// Kind 审计种类（上方常量之一），消费侧据此路由到对应审计表。
	Kind string `json:"kind"`
	// OccurredAt 事件时间（producer 侧 RFC3339Nano）——队列延迟后落库时
	// api/login 两类以此覆写 CreatedAt，保留事件时刻语义。
	OccurredAt string `json:"occurredAt"`
	// Data protojson 编码的审计 DTO（键为 proto json_name）。
	Data map[string]any `json:"data"`
}
