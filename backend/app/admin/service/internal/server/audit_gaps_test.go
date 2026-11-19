// 本文件补 internal/server 既有测试未覆盖的分支：
//   - enqueueAuditLog 的 nil enqueuer 守卫；
//   - auditLogSyncFallback（nil logger 无害跳过 / 桩 logger 收到 kind+err）；
//   - HandleAuditLogWrite 坏载荷：nil data / 空 kind / OccurredAt 非法回退 now；
//   - unmarshalAuditDTO：值类型不匹配报错、DiscardUnknown 语义（未知键忽略）；
//   - occurredAtToTimestamp；
//   - evalLoggingEngine.IsAuthorized 全链（enttest SQLite）：入队成功不直接落库、
//     入队失败回退同步落库、身份字段经 auth claims 注入；
//   - resolvePolicyCached 的 TTL 缓存：未过期命中不查库、过期重查。
package server

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	authzEngine "github.com/tx7do/kratos-authz/engine"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
	"go-wind-admin/pkg/task"
)

// int64Ptr 为 *int64 字段构造指针。
func int64Ptr(v int64) *int64 { return &v }

// TestEnqueueAuditLog_NilEnqueuer enqueuer 未配置时返回明确错误——
// 调用方据此走同步回退，而非静默丢弃审计记录。
func TestEnqueueAuditLog_NilEnqueuer(t *testing.T) {
	err := enqueueAuditLog(context.Background(), nil, task.AuditLogKindApi,
		&auditV1.ApiAuditLog{}, time.Now())
	require.ErrorContains(t, err, "audit log enqueuer is not configured")
}

// stubFallbackLogger 记录 Warnf 调用的桩。
type stubFallbackLogger struct {
	calls []string
}

func (s *stubFallbackLogger) Warnf(_ context.Context, format string, args ...any) {
	s.calls = append(s.calls, fmt.Sprintf(format, args...))
}

// TestAuditLogSyncFallback 回退告警：桩 logger 收到 kind 与原始错误；nil logger 无害跳过。
func TestAuditLogSyncFallback(t *testing.T) {
	// nil logger：不 panic，不产生任何行为。
	auditLogSyncFallback(context.Background(), nil, task.AuditLogKindApi, errors.New("boom"))

	stub := &stubFallbackLogger{}
	auditLogSyncFallback(context.Background(), stub, task.AuditLogKindLogin, errors.New("queue down"))
	require.Len(t, stub.calls, 1)
	require.Contains(t, stub.calls[0], "login")
	require.Contains(t, stub.calls[0], "queue down")
}

// TestHandleAuditLogWrite_BadPayload 坏载荷分支：nil data / 空 kind 报 kind 必填；
// OccurredAt 非法时回退 now 落库（不因解析失败而丢审计行）。
func TestHandleAuditLogWrite_BadPayload(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	handler := NewAuditLogWriteHandler(nil,
		data.NewApiAuditLogRepoForTest(entClient), nil, nil, nil, nil, nil)

	require.ErrorContains(t, handler.HandleAuditLogWrite(task.AuditLogWriteTaskType, nil), "kind is required")
	require.ErrorContains(t, handler.HandleAuditLogWrite(task.AuditLogWriteTaskType,
		&task.AuditLogWriteTaskData{Kind: ""}), "kind is required")

	// OccurredAt 非法：回退 now 落库成功。
	err := handler.HandleAuditLogWrite(task.AuditLogWriteTaskType, &task.AuditLogWriteTaskData{
		Kind:       task.AuditLogKindApi,
		OccurredAt: "not-a-timestamp",
		Data:       map[string]any{"httpMethod": "GET", "path": "/x"},
	})
	require.NoError(t, err)

	rows, err := entClient.Client().ApiAuditLog.Query().All(enttest.NewSystemContext(context.Background()))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.NotNil(t, rows[0].CreatedAt)
	require.WithinDuration(t, time.Now(), *rows[0].CreatedAt, 30*time.Second,
		"bad OccurredAt must fall back to now")
}

// TestUnmarshalAuditDTO 解码边界：值类型不匹配报错；DiscardUnknown 忽略未知键。
func TestUnmarshalAuditDTO(t *testing.T) {
	// 类型不匹配：bool 字段给了字符串。
	err := unmarshalAuditDTO(map[string]any{"success": "not-bool"}, &auditV1.ApiAuditLog{})
	require.ErrorContains(t, err, "unmarshal dto failed")

	// DiscardUnknown：未知键忽略，不报错。
	dto := &auditV1.ApiAuditLog{}
	require.NoError(t, unmarshalAuditDTO(map[string]any{
		"success": true, "unknownField": "whatever",
	}, dto))
	require.True(t, dto.GetSuccess())
}

