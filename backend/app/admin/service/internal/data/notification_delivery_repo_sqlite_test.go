// NotificationDeliveryRepo 的 SQLite 内存库测试（白盒，包内测试）。
//
// 台账是"投递事实"的记录（Create(SENDING) → MarkResult，外加 SweepStaleSending 补终态），
// 本仓刻意不提供 Update/Delete——改一条已发生的投递记录等于伪造事实。测试盯四件事：
//   - 枚举往返：event_type / channel / status 三个可空指针枚举列必须经 converter 双向命中
//     （实体侧 Optional().Nillable() 才让 copier 的指针对生效；读视图钉住即可，不回填）；
//   - 台账不可从旁边伪造：Create 不吃 attempts（它是结果列，只有 MarkAttempted/MarkResult
//     写得动，否则"台账不提供 Update"就从旁边漏了洞）；
//   - 结果回写只覆盖结果字段：MarkResult 不得篡改投递意图快照（event_type/target 等）；
//   - SweepStaleSending 的谓词语义：status==SENDING 与 created_at<before 都进 WHERE
//     （抢跑的异步 handler 不会被扫掉），以及 before 归一 UTC 的 SQLite 时区渲染修复。
package data

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/types/known/timestamppb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent/notificationdelivery"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newDelivery 构造一行最小合法台账载荷；request_id 不给则落 NULL（唯一索引对 NULL 不生效，
// 测试里不关心唯一性的行一律不传，需要幂等锚的用例显式传唯一值）。
func newDelivery(requestID string, channel notificationV1.Channel, createdAt *timestamppb.Timestamp) *notificationV1.NotificationDelivery {
	req := &notificationV1.NotificationDelivery{
		Channel: channel.Enum(),
	}
	if requestID != "" {
		req.RequestId = trans.Ptr(requestID)
	}
	if createdAt != nil {
		req.CreatedAt = createdAt
	}
	return req
}

