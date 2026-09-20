// 通知异步派发（规则行的 is_async + notification_dispatch 任务）的 SQLite 集成测试。
//
// 覆盖的是"什么时候离开请求、离开之后怎么闭环"这几件事：
//   - 异步事件在请求里**不拨号**：只入队，台账停在 SENDING，attempts 还是 0；
//   - 载荷只带 target/title/content，且 target 是明文（台账那一份是脱敏的，投不了递）；
//   - handler 的幂等门：结论已定案的行重投一次，不会再发第二封；
//   - 重试额度：还有额度的失败必须留在 SENDING（定案 FAILED 会让重投撞上幂等门，额度形同虚设）；
//   - 配置类错误走 asynq.SkipRetry，且载荷（带明文验证码）不进归档；
//   - 预检失败压根不入队，找回密码的回话照旧是当场结论；
//   - 入队失败回退成当场投递，通知不因为 Redis 挂了而丢；
//   - 同步事件（测试邮件 / 站内信）在有队列的情况下依然不入队。
//
// 与 notification_service_sqlite_test.go 的分工：那一个测"投递结果怎么写台账"，
// 这一个测"投递什么时候发生"。
package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"

	"go-wind-admin/app/admin/service/internal/data/channel"

	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"

	"go-wind-admin/pkg/task"
)

// notificationEnqueuer 是 TaskEnqueuer 替身：记下载荷本体，并可编排成"入队失败"。
// 与 internal_message 侧的 recordingTaskEnqueuer 分开，是因为这里要断言的是载荷内容而不只是次数。
type notificationEnqueuer struct {
	calls []enqueuedNotification
	err   error
}

type enqueuedNotification struct {
	typeName string
	data     *task.NotificationDispatchTaskData
}

func (e *notificationEnqueuer) NewTask(typeName string, msg any, _ ...asynq.Option) error {
	if e.err != nil {
		return e.err
	}
	payload, ok := msg.(*task.NotificationDispatchTaskData)
	if !ok {
		return fmt.Errorf("unexpected payload type %T", msg)
	}
	e.calls = append(e.calls, enqueuedNotification{typeName: typeName, data: payload})
	return nil
}

// dispatchEnv 复用 notification_service_sqlite_test.go 的替身装配，只多加一个入队替身。
type dispatchEnv struct {
	*notificationSvcEnv
	queue *notificationEnqueuer
}

func newDispatchEnv(t *testing.T) *dispatchEnv {
	t.Helper()

	e := &dispatchEnv{notificationSvcEnv: newNotificationServiceForTest(t)}
	e.queue = &notificationEnqueuer{}
	e.svc.taskEnqueuer = e.queue

	return e
}

// payload 取第 n 次入队的载荷，越界即失败（免得断言写成静默的 nil 检查）。
func (e *dispatchEnv) payload(t *testing.T, n int) *task.NotificationDispatchTaskData {
	t.Helper()
	require.Greater(t, len(e.queue.calls), n, "入队次数不足")
	return e.queue.calls[n].data
}

// TestAsyncDispatchEnqueuesWithoutSending 异步事件在请求里只入队：不拨号、台账停在 SENDING。
func TestNotificationServiceSqlite_AsyncDispatchEnqueuesWithoutSending(t *testing.T) {
	e := newDispatchEnv(t)

	resp, err := e.svc.SendDirect(e.ctx, directReq(notificationV1.EventType_PASSWORD_RESET_CODE))
	require.NoError(t, err, "找回密码请求不该等 SMTP")
	require.Equal(t, notificationV1.DeliveryStatus_SENDING, resp.GetStatus())
	require.Empty(t, e.email.calls, "请求里一次都不该拨号")

	require.Len(t, e.queue.calls, 1)
	require.Equal(t, task.NotificationDispatchTaskType, e.queue.calls[0].typeName)

	sent := e.payload(t, 0)
	require.Equal(t, resp.GetDeliveryId(), sent.DeliveryId)
	require.Equal(t, "bob@example.com", sent.Target, "载荷必须带明文地址：台账那一份是脱敏的，投不了递")
	require.Equal(t, "重置验证码", sent.Title)
	require.Equal(t, "您的验证码是 123456", sent.Content)

	got, err := e.svc.GetNotificationDelivery(e.ctx, &notificationV1.GetNotificationDeliveryRequest{Id: resp.GetDeliveryId()})
	require.NoError(t, err)
	require.Equal(t, notificationV1.DeliveryStatus_SENDING, got.GetStatus(), "结论由 handler 回写，请求这一趟不落")
	require.Equal(t, "b***@example.com", got.GetTarget(), "正文与明文地址只进 Redis，台账照旧脱敏")
	require.Zero(t, got.GetAttempts(), "还没尝试过")
	require.NotEmpty(t, got.GetRequestId(), "调用方不传也要有幂等锚")
}