func TestOccurredAtToTimestamp(t *testing.T) {
	at := time.Date(2026, 11, 9, 12, 0, 0, 0, time.Local)
	ts := occurredAtToTimestamp(at)
	require.NotNil(t, ts)
	require.True(t, ts.AsTime().Equal(at))
	require.NotNil(t, occurredAtToTimestamp(time.Time{}), "零值时间同样产出非 nil Timestamp（repo honor 语义）")
}

// --- evalLoggingEngine 全链 ---

// stubInnerAuthorizer 可编程的鉴权引擎桩（inner 引擎）。
type stubInnerAuthorizer struct {
	allowed bool
	err     error
}

func (s *stubInnerAuthorizer) Name() string { return "stub" }
func (s *stubInnerAuthorizer) ProjectsAuthorized(context.Context, authzEngine.Subjects, authzEngine.Action, authzEngine.Resource, authzEngine.Projects) (authzEngine.Projects, error) {
	return nil, nil
}
func (s *stubInnerAuthorizer) FilterAuthorizedPairs(context.Context, authzEngine.Subjects, authzEngine.Pairs) (authzEngine.Pairs, error) {
	return nil, nil
}
func (s *stubInnerAuthorizer) FilterAuthorizedProjects(context.Context, authzEngine.Subjects) (authzEngine.Projects, error) {
	return nil, nil
}
func (s *stubInnerAuthorizer) IsAuthorized(context.Context, authzEngine.Subject, authzEngine.Action, authzEngine.Resource, authzEngine.Project) (bool, error) {
	return s.allowed, s.err
}

// TestEvalLoggingEngine_EnqueueSuccessSkipsDirectWrite 入队成功：评估记录走异步，
// 不直接落库；记录内容（判定/身份/上下文）完整。
func TestEvalLoggingEngine_EnqueueSuccessSkipsDirectWrite(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := data.NewPolicyEvaluationLogRepoForTest(entClient)
	stub := &stubAuditEnqueuer{}
	eng := newEvalLoggingEngine(&stubInnerAuthorizer{allowed: true}, repo, stub)

	// 注入 auth claims（UserId/TenantId 来源）。
	payload := &authenticationV1.UserTokenPayload{UserId: 33, TenantId: int64Ptr(5)}
	ctx := auth.NewContext(context.Background(), payload)

	allowed, err := eng.IsAuthorized(ctx, "ROLE_ADMIN", "GET", "/admin/v1/roles", "")
	require.NoError(t, err)
	require.True(t, allowed)

	// 入队成功：payload kind 正确；DB 无直接落库行。
	require.Len(t, stub.typeNames, 1)
	require.Equal(t, task.AuditLogWriteTaskType, stub.typeNames[0])
	recPayload, ok := stub.payloads[0].(*task.AuditLogWriteTaskData)
	require.True(t, ok)
	require.Equal(t, task.AuditLogKindPolicyEvaluation, recPayload.Kind)

	count, err := entClient.Client().PolicyEvaluationLog.Query().Count(enttest.NewSystemContext(context.Background()))
	require.NoError(t, err)
	require.Zero(t, count, "enqueue success must not write the DB directly")

	logged := &permissionV1.PolicyEvaluationLog{}
	require.NoError(t, unmarshalAuditDTO(recPayload.Data, logged))
	require.True(t, logged.GetResult())
	require.Equal(t, int64(33), logged.GetUserId())
	require.Equal(t, int64(5), logged.GetTenantId())
	require.Equal(t, "/admin/v1/roles", logged.GetRequestPath())
	require.Equal(t, "GET", logged.GetRequestMethod())
	require.Contains(t, logged.GetEffectDetails(), "allowed")
	require.Contains(t, logged.GetEvaluationContext(), "ROLE_ADMIN")
}

// TestEvalLoggingEngine_EnqueueFailureFallsBackToSync 入队失败：回退同步落库，
// 判定结果不受日志链路影响。
func TestEvalLoggingEngine_EnqueueFailureFallsBackToSync(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := data.NewPolicyEvaluationLogRepoForTest(entClient)
	stub := &stubAuditEnqueuer{failNext: true}
	eng := newEvalLoggingEngine(&stubInnerAuthorizer{allowed: false}, repo, stub)

	allowed, err := eng.IsAuthorized(context.Background(), "ROLE_GUEST", "POST", "/admin/v1/roles", "")
	require.NoError(t, err, "日志链路失败不得影响鉴权判定")
	require.False(t, allowed)

	// 回退同步落库：行存在且结果为 denied。
	ctx := enttest.NewSystemContext(context.Background())
	rows, qerr := entClient.Client().PolicyEvaluationLog.Query().All(ctx)
	require.NoError(t, qerr)
	require.Len(t, rows, 1)
	require.NotNil(t, rows[0].Result)
	require.False(t, *rows[0].Result)
	require.NotNil(t, rows[0].EffectDetails)
	require.Contains(t, *rows[0].EffectDetails, "denied")
}