// TestNotificationDeliveryRepoSqlite_Create 验证字段级落库与三枚举双向映射：
// proto 枚举经 converter 落为实体枚举字面量、意图快照字段逐列落库、created_by/created_at
// 按载荷写入；Get 读回的 DTO 枚举必须落回正确的 proto 值（读视图一旦失配，台账页
// 渠道/事件全显示成 UNSPECIFIED，排障的人看到的是假账）。
func TestNotificationDeliveryRepoSqlite_Create(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewNotificationDeliveryRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	createdAt := time.Now().Add(-time.Minute)
	created, err := repo.Create(ctx, &notificationV1.NotificationDelivery{
		EventType:       notificationV1.EventType_PASSWORD_RESET_CODE.Enum(),
		Channel:         notificationV1.Channel_EMAIL.Enum(),
		ChannelId:       trans.Ptr(uint32(7)),
		RecipientUserId: trans.Ptr(uint32(1001)),
		RelatedId:       trans.Ptr(uint32(42)),
		Target:          trans.Ptr("a***@example.com"),
		Status:          notificationV1.DeliveryStatus_SENT.Enum(),
		RequestId:       trans.Ptr("req-create-full"),
		SentAt:          timestamppb.New(createdAt),
		CreatedBy:       trans.Ptr(uint32(1001)),
		CreatedAt:       timestamppb.New(createdAt),
	})
	require.NoError(t, err, "全字段载荷的 Create 应成功")
	require.NotZero(t, created.GetId())

	// 行侧：枚举按 proto 枚举名落为实体字面量，快照字段逐列命中
	row, err := entClient.Client().NotificationDelivery.Get(ctx, created.GetId())
	require.NoError(t, err)
	require.NotNil(t, row.EventType)
	require.Equal(t, notificationdelivery.EventTypePasswordResetCode, *row.EventType,
		"proto 枚举名应经 converter 落为实体枚举字面量")
	require.NotNil(t, row.Channel)
	require.Equal(t, notificationdelivery.ChannelEmail, *row.Channel)
	require.NotNil(t, row.ChannelID)
	require.Equal(t, uint32(7), *row.ChannelID)
	require.NotNil(t, row.RecipientUserID)
	require.Equal(t, uint32(1001), *row.RecipientUserID)
	require.NotNil(t, row.RelatedID)
	require.Equal(t, uint32(42), *row.RelatedID)
	require.NotNil(t, row.Target)
	require.Equal(t, "a***@example.com", *row.Target)
	require.NotNil(t, row.RequestID)
	require.Equal(t, "req-create-full", *row.RequestID)
	require.NotNil(t, row.Status)
	require.Equal(t, notificationdelivery.StatusSent, *row.Status)
	require.NotNil(t, row.SentAt)
	require.NotNil(t, row.CreatedBy)
	require.Equal(t, uint32(1001), *row.CreatedBy, "created_by 应按载荷落库")
	require.NotNil(t, row.CreatedAt, "created_at 必须写入：没有它台账页出空行、清扫也没锚")

	// 读视图：三枚举回映射为 proto 值（schema 那句 Optional().Nillable() 承诺的兑现）
	dto, err := repo.Get(ctx, created.GetId())
	require.NoError(t, err)
	require.Equal(t, notificationV1.EventType_PASSWORD_RESET_CODE, dto.GetEventType(),
		"实体枚举应经 converter 回映射为 proto 枚举，不能落 UNSPECIFIED 零值")
	require.Equal(t, notificationV1.Channel_EMAIL, dto.GetChannel())
	require.Equal(t, notificationV1.DeliveryStatus_SENT, dto.GetStatus())
	require.Equal(t, uint32(42), dto.GetRelatedId())
	require.Equal(t, "req-create-full", dto.GetRequestId())

	// 另一组枚举取值（EXT 渠道 + MONITOR_ALERT 事件 + SKIPPED 状态）：
	// 扩展渠道的台账统一记 EXT、具体渠道由 channel_id 回答——两侧字面量都要对得上
	created2, err := repo.Create(ctx, &notificationV1.NotificationDelivery{
		EventType: notificationV1.EventType_MONITOR_ALERT.Enum(),
		Channel:   notificationV1.Channel_EXT.Enum(),
		ChannelId: trans.Ptr(uint32(88)),
		Status:    notificationV1.DeliveryStatus_SKIPPED.Enum(),
		Target:    trans.Ptr("tg:***"),
	})
	require.NoError(t, err)
	row2, err := entClient.Client().NotificationDelivery.Get(ctx, created2.GetId())
	require.NoError(t, err)
	require.Equal(t, notificationdelivery.EventTypeMonitorAlert, *row2.EventType)
	require.Equal(t, notificationdelivery.ChannelExt, *row2.Channel)
	require.Equal(t, notificationdelivery.StatusSkipped, *row2.Status)

	// 入参守卫
	_, err = repo.Create(ctx, nil)
	require.Error(t, err, "nil 载荷应返回 BadRequest")
}

// TestNotificationDeliveryRepoSqlite_CreateDefaults 验证未指定字段的列默认与
// "attempts 不从 req 取"这条铁律：attempts 是结果列，新建时永远靠列默认 0——
// 否则调用方在 Create 里塞一个 attempts，"台账不提供 Update"就从旁边漏了个洞。
// 未指定 status 落列默认 SENDING；未指定 created_at 落 NULL（它来自调用方的意图时间，
// 不是 DB 时钟——这也决定了 sweep 的 CreatedAtLT 对 NULL 行永不命中）。
func TestNotificationDeliveryRepoSqlite_CreateDefaults(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewNotificationDeliveryRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	created, err := repo.Create(ctx, &notificationV1.NotificationDelivery{
		EventType: notificationV1.EventType_INTERNAL_MESSAGE.Enum(),
		Channel:   notificationV1.Channel_INTERNAL.Enum(),
		Attempts:  trans.Ptr(uint32(5)), // 载荷里的 attempts 必须被无视
	})
	require.NoError(t, err, "缺省载荷的 Create 应成功（除枚举外全部可空）")

	row, err := entClient.Client().NotificationDelivery.Get(ctx, created.GetId())
	require.NoError(t, err)
	require.NotNil(t, row.Status)
	require.Equal(t, notificationdelivery.StatusSending, *row.Status,
		"未指定 status 应落列默认 SENDING（投递前态）")
	require.NotNil(t, row.Attempts)
	require.Equal(t, uint32(0), *row.Attempts,
		"attempts 不从 req 取：载荷给 5 也必须落 0，它是 MarkAttempted/MarkResult 的专属列")
	require.Nil(t, row.CreatedAt, "未传意图时间时 created_at 应如实为 NULL，不该伪造 DB 时钟")
	require.Nil(t, row.CreatedBy, "无操作人上下文时不该写出一个不存在的 created_by")
	require.Nil(t, row.LastError)

	// 全空载荷也应成功（台账所有业务列可空），读回全部 nil 快照
	empty, err := repo.Create(ctx, &notificationV1.NotificationDelivery{})
	require.NoError(t, err, "全空载荷的 Create 应成功")
	dto, err := repo.Get(ctx, empty.GetId())
	require.NoError(t, err)
	require.Nil(t, dto.EventType)
	require.Nil(t, dto.Channel)
	require.Equal(t, notificationV1.DeliveryStatus_SENDING, dto.GetStatus(), "读视图应呈现列默认 SENDING")
	require.Nil(t, dto.Target)
	require.Nil(t, dto.SentAt)
}

