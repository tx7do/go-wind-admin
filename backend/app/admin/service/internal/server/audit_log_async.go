package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"go-wind-admin/pkg/task"
)

// 审计日志异步化（docs/audit-log-producer-design.md「写入是同步形态」的落地）：
// rest 中间件的审计回调先把 DTO 以 protojson 形态入队（asynq），消费侧解回 DTO
// 后落对应审计表。语义三要素：
//  1. at-least-once：asynq MaxRetry(3)，消费侧落库成功但 ack 失败的极小概率重复
//     被接受（审计场景宁可重复不可丢失）；
//  2. 回退：入队失败或 asynq 未配置（enqueuer 为 nil）时回退同步落库——宁可慢不可丢，
//     小负载部署不配 asynq 时行为与改造前完全一致；
//  3. 事件时间：payload 携带 producer 侧 OccurredAt，api/login 两类消费时覆写
//     DTO.CreatedAt（其 repo honor 非空 CreatedAt），其余三类保持落库时间语义。

// auditLogEnqueuer 审计入队所需的最小能力（*service.TaskService 天然实现）。
type auditLogEnqueuer interface {
	NewTask(typeName string, msg any, opts ...asynq.Option) error
}

// enqueueAuditLog 把一条审计 DTO 入队：protojson.Marshal → map 装载 → asynq。
// 入队失败返回错误，由调用方决定回退策略。
func enqueueAuditLog(ctx context.Context, enqueuer auditLogEnqueuer, kind string, dto proto.Message, occurredAt time.Time) error {
	if enqueuer == nil {
		return fmt.Errorf("audit log enqueuer is not configured")
	}
	raw, err := protojson.Marshal(dto)
	if err != nil {
		return fmt.Errorf("protojson marshal %s audit log failed: %w", kind, err)
	}
	data := map[string]any{}
	if err = json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("decode %s audit log json failed: %w", kind, err)
	}
	payload := &task.AuditLogWriteTaskData{
		Kind:       kind,
		OccurredAt: occurredAt.Format(time.RFC3339Nano),
		Data:       data,
	}
	if err = enqueuer.NewTask(task.AuditLogWriteTaskType, payload,
		asynq.MaxRetry(3), asynq.Timeout(30*time.Second)); err != nil {
		return fmt.Errorf("enqueue %s audit log failed: %w", kind, err)
	}
	return nil
}

// auditLogSyncFallback 入队失败时的统一告警（回退动作由调用方的同步落库闭包承担）。
func auditLogSyncFallback(ctx context.Context, log fallbackLogger, kind string, err error) {
	if log != nil {
		log.Warnf(ctx, "audit log [%s] enqueue failed, falling back to sync write: %s", kind, err.Error())
	}
}

// fallbackLogger 生产者回退告警所需的最小日志面（bLogger.Helper 满足）。
type fallbackLogger interface {
	Warnf(ctx context.Context, format string, args ...any)
}
