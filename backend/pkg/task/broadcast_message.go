package task

// BroadcastMessageTaskType 是站内信全员广播任务的类型常量。
// 该任务为一次性投递任务（非周期），由 InternalMessageService.SendMessage 在创建消息后入队，
// handler 为 InternalMessageService.AsyncBroadcastMessage。
// 广播逻辑从请求 goroutine 移入 asynq 任务，使进程重启后未完成的投递能自动重试，
// 而非静默丢失；重试幂等性由 (message_id, recipient_user_id) 唯一约束 +
// CreateBulk 的 ON CONFLICT DO NOTHING 保证。
const BroadcastMessageTaskType = "broadcast_message"

// BroadcastMessageTaskData 全员广播任务的载荷。
// 仅携带 messageId：消息本体（title/content）已在 SendMessage 中落库，
// handler 按 id 分页拉取用户后批量插入收件记录，无需在 payload 里携带大字段。
type BroadcastMessageTaskData struct {
	MessageId uint32 `json:"message_id"`
}
