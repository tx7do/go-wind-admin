package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/hibiken/asynq"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/timestamppb"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	taskV1 "go-wind-admin/api/gen/go/task/service/v1"
)

// systemTaskInspectMaxFailures 单任务类型保留的最近失败条数上限（防一张表被历史失败刷满）。
const systemTaskInspectMaxFailures = 10

// TaskMonitorService 系统级常驻任务监控（只读 asynq Inspector，平台管理员）。
//
// 方案 C'：零新表。调度计划与队列状态全部来自 asynq 自身——
//   - SchedulerEntries：cron 表达式 + Next/Prev 入队时间（Prev 零值 = 从未跑过）；
//   - 队列的 active/retry/archived 计数与失败明细（LastErr/LastFailedAt）。
//
// Inspector 用 server.asynq 同一 Redis URI 构造，看到的即本实例队列
// （asynq 队列无命名空间，部署共库错开 URI 后此语义正好正确）。
//
// asynq 给不了的：成功执行的持久历史（成功即出队消失）与领域级结果
// （那在领域台账里：通知投递表/审计归档 JSONL）。要看成功历史，配 asynq
// Retention 留存 completed 任务即可，仍是零表方案。
type TaskMonitorService struct {
	adminV1.TaskMonitorServiceHTTPServer

	asynqUri string
	queue    string
	log      *bLogger.Helper

	// inspector 懒构造：Redis 不可达时每次巡检重试，服务不因 asynq 掉线而拒启
	inspector    *asynq.Inspector
	inspectorErr error
}

func NewTaskMonitorService(ctx *bootstrap.Context) *TaskMonitorService {
	cfg := ctx.GetConfig()
	var asynqUri string
	var queue string
	if cfg != nil && cfg.GetServer() != nil && cfg.GetServer().GetAsynq() != nil {
		asynqUri = cfg.GetServer().GetAsynq().GetUri()
		queue = asynqQueueName(cfg.GetServer().GetAsynq().GetQueues())
	}

	return &TaskMonitorService{
		asynqUri: asynqUri,
		queue:    queue,
		log:      ctx.NewLoggerHelper("task-monitor/service/admin-service"),
	}
}

// asynqQueueName 推导本实例的队列名：NewPeriodicTask 未指定队列即落 asynq 默认队列，
// 配置里的多队列是权重而非路由；队列改名场景取配置里权重最高的一个。
// TaskMonitorService 与 NotificationService（归档清理）共用同一口径。
func asynqQueueName(queues map[string]int32) string {
	if len(queues) == 0 {
		return "default"
	}
	if _, ok := queues["default"]; ok {
		return "default"
	}
	return highestWeightQueue(queues)
}

// highestWeightQueue 取权重最高的队列名（配置改掉 default 时的兜底）。
func highestWeightQueue(queues map[string]int32) string {
	best, weight := "default", int32(-1)
	for name, w := range queues {
		if w > weight {
			best, weight = name, w
		}
	}
	return best
}

// getInspector 懒构造 Inspector：连接只在首次巡检时建立，失败留待下轮重试。
func (s *TaskMonitorService) getInspector() (*asynq.Inspector, error) {
	if s.inspector != nil {
		return s.inspector, nil
	}
	connOpt, err := asynq.ParseRedisURI(s.asynqUri)
	if err != nil {
		return nil, fmt.Errorf("parse asynq redis uri failed: %w", err)
	}
	s.inspector = asynq.NewInspector(connOpt)
	return s.inspector, nil
}

// InspectSystemTasks 一站式巡检（平台管理员守卫）。
func (s *TaskMonitorService) InspectSystemTasks(ctx context.Context, _ *taskV1.InspectSystemTasksRequest) (*taskV1.InspectSystemTasksResponse, error) {
	if err := requirePlatformAdmin(ctx, s.log, "task-monitor/inspect"); err != nil {
		return nil, err
	}

	inspector, err := s.getInspector()
	if err != nil {
		s.log.Errorf(ctx, "task monitor inspect: build inspector failed: %s", err.Error())
		return nil, adminV1.ErrorInternalServerError("build asynq inspector failed")
	}

	resp := &taskV1.InspectSystemTasksResponse{Queue: s.queue}

	entries, err := inspector.SchedulerEntries()
	if err != nil {
		s.log.Errorf(ctx, "task monitor inspect: list scheduler entries failed: %s", err.Error())
		return nil, adminV1.ErrorInternalServerError("list scheduler entries failed")
	}
	for _, e := range entries {
		resp.Schedules = append(resp.Schedules, scheduleEntryToDTO(e))
	}

	// 只巡检调度条目覆盖到的任务类型：队列里可能躺着别的项目/别的类型的任务
	// （队列无命名空间，共库错开 URI 后这里只看得到本实例，但保险起见仍以
	// 调度条目为准做过滤），未知类型的遗留任务在失败明细里仍可见。
	monitored := map[string]bool{}
	for _, e := range entries {
		monitored[e.Task.Type()] = true
	}

	summary, err := s.summarizeQueues(ctx, inspector, monitored)
	if err != nil {
		s.log.Errorf(ctx, "task monitor inspect: summarize queues failed: %s", err.Error())
		return nil, adminV1.ErrorInternalServerError("inspect queues failed")
	}
	resp.Summaries = summary

	return resp, nil
}