// TestNotificationDeliveryRepoSqlite_Get 验证命中回 DTO、id=0 的 BadRequest、
// 不存在 ID 回 NotFound（台账页按 ID 点开一条已清理的行该报"没有"，不该 500）。
func TestNotificationDeliveryRepoSqlite_Get(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewNotificationDeliveryRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	created, err := repo.Create(ctx, newDelivery("req-get", notificationV1.Channel_SMS, nil))
	require.NoError(t, err)

	dto, err := repo.Get(ctx, created.GetId())
	require.NoError(t, err, "存在的 ID 应命中")
	require.Equal(t, created.GetId(), dto.GetId())
	require.Equal(t, notificationV1.Channel_SMS, dto.GetChannel())
	require.Equal(t, notificationV1.DeliveryStatus_SENDING, dto.GetStatus())

	_, err = repo.Get(ctx, 0)
	require.Error(t, err, "id=0 应返回 BadRequest")

	_, err = repo.Get(ctx, 9999999)
	require.Error(t, err, "不存在的 ID 应返回 NotFound 而非空指针")
	require.Contains(t, err.Error(), "not found")
}

// TestNotificationDeliveryRepoSqlite_List 验证分页 total 与 target 的 contains 过滤；
// nil 请求返回 BadRequest。
func TestNotificationDeliveryRepoSqlite_List(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewNotificationDeliveryRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	mail := newDelivery("req-list-1", notificationV1.Channel_EMAIL, nil)
	mail.Target = trans.Ptr("user***@example.com")
	_, err := repo.Create(ctx, mail)
	require.NoError(t, err)
	_, err = repo.Create(ctx, newDelivery("req-list-2", notificationV1.Channel_WEBHOOK, nil))
	require.NoError(t, err)

	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.GetTotal())
	require.Len(t, all.GetItems(), 2)

	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: `{"target__contains":"example.com"}`,
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.GetTotal(), "contains 过滤应只统计命中行")
	require.Len(t, filtered.GetItems(), 1)
	require.Equal(t, notificationV1.Channel_EMAIL, filtered.GetItems()[0].GetChannel())

	_, err = repo.List(ctx, nil)
	require.Error(t, err, "nil 分页请求应返回 BadRequest")
}

