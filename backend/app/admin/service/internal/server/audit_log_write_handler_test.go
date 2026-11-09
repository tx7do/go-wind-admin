package server

import (
	"context"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/task"
)

// stubAuditEnqueuer 捕获入队载荷的桩。
type stubAuditEnqueuer struct {
	typeNames []string
	payloads  []any
	failNext  bool
}

func (s *stubAuditEnqueuer) NewTask(typeName string, msg any, _ ...asynq.Option) error {
	if s.failNext {
		return assertEnqueueFailure
	}
	s.typeNames = append(s.typeNames, typeName)
	s.payloads = append(s.payloads, msg)
	return nil
}

var assertEnqueueFailure = errStubEnqueue{}

type errStubEnqueue struct{}

func (errStubEnqueue) Error() string { return "enqueue failed (stub)" }

// TestEnqueueAuditLog_ProducerPayload producer 单测：DTO 经 protojson→map 往返，
// 载荷 kind/occurredAt/data 形态正确；入队失败向上返回错误。
func TestEnqueueAuditLog_ProducerPayload(t *testing.T) {
	stub := &stubAuditEnqueuer{}
	occurredAt := time.Date(2026, 11, 9, 12, 0, 0, 0, time.Local)

	dto := &auditV1.ApiAuditLog{
		HttpMethod: strPtr("GET"),
		Path:       strPtr("/admin/v1/ai/agents"),
		Success:    boolPtr(true),
	}
	require.NoError(t, enqueueAuditLog(context.Background(), stub, task.AuditLogKindApi, dto, occurredAt))
	require.Len(t, stub.typeNames, 1)
	require.Equal(t, task.AuditLogWriteTaskType, stub.typeNames[0])

	payload, ok := stub.payloads[0].(*task.AuditLogWriteTaskData)
	require.True(t, ok, "payload 应为 *AuditLogWriteTaskData")
	require.Equal(t, task.AuditLogKindApi, payload.Kind)
	require.NotEmpty(t, payload.OccurredAt)
	parsed, err := time.Parse(time.RFC3339Nano, payload.OccurredAt)
	require.NoError(t, err)
	require.True(t, parsed.Equal(occurredAt), "事件时间应无损保留")
	require.Equal(t, "GET", payload.Data["httpMethod"])
	require.Equal(t, true, payload.Data["success"])

	// 入队失败应向上返回错误（调用方据此回退同步写）
	stub.failNext = true
	require.Error(t, enqueueAuditLog(context.Background(), stub, task.AuditLogKindApi, dto, occurredAt))
}

func strPtr(s string) *string    { return &s }
func boolPtr(b bool) *bool       { return &b }
func uint32Ptr(v uint32) *uint32 { return &v }

// TestAuditLogWriteHandler_Sqlite_AllKinds 消费者逐 kind 回归：
// protojson map 解回 DTO → 落对应审计表；api/login 以 OccurredAt 覆写 CreatedAt；
// 未知 kind 报错。
func TestAuditLogWriteHandler_Sqlite_AllKinds(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	ctx := enttest.NewSystemContext(context.Background())
	handler := NewAuditLogWriteHandler(nil,
		data.NewApiAuditLogRepoForTest(entClient),
		data.NewLoginAuditLogRepoForTest(entClient),
		data.NewOperationAuditLogRepoForTest(entClient),
		data.NewPermissionAuditLogRepoForTest(entClient),
		data.NewDataAccessAuditLogRepoForTest(entClient),
		data.NewPolicyEvaluationLogRepoForTest(entClient),
	)

	occurredAt := time.Date(2026, 11, 9, 12, 0, 0, 0, time.Local)
	payloadOf := func(kind string, extra map[string]any) *task.AuditLogWriteTaskData {
		data := map[string]any{"tenantId": "7"}
		for k, v := range extra {
			data[k] = v
		}
		return &task.AuditLogWriteTaskData{Kind: kind, OccurredAt: occurredAt.Format(time.RFC3339Nano), Data: data}
	}

	// api：落库 + CreatedAt=事件时间
	require.NoError(t, handler.HandleAuditLogWrite(task.AuditLogWriteTaskType,
		payloadOf(task.AuditLogKindApi, map[string]any{"httpMethod": "GET", "path": "/x", "success": true})))
	apiRow, err := entClient.Client().ApiAuditLog.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "GET", *apiRow.HTTPMethod)
	require.Equal(t, occurredAt.Unix(), apiRow.CreatedAt.Unix(), "api CreatedAt 应为事件时间")

	// login：落库 + CreatedAt=事件时间
	require.NoError(t, handler.HandleAuditLogWrite(task.AuditLogWriteTaskType,
		payloadOf(task.AuditLogKindLogin, map[string]any{"username": "admin", "success": true})))
	loginRow, err := entClient.Client().LoginAuditLog.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "admin", *loginRow.Username)
	require.Equal(t, occurredAt.Unix(), loginRow.CreatedAt.Unix(), "login CreatedAt 应为事件时间")

	// operation / permission / data_access / policy_evaluation：各行落库
	require.NoError(t, handler.HandleAuditLogWrite(task.AuditLogWriteTaskType,
		payloadOf(task.AuditLogKindOperation, map[string]any{"httpMethod": "POST"})))
	require.NoError(t, handler.HandleAuditLogWrite(task.AuditLogWriteTaskType,
		payloadOf(task.AuditLogKindPermission, map[string]any{"resourceType": "role"})))
	require.NoError(t, handler.HandleAuditLogWrite(task.AuditLogWriteTaskType,
		payloadOf(task.AuditLogKindDataAccess, map[string]any{"sqlDigest": "dig-1"})))
	require.NoError(t, handler.HandleAuditLogWrite(task.AuditLogWriteTaskType,
		payloadOf(task.AuditLogKindPolicyEvaluation, map[string]any{"requestPath": "/x"})))

	// 未知 kind：报错
	require.Error(t, handler.HandleAuditLogWrite(task.AuditLogWriteTaskType,
		&task.AuditLogWriteTaskData{Kind: "unknown", Data: map[string]any{}}))
}
