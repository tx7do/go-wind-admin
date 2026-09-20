// NotificationService 的 SQLite 内存库集成测试（白盒，包内测试）。
//
// 覆盖目标：投递 → 台账的状态机（SENDING → SENT / FAILED / SKIPPED），以及三条容易被想当然的语义：
//   - 台账写的是脱敏目标，**真正发给渠道的是原文**（脱敏串一旦漏进 SendRequest，邮件永远发不到）；
//   - 读回路径三个枚举如实呈现（schema 上它们刻意做成 Optional().Nillable()，
//     换成值型枚举列就会踩 copier 的"指针↔指针对"失配，读回恒为零值）；
//   - 路由失败/入参不合法时**不产生台账行**——台账只记"确实要发的那一次"。
//
// 跳过项：EmailSender 的真实 SMTP 与渠道选择（依赖 sys_notification_channels 数据，
// 见 data/channel 侧；本批用替身 Sender 聚焦服务层结论）。
package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/channel"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/constants"

	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"

	appViewer "go-wind-admin/pkg/entgo/viewer"
)

// fakeSender 可编排的渠道替身：记录收到的每一次投递请求，按预设返回回执或错误。
type fakeSender struct {
	channel notificationV1.Channel
	receipt *channel.SendReceipt
	err     error

	// precheckErr 非空时 Precheck 报它：用来测异步派发的"入队前同步预检"分支。
	precheckErr error

	calls []*channel.SendRequest
}

func (f *fakeSender) Channel() notificationV1.Channel { return f.channel }

func (f *fakeSender) Send(_ context.Context, req *channel.SendRequest) (*channel.SendReceipt, error) {
	f.calls = append(f.calls, req)
	return f.receipt, f.err
}

func (f *fakeSender) Precheck(_ context.Context, _ uint32) error { return f.precheckErr }

type notificationSvcEnv struct {
	svc   *NotificationService
	ctx   context.Context
	email *fakeSender

	// client 是同一份内存库的 ent 客户端，只给测试注入故障用（见 notification_record_sqlite_test.go
	// 的 armRecordWriteFailure）——生产代码不从这里走。
	client *ent.Client
}

func newNotificationServiceForTest(t *testing.T) *notificationSvcEnv {
	t.Helper()

	entClient := enttest.NewEntClientForTest(t)
	email := &fakeSender{channel: notificationV1.Channel_EMAIL, receipt: &channel.SendReceipt{ChannelID: 42}}

	registry := channel.NewRegistry()
	registry.Register(email)

	svc := &NotificationService{
		log:          bLogger.NewHelper(bLogger.NopLogger()),
		deliveryRepo: data.NewNotificationDeliveryRepoForTest(entClient),
		ruleRepo:     newSeededRuleRepoForTest(t, entClient),
		channels:     registry,
	}

	return &notificationSvcEnv{
		svc:     svc,
		ctx:     appViewer.NewSystemViewerContext(context.Background()),
		email:   email,
		client:  entClient.Client(),
	}
}