// TestAsyncDispatchHandlerSettlesLedger handler 拿台账状态当幂等门：定案之后重投不再发第二次。
func TestNotificationServiceSqlite_AsyncDispatchHandlerSettlesLedger(t *testing.T) {
	e := newDispatchEnv(t)

	resp, err := e.svc.SendDirect(e.ctx, directReq(notificationV1.EventType_PASSWORD_RESET_CODE))
	require.NoError(t, err)
	sent := e.payload(t, 0)

	require.NoError(t, e.svc.AsyncNotificationDispatch(e.ctx, task.NotificationDispatchTaskType, sent))

	require.Len(t, e.email.calls, 1)
	require.Equal(t, "bob@example.com", e.email.calls[0].Target, "handler 投的是载荷里的明文")

	got, err := e.svc.GetNotificationDelivery(e.ctx, &notificationV1.GetNotificationDeliveryRequest{Id: resp.GetDeliveryId()})
	require.NoError(t, err)
	require.Equal(t, notificationV1.DeliveryStatus_SENT, got.GetStatus())
	require.Equal(t, uint32(1), got.GetAttempts())
	require.Equal(t, uint32(42), got.GetChannelId(), "回执的渠道 ID 由 handler 补进台账")
	require.NotNil(t, got.GetSentAt())

	// asynq 重投 / 归档后手工重跑：载荷一模一样，但台账已定案。
	require.NoError(t, e.svc.AsyncNotificationDispatch(e.ctx, task.NotificationDispatchTaskType, sent))
	require.Len(t, e.email.calls, 1, "已经发出去的通知不能因为重投再发一封")
	got2, err := e.svc.GetNotificationDelivery(e.ctx, &notificationV1.GetNotificationDeliveryRequest{Id: resp.GetDeliveryId()})
	require.NoError(t, err)
	require.Equal(t, uint32(1), got2.GetAttempts(), "被幂等门挡掉时连尝试计数都不该涨")
}

// TestAsyncDispatchKeepsRetryBudget 还有重试额度时不定案：留 SENDING，让重投真的再拨一次号。
func TestNotificationServiceSqlite_AsyncDispatchKeepsRetryBudget(t *testing.T) {
	e := newDispatchEnv(t)
	e.email.err = fmt.Errorf("smtp connect failed: dial tcp: i/o timeout")

	resp, err := e.svc.SendDirect(e.ctx, directReq(notificationV1.EventType_PASSWORD_RESET_CODE))
	require.NoError(t, err)
	sent := e.payload(t, 0)

	for attempt := 1; attempt < notificationDispatchMaxAttempts; attempt++ {
		require.Errorf(t, e.svc.AsyncNotificationDispatch(e.ctx, task.NotificationDispatchTaskType, sent),
			"第 %d 次失败必须返回 error，asynq 才会重试", attempt)

		got, getErr := e.svc.GetNotificationDelivery(e.ctx, &notificationV1.GetNotificationDeliveryRequest{Id: resp.GetDeliveryId()})
		require.NoError(t, getErr)
		require.Equal(t, uint32(attempt), got.GetAttempts())
		require.Equal(t, notificationV1.DeliveryStatus_SENDING, got.GetStatus(),
			"额度没用完就定案 FAILED，重投会被幂等门挡掉，额度等于没有")
		require.Contains(t, got.GetLastError(), "i/o timeout", "没定案也要留下这次的报错，否则排障只能看到 SENDING")

		// 第一次的回执把 channel_id 落进台账，之后每次重试都照着它显式钉住同一条配置：
		// 同一次投递中途换 SMTP 账号，"到底是哪条在抖"就再也问不出来了。
		// （渠道自选本身按 ID 升序取第一个启用的，本来也是确定性的，这里钉住的是同一个答案。）
		wantPinned := uint32(0)
		if attempt > 1 {
			wantPinned = 42
		}
		require.Equal(t, wantPinned, e.email.calls[attempt-1].ChannelID,
			"第 %d 次尝试钉住的渠道配置", attempt)
	}
	require.Len(t, e.email.calls, notificationDispatchMaxAttempts-1)

	// 最后一次：额度用尽，台账闭环为 FAILED。
	require.Error(t, e.svc.AsyncNotificationDispatch(e.ctx, task.NotificationDispatchTaskType, sent))

	got, err := e.svc.GetNotificationDelivery(e.ctx, &notificationV1.GetNotificationDeliveryRequest{Id: resp.GetDeliveryId()})
	require.NoError(t, err)
	require.Equal(t, notificationV1.DeliveryStatus_FAILED, got.GetStatus())
	require.Equal(t, uint32(notificationDispatchMaxAttempts), got.GetAttempts())
	require.Nil(t, got.GetSentAt())

	// 定案后再重投（归档手工重跑）不再拨号。
	require.NoError(t, e.svc.AsyncNotificationDispatch(e.ctx, task.NotificationDispatchTaskType, sent))
	require.Len(t, e.email.calls, notificationDispatchMaxAttempts)
}

