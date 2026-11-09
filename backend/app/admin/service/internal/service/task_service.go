package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	taskV1 "go-wind-admin/api/gen/go/task/service/v1"

	"github.com/tx7do/go-crud/viewer"
	"go-wind-admin/pkg/middleware/auth"
	"go-wind-admin/pkg/oss"
	"go-wind-admin/pkg/task"
)

// TaskScheduler 任务调度接口
type TaskScheduler interface {
	TaskTypeExists(taskType string) bool
	GetRegisteredTaskTypes() []string

	NewTask(typeName string, msg any, opts ...asynq.Option) error
	NewWaitResultTask(typeName string, msg any, opts ...asynq.Option) error
	NewPeriodicTask(cronSpec, typeName string, msg any, opts ...asynq.Option) (string, error)

	RemovePeriodicTask(id string) error
	// RemovePeriodicTaskByID 按 asynq 调度项 entryID 注销，不受 typeName 冲突影响：
	// 多次注册同一 typeName 产生的多个调度项可被独立移除（库 v1.3.14 起提供）。
	RemovePeriodicTaskByID(entryID string) error
	RemoveAllPeriodicTask()
}

// backupBucket 备份文件存放的 OSS 桶名。
const backupBucket = "backups"

// TaskService 任务服务
type TaskService struct {
	adminV1.TaskServiceHTTPServer

	log *bLogger.Helper

	taskScheduler TaskScheduler

	// periodicEntryIDs 定期任务 entryID 账本（ownerKey → 历次注册的 entryID）。
	// ownerKey：业务任务行用 "row:<sys_tasks.id>"，系统级常驻任务用 "sys:<typeName>"。
	// 库侧以 typeName 为去重/注销键、同名再注册会覆盖库内 entryIDs[typeName]（被覆盖的
	// 旧调度项仍在 asynq 里触发却从此注销不到，成孤儿）。本账本按行留住全部 entryID，
	// 使 ① 同名 PERIODIC 各租户各行可并存触发（多租户互不干扰，根治原互斥限制）；
	// ② stopTask/Update/Delete 按行精准注销；③ 全停时逐 ID 补刀，RestartAllTask 不再累积。
	periodicMu       sync.Mutex
	periodicEntryIDs map[string][]string

	userRepo            data.UserRepo
	taskRepo            *data.TaskRepo
	backupRepo          *data.BackupRepo
	tenantUsageRepo     *data.TenantUsageRepo
	auditLogArchiveRepo *data.AuditLogArchiveRepo
	mc                  *oss.MinIOClient
}

func NewTaskService(
	ctx *bootstrap.Context,
	taskRepo *data.TaskRepo,
	userRepo data.UserRepo,
	backupRepo *data.BackupRepo,
	tenantUsageRepo *data.TenantUsageRepo,
	auditLogArchiveRepo *data.AuditLogArchiveRepo,
	mc *oss.MinIOClient,
) *TaskService {
	svc := &TaskService{
		log:                 ctx.NewLoggerHelper("task/service/admin-service"),
		periodicEntryIDs:    make(map[string][]string),
		taskRepo:            taskRepo,
		userRepo:            userRepo,
		backupRepo:          backupRepo,
		tenantUsageRepo:     tenantUsageRepo,
		auditLogArchiveRepo: auditLogArchiveRepo,
		mc:                  mc,
	}

	return svc
}

func (s *TaskService) RegisterTaskScheduler(taskScheduler TaskScheduler) {
	s.taskScheduler = taskScheduler
}

// NewTask 委托 taskScheduler.NewTask 投递一次性 asynq 任务。
// 实现 service.TaskEnqueuer 接口，供 InternalMessageService 广播任务入队使用。
// asynq 未配置时 taskScheduler 为 nil，调用方（InternalMessageService）通过 nil 检查回退到 goroutine。
func (s *TaskService) NewTask(typeName string, msg any, opts ...asynq.Option) error {
	if !s.hasScheduler() {
		return errors.New("task scheduler is not configured")
	}
	return s.taskScheduler.NewTask(typeName, msg, opts...)
}

// hasScheduler 检查调度器是否可用（未配置 asynq 时为 nil）。
// 在所有调用 taskScheduler 的地方前置检查，避免 nil 解引用 panic。
func (s *TaskService) hasScheduler() bool {
	return s.taskScheduler != nil
}

func (s *TaskService) List(ctx context.Context, req *paginationV1.PagingRequest) (*taskV1.ListTaskResponse, error) {
	return s.taskRepo.List(ctx, req)
}

