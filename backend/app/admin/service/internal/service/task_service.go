package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/kratos-transport/broker"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"

	"go-wind-admin/pkg/middleware/auth"
	"go-wind-admin/pkg/task"
)

// TaskScheduler 任务调度接口
type TaskScheduler interface {
	TaskTypeExists(taskType string) bool
	GetRegisteredTaskTypes() []string

	NewTask(typeName string, msg broker.Any, opts ...asynq.Option) error
	NewWaitResultTask(typeName string, msg broker.Any, opts ...asynq.Option) error
	NewPeriodicTask(cronSpec, taskId, typeName string, msg broker.Any, opts ...asynq.Option) (string, error)

	RemovePeriodicTask(id string) error
	RemoveAllPeriodicTask()
}

// TaskService 任务服务
type TaskService struct {
	adminV1.TaskServiceHTTPServer

	log *log.Helper

	taskScheduler TaskScheduler

	userRepo data.UserRepo
	taskRepo *data.TaskRepo
}

func NewTaskService(
	ctx *bootstrap.Context,
	taskRepo *data.TaskRepo,
	userRepo data.UserRepo,
) *TaskService {
	svc := &TaskService{
		log:      ctx.NewLoggerHelper("task/service/admin-service"),
		taskRepo: taskRepo,
		userRepo: userRepo,
	}

	return svc
}

func (s *TaskService) RegisterTaskScheduler(taskScheduler TaskScheduler) {
	s.taskScheduler = taskScheduler
}

func (s *TaskService) List(ctx context.Context, req *paginationV1.PagingRequest) (*adminV1.ListTaskResponse, error) {
	return s.taskRepo.List(ctx, req)
}

func (s *TaskService) Get(ctx context.Context, req *adminV1.GetTaskRequest) (*adminV1.Task, error) {
	return s.taskRepo.Get(ctx, req)
}

func (s *TaskService) ListTaskTypeName(_ context.Context, _ *emptypb.Empty) (*adminV1.ListTaskTypeNameResponse, error) {
	typeNames := s.taskScheduler.GetRegisteredTaskTypes()
	return &adminV1.ListTaskTypeNameResponse{
		TypeNames: typeNames,
	}, nil
}