// TestAsyncDispatchSkipsRetryOnConfigError 配置类错误跳过重试：载荷带着明文验证码，不该躺进归档。
func TestNotificationServiceSqlite_AsyncDispatchSkipsRetryOnConfigError(t *testing.T) {
	e := newDispatchEnv(t)

	resp, err := e.svc.SendDirect(e.ctx, directReq(notificationV1.EventType_PASSWORD_RESET_CODE))
	require.NoError(t, err)

	// 入队之后、消费之前渠道被关掉：Send 报配置类错误。
	e.email.err = fmt.Errorf("%w: channel [3] is disabled", channel.ErrChannelNotConfigured)
	dispatchErr := e.svc.AsyncNotificationDispatch(e.ctx, task.NotificationDispatchTaskType, e.payload(t, 0))

	require.Error(t, dispatchErr)
	require.True(t, errors.Is(dispatchErr, asynq.SkipRetry),
		"重试不会改变结论，还多留几轮明文验证码在 Redis 里：%v", dispatchErr)

	got, getErr := e.svc.GetNotificationDelivery(e.ctx, &notificationV1.GetNotificationDeliveryRequest{Id: resp.GetDeliveryId()})
	require.NoError(t, getErr)
	require.Equal(t, notificationV1.DeliveryStatus_SKIPPED, got.GetStatus())
	require.Equal(t, uint32(1), got.GetAttempts())
}

// TestPrecheckFailureNeverEnqueues 预检失败当场给结论：找回密码的回话不变，队列里根本不进载荷。
func TestNotificationServiceSqlite_PrecheckFailureNeverEnqueues(t *testing.T) {
	e := newDispatchEnv(t)
	e.email.precheckErr = fmt.Errorf("%w: no enabled EMAIL channel", channel.ErrChannelNotConfigured)

	resp, err := e.svc.SendDirect(e.ctx, directReq(notificationV1.EventType_PASSWORD_RESET_CODE))
	require.Error(t, err, "调用方必须拿得到 error，否则会以为验证码发出去了")
	require.Equal(t, notificationV1.DeliveryStatus_SKIPPED, resp.GetStatus())

	require.Empty(t, e.queue.calls, "渠道根本用不了，不该把验证码塞进 Redis")
	require.Empty(t, e.email.calls, "预检不拨号")

	got, getErr := e.svc.GetNotificationDelivery(e.ctx, &notificationV1.GetNotificationDeliveryRequest{Id: resp.GetDeliveryId()})
	require.NoError(t, getErr)
	require.Equal(t, notificationV1.DeliveryStatus_SKIPPED, got.GetStatus())
	require.Contains(t, got.GetLastError(), "no enabled EMAIL channel")
}

// TestEnqueueFailureFallsBackToInline 队列不可用不能顺带把通知丢掉：当场投递，结论照旧即时。
func TestNotificationServiceSqlite_EnqueueFailureFallsBackToInline(t *testing.T) {
	e := newDispatchEnv(t)
	e.queue.err = fmt.Errorf("redis: connection refused")

	resp, err := e.svc.SendDirect(e.ctx, directReq(notificationV1.EventType_PASSWORD_RESET_CODE))
	require.NoError(t, err)
	require.Equal(t, notificationV1.DeliveryStatus_SENT, resp.GetStatus())
	require.Len(t, e.email.calls, 1)

	got, getErr := e.svc.GetNotificationDelivery(e.ctx, &notificationV1.GetNotificationDeliveryRequest{Id: resp.GetDeliveryId()})
	require.NoError(t, getErr)
	require.Equal(t, uint32(1), got.GetAttempts(), "当场投递也是一次尝试，attempts 不该只在异步路径有意义")
}