func (s *TaskService) Get(ctx context.Context, req *taskV1.GetTaskRequest) (*taskV1.Task, error) {
	return s.taskRepo.Get(ctx, req)
}

func (s *TaskService) ListTaskTypeName(_ context.Context, _ *emptypb.Empty) (*taskV1.ListTaskTypeNameResponse, error) {
	// nil 调度器防护：asynq 未配置时返回空列表而非 panic
	if !s.hasScheduler() {
		return &taskV1.ListTaskTypeNameResponse{}, nil
	}
	typeNames := s.taskScheduler.GetRegisteredTaskTypes()
	return &taskV1.ListTaskTypeNameResponse{
		TypeNames: typeNames,
	}, nil
}

// requireTaskPlatformAdmin 任务写入口的平台闸门。
//
// asynq 注册表里的任务全部是平台级 handler（backup/广播/审计归档/到期扫描/配额水位/
// 脚本桥……见 asynq_server.go），租户上下文只约束 sys_tasks 行归属，不约束 handler 以
// 什么身份执行——handler 内部自建 SystemContext。此前无闸门时，租户用户可建
// type_name:"backup" 的任务让 AsyncBackup 以 SystemContext 导出全平台核心表，或以
// broadcast_message(payload 可控) 读任意租户站内信并广播、StopAllTask 停摆平台全部
// 周期调度（到期处置/配额水位随之失效）。故全部写入口统一挡在平台管理员之内。
func (s *TaskService) requireTaskPlatformAdmin(ctx context.Context, action string) error {
	return requirePlatformAdmin(ctx, s.log, "task "+action)
}

