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
func NewAsynqServer(ctx *bootstrap.Context, taskService *service.TaskService) (*asynqServer.Server, error) {
	cfg := ctx.GetConfig()

	if cfg == nil || cfg.Server == nil || cfg.Server.Asynq == nil {
		return nil, nil
	}

	srv := bootstrapAsynq.NewAsynqServer(cfg.Server.Asynq)

	taskService.RegisterTaskScheduler(srv)

	var err error

	// 注册任务
	if err = asynqServer.RegisterSubscriber(srv, task.BackupTaskType, taskService.AsyncBackup); err != nil {
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
