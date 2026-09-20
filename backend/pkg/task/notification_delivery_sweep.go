package task

// NotificationDeliverySweepTaskType 是通知台账超时清扫任务的类型常量。
// 该任务为系统级常驻定时任务（不写入 sys_tasks 表），由 asynq 调度器按固定 cron 周期触发，
// handler 为 NotificationService.AsyncDeliverySweep。
//
// 它补的是异步派发留下的最后一个洞：台账先落 SENDING 再投递，所以"进程死在拨号中途"
// "结论回写失败""队列压根没消费者"这三种情况都会留下一行永远停在 SENDING 的记录，
// 而代码里此前没有任何地方会把非终态行推向终态（四个 SENDING 写入点全是"开始"，没有"收尸"）。
const NotificationDeliverySweepTaskType = "notification_delivery_sweep"

// NotificationDeliverySweepCronSpec 是台账清扫的系统级 cron 表达式（每 5 分钟）。
// 周期只决定"最坏多久被结算"，真正的判据是 NotificationService 里的超期阈值。
const NotificationDeliverySweepCronSpec = "*/5 * * * *"

// NotificationDeliverySweepTaskData 台账清扫任务的载荷（当前无参数；阈值走环境变量，
// 不放载荷是因为周期任务的载荷在建调度项时就固定了，改了也得重启才生效，不如环境变量）。
type NotificationDeliverySweepTaskData struct{}