func (s *TaskService) Create(ctx context.Context, req *taskV1.CreateTaskRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	if err := s.requireTaskPlatformAdmin(ctx, "create"); err != nil {
		return nil, err
	}

	// H8：校验 typeName 已在调度器注册，防止创建无 handler 的幽灵任务（每个 cron tick 报错）
	if !s.hasScheduler() || !s.taskScheduler.TaskTypeExists(req.Data.GetTypeName()) {
		return nil, adminV1.ErrorBadRequest("task type [%s] is not registered", req.Data.GetTypeName())
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	var t *taskV1.Task
	if t, err = s.taskRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	// 禁用状态的任务只需落库，不进入调度器
	if !t.GetEnable() {
		return &emptypb.Empty{}, nil
	}

	if err = s.startTask(t); err != nil {
		// 调度失败不掩盖：DB 记录已建，但任务实际不会运行，需明确告知
		s.log.Errorf(ctx, "create task [%s] succeeded but scheduling failed: %s", t.GetTypeName(), err.Error())
		return nil, adminV1.ErrorInternalServerError("task scheduling failed: %s", err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *TaskService) Update(ctx context.Context, req *taskV1.UpdateTaskRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	if err := s.requireTaskPlatformAdmin(ctx, "update"); err != nil {
		return nil, err
	}

	// 获取更新前的任务，用于判断调度器中是否有正在运行的注册项。
	// 此处不可吞错：若 Get 失败则 oldTask==nil，下方 remove 会因 oldTask==nil 被跳过，
	// 导致旧调度项残留（任务"停用/更新后仍运行"）。故失败直接返回。
	oldTask, err := s.taskRepo.Get(ctx, &taskV1.GetTaskRequest{QueryBy: &taskV1.GetTaskRequest_Id{Id: req.GetId()}})
	if err != nil {
		return nil, err
	}

	// H8：校验 typeName 已在调度器注册。
	// 列表行的启停开关只提交 {enable}，不带 typeName；此时从库中回填原值再校验，
	// 否则开关永远 400 "task type [] is not registered"。
	if req.Data.GetTypeName() == "" && oldTask != nil {
		req.Data.TypeName = trans.Ptr(oldTask.GetTypeName())
	}
	if !s.hasScheduler() || !s.taskScheduler.TaskTypeExists(req.Data.GetTypeName()) {
		return nil, adminV1.ErrorBadRequest("task type [%s] is not registered", req.Data.GetTypeName())
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.Id = trans.Ptr(req.GetId())

	req.Data.UpdatedBy = trans.Ptr(operator.UserId)
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	var t *taskV1.Task
	if t, err = s.taskRepo.Update(ctx, req); err != nil {

		return nil, err
	}

	// 先移除调度器中旧的注册项（若存在），避免停用后仍运行、或启用时重复注册。
	// 直接调用注销绕过 stopTask 内部的 enable 保护，
	// 因为 oldTask 可能已是禁用状态，但其注册项可能仍残留在调度器中。
	// 按行（ownerKey=row:<id>）精准注销：多租户同名任务只注销本行，不波及他行。
	if s.hasScheduler() && oldTask != nil && oldTask.GetType() == taskV1.Task_PERIODIC && oldTask.GetTypeName() != "" {
		s.removePeriodicEntries(oldTask)
	}

	// 根据更新后的 enable 状态决定是否重新启动。
	// 注意：disable 也是一种合法的更新意图（更新即停用），不应视为调度失败。
	// startTask 内部对 enable==false 直接返回 error，若此处无差别调用会把
	// "成功停用"误报成 InternalServerError。因此仅在 enable 时才启动。
	if t.GetEnable() {
		if err = s.startTask(t); err != nil {
			s.log.Error(ctx, err.Error())
			return nil, adminV1.ErrorInternalServerError("task scheduling failed: %s", err.Error())
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *TaskService) Delete(ctx context.Context, req *taskV1.DeleteTaskRequest) (*emptypb.Empty, error) {
	if err := s.requireTaskPlatformAdmin(ctx, "delete"); err != nil {
		return nil, err
	}

	var err error
	var t *taskV1.Task
	// 获取待删除任务用于清理调度项。失败必须返回：否则 t==nil 会跳过调度清理，
	// 造成 DB 已删但调度器里周期任务仍触发的"幽灵任务"泄漏。
	if t, err = s.taskRepo.Get(ctx, &taskV1.GetTaskRequest{QueryBy: &taskV1.GetTaskRequest_Id{Id: req.GetId()}}); err != nil {
		return nil, err
	}

	if err = s.taskRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	// DB 删除成功后清理调度项。stopTask 在 enable=false 时会返回错误并跳过清理，
	// 但 disabled 任务的调度项仍可能残留（经由直接改库或历史 Update 路径），
	// 因此这里绕过 enable 保护，按行（ownerKey）逐 entryID 精准注销。
	if s.hasScheduler() && t.GetType() == taskV1.Task_PERIODIC && t.GetTypeName() != "" {
		s.removePeriodicEntries(t)
	}

	return &emptypb.Empty{}, nil
}

// ControlTask 控制调度任务
func (s *TaskService) ControlTask(ctx context.Context, req *taskV1.ControlTaskRequest) (*emptypb.Empty, error) {
	if err := s.requireTaskPlatformAdmin(ctx, "control"); err != nil {
		return nil, err
	}

	t, err := s.taskRepo.Get(ctx, &taskV1.GetTaskRequest{QueryBy: &taskV1.GetTaskRequest_TypeName{TypeName: req.GetTypeName()}})
	if err != nil {
		s.log.Errorf(ctx, "获取任务失败[%s]", err.Error())
		return nil, err
	}

	switch req.GetControlType() {
	case taskV1.ControlTaskRequest_Restart:
		if err = s.stopTask(t); err != nil {
			return nil, err
		}

		if err = s.startTask(t); err != nil {
			return nil, err
		}

	case taskV1.ControlTaskRequest_Stop:
		err = s.stopTask(t)
		return nil, err

	case taskV1.ControlTaskRequest_Start:
		err = s.startTask(t)
		return nil, err

	default:
		// H8：未知控制类型必须明确拒绝，避免静默返回成功
		return nil, adminV1.ErrorBadRequest("invalid control type")
	}

	return &emptypb.Empty{}, nil
}

// StopAllTask 停止所有的调度任务
func (s *TaskService) StopAllTask(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.requireTaskPlatformAdmin(ctx, "stop-all"); err != nil {
		return nil, err
	}

	s.stopAllTask()
	return &emptypb.Empty{}, nil
}

// StartAllTask 启动所有的调度任务
func (s *TaskService) StartAllTask(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.requireTaskPlatformAdmin(ctx, "start-all"); err != nil {
		return nil, err
	}

	_, err := s.startAllTask(ctx)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// BootstrapStartAllTask 服务启动期装载全部周期任务的内部入口（不经 HTTP、无操作者上下文）。
// StartAllTask 已加平台管理员闸门，启动装配（asynq_server.go）改走本方法，
// 避免闸门把无操作者的系统内部调用一并拒绝。
func (s *TaskService) BootstrapStartAllTask(ctx context.Context) error {
	_, err := s.startAllTask(ctx)
	return err
}

// RestartAllTask 重启所有的调度任务
func (s *TaskService) RestartAllTask(ctx context.Context, _ *emptypb.Empty) (*taskV1.RestartAllTaskResponse, error) {
	if err := s.requireTaskPlatformAdmin(ctx, "restart-all"); err != nil {
		return nil, err
	}

	// 停止所有的任务
	s.stopAllTask()

	// 重新启动所有的任务
	count, err := s.startAllTask(ctx)

	return &taskV1.RestartAllTaskResponse{
		Count: count,
	}, err
}

// startAllTask 启动所有的任务
func (s *TaskService) startAllTask(ctx context.Context) (int32, error) {
	resp, err := s.List(ctx, &paginationV1.PagingRequest{
		NoPaging: trans.Ptr(true),
	})
	if err != nil {
		s.log.Errorf(ctx, "获取任务列表失败[%s]", err.Error())
		return 0, err
	}

	s.log.Infof(ctx, "开始开启定时任务，总计[%d]个", resp.GetTotal())

	// 重新启动任务
	//
	// 注意（调度器限制与根治）：底层 asynq 调度器用 typeName 做 handler 路由键，
	// 同名多 entry 本身可并存触发（各 entry 各自的 cron/payload）；此前"后注册覆盖
	// entryIDs 致先注册项成孤儿"的问题已由 entryID 账本根治（见 TaskService.periodicEntryIDs
	// 与 removePeriodicEntries）——同名 PERIODIC 各租户各行独立注册、按行注销，互不影响。
	// handler 路由仍按 typeName，执行数据来自各自 entry 的载荷，多租户互不干扰。
	var count int32
	for _, t := range resp.GetItems() {
		if s.startTask(t) != nil {
			continue
		} else {
			count++
		}
	}

	s.log.Infof(ctx, "总共成功开启定时任务[%d]个", count)

	// 注册系统级常驻任务：租户到期扫描（不依赖 sys_tasks 表）。
	// 放在 startAllTask 末尾，确保初始启动与 RestartAllTask（先 RemoveAllPeriodicTask 再
	// startAllTask）后均能恢复调度项。handler 已在 NewAsynqServer 中通过 RegisterSubscriber 注册。
	if s.hasScheduler() {
		if entryID, err := s.taskScheduler.NewPeriodicTask(
			task.TenantExpiryScanCronSpec,
			task.TenantExpiryScanTaskType,
			&task.TenantExpiryScanTaskData{},
		); err != nil {
			s.log.Errorf(ctx, "注册系统级到期扫描定时任务失败: %s", err.Error())
		} else {
			s.recordPeriodicEntry("sys:"+task.TenantExpiryScanTaskType, entryID)
			s.log.Infof(ctx, "系统级到期扫描定时任务已注册（cron=%s）", task.TenantExpiryScanCronSpec)
		}

		// 审计日志归档（等保 ≥6 个月留存）：超保留期行导出 JSONL 后删除，
		// 目录/保留期走环境变量（见 AsyncAuditLogArchive）。
		if entryID, err := s.taskScheduler.NewPeriodicTask(
			task.AuditLogArchiveCronSpec,
			task.AuditLogArchiveTaskType,
			&task.AuditLogArchiveTaskData{},
		); err != nil {
			s.log.Errorf(ctx, "注册审计日志归档定时任务失败: %s", err.Error())
		} else {
			s.recordPeriodicEntry("sys:"+task.AuditLogArchiveTaskType, entryID)
			s.log.Infof(ctx, "审计日志归档定时任务已注册（cron=%s）", task.AuditLogArchiveCronSpec)
		}

		// 通知台账超时清扫：把超期仍停在 SENDING 的投递行结算为 FAILED。
		// handler 属通知域（NotificationService.AsyncDeliverySweep），调度项仍在这里注册，
		// 因为只有本函数会在 RestartAllTask（先 RemoveAllPeriodicTask）之后被再次调用。
		if entryID, err := s.taskScheduler.NewPeriodicTask(
			task.NotificationDeliverySweepCronSpec,
			task.NotificationDeliverySweepTaskType,
			&task.NotificationDeliverySweepTaskData{},
		); err != nil {
			s.log.Errorf(ctx, "注册通知台账清扫定时任务失败: %s", err.Error())
		} else {
			s.recordPeriodicEntry("sys:"+task.NotificationDeliverySweepTaskType, entryID)
			s.log.Infof(ctx, "通知台账清扫定时任务已注册（cron=%s）", task.NotificationDeliverySweepCronSpec)
		}

		// 监控告警扫描：每 5 分钟评估全部启用的告警规则（handler 属监控告警域）。
		if entryID, err := s.taskScheduler.NewPeriodicTask(
			task.MonitorAlertScanCronSpec,
			task.MonitorAlertScanTaskType,
			&task.MonitorAlertScanTaskData{},
		); err != nil {
			s.log.Errorf(ctx, "注册监控告警扫描定时任务失败: %s", err.Error())
		} else {
			s.recordPeriodicEntry("sys:"+task.MonitorAlertScanTaskType, entryID)
			s.log.Infof(ctx, "监控告警扫描定时任务已注册（cron=%s）", task.MonitorAlertScanCronSpec)
		}

		// 审计日报 AI 摘要：昨日操作审计统计 → 默认模型摘要 → 站内信投递平台用户。
		// handler 属 AI 域（AiDigestService.AsyncAiAuditDigest），调度项在此注册（同上理由）。
		if entryID, err := s.taskScheduler.NewPeriodicTask(
			task.AiAuditDigestCronSpec,
			task.AiAuditDigestTaskType,
			&task.AiAuditDigestTaskData{},
		); err != nil {
			s.log.Errorf(ctx, "注册审计日报 AI 摘要定时任务失败: %s", err.Error())
		} else {
			s.recordPeriodicEntry("sys:"+task.AiAuditDigestTaskType, entryID)
			s.log.Infof(ctx, "审计日报 AI 摘要定时任务已注册（cron=%s）", task.AiAuditDigestCronSpec)
		}

		// 套餐配额水位扫描：扫全部 ON 租户四类配额水位，命中租户的告警经站内信
		// 投递其管理员。handler 属套餐计费域（PlanQuotaWatermarkService.
		// AsyncPlanQuotaWatermarkScan），调度项在此注册（同上理由）。
		if entryID, err := s.taskScheduler.NewPeriodicTask(
			task.PlanQuotaWatermarkCronSpec,
			task.PlanQuotaWatermarkTaskType,
			&task.PlanQuotaWatermarkTaskData{},
		); err != nil {
			s.log.Errorf(ctx, "注册套餐配额水位扫描定时任务失败: %s", err.Error())
		} else {
			s.recordPeriodicEntry("sys:"+task.PlanQuotaWatermarkTaskType, entryID)
			s.log.Infof(ctx, "套餐配额水位扫描定时任务已注册（cron=%s）", task.PlanQuotaWatermarkCronSpec)
		}

		// 通知归档集清理：把归档集里超保留期的 notification_dispatch 任务删掉
		// （载荷带明文验证码，不能永久躺在 Redis）。handler 属通知域
		// （NotificationService.AsyncArchivedCleanup），调度项在此注册（同上理由）。
		if entryID, err := s.taskScheduler.NewPeriodicTask(
			task.NotificationArchivedCleanupCronSpec,
			task.NotificationArchivedCleanupTaskType,
			&task.NotificationArchivedCleanupTaskData{},
		); err != nil {
			s.log.Errorf(ctx, "注册通知归档集清理定时任务失败: %s", err.Error())
		} else {
			s.recordPeriodicEntry("sys:"+task.NotificationArchivedCleanupTaskType, entryID)
			s.log.Infof(ctx, "通知归档集清理定时任务已注册（cron=%s）", task.NotificationArchivedCleanupCronSpec)
		}
	}

	return count, nil
}

// stopAllTask 停止所有的任务
func (s *TaskService) stopAllTask() {
	// nil 调度器防护
	if !s.hasScheduler() {
		s.log.Warnf(context.Background(), "task scheduler is not configured, skip stopAllTask")
		return
	}

	s.log.Infof(context.Background(), "开始清除所有的定时任务...")

	// 清除所有的定时任务：RemoveAllPeriodicTask 只够得着库内 entryIDs 映射的尾值，
	// 被同名覆盖的旧调度项要靠账本逐 entryID 补刀（孤儿缓解），最后再全量清一次。
	s.taskScheduler.RemoveAllPeriodicTask()
	s.clearPeriodicEntries()

	s.log.Infof(context.Background(), "完成清除所有的定时任务")
}

// stopTask 停止一个任务
func (s *TaskService) stopTask(t *taskV1.Task) error {
	if t == nil {
		return errors.New("task is nil")
	}

	if !t.GetEnable() {
		return errors.New("task is not enable")
	}

	// nil 调度器防护
	if !s.hasScheduler() {
		return errors.New("task scheduler is not configured")
	}

	switch t.GetType() {
	case taskV1.Task_PERIODIC:
		s.removePeriodicEntries(t)
		return nil

	// DELAY/WAIT_RESULT 是已投递到 asynq 队列的一次性任务，底层 transport
	// 未暴露 Inspector（删除排队任务/取消在途任务的能力），无法安全停止。
	// 此前空 case 静默返回成功，用户会以为已停止——必须诚实报错。
	case taskV1.Task_DELAY:
		return errors.New("queued one-shot task cannot be stopped, it will run at its scheduled time")
	case taskV1.Task_WAIT_RESULT:
		return errors.New("in-flight task cannot be stopped, please wait for it to finish")
	}

	return nil
}

// convertTaskOption 转换任务选项
func (s *TaskService) convertTaskOption(t *taskV1.Task) (opts []asynq.Option, payload any) {
	if t == nil {
		return
	}

	if len(t.GetTaskPayload()) > 0 {
		_ = json.Unmarshal([]byte(t.GetTaskPayload()), &payload)
	}

	// asynq 拒绝 nil message（"message is nil"）；任务数据留空时兜底为空对象
	if payload == nil {
		payload = map[string]any{}
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
		opts = append(opts, asynq.Group(t.GetTaskOptions().GetGroup()))
		opts = append(opts, asynq.TaskID(t.GetTaskOptions().GetTaskId()))
	}

	return
}

// recordPeriodicEntry 把一次成功注册的调度项 entryID 记入账本（按 ownerKey 分桶）。
// 见 TaskService.periodicEntryIDs 注释——账本是多租户同名并存与孤儿根治的核心。
func (s *TaskService) recordPeriodicEntry(ownerKey, entryID string) {
	if entryID == "" {
		return
	}
	s.periodicMu.Lock()
	defer s.periodicMu.Unlock()
	if s.periodicEntryIDs == nil {
		s.periodicEntryIDs = make(map[string][]string)
	}
	s.periodicEntryIDs[ownerKey] = append(s.periodicEntryIDs[ownerKey], entryID)
}

// periodicOwnerKey 账本分桶键：业务任务行用 "row:<sys_tasks.id>"（每行独立记账，
// 多租户同名互不影响）；无 ID 的场景回退 "type:<typeName>"（防御分支，正常路径不会走到）。
func (s *TaskService) periodicOwnerKey(t *taskV1.Task) string {
	if t != nil && t.GetId() > 0 {
		return fmt.Sprintf("row:%d", t.GetId())
	}
	return "type:" + t.GetTypeName()
}

// removePeriodicEntries 按 ownerKey 注销账本里的全部 entryID，最后按 typeName 兜底
// 注销一次（清理库内 entryIDs 映射的尾值，不影响账本里已按 ID 注销的其他行）。
// 注销失败只告警不中断——留孤儿调度项（永久触发）必须可观测。
func (s *TaskService) removePeriodicEntries(t *taskV1.Task) {
	if !s.hasScheduler() || t == nil {
		return
	}
	ownerKey := s.periodicOwnerKey(t)
	typeName := t.GetTypeName()

	s.periodicMu.Lock()
	ids := s.periodicEntryIDs[ownerKey]
	delete(s.periodicEntryIDs, ownerKey)
	s.periodicMu.Unlock()

	for _, id := range ids {
		if err := s.taskScheduler.RemovePeriodicTaskByID(id); err != nil {
			s.log.Warnf(context.Background(), "按 entryID 注销定时任务失败[%s/%s]: %v", ownerKey, id, err)
		}
	}
	if err := s.taskScheduler.RemovePeriodicTask(typeName); err != nil {
		s.log.Warnf(context.Background(), "按 typeName 兜底注销定时任务失败[%s]: %v", typeName, err)
	}
}

// clearPeriodicEntries 全停后清扫账本残留：RemoveAllPeriodicTask 只能注销库内
// entryIDs 映射里"每 typeName 最新一个"，被覆盖的旧调度项必须逐 entryID 补刀，
// 否则 RestartAllTask（先 stopAllTask 再 startAllTask）会一轮一轮累积孤儿。
func (s *TaskService) clearPeriodicEntries() {
	s.periodicMu.Lock()
	defer s.periodicMu.Unlock()
	if !s.hasScheduler() {
		s.periodicEntryIDs = make(map[string][]string)
		return
	}
	for ownerKey, ids := range s.periodicEntryIDs {
		for _, id := range ids {
			if err := s.taskScheduler.RemovePeriodicTaskByID(id); err != nil {
				s.log.Warnf(context.Background(), "全停清扫按 entryID 注销失败[%s/%s]: %v", ownerKey, id, err)
			}
		}
	}
	s.periodicEntryIDs = make(map[string][]string)
}

// startTask 启动一个任务
func (s *TaskService) startTask(t *taskV1.Task) error {
	if t == nil {
		return errors.New("task is nil")
	}

	if !t.GetEnable() {
		return errors.New("task is not enable")
	}

	// nil 调度器防护
	if !s.hasScheduler() {
		return errors.New("task scheduler is not configured")
	}

	var opts []asynq.Option
	var payload any
	var err error

	switch t.GetType() {
	case taskV1.Task_PERIODIC:
		opts, payload = s.convertTaskOption(t)
		var entryID string
		if entryID, err = s.taskScheduler.NewPeriodicTask(t.GetCronSpec(), t.GetTypeName(), payload, opts...); err != nil {
			s.log.Errorf(context.Background(), "[%s] 创建定时任务失败[%s]", t.GetTypeName(), err.Error())
			return err
		}
		s.recordPeriodicEntry(s.periodicOwnerKey(t), entryID)

	case taskV1.Task_DELAY:
		opts, payload = s.convertTaskOption(t)
		if err = s.taskScheduler.NewTask(t.GetTypeName(), payload, opts...); err != nil {
			s.log.Errorf(context.Background(), "[%s] 创建延迟任务失败[%s]", t.GetTypeName(), err.Error())
			return err
		}

	case taskV1.Task_WAIT_RESULT:
		opts, payload = s.convertTaskOption(t)
		if err = s.taskScheduler.NewWaitResultTask(t.GetTypeName(), payload, opts...); err != nil {
			s.log.Errorf(context.Background(), "[%s] 创建等待结果任务失败[%s]", t.GetTypeName(), err.Error())
			return err
		}
	}

	return nil
}

// AsyncAuditLogArchive 审计日志归档任务：把超过保留期的审计行导出为本地
// JSONL 文件后从库中删除（等保要求日志留存 ≥6 个月）。
// 目录：AUDIT_ARCHIVE_DIR（默认 ./data/audit-archive）；
// 保留天数：AUDIT_RETENTION_DAYS（默认 180）。
func (s *TaskService) AsyncAuditLogArchive(taskType string, taskData *task.AuditLogArchiveTaskData) error {
	s.log.Infof(context.Background(), "AsyncAuditLogArchive [%s] [%+v]", taskType, taskData)

	if s.auditLogArchiveRepo == nil {
		return errors.New("audit log archive repo is not configured")
	}

	outDir := os.Getenv("AUDIT_ARCHIVE_DIR")
	if outDir == "" {
		outDir = "./data/audit-archive"
	}
	retentionDays := 180
	if v := os.Getenv("AUDIT_RETENTION_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			retentionDays = n
		}
	}
	before := time.Now().AddDate(0, 0, -retentionDays)

	ctx := viewer.WithSystemContext(context.Background())
	results, err := s.auditLogArchiveRepo.ArchiveExpired(ctx, before, outDir)
	if err != nil {
		s.log.Errorf(ctx, "audit log archive failed: %s", err.Error())
		return err
	}
	for table, n := range results {
		s.log.Infof(ctx, "audit log archived: %s -> %d rows (dir=%s)", table, n, outDir)
	}
	return nil
}

// AsyncBackup 异步备份任务的实际执行逻辑。
//
// H8：实现真正的备份——导出核心业务表为 JSON，gzip 压缩后上传到 OSS 的 backups 桶。
// 纯 Go 实现，跨数据库驱动（MySQL/PostgreSQL/SQLite）。
// 当前覆盖核心身份/权限/组织表；审计日志等大体量表暂不纳入（可按需扩展 BackupRepo.ExportCoreTables）。
func (s *TaskService) AsyncBackup(taskType string, taskData *task.BackupTaskData) error {
	// taskData 可能为 nil（下方有判空分支），直接解引用 taskData.Name 会在 nil 时 panic
	backupName := ""
	if taskData != nil {
		backupName = taskData.Name
	}
	s.log.Infof(context.Background(), "AsyncBackup [%s] [%+v] [%s]", taskType, taskData, backupName)

	// 用 SystemContext 包裹：备份需要导出全部租户的核心表，
	// 而 TenantPrivacy 在 viewer 缺失时会返回 error 导致带 tenant_id 的表全部查询失败。
	// SystemContext 的 IsSystemContext()==true，使 TenantPrivacy.EvalQuery 放行全量数据。
	ctx := viewer.WithSystemContext(context.Background())
	if backupName == "" {
		backupName = fmt.Sprintf("backup-%s", time.Now().UTC().Format("20060102-150405"))
	}

	if s.backupRepo == nil || !s.backupRepo.IsConfigured() || s.mc == nil {
		return fmt.Errorf("backup dependencies not configured (backupRepo or minio is nil)")
	}

	// 1. 导出核心表（ent 访问收敛在 data 层的 BackupRepo 内）
	tables, err := s.backupRepo.ExportCoreTables(ctx)
	if err != nil {
		return fmt.Errorf("export core tables failed: %w", err)
	}

	// 2. 序列化为 JSON
	jsonBytes, err := json.MarshalIndent(map[string]any{
		"_meta": map[string]any{
			"exportedAt": time.Now().UTC().Format(time.RFC3339),
			"name":       backupName,
			"tables":     tableNamesOf(tables),
		},
		"data": tables,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal backup json failed: %w", err)
	}

	// 3. gzip 压缩
	compressed, err := gzipBytes(jsonBytes)
	if err != nil {
		return fmt.Errorf("gzip backup failed: %w", err)
	}

	// 4. 上传到 OSS
	objectName := fmt.Sprintf("%s/%s-%s.json.gz",
		time.Now().UTC().Format("2006/01/02"),
		backupName,
		time.Now().UTC().Format("150405"),
	)

	s.log.Infof(context.Background(), "backup: uploading %s (%d bytes raw, %d bytes compressed) to bucket %q",
		objectName, len(jsonBytes), len(compressed), backupBucket)

	if _, _, _, err = s.mc.UploadFile(ctx, backupBucket, objectName, "application/gzip", compressed); err != nil {
		s.log.Errorf(context.Background(), "backup: upload to oss failed: %s", err.Error())
		return fmt.Errorf("upload backup to oss failed: %w", err)
	}

	s.log.Infof(context.Background(), "backup: completed successfully, object=%s", objectName)

	// 5. 顺手清理超过保留期的旧备份（best-effort）：备份本体已成功，
	// 清理失败只留日志，不影响任务结论（否则告警指向错误的故障点）。
	if cleaner := newMinioBackupCleaner(s.mc); cleaner != nil {
		s.cleanupOldBackups(ctx, cleaner)
	}

	return nil
}

// tableNamesOf 返回 map 的键列表（用于备份元信息）。
func tableNamesOf(tables map[string]any) []string {
	names := make([]string, 0, len(tables))
	for k := range tables {
		names = append(names, k)
	}
	return names
}

// gzipBytes 使用标准库 gzip 压缩字节切片。
func gzipBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		_ = gz.Close()
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// AsyncTenantExpiryScan 租户到期扫描任务的实际执行逻辑。
//
// 系统级常驻定时任务（每小时整点），由 NewAsynqServer 启动时注册，不写入 sys_tasks 表。
// 逻辑详见 TenantUsageRepo.EnforceExpiryPolicies：扫描 status==ON 且 expired_at<=now 的租户，
// 按其套餐的 expiry_policy 修改租户状态（BLOCK_LOGIN→EXPIRED、FREEZE→FREEZE、READONLY 保持 ON），
// 并吊销受影响租户全部用户的在线令牌。
//
// READONLY 策略的即时读写拦截由 TenantAccessChecker 中间件承担，不需要等待本扫描任务。
func (s *TaskService) AsyncTenantExpiryScan(taskType string, taskData *task.TenantExpiryScanTaskData) error {
	s.log.Infof(context.Background(), "AsyncTenantExpiryScan [%s] [%+v]", taskType, taskData)

	if s.tenantUsageRepo == nil {
		s.log.Errorf(context.Background(), "AsyncTenantExpiryScan: tenantUsageRepo is nil, aborting")
		return errors.New("tenantUsageRepo is not configured")
	}

	// 用 SystemContext 包裹：到期扫描需要跨租户查询。
	ctx := viewer.WithSystemContext(context.Background())

	count, err := s.tenantUsageRepo.EnforceExpiryPolicies(ctx)
	if err != nil {
		s.log.Errorf(context.Background(), "AsyncTenantExpiryScan: enforce failed: %v", err)
		return err
	}

	s.log.Infof(context.Background(), "AsyncTenantExpiryScan: completed, %d tenants enforced", count)
	return nil
}
