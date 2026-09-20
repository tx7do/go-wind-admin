// 台账结论回写失败（markResult 上抛）的 SQLite 集成测试。
//
// 钉住的是 P2-5 那条策略表：同一个"回写失败"在四个出口上三种去处，写下来免得下次有人顺手统一成一种。
//   - 异步 + 发送成功 → 错误交给 asynq（重试才可能把结论落对），且不许带 SkipRetry；
//   - 异步 + SkipRetry 两支（渠道未注册 / 配置类 SKIPPED）→ 回写失败也不把 SkipRetry 挤掉：
//     从没拨过号的行不值得让明文验证码载荷在队列里多留一轮，代价由清扫兜；
//   - 同步 + 发送成功 → 绝不回错给业务侧（找回密码的调用方一看错就再生成一封验证码），
//     台账那一行停在 SENDING 由 notification_delivery_sweep 定案；
//   - 发送本来就失败（同步 / 重试窗口 / 额度用尽）→ errors.Join 带上两个事实，
//     且 errors.Is 仍认得原始发送错误（调用方靠它分文案）。
//
// 故障注入用 ent 的 mutation hook（armRecordWriteFailure），不用"把行删掉"：
// 删行会让 Get / MarkAttempted 一起失败，测试就分不清"结论没落库"和"这一行不存在"两件事。
//
// 与 notification_dispatch_sqlite_test.go 的分工：那一个测"结论该写什么"，这一个测"写不进去怎么办"。
package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"

	"go-wind-admin/app/admin/service/internal/data/channel"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/notificationdelivery"

	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"

	"go-wind-admin/pkg/task"
)

// errInjectedRecordWrite 注入用的错误。repo 会把它包成自己的 500，所以断言不认它的身份，
// 只认"回写确实失败了"这个事实（fired() + 台账状态还停在 SENDING）。
var errInjectedRecordWrite = errors.New("injected: notification delivery result write failed")

// armRecordWriteFailure 让**下一次**"写台账结论"的 mutation 失败。
//
// 只命中 NotificationDelivery 上带 status 列的 UpdateOne：异步链路前两步（Get 是查询、
// MarkAttempted 只写 attempts）因此照常走，测试能干净地只掐掉回写那一下。
// 返回值是"注入是否真的打中过"——每条测试都必须断言它，否则注入悄悄失效会变成空跑的绿灯。
func (e *notificationSvcEnv) armRecordWriteFailure(t *testing.T) func() bool {
	t.Helper()

	fired := false
	e.client.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			if !fired && m.Type() == ent.TypeNotificationDelivery &&
				m.Op() == ent.OpUpdateOne &&
				slices.Contains(m.Fields(), notificationdelivery.FieldStatus) {
				fired = true

				return nil, errInjectedRecordWrite
			}

			return next.Mutate(ctx, m)
		})
	})

	return func() bool { return fired }
}

func (e *notificationSvcEnv) getDelivery(t *testing.T, id uint32) *notificationV1.NotificationDelivery {
	t.Helper()

	got, err := e.svc.GetNotificationDelivery(e.ctx, &notificationV1.GetNotificationDeliveryRequest{Id: id})
	require.NoError(t, err)

	return got
}

// TestAsyncSendRecordFailureGoesToAsynq 信已发出而结论没落库：错误必须交给 asynq，
// 重投之后台账才可能对得上（代价是再拨一次号 = 第二封信，这是 P2-3 认下的至少一次投递）。
func TestNotificationServiceSqlite_AsyncSendRecordFailureGoesToAsynq(t *testing.T) {
	e := newDispatchEnv(t)
	fired := e.armRecordWriteFailure(t)

	resp, err := e.svc.SendDirect(e.ctx, directReq(notificationV1.EventType_PASSWORD_RESET_CODE))
	require.NoError(t, err)
	sent := e.payload(t, 0)

	err = e.svc.AsyncNotificationDispatch(e.ctx, task.NotificationDispatchTaskType, sent)
	require.Error(t, err, "回写失败必须上抛：吞掉它 = 这一行永远停在 SENDING 而队列里已经没有任务了")
	require.False(t, errors.Is(err, asynq.SkipRetry), "这一行值得重投，SkipRetry 会把唯一能修好它的机会挡掉")
	require.True(t, fired(), "注入没打中，这条测试是空跑的")
	require.Len(t, e.email.calls, 1, "错误是在拨号之后才产生的")

	got := e.getDelivery(t, resp.GetDeliveryId())
	require.Equal(t, notificationV1.DeliveryStatus_SENDING, got.GetStatus(), "结论没落库，行还停在 SENDING")
	require.Equal(t, uint32(1), got.GetAttempts(), "MarkAttempted 不写 status，不受注入影响")

	// 模拟 asynq 的重投：注入是一次性的，这一次结论该落对。
	require.NoError(t, e.svc.AsyncNotificationDispatch(e.ctx, task.NotificationDispatchTaskType, sent))
	require.Len(t, e.email.calls, 2, "重投会再拨一次号：这是「至少一次投递」的代价，写死在这里")

	got2 := e.getDelivery(t, resp.GetDeliveryId())
	require.Equal(t, notificationV1.DeliveryStatus_SENT, got2.GetStatus(), "重试后台账终于与事实一致")
	require.Equal(t, uint32(2), got2.GetAttempts())
}

