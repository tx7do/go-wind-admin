// 站内信 ⇄ 通知域接缝的 SQLite 集成测试（白盒，包内测试）。
//
// 钉住 P2 第一块的四件事，全是"结构审查看不出来、只有跑一次才看得见"的那类：
//   - 定向发送确实经过了缝：台账里有一行 INTERNAL_MESSAGE / INTERNAL / SENT，
//     并且 related_id 指回消息本体（没有它，台账行回答不了"发的是哪条消息"）；
//   - 台账写的是可读的用户 ID，发给渠道的是同一个值（INTERNAL 不脱敏，见 maskTarget）；
//   - 缝没接上（notifier 仍是未装配占位）时不写收件行、留下错误，而不是静默成功；
//   - 全员广播的 SSE 载荷带收件行主键——缺 id 时 vue-element 的通知面板在第一行守卫处
//     return，整场广播不弹桌面通知、不涨未读数，且不报任何错。
package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-crud/viewer"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/tx7do/kratos-transport/transport/sse"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/channel"
	"go-wind-admin/app/admin/service/internal/data/enttest"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	internalMessageV1 "go-wind-admin/api/gen/go/internal_message/service/v1"
	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"

	appViewer "go-wind-admin/pkg/entgo/viewer"
	"go-wind-admin/pkg/middleware/auth"
)

// payloadRecordingPublisher 记下每次 TryPublish 的事件体，供"载荷到底长什么样"的断言使用。
type payloadRecordingPublisher struct {
	events []sse.Event
}

func (p *payloadRecordingPublisher) Publish(_ context.Context, _ sse.StreamID, event *sse.Event) {
	if event != nil {
		p.events = append(p.events, *event)
	}
}

func (p *payloadRecordingPublisher) TryPublish(_ context.Context, _ sse.StreamID, event *sse.Event) bool {
	p.Publish(nil, "", event)
	return true
}

// ssePayloadAt 把第 n 条推送的 JSON 解成 map：断言键存在性时用，不许比对字节
// （protojson 会在字段间插入随机空白）。
func ssePayloadAt(t *testing.T, p *payloadRecordingPublisher, idx int) map[string]any {
	t.Helper()
	require.Greater(t, len(p.events), idx, "期望的 SSE 推送不存在")

	var got map[string]any
	require.NoError(t, json.Unmarshal(p.events[idx].Data, &got))
	return got
}

// notifySeamEnv 按生产装配的形状把两侧接起来：Registry 里放 InternalMessageSender，
// InternalMessageService 的 notifier 换成 NotificationService。
type notifySeamEnv struct {
	im       *InternalMessageService
	notifier *NotificationService
	ctx      context.Context
	pub      *payloadRecordingPublisher
}

func newNotifySeamEnv(t *testing.T) *notifySeamEnv {
	t.Helper()

	entClient := enttest.NewEntClientForTest(t)
	pub := &payloadRecordingPublisher{}

	im := &InternalMessageService{
		log:                          bLogger.NewHelper(bLogger.NopLogger()),
		internalMessageRepo:          data.NewInternalMessageRepoForTest(entClient),
		internalMessageCategoryRepo:  data.NewInternalMessageCategoryRepoForTest(entClient),
		internalMessageRecipientRepo: data.NewInternalMessageRecipientRepoForTest(entClient),
		userRepo: &internalMessageServiceUserRepoStub{
			// 定向投递的收件人 1024 属于租户 5，而本测试的 ctx 是 SystemViewer（租户 0）：
			// 收件行落在 5 才说明打标跟的是收件人，不是操作人/viewer。
			tenantByUserID: map[uint32]uint32{1024: 5},
		},
		internalMessagePublisher: pub,
		notifier:                 unwiredNotifier{},
	}

	registry := channel.NewRegistry()
	registry.Register(NewInternalMessageSender(im))

	notifier := &NotificationService{
		log:          bLogger.NewHelper(bLogger.NopLogger()),
		deliveryRepo: data.NewNotificationDeliveryRepoForTest(entClient),
		ruleRepo:     newSeededRuleRepoForTest(t, entClient),
		channels:     registry,
	}
	im.RegisterNotifier(notifier)

	ctx := auth.NewContext(
		enttest.NewSystemViewerCtx(context.Background()),
		&authenticationV1.UserTokenPayload{UserId: 88},
	)

	return &notifySeamEnv{im: im, notifier: notifier, ctx: ctx, pub: pub}
}