// TestNotificationDeliveryRepoSqlite_MarkResult 验证结果回写的两条边界：
//   - 只覆盖结果字段：status/channel_id/attempts/sent_at/provider_id/last_error 落位，
//     意图快照（event_type/target/recipient）保持 Create 时的原值——回写不得篡改事实；
//   - last_error 空串不写：FAILED 但没有错误详情时不得抹掉上一次的真实报错。
func TestNotificationDeliveryRepoSqlite_MarkResult(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewNotificationDeliveryRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	created, err := repo.Create(ctx, &notificationV1.NotificationDelivery{
		EventType:       notificationV1.EventType_PASSWORD_RESET_CODE.Enum(),
		RecipientUserId: trans.Ptr(uint32(1001)),
		Target:          trans.Ptr("u***@example.com"),
		RequestId:       trans.Ptr("req-mark"),
	})
	require.NoError(t, err)
	require.NoError(t, repo.MarkAttempted(ctx, created.GetId(), 1), "拨号前先记一次尝试")

	sentAt := time.Now()
	require.NoError(t, repo.MarkResult(ctx, created.GetId(), &DeliveryOutcome{
		Status:     notificationV1.DeliveryStatus_SENT,
		ChannelID:  trans.Ptr(uint32(9)), // 自选渠道：Create 时不知道，回写补上
		SentAt:     &sentAt,
		ProviderID: trans.Ptr("prov-msg-123"),
		// Attempts 给 nil：异步路径已在 MarkAttempted 写过，回写不再碰
	}))

	row, err := entClient.Client().NotificationDelivery.Get(ctx, created.GetId())
	require.NoError(t, err)
	require.NotNil(t, row.Status)
	require.Equal(t, notificationdelivery.StatusSent, *row.Status)
	require.NotNil(t, row.ChannelID)
	require.Equal(t, uint32(9), *row.ChannelID)
	require.NotNil(t, row.Attempts)
	require.Equal(t, uint32(1), *row.Attempts, "outcome.Attempts=nil 时回写不得覆盖 MarkAttempted 的计数")
	require.NotNil(t, row.SentAt)
	require.WithinDuration(t, sentAt, *row.SentAt, 2*time.Second, "投递完成时间应落库")
	require.NotNil(t, row.ProviderID)
	require.Equal(t, "prov-msg-123", *row.ProviderID)
	require.Nil(t, row.LastError, "成功回写不该留错误信息")
	// 意图快照不得被结果回写篡改
	require.Equal(t, notificationdelivery.EventTypePasswordResetCode, *row.EventType)
	require.Equal(t, "u***@example.com", *row.Target, "回写只覆盖结果字段，快照列不得动")
	require.Equal(t, uint32(1001), *row.RecipientUserID)

	// FAILED + last_error：失败详情落列，sent_at 保持空（FAILED 没有完成时间）
	failed, err := repo.Create(ctx, newDelivery("req-mark-failed", notificationV1.Channel_EMAIL, nil))
	require.NoError(t, err)
	require.NoError(t, repo.MarkResult(ctx, failed.GetId(), &DeliveryOutcome{
		Status:    notificationV1.DeliveryStatus_FAILED,
		LastError: "dial tcp: connection refused",
	}))
	frow, err := entClient.Client().NotificationDelivery.Get(ctx, failed.GetId())
	require.NoError(t, err)
	require.Equal(t, notificationdelivery.StatusFailed, *frow.Status)
	require.NotNil(t, frow.LastError)
	require.Equal(t, "dial tcp: connection refused", *frow.LastError)
	require.Nil(t, frow.SentAt, "FAILED/SKIPPED 不写完成时间")

	// last_error 空串不覆盖：空串表示"没有新详情"，不得抹掉上一次的真实报错
	require.NoError(t, repo.MarkResult(ctx, failed.GetId(), &DeliveryOutcome{
		Status:    notificationV1.DeliveryStatus_FAILED,
		LastError: "",
	}))
	frow, err = entClient.Client().NotificationDelivery.Get(ctx, failed.GetId())
	require.NoError(t, err)
	require.Equal(t, "dial tcp: connection refused", *frow.LastError,
		"空串 last_error 不应覆盖既有报错")

	// 入参守卫与不存在 ID
	require.Error(t, repo.MarkResult(ctx, created.GetId(), nil), "nil outcome 应返回 BadRequest")
	require.Error(t, repo.MarkResult(ctx, 0, &DeliveryOutcome{Status: notificationV1.DeliveryStatus_SENT}),
		"id=0 应返回 BadRequest")
	err = repo.MarkResult(ctx, 9999999, &DeliveryOutcome{Status: notificationV1.DeliveryStatus_SENT})
	require.Error(t, err, "对不存在 ID 的回写应报错而非静默")
	require.Contains(t, err.Error(), "mark notification delivery result failed")
}