// TestSyncSendNotFailedByRecordFailure 同步路径的"不回错"：信出去了就不许把回写失败说成发送失败，
// 否则找回密码的调用方会再生成一封验证码 —— 那比台账缺一格严重。
func TestNotificationServiceSqlite_SyncSendNotFailedByRecordFailure(t *testing.T) {
	e := newNotificationServiceForTest(t)
	fired := e.armRecordWriteFailure(t)

	resp, err := e.svc.SendDirect(e.ctx, directReq(notificationV1.EventType_CHANNEL_TEST_EMAIL))
	require.NoError(t, err, "发送成功的结论不许被回写失败污染")
	require.Equal(t, notificationV1.DeliveryStatus_SENT, resp.GetStatus())
	require.Len(t, e.email.calls, 1, "上抛策略不该导致当场重发")
	require.True(t, fired())

	got := e.getDelivery(t, resp.GetDeliveryId())
	require.Equal(t, notificationV1.DeliveryStatus_SENDING, got.GetStatus(),
		"这一行留给 notification_delivery_sweep 定案")
	// 同步路径的 attempts 是**和结论一次写**的（deliver 里 outcome.Attempts=1），所以结论写不进去时
	// "拨过号"这条线索一起丢了 —— 异步路径先 MarkAttempted 再拨号才留得下来。
	// 这条不对称是既有设计（同步不必多一次写），清扫读到的就是 attempts=0 的行；记在这里免得被读成 bug。
	require.Equal(t, uint32(0), got.GetAttempts())
}

// TestSyncSendFailureCarriesBothFacts 发送本来就失败时，两个事实一起报，且调用方仍认得原始发送错误。
func TestNotificationServiceSqlite_SyncSendFailureCarriesBothFacts(t *testing.T) {
	e := newNotificationServiceForTest(t)
	busyErr := fmt.Errorf("smtp rejected: 450 mailbox busy")
	e.email.err = busyErr
	fired := e.armRecordWriteFailure(t)

	resp, err := e.svc.SendDirect(e.ctx, directReq(notificationV1.EventType_CHANNEL_TEST_EMAIL))
	require.Error(t, err)
	require.Equal(t, notificationV1.DeliveryStatus_FAILED, resp.GetStatus())
	require.True(t, errors.Is(err, busyErr), "errors.Join 之后调用方仍要能按原始错误分文案")
	require.Contains(t, err.Error(), "mark notification delivery result failed",
		"回写失败这条事实也得带上：这次调用已经在报错了，多一行真相不会诱导任何人重发")
	require.True(t, fired())
}

// TestSkippedBranchKeepsSkipRetryWhenRecordFails 配置类结论回写失败也不改成重投：
// 这一行从没拨过号，为重投把明文验证码在队列里多留一轮不值（代价：台账由清扫定案）。
func TestNotificationServiceSqlite_SkippedBranchKeepsSkipRetryWhenRecordFails(t *testing.T) {
	e := newDispatchEnv(t)
	e.email.err = channel.ErrChannelNotConfigured
	fired := e.armRecordWriteFailure(t)

	resp, err := e.svc.SendDirect(e.ctx, directReq(notificationV1.EventType_PASSWORD_RESET_CODE))
	require.NoError(t, err, "预检放行才会入队，这一格测的是 handler 里渠道刚被关掉的分支")
	sent := e.payload(t, 0)

	err = e.svc.AsyncNotificationDispatch(e.ctx, task.NotificationDispatchTaskType, sent)
	require.Error(t, err)
	require.True(t, errors.Is(err, asynq.SkipRetry), "SkipRetry 必须活过 errors.Join")
	require.Contains(t, err.Error(), "mark notification delivery result failed")
	require.True(t, fired())

	got := e.getDelivery(t, resp.GetDeliveryId())
	require.Equal(t, notificationV1.DeliveryStatus_SENDING, got.GetStatus())
	require.Equal(t, uint32(1), got.GetAttempts())
	// 不断言"压根没拨号"：真发件人是 EmailSender 在拨号前 checkUsable（data/channel 的测试覆盖那一段），
	// 这里的替身只会把错误原样交回来，calls 记不记都说明不了渠道有没有真的连过。
}

// TestRetryWindowRecordFailureStillRetries 重试窗口那一格回写的是"上一次为什么失败"，
// 写丢了不致命（下一次尝试会带着 sendErr 再来一遍），但错误身份必须仍然是"可重试"。
func TestNotificationServiceSqlite_RetryWindowRecordFailureStillRetries(t *testing.T) {
	e := newDispatchEnv(t)
	resetErr := fmt.Errorf("smtp connect failed: connection reset by peer")
	e.email.err = resetErr
	fired := e.armRecordWriteFailure(t)

	id := e.seedSendingRow(t, 0, 1)
	err := e.svc.AsyncNotificationDispatch(e.ctx, task.NotificationDispatchTaskType,
		&task.NotificationDispatchTaskData{DeliveryId: id, Target: "bob@example.com", Title: "重置验证码", Content: "您的验证码是 123456"})

	require.Error(t, err)
	require.True(t, errors.Is(err, resetErr), "发送错误仍然是主错误")
	require.False(t, errors.Is(err, asynq.SkipRetry), "还有额度，回写失败不许把它变成不重试")
	require.Contains(t, err.Error(), "mark notification delivery result failed")
	require.True(t, fired())

	got := e.getDelivery(t, id)
	require.Equal(t, notificationV1.DeliveryStatus_SENDING, got.GetStatus())
	require.Equal(t, uint32(2), got.GetAttempts(), "这次尝试的记账与报错是两次写，只丢后者")
	require.Empty(t, got.GetLastError(), "被掐掉的正是「这次的报错写进台账」那一下")
}