func (e *notifySeamEnv) deliveries(t *testing.T) []*notificationV1.NotificationDelivery {
	t.Helper()
	resp, err := e.notifier.deliveryRepo.List(e.ctx, &paginationV1.PagingRequest{
		Page:     trans.Ptr(uint32(1)),
		PageSize: trans.Ptr(uint32(50)),
	})
	require.NoError(t, err)
	return resp.GetItems()
}

// TestNotifySeamDirectedSend 定向发送：收件行 + 一条 INTERNAL 台账 + 带主键的 SSE 载荷。
func TestNotifySeamDirectedSend(t *testing.T) {
	e := newNotifySeamEnv(t)

	resp, err := e.im.SendMessage(e.ctx, &internalMessageV1.SendMessageRequest{
		Type:            internalMessageV1.InternalMessage_NOTIFICATION,
		Title:           trans.Ptr("接缝测试标题"),
		Content:         "接缝测试正文",
		RecipientUserId: trans.Ptr(uint32(1024)),
	})
	require.NoError(t, err)

	items := e.deliveries(t)
	require.Len(t, items, 1, "站内信投递必须在台账里留一行——这正是 P2 接进来的目的")
	row := items[0]
	require.Equal(t, notificationV1.EventType_INTERNAL_MESSAGE, row.GetEventType())
	require.Equal(t, notificationV1.Channel_INTERNAL, row.GetChannel(),
		"渠道由 sys_notification_rules 的 INTERNAL_MESSAGE 一行决定（调用方没有显式传 Channel）")
	require.Equal(t, resp.GetMessageId(), row.GetRelatedId(), "台账要能跳回消息本体")
	require.Equal(t, uint32(1024), row.GetRecipientUserId())
	require.Equal(t, "1024", row.GetTarget(), "INTERNAL 的目标是用户 ID 原文，不脱敏")
	require.Equal(t, notificationV1.DeliveryStatus_SENT, row.GetStatus())
	require.Zero(t, row.GetChannelId(), "站内信不吃渠道花名册，channel_id 保持空")
	require.Equal(t, uint32(88), row.GetCreatedBy(), "操作人由调用方显式传入")

	inbox, err := e.im.internalMessageRecipientRepo.List(e.ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Len(t, inbox.GetItems(), 1, "缝的另一端确实落了收件行")
	require.Equal(t, resp.GetMessageId(), inbox.GetItems()[0].GetMessageId())
	require.Equal(t, uint32(1024), inbox.GetItems()[0].GetRecipientUserId())
	require.Equal(t, uint32(5), inbox.GetItems()[0].GetTenantId(),
		"收件行必须打在收件人的租户上：viewer 是租户 0 的 SystemViewer，落在 5 才证明打标跟的是受众")

	payload := ssePayloadAt(t, e.pub, 0)
	require.NotZero(t, payload["id"], "SSE 载荷必须带收件行主键")
	require.Equal(t, float64(resp.GetMessageId()), payload["messageId"])
	// title/content 在收件表里没有列（读侧由消息本体回填），只有推送载荷带得到值。
	require.Equal(t, "接缝测试标题", payload["title"])
}

// TestNotifySeamUnwired 缝未装配时不能"看起来成功"：没有收件行，也没有台账。
func TestNotifySeamUnwired(t *testing.T) {
	e := newNotifySeamEnv(t)
	e.im.RegisterNotifier(unwiredNotifier{})

	err := e.im.deliverViaNotifier(e.ctx, 1, 1024, 88, "标题", "正文")
	require.Error(t, err, "未装配的 Notifier 必须报错，不能静默吞掉一次投递")

	inbox, err := e.im.internalMessageRecipientRepo.List(e.ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Empty(t, inbox.GetItems(), "报错发生在落库之前，不该留下半条投递")
}

// TestInternalMessageSenderRequiresMessageId 缺 related_id：渠道报错 → 台账 FAILED。
//
// 站内信没有"地址"这种独立目标，收件行必须挂在一个已存在的消息本体上；
// 少了 related_id 就是调用方的 bug，与"渠道用不了"（SKIPPED）区分开。
func TestInternalMessageSenderRequiresMessageId(t *testing.T) {
	e := newNotifySeamEnv(t)

	_, err := e.notifier.SendDirect(e.ctx, &notificationV1.SendDirectNotificationRequest{
		EventType:       notificationV1.EventType_INTERNAL_MESSAGE,
		Target:          "1024",
		Title:           "缺关联的投递",
		Content:         "正文",
		RecipientUserId: trans.Ptr(uint32(1024)),
	})
	require.Error(t, err)

	items := e.deliveries(t)
	require.Len(t, items, 1, "失败也要留台账行：它是这次尝试唯一的存在证明")
	require.Equal(t, notificationV1.DeliveryStatus_FAILED, items[0].GetStatus())
	require.Contains(t, items[0].GetLastError(), "related_id")
}

// TestBroadcastSsePayloadCarriesRecipientId 全员广播的推送载荷必须带收件行主键。
//
// 回归的是 CreateBulk（ON CONFLICT DO NOTHING）不返回实体这一步：收件行 id 留 0 时
// protojson 直接省略该键，vue-element 的 handleSseNotification 在
// `if (!data.id || !data.messageId) return` 处退出——公告发出去了、落了库，
// 但租户用户那边桌面通知、未读数、下拉列表全都不动，也没有任何错误。
func TestBroadcastSsePayloadCarriesRecipientId(t *testing.T) {
	e := newNotifySeamEnv(t)

	mkUsers := func(ids ...uint32) []*identityV1.User {
		users := make([]*identityV1.User, 0, len(ids))
		for _, id := range ids {
			users = append(users, &identityV1.User{Id: trans.Ptr(id)})
		}
		return users
	}
	e.im.userRepo = &broadcastUserRepoStub{
		usersByTenant: map[uint32][]*identityV1.User{7: mkUsers(701, 702, 703)},
	}

	msg, err := e.im.internalMessageRepo.Create(e.ctx, &internalMessageV1.CreateInternalMessageRequest{
		Data: &internalMessageV1.InternalMessage{
			TenantId:  trans.Ptr(uint32(7)),
			Title:     trans.Ptr("广播标题"),
			Content:   trans.Ptr("广播正文"),
			Status:    internalMessageV1.InternalMessage_PUBLISHED.Enum(),
			Type:      internalMessageV1.InternalMessage_NOTIFICATION.Enum(),
			CreatedBy: trans.Ptr(uint32(1)),
		},
	})
	require.NoError(t, err)

	// 广播在 asynq handler 的 ctx 上跑：viewer 按任务 payload 的租户重建，与 HTTP 请求 ctx 无关。
	broadcastCtx := viewer.WithContext(context.Background(), appViewer.NewUserViewer(0, 7, 0, "", nil))
	e.im.executeBroadcast(broadcastCtx, msg.GetId(), 1, "广播标题", "广播正文")

	require.Len(t, e.pub.events, 3, "三个收件人各推一条")
	for i := range e.pub.events {
		payload := ssePayloadAt(t, e.pub, i)
		require.NotZero(t, payload["id"], "第 %d 条推送缺收件行主键", i)
		require.Equal(t, float64(msg.GetId()), payload["messageId"])
	}
}
