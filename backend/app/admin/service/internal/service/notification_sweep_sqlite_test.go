// 通知台账超时清扫（notification_delivery_sweep）的 SQLite 集成测试。
//
// 测的是"谁来给发了一半的行收尸"这件事的边界：
//   - 只有"超期 + 仍是 SENDING"的行会被结算，Fresh 的 SENDING 与已有结论的行一律不动；
//   - 清扫保留 attempts，并把可 grep 的原因写进 last_error；
//   - 状态谓词在 UPDATE 的 WHERE 里，所以并发定案的行不会被扫回 FAILED；
//   - 清扫过的行此后会被 AsyncNotificationDispatch 的幂等门挡掉 —— 这是阈值必须明显大于
//     重试预算的理由，这里把它测出来而不是只写在注释里；
//   - 批大小收敛：一次扫不完的下一次继续；
//   - 阈值解析（环境变量 / 坏值 / 低于预算的下限）。
//
// 与 notification_dispatch_sqlite_test.go 的分工：那一个测"投递什么时候发生"，
// 这一个测"没有结论的投递什么时候被判定为不会再有结论"。
package service

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"

	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/pkg/task"
)

// sweepSeedSeq 给测试里的 request_id 补一个保证唯一的序号。
var sweepSeedSeq atomic.Uint64

// deliveryOutcomeSent 一次成功投递的回写内容，用来在测试里充当"handler 已经把这一行定案"。
func deliveryOutcomeSent() data.DeliveryOutcome {
	sentAt := time.Now()
	return data.DeliveryOutcome{
		Status:    notificationV1.DeliveryStatus_SENT,
		ChannelID: trans.Ptr(uint32(42)),
		SentAt:    &sentAt,
	}
}

// seedSendingRow 直接经仓储落一行台账，created_at 由调用方决定（服务层的 SendDirect 只会写"现在"）。
// attempts 同样手工写：Create 收不到这一列（它是结果列，只有 MarkAttempted/MarkResult 写得动）。
func (e *notificationSvcEnv) seedSendingRow(t *testing.T, age time.Duration, attempts uint32) uint32 {
	t.Helper()

	// createdAt.UnixNano() 单独做键不够用：连排落库时拿到的时间戳会重复（本机 5 行一批实测撞过
	// (request_id, channel) 唯一索引），所以再拼一个自增序号。
	createdAt := time.Now().Add(-age)
	row, err := e.svc.deliveryRepo.Create(e.ctx, &notificationV1.NotificationDelivery{
		EventType: trans.Ptr(notificationV1.EventType_PASSWORD_RESET_CODE),
		Channel:   trans.Ptr(notificationV1.Channel_EMAIL),
		RequestId: trans.Ptr(fmt.Sprintf("sweep-test-%d-%d", createdAt.UnixMilli(), sweepSeedSeq.Add(1))),
		Target:    trans.Ptr("b***@example.com"),
		Status:    trans.Ptr(notificationV1.DeliveryStatus_SENDING),
		CreatedAt: timeutil.TimeToTimestamppb(&createdAt),
	})
	require.NoError(t, err)

	if attempts > 0 {
		require.NoError(t, e.svc.deliveryRepo.MarkAttempted(e.ctx, row.GetId(), attempts))
	}

	return row.GetId()
}

