package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	monitorAlertV1 "go-wind-admin/api/gen/go/monitor_alert/service/v1"
	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
	redisCacheV1 "go-wind-admin/api/gen/go/redis_cache/service/v1"
	serverMonitorV1 "go-wind-admin/api/gen/go/server_monitor/service/v1"

	"github.com/tx7do/go-crud/viewer"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// alertRecordingNotifier 记录每次 SendDirect 的调用，供"告警/恢复/冷却"断言使用。
type alertRecordingNotifier struct {
	calls []*notificationV1.SendDirectNotificationRequest
}

func (r *alertRecordingNotifier) SendDirect(_ context.Context, req *notificationV1.SendDirectNotificationRequest) (*notificationV1.SendNotificationResponse, error) {
	r.calls = append(r.calls, req)
	return &notificationV1.SendNotificationResponse{
		DeliveryId: 1,
		Status:     notificationV1.DeliveryStatus_SENT,
	}, nil
}

type monitorAlertEnv struct {
	svc         *MonitorAlertService
	repo        *data.MonitorAlertRuleRepo
	notifier    *alertRecordingNotifier
	ctx         context.Context
	serverInfo  *serverMonitorV1.ServerMonitorInfo
	redisInfo   *redisCacheV1.RedisCacheMonitorInfo
	serverFails bool
	redisFails  bool
}

func newMonitorAlertEnv(t *testing.T) *monitorAlertEnv {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	repo := data.NewMonitorAlertRuleRepoForTest(entClient)
	notifier := &alertRecordingNotifier{}

	svc := &MonitorAlertService{
		repo:     repo,
		notifier: notifier,
		log:      bLogger.NewHelper(bLogger.NopLogger()),
	}
	env := &monitorAlertEnv{
		svc:      svc,
		repo:     repo,
		notifier: notifier,
		ctx:      viewer.WithSystemContext(context.Background()),
		serverInfo: &serverMonitorV1.ServerMonitorInfo{
			Go:       &serverMonitorV1.GoRuntimeInfo{NumGoroutine: trans.Ptr(uint32(100)), MemAllocBytes: trans.Ptr(int64(512 << 20))},
			Database: &serverMonitorV1.DatabaseInfo{PingOk: trans.Ptr(true), OpenConnections: trans.Ptr(uint32(10))},
		},
		redisInfo: &redisCacheV1.RedisCacheMonitorInfo{DbSize: 500},
	}
	svc.collectServerFn = func(_ context.Context) (*serverMonitorV1.ServerMonitorInfo, error) {
		if env.serverFails {
			return nil, context.DeadlineExceeded
		}
		return env.serverInfo, nil
	}
	svc.collectRedisFn = func(_ context.Context) (*redisCacheV1.RedisCacheMonitorInfo, error) {
		if env.redisFails {
			return nil, context.DeadlineExceeded
		}
		return env.redisInfo, nil
	}
	return env
}

// TestMonitorAlertEvaluateLifecycle 钉住评估内核的状态机：
// 首次越限告警 → 持续越限冷却内不重发 → 冷却后重发 → 回到阈值内发恢复 → 平稳轮静默。
func TestMonitorAlertEvaluateLifecycle(t *testing.T) {
	env := newMonitorAlertEnv(t)
	ctx := env.ctx

	ruleID, err := env.repo.Create(ctx, &monitorAlertV1.MonitorAlertRule{
		Name:            trans.Ptr("goroutine 告警"),
		Metric:          monitorAlertV1.MonitorMetric_GO_GOROUTINES.Enum(),
		Op:              monitorAlertV1.AlertOp_GE.Enum(),
		Threshold:       trans.Ptr(float64(1000)),
		CooldownMinutes: trans.Ptr(uint32(30)),
		Channel:         notificationV1.Channel_EMAIL.Enum(),
		Target:          trans.Ptr("ops@example.com"),
		IsEnabled:       trans.Ptr(true),
	}, 1)
	require.NoError(t, err)

	// 第 1 轮：未越限 → 静默
	outcomes := env.svc.evaluateOnce(ctx)
	require.Len(t, outcomes, 1)
	require.False(t, outcomes[0].firing)
	require.False(t, outcomes[0].notified)
	require.Empty(t, env.notifier.calls)

	// 第 2 轮：越限（goroutine 2000 ≥ 1000）→ 首次告警
	env.serverInfo.Go.NumGoroutine = trans.Ptr(uint32(2000))
	outcomes = env.svc.evaluateOnce(ctx)
	require.True(t, outcomes[0].firing)
	require.True(t, outcomes[0].notified)
	require.Len(t, env.notifier.calls, 1)
	require.Contains(t, env.notifier.calls[0].Title, "告警")
	require.Equal(t, notificationV1.EventType_MONITOR_ALERT, env.notifier.calls[0].EventType)
	require.Equal(t, "ops@example.com", env.notifier.calls[0].Target)

	// 第 3 轮：持续越限、冷却未到 → 不重发
	outcomes = env.svc.evaluateOnce(ctx)
	require.True(t, outcomes[0].firing)
	require.False(t, outcomes[0].notified)
	require.Len(t, env.notifier.calls, 1)

	// 第 4 轮：恢复 → 发恢复通知，firing 清零
	env.serverInfo.Go.NumGoroutine = trans.Ptr(uint32(100))
	outcomes = env.svc.evaluateOnce(ctx)
	require.False(t, outcomes[0].firing)
	require.True(t, outcomes[0].notified)
	require.Len(t, env.notifier.calls, 2)
	require.Contains(t, env.notifier.calls[1].Title, "已恢复")

	// 第 5 轮：平稳 → 静默
	outcomes = env.svc.evaluateOnce(ctx)
	require.False(t, outcomes[0].firing)
	require.False(t, outcomes[0].notified)
	require.Len(t, env.notifier.calls, 2)

	_ = ruleID
}