// TestNotificationDeliveryRepoSqlite_MarkAttempted 验证拨号前的尝试计数：
// attempts 递增落库、status 不得被顺手动（进程死在 SMTP 握手中时，行停在
// SENDING + attempts>0 就是"拨过号但没结论"的唯一线索）。
func TestNotificationDeliveryRepoSqlite_MarkAttempted(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewNotificationDeliveryRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	created, err := repo.Create(ctx, newDelivery("req-attempt", notificationV1.Channel_SMS, nil))
	require.NoError(t, err)

	require.NoError(t, repo.MarkAttempted(ctx, created.GetId(), 1), "首次尝试应成功落库")
	require.NoError(t, repo.MarkAttempted(ctx, created.GetId(), 2), "重试计数应可递增")

	row, err := entClient.Client().NotificationDelivery.Get(ctx, created.GetId())
	require.NoError(t, err)
	require.NotNil(t, row.Attempts)
	require.Equal(t, uint32(2), *row.Attempts)
	require.Equal(t, notificationdelivery.StatusSending, *row.Status,
		"MarkAttempted 只写 attempts：进程死在半路时行必须还停在 SENDING")

	// 入参守卫与不存在 ID
	require.Error(t, repo.MarkAttempted(ctx, 0, 1), "id=0 应返回 BadRequest")
	err = repo.MarkAttempted(ctx, 9999999, 1)
	require.Error(t, err, "对不存在 ID 的尝试回写应报错而非静默")
	require.Contains(t, err.Error(), "mark notification delivery attempt failed")
}

// TestNotificationDeliveryRepoSqlite_SweepStaleSending 验证清扫的 WHERE 语义：
// 只有"status 仍 SENDING 且 created_at 早于 cutoff"的行被落 FAILED；已有结论的行、
// 未到期的行、created_at 为 NULL 的行都不动；被扫行的 attempts 保留（那是"发了一半"
// 唯一留得下来的痕迹）。fresh 行用 cutoff 之后的意图时间，顺带钉住 before 归一 UTC
// 的 SQLite 时区渲染修复——若归一被去掉，fresh 行会被字典序比较误判成超期并被扫掉。
func TestNotificationDeliveryRepoSqlite_SweepStaleSending(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewNotificationDeliveryRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	before := time.Now()
	staleAt := timestamppb.New(before.Add(-15 * time.Minute))
	freshAt := timestamppb.New(before.Add(5 * time.Minute)) // 故意取 cutoff 之后，免时钟竞态

	staleID, err := repo.Create(ctx, newDelivery("req-sweep-stale", notificationV1.Channel_EMAIL, staleAt))
	require.NoError(t, err)
	require.NoError(t, repo.MarkAttempted(ctx, staleID.GetId(), 3), "拨过 3 次、没等到结论的行")

	freshID, err := repo.Create(ctx, newDelivery("req-sweep-fresh", notificationV1.Channel_EMAIL, freshAt))
	require.NoError(t, err)
	sentID, err := repo.Create(ctx, &notificationV1.NotificationDelivery{
		Channel:   notificationV1.Channel_EMAIL.Enum(),
		Status:    notificationV1.DeliveryStatus_SENT.Enum(), // 已有结论
		CreatedAt: staleAt,                                   // 虽然够老，但状态谓词不该命中它
		RequestId: trans.Ptr("req-sweep-sent"),
	})
	require.NoError(t, err)
	nullTimeID, err := repo.Create(ctx, newDelivery("req-sweep-nulltime", notificationV1.Channel_SMS, nil))
	require.NoError(t, err)

	count, err := repo.SweepStaleSending(ctx, before, "sweep: stale sending beyond budget", 100)
	require.NoError(t, err)
	require.Equal(t, 1, count, "只有 stale 一行该被扫掉")

	staleRow, err := entClient.Client().NotificationDelivery.Get(ctx, staleID.GetId())
	require.NoError(t, err)
	require.Equal(t, notificationdelivery.StatusFailed, *staleRow.Status, "超期未结算应落 FAILED 终态")
	require.NotNil(t, staleRow.LastError)
	require.Equal(t, "sweep: stale sending beyond budget", *staleRow.LastError)
	require.NotNil(t, staleRow.Attempts)
	require.Equal(t, uint32(3), *staleRow.Attempts, "清扫不得抹掉最后一次真实尝试计数")

	freshRow, err := entClient.Client().NotificationDelivery.Get(ctx, freshID.GetId())
	require.NoError(t, err)
	require.Equal(t, notificationdelivery.StatusSending, *freshRow.Status,
		"未到 cutoff 的行不得被扫：它可能正被异步 handler 收尾")
	require.Nil(t, freshRow.LastError)

	sentRow, err := entClient.Client().NotificationDelivery.Get(ctx, sentID.GetId())
	require.NoError(t, err)
	require.Equal(t, notificationdelivery.StatusSent, *sentRow.Status,
		"已有结论的行不得被扫：status==SENDING 在 WHERE 里，改写结论不是清扫的活")

	nullRow, err := entClient.Client().NotificationDelivery.Get(ctx, nullTimeID.GetId())
	require.NoError(t, err)
	require.Equal(t, notificationdelivery.StatusSending, *nullRow.Status,
		"created_at 为 NULL 的行不参与比较，也不该被扫")

	// 再扫一遍应无事可做（幂等，不再产出新终态）
	count, err = repo.SweepStaleSending(ctx, before, "sweep again", 100)
	require.NoError(t, err)
	require.Equal(t, 0, count, "没有新的 stale 行时清扫应返回 0")

	// 入参守卫
	_, err = repo.SweepStaleSending(ctx, before, "x", 0)
	require.Error(t, err, "limit=0 应返回 BadRequest")
	_, err = repo.SweepStaleSending(ctx, before, "x", -1)
	require.Error(t, err, "limit<0 应返回 BadRequest")
}

