package server

import (
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
func NewAsynqServer(ctx *bootstrap.Context, taskService *service.TaskService, internalMessageService *service.InternalMessageService) (*asynqServer.Server, error) {
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

	// 注册站内信广播任务 handler。该任务为一次性投递任务（非周期），
	// 由 InternalMessageService.SendMessage 在消息落库后入队，handler 从 DB 取回消息本体后执行 fan-out。
	// 重试幂等性由 (message_id, recipient_user_id) 唯一约束 + CreateBulk 的 ON CONFLICT DO NOTHING 保证。
	if err = asynqServer.RegisterSubscriber(srv, task.BroadcastMessageTaskType, internalMessageService.AsyncBroadcastMessage); err != nil {
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