// TestMonitorAlertCooldownElapsed 冷却到期重发：把 last_alerted_at 拨回冷却窗口之外。
func TestMonitorAlertCooldownElapsed(t *testing.T) {
	env := newMonitorAlertEnv(t)
	ctx := env.ctx

	_, err := env.repo.Create(ctx, &monitorAlertV1.MonitorAlertRule{
		Name:            trans.Ptr("内存告警"),
		Metric:          monitorAlertV1.MonitorMetric_GO_MEM_ALLOC_MB.Enum(),
		Op:              monitorAlertV1.AlertOp_GE.Enum(),
		Threshold:       trans.Ptr(float64(100)), // MB
		CooldownMinutes: trans.Ptr(uint32(30)),
		Channel:         notificationV1.Channel_EMAIL.Enum(),
		Target:          trans.Ptr("ops@example.com"),
		IsEnabled:       trans.Ptr(true),
	}, 1)
	require.NoError(t, err)

	// 首次越限告警（512MB ≥ 100MB）
	outcomes := env.svc.evaluateOnce(ctx)
	require.True(t, outcomes[0].notified)
	require.Len(t, env.notifier.calls, 1)

	// 把 last_alerted_at 拨到 31 分钟前 → 冷却已过，重发
	rows, err := env.repo.List(ctx, &paginationV1.PagingRequest{NoPaging: trans.Ptr(true)})
	require.NoError(t, err)
	require.Len(t, rows.Items, 1)
	// 评估内核从 DB 读 last_alerted_at；直接借 MarkFiring 把时间拨回
	thirtyOneMinAgo := time.Now().Add(-31 * time.Minute)
	require.NoError(t, env.repo.MarkFiring(ctx, rows.Items[0].GetId(), true, trans.Ptr(512.0), &thirtyOneMinAgo))

	outcomes = env.svc.evaluateOnce(ctx)
	require.True(t, outcomes[0].firing)
	require.True(t, outcomes[0].notified)
	require.Contains(t, outcomes[0].reason, "cooldown")
	require.Len(t, env.notifier.calls, 2)
}

// TestMonitorAlertPingFailAndResolve DB_PING_FAIL 布尔指标：ping 失败即告警，恢复即发恢复。
func TestMonitorAlertPingFailAndResolve(t *testing.T) {
	env := newMonitorAlertEnv(t)
	ctx := env.ctx

	_, err := env.repo.Create(ctx, &monitorAlertV1.MonitorAlertRule{
		Name:            trans.Ptr("数据库连通性"),
		Metric:          monitorAlertV1.MonitorMetric_DB_PING_FAIL.Enum(),
		Op:              monitorAlertV1.AlertOp_GE.Enum(),
		Threshold:       trans.Ptr(float64(1)),
		CooldownMinutes: trans.Ptr(uint32(30)),
		Channel:         notificationV1.Channel_WEBHOOK.Enum(),
		Target:          trans.Ptr("https://hooks.example.com/alert"),
		IsEnabled:       trans.Ptr(true),
	}, 1)
	require.NoError(t, err)

	// ping 失败 → 告警
	env.serverInfo.Database.PingOk = trans.Ptr(false)
	outcomes := env.svc.evaluateOnce(ctx)
	require.True(t, outcomes[0].firing)
	require.True(t, outcomes[0].notified)
	require.Len(t, env.notifier.calls, 1)
	require.Equal(t, notificationV1.Channel_WEBHOOK, env.notifier.calls[0].GetChannel())

	// ping 恢复 → 恢复通知
	env.serverInfo.Database.PingOk = trans.Ptr(true)
	outcomes = env.svc.evaluateOnce(ctx)
	require.False(t, outcomes[0].firing)
	require.True(t, outcomes[0].notified)
	require.Len(t, env.notifier.calls, 2)
}

// TestMonitorAlertCollectFailKeepsState 采集失败轮：跳过规则且不发"恢复"（抖动不是恢复）。
func TestMonitorAlertCollectFailKeepsState(t *testing.T) {
	env := newMonitorAlertEnv(t)
	ctx := env.ctx

	_, err := env.repo.Create(ctx, &monitorAlertV1.MonitorAlertRule{
		Name:            trans.Ptr("redis 规模"),
		Metric:          monitorAlertV1.MonitorMetric_REDIS_DB_SIZE.Enum(),
		Op:              monitorAlertV1.AlertOp_GE.Enum(),
		Threshold:       trans.Ptr(float64(100)),
		CooldownMinutes: trans.Ptr(uint32(30)),
		Channel:         notificationV1.Channel_EMAIL.Enum(),
		Target:          trans.Ptr("ops@example.com"),
		IsEnabled:       trans.Ptr(true),
	}, 1)
	require.NoError(t, err)

	// 越限告警
	outcomes := env.svc.evaluateOnce(ctx)
	require.True(t, outcomes[0].notified)
	require.Len(t, env.notifier.calls, 1)

	// Redis 采集失败一轮：跳过（沿用上次 firing）且不误发恢复
	env.redisFails = true
	outcomes = env.svc.evaluateOnce(ctx)
	require.True(t, outcomes[0].firing, "采集失败轮应沿用上次的 firing 状态")
	require.False(t, outcomes[0].notified)
	require.Len(t, env.notifier.calls, 1)

	// 采集恢复：仍越限，冷却未到 → 静默
	env.redisFails = false
	outcomes = env.svc.evaluateOnce(ctx)
	require.True(t, outcomes[0].firing)
	require.False(t, outcomes[0].notified)
	require.Len(t, env.notifier.calls, 1)
}
