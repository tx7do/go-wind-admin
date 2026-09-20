package server

import (
	"errors"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bootstrapAsynq "github.com/tx7do/kratos-bootstrap/transport/asynq"
	asynqServer "github.com/tx7do/kratos-transport/transport/asynq"

	"go-wind-admin/app/admin/service/internal/service"

	appViewer "go-wind-admin/pkg/entgo/viewer"
	"go-wind-admin/pkg/task"
)

// NewAsynqServer creates a new asynq server.
func NewAsynqServer(ctx *bootstrap.Context, taskService *service.TaskService, internalMessageService *service.InternalMessageService, notificationService *service.NotificationService, scriptRuntime *service.ScriptRuntime) (*asynqServer.Server, error) {
	cfg := ctx.GetConfig()

	if cfg == nil || cfg.Server == nil || cfg.Server.Asynq == nil {
		return nil, nil
	}

	srv := bootstrapAsynq.NewAsynqServer(
		cfg.Server.Asynq,
		asynqServer.WithEnableKeepAlive(false),
	)

	taskService.RegisterTaskScheduler(srv)
	// 注入 asynq 任务入队能力，使广播 fan-out 改走 asynq 任务（可重试、断点恢复）。
	// asynq 未配置时本函数在上方 return nil，此行不会执行，internalMessageService.taskEnqueuer 保持 nil。
	internalMessageService.RegisterTaskEnqueuer(taskService)
	// 通知投递同样拿这份入队能力：命中 asyncDispatchEvents 的事件（找回密码/换绑验证码）
	// 入队后立即返回，SMTP 往返不再占住 HTTP。未配置 asynq 时保持 nil，全部事件退回同步投递。
	notificationService.RegisterTaskEnqueuer(taskService)

	// 脚本任务桥：注册固定分发类型（task.ScriptTaskDispatchType）的订阅。
	// asynq 的 mux 拒绝 Start 后注册 handler，而脚本处理器运行期动态变化，
	// 故订阅在启动期一次注册，处理器名经消息载荷 handler 字段分发（见 ScriptRuntime.RunScriptTaskHandler）。
	// sys_tasks 侧：type=PERIODIC，type_name="script_task"，task_payload 带 handler/params。
	if scriptRuntime != nil {
		if err := scriptRuntime.AttachScriptTaskRegistrar(
			func(taskType string, fn func(string, *task.ScriptTaskData) error) error {
				return srv.RegisterSubscriber(
					taskType,
					func(taskType string, payload asynqServer.MessagePayload) error {
						data, ok := payload.(*task.ScriptTaskData)
						if !ok {
							return errors.New("invalid script task payload type")
						}
						return fn(taskType, data)
					},
					func() any { return &task.ScriptTaskData{} },
				)
			},
		); err != nil {
			log.Error(err)
			return nil, err
		}
		scriptRuntime.RegisterScriptTaskSubscriber()
	}

	var err error

	// 注册任务
	if err = asynqServer.RegisterSubscriber(srv, task.BackupTaskType, taskService.AsyncBackup); err != nil {
		log.Error(err)
		return nil, err
	}

	// 注册租户到期扫描任务（系统级常驻任务，不写入 sys_tasks 表）。
	// 该任务每小时整点扫描 status==ON 且 expired_at<=now 的租户，按套餐 expiry_policy 修改状态并吊销令牌。
	// READONLY 策略的即时读写拦截由 TenantAccessChecker 中间件承担，不依赖本扫描任务。
	// 注意：仅在此注册 handler（subscriber），周期调度在 startAllTask 末尾注册，
	// 以确保 RestartAllTask（先 RemoveAllPeriodicTask 再 startAllTask）后仍能恢复。
	if err = asynqServer.RegisterSubscriber(srv, task.TenantExpiryScanTaskType, taskService.AsyncTenantExpiryScan); err != nil {
		log.Error(err)
		return nil, err
	}

	// 注册审计日志归档任务 handler（等保 ≥6 个月留存：超期行导出 JSONL 后删除）。
	// 周期调度同样在 startAllTask 末尾注册。
	if err = asynqServer.RegisterSubscriber(srv, task.AuditLogArchiveTaskType, taskService.AsyncAuditLogArchive); err != nil {
		log.Error(err)
		return nil, err
	}

	// 注册站内信广播任务 handler。该任务为一次性投递任务（非周期），
	// 由 InternalMessageService.SendMessage 在消息落库后入队，handler 从 DB 取回消息本体后执行 fan-out。
	// 重试幂等性由 (message_id, recipient_user_id) 唯一约束 + CreateBulk 的 ON CONFLICT DO NOTHING 保证。
	if err = asynqServer.RegisterSubscriber(srv, task.BroadcastMessageTaskType, internalMessageService.AsyncBroadcastMessage); err != nil {
		log.Error(err)
		return nil, err
	}

	// 注册通知异步派发 handler（找回密码/换绑验证码的 SMTP 投递）。
	// 用 WithCtx 变体：asynq 的任务 ctx 带着 asynq.Timeout 的 deadline，而 mailer 走 DialContext
	// 拨号——只有把 ctx 传到 Send，超时才真能掐断一次卡死的 SMTP 握手（否则任务超时只杀任务不杀连接）。
	// 重试幂等性由台账状态把关：status 已非 SENDING 即视为已处理（见 AsyncNotificationDispatch）。
	if err = asynqServer.RegisterSubscriberWithCtx(srv, task.NotificationDispatchTaskType, notificationService.AsyncNotificationDispatch); err != nil {
		log.Error(err)
		return nil, err
	}

	// 通知台账超时清扫 handler（系统级常驻任务，不写入 sys_tasks 表）。
	// 周期调度在 startAllTask 末尾注册，与其他系统级任务同构。
	// 它结算的是"再也不会有结论"的 SENDING 行（进程死在拨号中途 / 结论回写失败 / 队列没有消费者）。
	if err = asynqServer.RegisterSubscriber(srv, task.NotificationDeliverySweepTaskType, notificationService.AsyncDeliverySweep); err != nil {
		log.Error(err)
		return nil, err
	}

	// 启动所有的任务
	if _, err = taskService.StartAllTask(appViewer.NewSystemViewerContext(ctx.Context()), &emptypb.Empty{}); err != nil {
		log.Error(err)
		return nil, err
	}

	return srv, nil
}