// summarizeQueues 按任务类型聚合 active/pending/retry/archived 计数与最近失败。
func (s *TaskMonitorService) summarizeQueues(ctx context.Context, inspector *asynq.Inspector, monitored map[string]bool) ([]*taskV1.SystemTaskStateSummary, error) {
	// active + pending + retry + archived 四个状态各拉一遍（分页取首页即可，
	// 计数用 QueueStats 的总量、明细用 ListTasks 首页）
	byType := map[string]*taskV1.SystemTaskStateSummary{}
	get := func(t string) *taskV1.SystemTaskStateSummary {
		if summary, ok := byType[t]; ok {
			return summary
		}
		summary := &taskV1.SystemTaskStateSummary{TaskType: t}
		byType[t] = summary
		return summary
	}

	// 队列计数总量按 TaskInfo 逐条归到类型上：把四种状态的每条任务各读一遍太重，
	// 这里用 ListTasks 首页（前 N 条）+ QueueStats 总数组合——计数只做"有没有存货"
	// 的量级提示，失败明细才是运营要看的主体。
	active, err := inspector.ListActiveTasks(s.queue)
	if err != nil {
		return nil, err
	}
	for _, ti := range active {
		get(ti.Type).Active++
	}
	pending, err := inspector.ListPendingTasks(s.queue)
	if err != nil {
		return nil, err
	}
	for _, ti := range pending {
		get(ti.Type).Pending++
	}
	retry, err := inspector.ListRetryTasks(s.queue)
	if err != nil {
		return nil, err
	}
	for _, ti := range retry {
		summary := get(ti.Type)
		summary.Retry++
		if monitored[ti.Type] || len(monitored) == 0 {
			summary.RecentFailures = append(summary.RecentFailures, failureToDTO(ti, "retry"))
		}
	}
	archived, err := inspector.ListArchivedTasks(s.queue)
	if err != nil {
		return nil, err
	}
	for _, ti := range archived {
		summary := get(ti.Type)
		summary.Archived++
		if monitored[ti.Type] || len(monitored) == 0 {
			summary.RecentFailures = append(summary.RecentFailures, failureToDTO(ti, "archived"))
		}
	}

	// 计数与总量对齐：QueueStats 的全队列计数只在单类型场景下才等于类型计数，
	// 这里不用它，避免共库时把别的项目任务数混进来。

	for _, summary := range byType {
		sortFailures(summary.RecentFailures)
		if len(summary.RecentFailures) > systemTaskInspectMaxFailures {
			summary.RecentFailures = summary.RecentFailures[:systemTaskInspectMaxFailures]
		}
	}

	// 稳定输出：按任务类型名排序
	result := make([]*taskV1.SystemTaskStateSummary, 0, len(byType))
	for _, summary := range byType {
		result = append(result, summary)
	}
	sortSummaries(result)

	return result, nil
}

// ==== 纯映射/排序 helpers（单元测试覆盖） ====

func scheduleEntryToDTO(e *asynq.SchedulerEntry) *taskV1.SystemTaskScheduleEntry {
	return &taskV1.SystemTaskScheduleEntry{
		TaskType:      e.Task.Type(),
		CronSpec:      e.Spec,
		NextEnqueueAt: zeroTimeToNil(e.Next),
		// Prev 零值时间（从未入队）不转：proto 里保持空，前端按"从未运行"渲染
		PrevEnqueueAt: zeroTimeToNil(e.Prev),
	}
}

func failureToDTO(ti *asynq.TaskInfo, state string) *taskV1.SystemTaskFailure {
	return &taskV1.SystemTaskFailure{
		TaskType:      ti.Type,
		Queue:         ti.Queue,
		State:         state,
		LastError:     ti.LastErr,
		LastFailedAt:  zeroTimeToNil(ti.LastFailedAt),
		Retried:       uint32(ti.Retried),
		MaxRetry:      uint32(ti.MaxRetry),
		NextProcessAt: zeroTimeToNil(ti.NextProcessAt),
	}
}

func zeroTimeToNil(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timeutil.TimeToTimestamppb(&t)
}

func sortFailures(failures []*taskV1.SystemTaskFailure) {
	// 按失败时间倒序（零值排最后）
	sort.SliceStable(failures, func(i, j int) bool {
		a := timeutil.TimestamppbToTime(failures[i].GetLastFailedAt())
		b := timeutil.TimestamppbToTime(failures[j].GetLastFailedAt())
		if a == nil {
			return false
		}
		if b == nil {
			return true
		}
		return a.After(*b)
	})
}

func sortSummaries(summaries []*taskV1.SystemTaskStateSummary) {
	sort.SliceStable(summaries, func(i, j int) bool {
		return summaries[i].GetTaskType() < summaries[j].GetTaskType()
	})
}