// TestEvalLoggingEngine_InnerErrorCarried inner 引擎报错：判定与错误原样透传，
// effect_details 附错误文本。
func TestEvalLoggingEngine_InnerErrorCarried(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := data.NewPolicyEvaluationLogRepoForTest(entClient)
	innerErr := errors.New("policy store unavailable")
	eng := newEvalLoggingEngine(&stubInnerAuthorizer{allowed: false, err: innerErr}, repo, nil)

	allowed, err := eng.IsAuthorized(context.Background(), "ROLE_X", "GET", "/p", "")
	require.ErrorIs(t, err, innerErr)
	require.False(t, allowed)

	rows, qerr := entClient.Client().PolicyEvaluationLog.Query().All(enttest.NewSystemContext(context.Background()))
	require.NoError(t, qerr)
	require.Len(t, rows, 1)
	require.NotNil(t, rows[0].EffectDetails)
	require.Contains(t, *rows[0].EffectDetails, "policy store unavailable")
}

// TestEvalLoggingEngine_NilGuards newEvalLoggingEngine 的 nil 守卫：repo 缺失时原样返回 inner。
func TestEvalLoggingEngine_NilGuards(t *testing.T) {
	inner := &stubInnerAuthorizer{}
	// repo 为 nil：原样返回 inner（不埋点）。
	require.Same(t, inner, newEvalLoggingEngine(inner, nil, nil))
}

// TestResolvePolicyCached_TTL TTL 缓存语义：未过期命中不查库（空库下命中预置值即证明）、
// 过期条目重查、查询后写缓存。
func TestResolvePolicyCached_TTL(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := data.NewPolicyEvaluationLogRepoForTest(entClient)
	eng, ok := newEvalLoggingEngine(&stubInnerAuthorizer{}, repo, nil).(*evalLoggingEngine)
	require.True(t, ok)
	ctx := enttest.NewSystemContext(context.Background())

	// 预置未过期缓存条目：命中即返回预置值（空库直查只会得到 0,0）。
	key := "ROLE_ADMIN|/admin/v1/roles|GET"
	eng.resolveCache[key] = resolveResult{
		permissionID: 11, policyID: 22,
		expiresAt: time.Now().Add(time.Hour),
	}
	pid, pol := eng.resolvePolicyCached(ctx, "ROLE_ADMIN", "/admin/v1/roles", "get")
	require.Equal(t, int64(11), pid)
	require.Equal(t, int64(22), pol)

	// 过期条目：重查（空库 → 0,0）并写回缓存。
	eng.resolveCache[key] = resolveResult{
		permissionID: 11, policyID: 22,
		expiresAt: time.Now().Add(-time.Minute),
	}
	pid, pol = eng.resolvePolicyCached(ctx, "ROLE_ADMIN", "/admin/v1/roles", "GET")
	require.Zero(t, pid)
	require.Zero(t, pol)

	eng.resolveMu.Lock()
	cached, exists := eng.resolveCache[key]
	eng.resolveMu.Unlock()
	require.True(t, exists)
	require.Zero(t, cached.permissionID)
	require.True(t, cached.expiresAt.After(time.Now()), "re-resolved entry must carry a fresh TTL")
}

// --- 基准 ---

// BenchmarkEnqueueAuditLog 每条审计记录的 protojson→map→载荷构造成本。
func BenchmarkEnqueueAuditLog(b *testing.B) {
	stub := &stubAuditEnqueuer{}
	dto := &auditV1.ApiAuditLog{
		HttpMethod: strPtr("GET"),
		Path:       strPtr("/admin/v1/users"),
		Success:    boolPtr(true),
	}
	at := timestamppb.Now()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := enqueueAuditLog(ctx, stub, task.AuditLogKindApi, dto, at.AsTime()); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshalAuditDTO 消费侧每条记录的 map→typed DTO 解码成本。
func BenchmarkUnmarshalAuditDTO(b *testing.B) {
	dataMap := map[string]any{
		"httpMethod": "GET", "path": "/admin/v1/users", "success": true,
		"tenantId": "7", "userId": "42", "traceId": "0af7651916cd43dd8448eb211c80319c",
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := unmarshalAuditDTO(dataMap, &auditV1.ApiAuditLog{}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkResolvePolicyCached 反查缓存命中路径（含互斥锁与 key 拼接）。
func BenchmarkResolvePolicyCached(b *testing.B) {
	entClient := enttest.NewEntClientForTest(b)
	repo := data.NewPolicyEvaluationLogRepoForTest(entClient)
	eng, ok := newEvalLoggingEngine(&stubInnerAuthorizer{}, repo, nil).(*evalLoggingEngine)
	require.True(b, ok)
	key := "ROLE_ADMIN|/admin/v1/roles|GET"
	eng.resolveCache[key] = resolveResult{
		permissionID: 1, policyID: 1,
		expiresAt: time.Now().Add(time.Hour),
	}
	ctx := enttest.NewSystemContext(context.Background())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eng.resolvePolicyCached(ctx, "ROLE_ADMIN", "/admin/v1/roles", "GET")
	}
}
