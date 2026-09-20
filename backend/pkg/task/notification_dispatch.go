package task

// NotificationDispatchTaskType 是通知异步派发任务的类型常量。
// 一次性投递任务（非周期），由 NotificationService.SendDirect 在台账落 SENDING 之后入队，
// handler 为 NotificationService.AsyncNotificationDispatch。
//
// 异步化只针对"外呼慢且无需即时结论"的事件（找回密码/换绑验证码的 SMTP 拨号）：
// 请求不再等 SMTP 往返，慢邮箱服务器拖不住 HTTP。路由与台账语义不变。
const NotificationDispatchTaskType = "notification_dispatch"

// NotificationDispatchTaskData 通知派发任务的载荷。
//
// 为什么把 target/title/content 带进载荷、其余字段（渠道、事件、收件人、操作人）从台账读回：
// 台账的 target 是脱敏后的（`a***@x.com`），且压根没有正文列 —— 正文只活在内存里，
// handler 取不回来。异步化若不带上正文，就只能给台账加明文正文列，那是更糟的取舍
// （台账是管理员可见的全平台投递清单）。
//
// 代价写清楚：**验证码明文会在 Redis（asynq 队列）里停留到投递完成**。缓解是三条，
// 都在 NotificationService 里：入队前先做渠道可用性预检（配置不对根本不入队）、
// 重试上限 3 次、配置类错误返回 asynq.SkipRetry（不进重试/归档）。
// 载荷里只放投递所需的三元组，收件用户 ID 之类的身份字段一律从台账行读，
// 减少 Redis 里的可关联信息。
type NotificationDispatchTaskData struct {
	// DeliveryId 台账行主键（sys_notification_deliveries.id）：重试幂等的锚，
	// 也是"这一行现在到底是什么状态"的唯一权威来源。
	DeliveryId uint32 `json:"delivery_id"`
	Target     string `json:"target"`
	Title      string `json:"title"`
	Content    string `json:"content"`
}