// TestSyncEventsNeverEnqueue 测试邮件与站内信在有队列时依然同步：前者的产物是报错原文，
// 后者要在同一趟里拿收件行主键。
func TestNotificationServiceSqlite_SyncEventsNeverEnqueue(t *testing.T) {
	e := newDispatchEnv(t)
	internal := &fakeSender{channel: notificationV1.Channel_INTERNAL, receipt: &channel.SendReceipt{}}
	e.svc.channels.Register(internal)

	_, err := e.svc.SendDirect(e.ctx, &notificationV1.SendDirectNotificationRequest{
		EventType: notificationV1.EventType_CHANNEL_TEST_EMAIL,
		ChannelId: trans.Ptr(uint32(7)),
		Target:    "ops@example.com",
		Title:     "t",
		Content:   "c",
	})
	require.NoError(t, err)

	_, err = e.svc.SendDirect(e.ctx, &notificationV1.SendDirectNotificationRequest{
		EventType:       notificationV1.EventType_INTERNAL_MESSAGE,
		Target:          "1024",
		Title:           "t",
		Content:         "c",
		RecipientUserId: trans.Ptr(uint32(1024)),
		RelatedId:       trans.Ptr(uint32(55)),
	})
	require.NoError(t, err)

	require.Empty(t, e.queue.calls, "这两个事件的调用方都需要当场结论")
	require.Len(t, e.email.calls, 1)
	require.Len(t, internal.calls, 1)
}

// TestRequestIdAnchor 幂等锚：调用方给的值原样落档，且 (request_id, channel) 唯一。
func TestNotificationServiceSqlite_RequestIdAnchor(t *testing.T) {
	e := newDispatchEnv(t)
	req := directReq(notificationV1.EventType_PASSWORD_RESET_CODE)
	req.RequestId = trans.Ptr("pwd-reset-user-2")

	resp, err := e.svc.SendDirect(e.ctx, req)
	require.NoError(t, err)
	got, err := e.svc.GetNotificationDelivery(e.ctx, &notificationV1.GetNotificationDeliveryRequest{Id: resp.GetDeliveryId()})
	require.NoError(t, err)
	require.Equal(t, "pwd-reset-user-2", got.GetRequestId())

	_, err = e.svc.SendDirect(e.ctx, req)
	require.Error(t, err, "同一次意图撞同一个 (request_id, channel)，不该变成第二封真实投递")
	require.Equal(t, uint64(1), e.countDeliveries(t))

	// 组合唯一而不是单列唯一：一次意图扇出到另一个渠道是允许的（§3.3 的行粒度）。
	webhook := &fakeSender{channel: notificationV1.Channel_WEBHOOK, receipt: &channel.SendReceipt{}}
	e.svc.channels.Register(webhook)
	other := directReq(notificationV1.EventType_CONTACT_BIND_CODE)
	other.RequestId = trans.Ptr("pwd-reset-user-2")
	other.Channel = trans.Ptr(notificationV1.Channel_WEBHOOK)
	other.Target = "https://example.com/hook"
	_, err = e.svc.SendDirect(e.ctx, other)
	require.NoError(t, err, "同一个 request_id 换渠道应该放行")
	require.Equal(t, uint64(2), e.countDeliveries(t))
}

// handler 的载荷缺行时把错误交给 asynq 重试（台账行是入队前单独提交的，读不到＝DB 侧临时故障）。
func TestNotificationServiceSqlite_AsyncDispatchUnknownDeliveryRetries(t *testing.T) {
	e := newDispatchEnv(t)

	require.Error(t, e.svc.AsyncNotificationDispatch(e.ctx, task.NotificationDispatchTaskType,
		&task.NotificationDispatchTaskData{DeliveryId: 999999, Target: "bob@example.com"}))
	require.Empty(t, e.email.calls)

	require.Error(t, e.svc.AsyncNotificationDispatch(e.ctx, task.NotificationDispatchTaskType, nil), "空载荷不该被当成投递")
}
