package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/go-crud/viewer"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/pkg/audit"
	"go-wind-admin/pkg/task"
)

// AuditLogWriteHandler 审计日志异步写任务的消费者：按 payload.kind 把
// protojson 编码的审计 DTO 解回 typed 结构，落对应审计表。
// 落库 ctx 保持与同步路径相同的不变量（SystemContext + audit.SinkKey 防递归标记）。
type AuditLogWriteHandler struct {
	apiAuditLogRepo         *data.ApiAuditLogRepo
	loginLogRepo            *data.LoginAuditLogRepo
	operationAuditLogRepo   *data.OperationAuditLogRepo
	permissionAuditLogRepo  *data.PermissionAuditLogRepo
	dataAccessAuditLogRepo  *data.DataAccessAuditLogRepo
	policyEvaluationLogRepo *data.PolicyEvaluationLogRepo
}

func NewAuditLogWriteHandler(
	ctx *bootstrap.Context,
	apiAuditLogRepo *data.ApiAuditLogRepo,
	loginLogRepo *data.LoginAuditLogRepo,
	operationAuditLogRepo *data.OperationAuditLogRepo,
	permissionAuditLogRepo *data.PermissionAuditLogRepo,
	dataAccessAuditLogRepo *data.DataAccessAuditLogRepo,
	policyEvaluationLogRepo *data.PolicyEvaluationLogRepo,
) *AuditLogWriteHandler {
	return &AuditLogWriteHandler{
		apiAuditLogRepo:         apiAuditLogRepo,
		loginLogRepo:            loginLogRepo,
		operationAuditLogRepo:   operationAuditLogRepo,
		permissionAuditLogRepo:  permissionAuditLogRepo,
		dataAccessAuditLogRepo:  dataAccessAuditLogRepo,
		policyEvaluationLogRepo: policyEvaluationLogRepo,
	}
}

// occurredAtToTimestamp 事件时间转 proto Timestamp（值恒非 nil，便于 repo honor）。
func occurredAtToTimestamp(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}

// HandleAuditLogWrite asynq 消费入口。
// at-least-once：落库失败返回 error 交 asynq 重试（极小概率重复被接受，审计场景
// 宁可重复不可丢失）；落库成功后返回 nil。
func (h *AuditLogWriteHandler) HandleAuditLogWrite(taskType string, data *task.AuditLogWriteTaskData) error {
	if data == nil || data.Kind == "" {
		return fmt.Errorf("audit log write: kind is required")
	}

	// 与同步路径相同的落库不变量：防递归 SinkKey + 系统 viewer。
	ctx := viewer.WithSystemContext(context.WithValue(context.Background(), audit.SinkKey(), true))

	occurredAt, parErr := time.Parse(time.RFC3339Nano, data.OccurredAt)
	if parErr != nil {
		occurredAt = time.Now()
	}

	switch data.Kind {
	case task.AuditLogKindApi:
		dto := &auditV1.ApiAuditLog{}
		if err := unmarshalAuditDTO(data.Data, dto); err != nil {
			return err
		}
		dto.CreatedAt = occurredAtToTimestamp(occurredAt) // 事件时间（repo honor 非空 CreatedAt）
		return h.apiAuditLogRepo.Create(ctx, &auditV1.CreateApiAuditLogRequest{Data: dto})

	case task.AuditLogKindLogin:
		dto := &auditV1.LoginAuditLog{}
		if err := unmarshalAuditDTO(data.Data, dto); err != nil {
			return err
		}
		dto.CreatedAt = occurredAtToTimestamp(occurredAt)
		return h.loginLogRepo.Create(ctx, &auditV1.CreateLoginAuditLogRequest{Data: dto})

	case task.AuditLogKindOperation:
		dto := &auditV1.OperationAuditLog{}
		if err := unmarshalAuditDTO(data.Data, dto); err != nil {
			return err
		}
		return h.operationAuditLogRepo.Create(ctx, &auditV1.CreateOperationAuditLogRequest{Data: dto})

	case task.AuditLogKindPermission:
		dto := &auditV1.PermissionAuditLog{}
		if err := unmarshalAuditDTO(data.Data, dto); err != nil {
			return err
		}
		return h.permissionAuditLogRepo.Create(ctx, &auditV1.CreatePermissionAuditLogRequest{Data: dto})

	case task.AuditLogKindDataAccess:
		dto := &auditV1.DataAccessAuditLog{}
		if err := unmarshalAuditDTO(data.Data, dto); err != nil {
			return err
		}
		return h.dataAccessAuditLogRepo.Create(ctx, &auditV1.CreateDataAccessAuditLogRequest{Data: dto})

	case task.AuditLogKindPolicyEvaluation:
		dto := &permissionV1.PolicyEvaluationLog{}
		if err := unmarshalAuditDTO(data.Data, dto); err != nil {
			return err
		}
		return h.policyEvaluationLogRepo.Create(ctx, &permissionV1.CreatePolicyEvaluationLogRequest{Data: dto})

	default:
		return fmt.Errorf("audit log write: unknown kind %q", data.Kind)
	}
}

// unmarshalAuditDTO 把 payload 里的 protojson map 解回 typed DTO。
func unmarshalAuditDTO(data map[string]any, dto proto.Message) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("audit log write: re-marshal payload failed: %w", err)
	}
	opts := protojson.UnmarshalOptions{DiscardUnknown: true}
	if uerr := opts.Unmarshal(raw, dto); uerr != nil {
		return fmt.Errorf("audit log write: unmarshal dto failed: %w", uerr)
	}
	return nil
}