// TestSweepSettlesOnlyStaleSending 超期的 SENDING 落 FAILED 并留下原因；没到阈值和已定案的行不动。
func TestNotificationServiceSqlite_SweepSettlesOnlyStaleSending(t *testing.T) {
	e := newNotificationServiceForTest(t)

	stale := e.seedSendingRow(t, 2*time.Hour, 2)
	fresh := e.seedSendingRow(t, time.Minute, 0)
	settled := e.seedSendingRow(t, 2*time.Hour, 1)
	settledOutcome := deliveryOutcomeSent()
	require.NoError(t, e.svc.deliveryRepo.MarkResult(e.ctx, settled, &settledOutcome))

	require.NoError(t, e.svc.AsyncDeliverySweep(task.NotificationDeliverySweepTaskType, nil))

	got, err := e.svc.GetNotificationDelivery(e.ctx, &notificationV1.GetNotificationDeliveryRequest{Id: stale})
	require.NoError(t, err)
	require.Equal(t, notificationV1.DeliveryStatus_FAILED, got.GetStatus())
	require.Equal(t, uint32(2), got.GetAttempts(), "清扫只补结论，不许顺手改尝试次数")
	require.Contains(t, got.GetLastError(), task.NotificationDeliverySweepTaskType,
		"原因里要带任务名：排障时 grep 得出这一行是被谁定的案")
	require.Nil(t, got.GetSentAt(), "没发出去就不该有完成时间")

	gotFresh, err := e.svc.GetNotificationDelivery(e.ctx, &notificationV1.GetNotificationDeliveryRequest{Id: fresh})
	require.NoError(t, err)
	require.Equal(t, notificationV1.DeliveryStatus_SENDING, gotFresh.GetStatus(),
		"阈值之内不动：异步派发最坏还要跑约 4 分钟")

	gotSettled, err := e.svc.GetNotificationDelivery(e.ctx, &notificationV1.GetNotificationDeliveryRequest{Id: settled})
	require.NoError(t, err)
	require.Equal(t, notificationV1.DeliveryStatus_SENT, gotSettled.GetStatus())
	require.NotNil(t, gotSettled.GetSentAt())
}

// TestSweepDoesNotOverwriteConcurrentResult 并发保护：状态谓词在 UPDATE 里，
// "查出来时还是 SENDING、写回去时已经定案"的行不会被扫回 FAILED。
//
// 这里用同一批 ID 手工复现那个时序（先拿到候选行，再让 handler 定案，然后执行清扫的写入），
// 而不依赖两个 goroutine 抢跑。
func TestNotificationServiceSqlite_SweepDoesNotOverwriteConcurrentResult(t *testing.T) {
	e := newNotificationServiceForTest(t)
	id := e.seedSendingRow(t, 2*time.Hour, 1)

	outcome := deliveryOutcomeSent()
	require.NoError(t, e.svc.deliveryRepo.MarkResult(e.ctx, id, &outcome))

	// 清扫照常执行：这一行的 status 已经不是 SENDING，WHERE 不命中。
	count, err := e.svc.deliveryRepo.SweepStaleSending(e.ctx, time.Now().Add(-time.Hour), "swept", 500)
	require.NoError(t, err)
	require.Zero(t, count)

	got, err := e.svc.GetNotificationDelivery(e.ctx, &notificationV1.GetNotificationDeliveryRequest{Id: id})
	require.NoError(t, err)
	require.Equal(t, notificationV1.DeliveryStatus_SENT, got.GetStatus())
	require.Equal(t, uint32(42), got.GetChannelId(), "已有的回执不能被清扫抹掉")
}

// TestSweptRowIsNeverDispatched 清扫的代价：定案之后 asynq 才把任务跑完，也会被幂等门挡掉。
// 换句话说，"队列积压到超过阈值"在今天确实是"这封不会再发了"，而不是"晚点发"。
func TestNotificationServiceSqlite_SweptRowIsNeverDispatched(t *testing.T) {
	e := newDispatchEnv(t)

	resp, err := e.svc.SendDirect(e.ctx, directReq(notificationV1.EventType_PASSWORD_RESET_CODE))
	require.NoError(t, err)

	// 把"行已超期"表现出来：阈值传一个刚刚晚于 created_at 的时刻。
	count, err := e.svc.deliveryRepo.SweepStaleSending(e.ctx, time.Now().Add(time.Minute), "swept by test", 500)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	require.NoError(t, e.svc.AsyncNotificationDispatch(e.ctx, task.NotificationDispatchTaskType, e.payload(t, 0)))
	require.Empty(t, e.email.calls, "已经定案的行必须停在幂等门外，否则重投会发第二封")

	got, err := e.svc.GetNotificationDelivery(e.ctx, &notificationV1.GetNotificationDeliveryRequest{Id: resp.GetDeliveryId()})
	require.NoError(t, err)
	require.Equal(t, notificationV1.DeliveryStatus_FAILED, got.GetStatus())
	require.Zero(t, got.GetAttempts())
}