// TestNotificationDeliveryRepoSqlite_SweepLimit 验证 limit 只放行一批：
// 3 行 stale、limit=2 时本次扫 2 行，剩 1 行留给下一轮（清扫是限批的例行任务，
// 一次全表扫会跟在线投递抢写锁）。
func TestNotificationDeliveryRepoSqlite_SweepLimit(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewNotificationDeliveryRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	before := time.Now()
	staleAt := timestamppb.New(before.Add(-30 * time.Minute))
	for _, rid := range []string{"req-limit-1", "req-limit-2", "req-limit-3"} {
		_, err := repo.Create(ctx, newDelivery(rid, notificationV1.Channel_WEBHOOK, staleAt))
		require.NoError(t, err)
	}

	count, err := repo.SweepStaleSending(ctx, before, "sweep batch", 2)
	require.NoError(t, err)
	require.Equal(t, 2, count, "limit=2 时本次最多扫 2 行")

	left, err := entClient.Client().NotificationDelivery.Query().
		Where(notificationdelivery.StatusEQ(notificationdelivery.StatusSending)).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, left, "剩余 1 行 SENDING 留给下一轮清扫")

	// 下一轮把尾巴收掉
	count, err = repo.SweepStaleSending(ctx, before, "sweep batch", 2)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

// TestNotificationDeliveryRepoSqlite_RequestChannelUnique 验证 (request_id, channel)
// 唯一索引的语义：同一次派发意图 × 同一渠道不许变成两行真实投递（重复入队在写入这步
// 被 DB 挡下）；扇出到第二个渠道是合法的两行；换意图同渠道也合法。
func TestNotificationDeliveryRepoSqlite_RequestChannelUnique(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewNotificationDeliveryRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	_, err := repo.Create(ctx, newDelivery("intent-1", notificationV1.Channel_EMAIL, nil))
	require.NoError(t, err, "首行应正常落库")

	// 同一意图 × 同一渠道：第二行必须被唯一索引拒绝
	_, err = repo.Create(ctx, newDelivery("intent-1", notificationV1.Channel_EMAIL, nil))
	require.Error(t, err, "重复的 (request_id, channel) 应被 DB 挡下")
	require.Contains(t, err.Error(), "insert notification delivery failed")

	// 同一意图扇出到第二个渠道：合法的两行台账
	_, err = repo.Create(ctx, newDelivery("intent-1", notificationV1.Channel_WEBHOOK, nil))
	require.NoError(t, err, "同意图换渠道（扇出）应允许成行")

	// 换意图 × 同渠道：合法
	_, err = repo.Create(ctx, newDelivery("intent-2", notificationV1.Channel_EMAIL, nil))
	require.NoError(t, err)

	cnt, err := entClient.Client().NotificationDelivery.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, cnt, "最终应恰有 3 行台账")
}