func (s *TaskService) Create(ctx context.Context, req *adminV1.CreateTaskRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	var t *adminV1.Task
	if t, err = s.taskRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	if err = s.startTask(t); err != nil {
		s.log.Error(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *TaskService) Update(ctx context.Context, req *adminV1.UpdateTaskRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.UpdatedBy = trans.Ptr(operator.UserId)

	var t *adminV1.Task
	if t, err = s.taskRepo.Update(ctx, req); err != nil {

		return nil, err
	}

	if err = s.startTask(t); err != nil {
		s.log.Error(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *TaskService) Delete(ctx context.Context, req *adminV1.DeleteTaskRequest) (*emptypb.Empty, error) {
	var err error
	var t *adminV1.Task
	if t, err = s.taskRepo.Get(ctx, &adminV1.GetTaskRequest{QueryBy: &adminV1.GetTaskRequest_Id{Id: req.GetId()}}); err != nil {
		s.log.Error(err)
	}

	if err = s.taskRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	if t != nil {
		_ = s.stopTask(t)
	}

	return &emptypb.Empty{}, nil
}

// ControlTask 控制调度任务
func (s *TaskService) ControlTask(ctx context.Context, req *adminV1.ControlTaskRequest) (*emptypb.Empty, error) {
	t, err := s.taskRepo.Get(ctx, &adminV1.GetTaskRequest{QueryBy: &adminV1.GetTaskRequest_TypeName{TypeName: req.GetTypeName()}})
	if err != nil {
		s.log.Errorf("获取任务失败[%s]", err.Error())
		return nil, err
	}

	switch req.GetControlType() {
	case adminV1.ControlTaskRequest_Restart:
		if err = s.stopTask(t); err != nil {
			return nil, err
		}

		if err = s.startTask(t); err != nil {
			return nil, err
		}

	case adminV1.ControlTaskRequest_Stop:
		err = s.stopTask(t)
		return nil, err

	case adminV1.ControlTaskRequest_Start:
		err = s.startTask(t)
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// StopAllTask 停止所有的调度任务
func (s *TaskService) StopAllTask(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	s.stopAllTask()
	return &emptypb.Empty{}, nil
}

// StartAllTask 启动所有的调度任务
func (s *TaskService) StartAllTask(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	_, err := s.startAllTask(ctx)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// RestartAllTask 重启所有的调度任务
func (s *TaskService) RestartAllTask(ctx context.Context, _ *emptypb.Empty) (*adminV1.RestartAllTaskResponse, error) {
	// 停止所有的任务
	s.stopAllTask()

	// 重新启动所有的任务
	count, err := s.startAllTask(ctx)

	return &adminV1.RestartAllTaskResponse{
		Count: count,
	}, err
}

// StartAllTask 启动所有的任务
func (s *TaskService) startAllTask(ctx context.Context) (int32, error) {
	//_, _ = s.asynqServer.NewPeriodicTask("*/1 * * * ?", task.BackupTaskType, task.BackupTaskData{Name: "test"})

	resp, err := s.List(ctx, &paginationV1.PagingRequest{
		NoPaging: trans.Ptr(true),
	})
	if err != nil {
		s.log.Errorf("获取任务列表失败[%s]", err.Error())
		return 0, err
	}

	s.log.Infof("开始开启定时任务，总计[%d]个", resp.GetTotal())

	// 重新启动任务
	var count int32
	for _, t := range resp.GetItems() {
		if s.startTask(t) != nil {
			continue
		} else {
			count++
		}
	}

	s.log.Infof("总共成功开启定时任务[%d]个", count)

	return count, nil
}

// stopAllTask 停止所有的任务
func (s *TaskService) stopAllTask() {
	s.log.Infof("开始清除所有的定时任务...")

	// 清除所有的定时任务
	s.taskScheduler.RemoveAllPeriodicTask()

	s.log.Infof("完成清除所有的定时任务")
}

// stopTask 停止一个任务
func (s *TaskService) stopTask(t *adminV1.Task) error {
	if t == nil {
		return errors.New("task is nil")
	}

	if t.GetEnable() == false {
		return errors.New("task is not enable")
	}

	switch t.GetType() {
	case adminV1.Task_PERIODIC:
		return s.taskScheduler.RemovePeriodicTask(t.GetTypeName())

	case adminV1.Task_DELAY:

	case adminV1.Task_WAIT_RESULT:
	}

	return nil
}

// convertTaskOption 转换任务选项
func (s *TaskService) convertTaskOption(t *adminV1.Task) (opts []asynq.Option, payload broker.Any) {
	if t == nil {
		return
	}

	if len(t.GetTaskPayload()) > 0 {
		_ = json.Unmarshal([]byte(t.GetTaskPayload()), &payload)
	}

	if t.TaskOptions != nil {
		if t.GetTaskOptions().GetMaxRetry() > 0 {
			opts = append(opts, asynq.MaxRetry(int(t.GetTaskOptions().GetMaxRetry())))
		}
		if t.GetTaskOptions().Timeout != nil {
			opts = append(opts, asynq.Timeout(t.GetTaskOptions().GetTimeout().AsDuration()))
		}
		if t.GetTaskOptions().Deadline != nil {
			opts = append(opts, asynq.Deadline(t.GetTaskOptions().GetDeadline().AsTime()))
		}
		if t.GetTaskOptions().ProcessIn != nil {
			opts = append(opts, asynq.ProcessIn(t.GetTaskOptions().GetProcessIn().AsDuration()))
		}
		if t.GetTaskOptions().ProcessAt != nil {
			opts = append(opts, asynq.ProcessAt(t.GetTaskOptions().GetProcessAt().AsTime()))
		}
		if t.GetTaskOptions().UniqueTtl != nil {
			opts = append(opts, asynq.Unique(t.GetTaskOptions().GetUniqueTtl().AsDuration()))
		}
		if t.GetTaskOptions().Retention != nil {
			opts = append(opts, asynq.Retention(t.GetTaskOptions().GetRetention().AsDuration()))
		}
		if t.GetTaskOptions().Group != nil {
			opts = append(opts, asynq.Group(t.GetTaskOptions().GetGroup()))
		}
		if t.GetTaskOptions().TaskId != nil {
			opts = append(opts, asynq.TaskID(t.GetTaskOptions().GetTaskId()))
		}
	}

	return
}

// startTask 启动一个任务
func (s *TaskService) startTask(t *adminV1.Task) error {
	if t == nil {
		return errors.New("task is nil")
	}

	if t.GetEnable() == false {
		return errors.New("task is not enable")
	}

	var opts []asynq.Option
	var payload broker.Any
	var err error

	switch t.GetType() {
	case adminV1.Task_PERIODIC:
		opts, payload = s.convertTaskOption(t)
		if _, err = s.taskScheduler.NewPeriodicTask(t.GetCronSpec(), task.CreateBackupTaskID(t.GetId()), t.GetTypeName(), payload, opts...); err != nil {
			s.log.Errorf("[%s] 创建定时任务失败[%s]", t.GetTypeName(), err.Error())
			return err
		}

	case adminV1.Task_DELAY:
		opts, payload = s.convertTaskOption(t)
		if err = s.taskScheduler.NewTask(t.GetTypeName(), payload, opts...); err != nil {
			s.log.Errorf("[%s] 创建延迟任务失败[%s]", t.GetTypeName(), err.Error())
			return err
		}

	case adminV1.Task_WAIT_RESULT:
		opts, payload = s.convertTaskOption(t)
		if err = s.taskScheduler.NewWaitResultTask(t.GetTypeName(), payload, opts...); err != nil {
			s.log.Errorf("[%s] 创建等待结果任务失败[%s]", t.GetTypeName(), err.Error())
			return err
		}
	}

	return nil
}

// AsyncBackup 异步备份
func (s *TaskService) AsyncBackup(taskType string, taskData *task.BackupTaskData) error {
	s.log.Infof("AsyncBackup [%s] [%+v] [%s]", taskType, taskData, taskData.Name)
	return nil
}