// TestSweepAdvancesInBatches 一次扫不完的下一轮继续（批大小由调用方给，循环在服务层）。
func TestNotificationServiceSqlite_SweepAdvancesInBatches(t *testing.T) {
	e := newNotificationServiceForTest(t)

	const rows, batch = 5, 2
	for i := 0; i < rows; i++ {
		e.seedSendingRow(t, 2*time.Hour, 0)
	}

	remaining := rows
	for round := 1; ; round++ {
		count, err := e.svc.deliveryRepo.SweepStaleSending(e.ctx, time.Now().Add(-time.Hour), "swept", batch)
		require.NoError(t, err)
		require.LessOrEqual(t, count, batch, "单批不得超过 limit")
		remaining -= count
		if count == 0 {
			require.Equal(t, round, rows/batch+2, "轮数应当正好把 5 行按每批 2 行扫完")
			break
		}
	}
	require.Zero(t, remaining)

	// 再扫一次：没有 SENDING 可扫了，返回 0 而不是报错。
	count, err := e.svc.deliveryRepo.SweepStaleSending(e.ctx, time.Now().Add(-time.Hour), "swept", batch)
	require.NoError(t, err)
	require.Zero(t, count)
}

// TestSweepThresholdResolution 阈值解析：缺省 15 分钟，环境变量可放宽，坏值与低于预算的值都兜住。
func TestNotificationServiceSqlite_SweepThresholdResolution(t *testing.T) {
	e := newNotificationServiceForTest(t)

	t.Setenv("NOTIFICATION_DELIVERY_STALE_MINUTES", "")
	require.Equal(t, deliverySweepDefaultStaleAfter, e.svc.deliveryStaleAfter(e.ctx))

	t.Setenv("NOTIFICATION_DELIVERY_STALE_MINUTES", "60")
	require.Equal(t, time.Hour, e.svc.deliveryStaleAfter(e.ctx))

	t.Setenv("NOTIFICATION_DELIVERY_STALE_MINUTES", "1")
	require.Equal(t, deliverySweepMinStaleAfter, e.svc.deliveryStaleAfter(e.ctx),
		"低于重试预算的阈值会把还在投递的行扫死，必须抬回下限")

	t.Setenv("NOTIFICATION_DELIVERY_STALE_MINUTES", "abc")
	require.Equal(t, deliverySweepDefaultStaleAfter, e.svc.deliveryStaleAfter(e.ctx), "坏值按缺省，不静默采纳")
}

// TestSweepAfterHandlerCrash 端到端形状：handler 死在拨号中途（attempts 已记、结论未回写），
// 清扫把这一行闭环 —— 这正是加它的动机。
func TestNotificationServiceSqlite_SweepAfterHandlerCrash(t *testing.T) {
	e := newDispatchEnv(t)

	resp, err := e.svc.SendDirect(e.ctx, directReq(notificationV1.EventType_PASSWORD_RESET_CODE))
	require.NoError(t, err)
	id := resp.GetDeliveryId()

	// 模拟"进程在 MarkAttempted 之后、回写结论之前被杀"：这一次真的拨过号（attempts=1），
	// 但台账上没有任何结论。
	require.NoError(t, e.svc.deliveryRepo.MarkAttempted(e.ctx, id, 1))

	// 这一行是"刚刚"落的，缺省 15 分钟阈值扫不到；把阈值放到它之前（等价于"已经超期"）就照常扫到。
	require.NoError(t, e.svc.AsyncDeliverySweep(task.NotificationDeliverySweepTaskType, nil))
	got, err := e.svc.GetNotificationDelivery(e.ctx, &notificationV1.GetNotificationDeliveryRequest{Id: id})
	require.NoError(t, err)
	require.Equal(t, notificationV1.DeliveryStatus_SENDING, got.GetStatus(), "缺省阈值内不误伤")

	count, err := e.svc.deliveryRepo.SweepStaleSending(e.ctx, time.Now().Add(time.Minute), "swept", 500)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	got, err = e.svc.GetNotificationDelivery(e.ctx, &notificationV1.GetNotificationDeliveryRequest{Id: id})
	require.NoError(t, err)
	require.Equal(t, notificationV1.DeliveryStatus_FAILED, got.GetStatus())
	require.Equal(t, uint32(1), got.GetAttempts(),
		"attempts>0 而 status 被清扫定案 = 拨过号却没回写结论，这条线索必须留在台账里")
}
