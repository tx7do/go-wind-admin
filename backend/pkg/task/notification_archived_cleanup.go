package task

// NotificationArchivedCleanupTaskType 是通知归档集清理任务的类型常量。
// 该任务为系统级常驻定时任务（不写入 sys_tasks 表），由 asynq 调度器按固定 cron 周期触发，
// handler 为 NotificationService.AsyncArchivedCleanup。
//
// 它收的是异步派发的最后一个尾：重试耗尽的通知派发任务会被 asynq 移进归档集
// （asynq:{<queue>}:archived）后没有任何代码再碰它，而载荷带明文一次性验证码，
// 只能靠周期清理把超保留期的删掉（台账侧的终态由 notification_delivery_sweep 结算，两边不同步是有意的）。
const NotificationArchivedCleanupTaskType = "notification_archived_cleanup"

// NotificationArchivedCleanupCronSpec 是归档清理的系统级 cron 表达式（每小时整点）。
// 归档集堆积不紧急（每任务 MaxRetry 只有 3），但载荷含明文 OTP，不宜留到每日一清。
const NotificationArchivedCleanupCronSpec = "0 * * * *"

// NotificationArchivedCleanupTaskData 归档清理任务的载荷（当前无参数；保留期走环境变量，
// 不放载荷与 notification_delivery_sweep 同理——周期任务的载荷在建调度项时就固定了）。
type NotificationArchivedCleanupTaskData struct{}