// newSeededRuleRepoForTest 造一个已播种默认路由的规则仓储。
//
// 自 P2-C 起每次投递都要读 sys_notification_rules 才知道走哪个渠道，服务层测试因此必须
// 先有这几行：否则每一条 SendDirect 都撞在"没有路由规则"上，测到的全是路由缺失分支。
// 播种走生产同一份 constants.DefaultNotificationRules，两边不可能漂移。
func newSeededRuleRepoForTest(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *data.NotificationRuleRepo {
	t.Helper()

	repo := data.NewNotificationRuleRepoForTest(entClient)
	ctx := appViewer.NewSystemViewerContext(context.Background())

	for _, rule := range constants.DefaultNotificationRules {
		if _, err := repo.Create(ctx, rule, 0); err != nil {
			t.Fatalf("seed rule [%s] failed: %v", rule.GetEventType().String(), err)
		}
	}

	return repo
}

func directReq(eventType notificationV1.EventType) *notificationV1.SendDirectNotificationRequest {
	return &notificationV1.SendDirectNotificationRequest{
		EventType: eventType,
		Target:    "bob@example.com",
		Title:     "重置验证码",
		Content:   "您的验证码是 123456",
	}
}

// countDeliveries 台账当前行数：用来断言"没发就不该留痕"。
func (e *notificationSvcEnv) countDeliveries(t *testing.T) uint64 {
	t.Helper()
	resp, err := e.svc.deliveryRepo.List(e.ctx, &paginationV1.PagingRequest{
		Page:     trans.Ptr(uint32(1)),
		PageSize: trans.Ptr(uint32(10)),
	})
	require.NoError(t, err)
	return resp.GetTotal()
}

// TestNotificationServiceSqlite_SendSent 成功投递：回执与结果如实落台账。
func TestNotificationServiceSqlite_SendSent(t *testing.T) {
	e := newNotificationServiceForTest(t)

	resp, err := e.svc.SendDirect(e.ctx, directReq(notificationV1.EventType_PASSWORD_RESET_CODE))
	require.NoError(t, err)
	require.Equal(t, notificationV1.DeliveryStatus_SENT, resp.GetStatus())

	// 路由表：PASSWORD_RESET_CODE → EMAIL
	require.Len(t, e.email.calls, 1)
	sent := e.email.calls[0]
	require.Equal(t, "bob@example.com", sent.Target,
		"发给渠道的必须是原始地址：脱敏只作用于台账，漏进这里邮件永远发不到")
	require.Equal(t, "重置验证码", sent.Title)
	require.Equal(t, "您的验证码是 123456", sent.Content)
	require.Zero(t, sent.ChannelID, "未显式指定渠道时传 0，由渠道自选策略处理")

	got, err := e.svc.GetNotificationDelivery(e.ctx, &notificationV1.GetNotificationDeliveryRequest{Id: resp.GetDeliveryId()})
	require.NoError(t, err)
	require.Equal(t, notificationV1.DeliveryStatus_SENT, got.GetStatus())
	require.Equal(t, notificationV1.EventType_PASSWORD_RESET_CODE, got.GetEventType(),
		"枚举读回应如实（schema 侧 Optional().Nillable() 是这一条的前提）")
	require.Equal(t, notificationV1.Channel_EMAIL, got.GetChannel())
	require.Equal(t, uint32(42), got.GetChannelId(), "自选渠道的结果要落档，排障时才答得出走的哪个 SMTP")
	require.Equal(t, "b***@example.com", got.GetTarget())
	require.NotNil(t, got.GetSentAt())
	require.Empty(t, got.GetLastError())
}

// TestNotificationServiceSqlite_SendFailed 渠道报错：FAILED + 原因，且不落 sent_at。
func TestNotificationServiceSqlite_SendFailed(t *testing.T) {
	e := newNotificationServiceForTest(t)
	e.email.err = fmt.Errorf("smtp auth failed: 535")

	resp, err := e.svc.SendDirect(e.ctx, directReq(notificationV1.EventType_CONTACT_BIND_CODE))
	require.Error(t, err, "调用方必须拿得到 error，否则会以为发出去了")
	require.Equal(t, notificationV1.DeliveryStatus_FAILED, resp.GetStatus())

	got, getErr := e.svc.GetNotificationDelivery(e.ctx, &notificationV1.GetNotificationDeliveryRequest{Id: resp.GetDeliveryId()})
	require.NoError(t, getErr)
	require.Equal(t, notificationV1.DeliveryStatus_FAILED, got.GetStatus())
	require.Contains(t, got.GetLastError(), "smtp auth failed")
	require.Nil(t, got.GetSentAt(), "没发出去就不该有完成时间")
	require.Nil(t, got.ChannelId, "发送前就失败时渠道未确认，不得凭空补 ID")
}

// TestNotificationServiceSqlite_SendSkipped 渠道没配/没启用：SKIPPED，与"渠道报错"区分开。
func TestNotificationServiceSqlite_SendSkipped(t *testing.T) {
	e := newNotificationServiceForTest(t)
	e.email.err = fmt.Errorf("%w: no enabled EMAIL channel", channel.ErrChannelNotConfigured)

	resp, err := e.svc.SendDirect(e.ctx, directReq(notificationV1.EventType_PASSWORD_RESET_CODE))
	require.Error(t, err)
	require.Equal(t, notificationV1.DeliveryStatus_SKIPPED, resp.GetStatus(),
		"一条都没尝试发＝配置问题（SKIPPED），不是投递失败（FAILED）：找回密码链路按这个区分回错误文案")

	got, getErr := e.svc.GetNotificationDelivery(e.ctx, &notificationV1.GetNotificationDeliveryRequest{Id: resp.GetDeliveryId()})
	require.NoError(t, getErr)
	require.Equal(t, notificationV1.DeliveryStatus_SKIPPED, got.GetStatus())
}

// TestNotificationServiceSqlite_UnregisteredChannel 路由指向没有实现的渠道：FAILED。
func TestNotificationServiceSqlite_UnregisteredChannel(t *testing.T) {
	e := newNotificationServiceForTest(t)

	// 显式请求 SMS，注册表里只有 EMAIL —— 属于"代码/配置不自洽"，必须留下痕迹而不是静默丢弃。
	resp, err := e.svc.SendDirect(e.ctx, &notificationV1.SendDirectNotificationRequest{
		EventType: notificationV1.EventType_CHANNEL_TEST_EMAIL,
		Channel:   trans.Ptr(notificationV1.Channel_SMS),
		Target:    "13800000000",
		Title:     "t",
		Content:   "c",
	})
	require.Error(t, err)
	require.Equal(t, notificationV1.DeliveryStatus_FAILED, resp.GetStatus())

	got, getErr := e.svc.GetNotificationDelivery(e.ctx, &notificationV1.GetNotificationDeliveryRequest{Id: resp.GetDeliveryId()})
	require.NoError(t, getErr)
	require.Contains(t, got.GetLastError(), "not registered")
}

// TestNotificationServiceSqlite_ExplicitChannelBeatsRouting 请求显式带渠道时优先生效。
func TestNotificationServiceSqlite_ExplicitChannelBeatsRouting(t *testing.T) {
	e := newNotificationServiceForTest(t)

	webhook := &fakeSender{channel: notificationV1.Channel_WEBHOOK, receipt: &channel.SendReceipt{}}
	e.svc.channels.Register(webhook)

	// CHANNEL_TEST_EMAIL 的路由默认是 EMAIL，显式指定 WEBHOOK 应覆盖它。
	_, err := e.svc.SendDirect(e.ctx, &notificationV1.SendDirectNotificationRequest{
		EventType: notificationV1.EventType_CHANNEL_TEST_EMAIL,
		Channel:   trans.Ptr(notificationV1.Channel_WEBHOOK),
		Target:    "https://example.com/hook",
		Title:     "t",
		Content:   "c",
	})
	require.NoError(t, err)
	require.Len(t, webhook.calls, 1, "显式渠道应命中该渠道的实现")
	require.Empty(t, e.email.calls, "EMAIL 不该被叫到")
}

// TestNotificationServiceSqlite_NoLedgerRowWhenNotAttempted 未进入投递就不留痕。
func TestNotificationServiceSqlite_NoLedgerRowWhenNotAttempted(t *testing.T) {
	e := newNotificationServiceForTest(t)

	// 入参不全
	_, err := e.svc.SendDirect(e.ctx, &notificationV1.SendDirectNotificationRequest{
		EventType: notificationV1.EventType_PASSWORD_RESET_CODE,
		Target:    "bob@example.com",
	})
	require.Error(t, err, "缺标题/正文应报错")

	// 路由表里没有这个事件（UNSPECIFIED）
	_, err = e.svc.SendDirect(e.ctx, directReq(notificationV1.EventType_EVENT_TYPE_UNSPECIFIED))
	require.Error(t, err)

	require.Zero(t, e.countDeliveries(t), "这两种情况一条都没发，台账不该出现空转记录")
}

// updateRule 走仓储改规则行（与规则管理页同一条写路径），mask 指定字段掩码。
func (e *notificationSvcEnv) updateRule(t *testing.T, eventType notificationV1.EventType, data *notificationV1.NotificationRule, mask ...string) {
	t.Helper()

	rule, err := e.svc.ruleRepo.GetByEventType(e.ctx, eventType)
	require.NoError(t, err)
	require.NotNil(t, rule, "默认路由里应当有 %s 这一行", eventType.String())

	require.NoError(t, e.svc.ruleRepo.Update(e.ctx, &notificationV1.UpdateNotificationRuleRequest{
		Id:         rule.GetId(),
		UpdateMask: &fieldmaskpb.FieldMask{Paths: mask},
		Data:       data,
	}, 0), "更新规则行应成功")
}

// TestNotificationServiceSqlite_RuleRowDecidesChannel 改库里的规则行立刻换渠道：
// 这一条测的是"DB 是唯一真相、进程内没有缓存"。Go 静态表时代改一行配置要重启，
// 而留着缓存会让"改完之后的行为"取决于进程何时启动过——比原来的常量更难排。
func TestNotificationServiceSqlite_RuleRowDecidesChannel(t *testing.T) {
	e := newNotificationServiceForTest(t)
	webhook := &fakeSender{channel: notificationV1.Channel_WEBHOOK, receipt: &channel.SendReceipt{}}
	e.svc.channels.Register(webhook)

	// 播种值：PASSWORD_RESET_CODE 走 EMAIL、异步（但本 env 没有队列，会当场投）。
	_, err := e.svc.SendDirect(e.ctx, directReq(notificationV1.EventType_PASSWORD_RESET_CODE))
	require.NoError(t, err)
	require.Len(t, e.email.calls, 1, "默认路由应命中 EMAIL 实现")

	e.updateRule(t, notificationV1.EventType_PASSWORD_RESET_CODE,
		&notificationV1.NotificationRule{Channel: notificationV1.Channel_WEBHOOK.Enum()}, "channel")

	second, err := e.svc.SendDirect(e.ctx, directReq(notificationV1.EventType_PASSWORD_RESET_CODE))
	require.NoError(t, err)
	require.Len(t, webhook.calls, 1, "改完规则的下一次投递就该走新渠道")
	require.Len(t, e.email.calls, 1, "旧渠道不该再被叫到")

	got, getErr := e.svc.GetNotificationDelivery(e.ctx, &notificationV1.GetNotificationDeliveryRequest{
		Id: second.GetDeliveryId(),
	})
	require.NoError(t, getErr)
	require.Equal(t, notificationV1.Channel_WEBHOOK, got.GetChannel(), "台账记的是实际用的渠道，不是播种值")
}

// TestNotificationServiceSqlite_RuleRowDecidesAsync 规则行的 is_async 决定"要不要离开请求"：
// 关掉异步 → 找回密码当场投递并拿到结论；打开异步 → 测试邮件也入队。
// 这一列原先是 Go 常量 asyncDispatchEvents，因此两侧都要能改得动才算真的平移完了。
func TestNotificationServiceSqlite_RuleRowDecidesAsync(t *testing.T) {
	e := newDispatchEnv(t)

	e.updateRule(t, notificationV1.EventType_PASSWORD_RESET_CODE,
		&notificationV1.NotificationRule{IsAsync: trans.Ptr(false)}, "is_async")

	resp, err := e.svc.SendDirect(e.ctx, directReq(notificationV1.EventType_PASSWORD_RESET_CODE))
	require.NoError(t, err)
	require.Equal(t, notificationV1.DeliveryStatus_SENT, resp.GetStatus(), "同步派发应在请求里就拿到结论")
	require.Len(t, e.email.calls, 1, "关掉异步后当场拨号")
	require.Empty(t, e.queue.calls, "一次都不该入队")

	// 反向：把原先同步的 CHANNEL_TEST_EMAIL 改成异步
	e.updateRule(t, notificationV1.EventType_CHANNEL_TEST_EMAIL,
		&notificationV1.NotificationRule{IsAsync: trans.Ptr(true)}, "is_async")

	resp, err = e.svc.SendDirect(e.ctx, directReq(notificationV1.EventType_CHANNEL_TEST_EMAIL))
	require.NoError(t, err)
	require.Equal(t, notificationV1.DeliveryStatus_SENDING, resp.GetStatus())
	require.Len(t, e.queue.calls, 1, "规则行标为异步的事件必须入队")
	require.Len(t, e.email.calls, 1, "入队的那一趟不该再拨号")
}

// TestNotificationServiceSqlite_DisabledRuleNeedsExplicitChannel 规则停用 = 这条路由不生效：
// 没点名渠道时报错（不静默丢通知），点名了渠道则放行——"点名的渠道 + 同步投递"是一个完整决定。
// 缺行（删掉规则）与停用走同一分支，故一并测。
func TestNotificationServiceSqlite_DisabledRuleNeedsExplicitChannel(t *testing.T) {
	e := newNotificationServiceForTest(t)

	e.updateRule(t, notificationV1.EventType_PASSWORD_RESET_CODE,
		&notificationV1.NotificationRule{IsEnabled: trans.Ptr(false)}, "is_enabled")

	_, err := e.svc.SendDirect(e.ctx, directReq(notificationV1.EventType_PASSWORD_RESET_CODE))
	require.Error(t, err, "规则停用又没点名渠道时必须报错，而不是退回旧的 Go 表")
	require.Contains(t, err.Error(), "no enabled channel routing rule for event type PASSWORD_RESET_CODE")
	require.Contains(t, err.Error(), "notification-rules page", "错误要指出去哪儿修")

	// 显式点名渠道：放行，且按同步投递（规则行没生效，is_async 也就无从谈起）
	resp, err := e.svc.SendDirect(e.ctx, &notificationV1.SendDirectNotificationRequest{
		EventType: notificationV1.EventType_PASSWORD_RESET_CODE,
		Channel:   trans.Ptr(notificationV1.Channel_EMAIL),
		Target:    "bob@example.com",
		Title:     "t",
		Content:   "c",
	})
	require.NoError(t, err, "点名了渠道的决定是完整的，规则缺位不该拦住它")
	require.Equal(t, notificationV1.DeliveryStatus_SENT, resp.GetStatus())

	// 删掉整行：与停用同语义
	rule, getErr := e.svc.ruleRepo.GetByEventType(e.ctx, notificationV1.EventType_CONTACT_BIND_CODE)
	require.NoError(t, getErr)
	require.NotNil(t, rule)
	require.NoError(t, e.svc.ruleRepo.Delete(e.ctx, rule.GetId()))

	_, err = e.svc.SendDirect(e.ctx, directReq(notificationV1.EventType_CONTACT_BIND_CODE))
	require.Error(t, err, "规则行被删掉后该事件不再通知，且要说人话")
	require.Contains(t, err.Error(), "no enabled channel routing rule for event type CONTACT_BIND_CODE")
}

// TestMaskTarget 投递目标脱敏形态：站外地址掩，站内收件人 ID 原样留。
func TestMaskTarget(t *testing.T) {
	cases := []struct {
		ch   notificationV1.Channel
		in   string
		want string
	}{
		{notificationV1.Channel_EMAIL, "bob@example.com", "b***@example.com"},
		{notificationV1.Channel_EMAIL, "a.b+tag@example.com", "a***@example.com"},
		{notificationV1.Channel_WEBHOOK, "https://hooks.example.com/abc123", "****c123"},
		{notificationV1.Channel_SMS, "13800000000", "****0000"},
		{notificationV1.Channel_SMS, "", "****"},
		// INTERNAL 的"地址"就是本平台用户主键：掩成 ****1024 会把台账里唯一可读的字段变成噪音。
		{notificationV1.Channel_INTERNAL, "1024", "1024"},
		{notificationV1.Channel_INTERNAL, "7", "7"},
	}
	for _, c := range cases {
		require.Equal(t, c.want, maskTarget(c.ch, c.in), "渠道 %s 输入 %q", c.ch.String(), c.in)
	}
}
