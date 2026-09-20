// NotificationRuleService 的 SQLite 内存库测试（白盒，包内测试）。
//
// 覆盖两件只有接真库才成立的事：
//   - 播种守卫是 count==0 而非"按键补缺"：管理员删掉一行规则是"这个事件不再通知"的决定，
//     每次启动补回来等于禁用无效。代价（新增事件类型不会自动出现在已部署实例里）也在这里钉住；
//   - 测试投递按钮的语义：它必须把"这次投递走哪条路由"交回给规则行决定，
//     自己只补目标地址——所以 SendDirect 载荷里带 EventType 与可选 ChannelId，唯独不带 Channel。
//     一旦哪天有人顺手写死渠道，这个按钮测的就不再是那条规则了。
//
// 投递本身（台账、脱敏、SKIPPED/FAILED 归类）由 notification_service_sqlite_test.go 覆盖，
// 这里用 recordingNotifier 替身，只断言"按钮到底把什么交给了缝"。
package service

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	notificationChannelV1 "go-wind-admin/api/gen/go/notification_channel/service/v1"
	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/constants"
	"go-wind-admin/pkg/middleware/auth"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
)

// recordingNotifier Notifier 替身：记下收到的每一次 SendDirect 请求，按预设回结论或错误。
type recordingNotifier struct {
	calls []*notificationV1.SendDirectNotificationRequest
	resp  *notificationV1.SendNotificationResponse
	err   error
}

func (f *recordingNotifier) SendDirect(_ context.Context, req *notificationV1.SendDirectNotificationRequest) (*notificationV1.SendNotificationResponse, error) {
	f.calls = append(f.calls, req)
	return f.resp, f.err
}

type ruleSvcEnv struct {
	svc      *NotificationRuleService
	notifier *recordingNotifier
	repo     *data.NotificationRuleRepo
	channel  *data.NotificationChannelRepo
	ctx      context.Context
	opCtx    context.Context
}

// newRuleSvcEnv 白盒构造规则服务：log 换 NopLogger，repo/channel 走真仓储（同库），
// notifier 用替身。不调用 svc.init()——播种本身是被测对象，各测试自己决定调不调。
func newRuleSvcEnv(t *testing.T) *ruleSvcEnv {
	t.Helper()

	entClient := enttest.NewEntClientForTest(t)
	notifier := &recordingNotifier{resp: &notificationV1.SendNotificationResponse{
		DeliveryId: 777,
		Status:     notificationV1.DeliveryStatus_SENT,
	}}

	env := &ruleSvcEnv{
		repo:     data.NewNotificationRuleRepoForTest(entClient),
		channel:  data.NewNotificationChannelRepoForTest(entClient),
		notifier: notifier,
	}
	env.svc = &NotificationRuleService{
		log:      bLogger.NewHelper(bLogger.NopLogger()),
		repo:     env.repo,
		channel:  env.channel,
		notifier: notifier,
	}

	// 规则表是平台级配置，服务层 requirePlatformAdmin 挡住租户侧，所以这个 ctx 带平台管理员标志。
	// ctx 与 opCtx 取同一个值："无操作人"那一格由 Crud 现造裸 ctx 来测。
	env.ctx = auth.NewContext(enttest.NewSystemViewerCtx(context.Background()),
		&authenticationV1.UserTokenPayload{UserId: 88, IsPlatformAdmin: trans.Ptr(true)})
	env.opCtx = env.ctx

	return env
}

// seedViaInit 走生产播种路径，返回规则服务本身。
func (e *ruleSvcEnv) seedViaInit(t *testing.T) {
	t.Helper()
	e.svc.init()
}

// createWebhookChannel 落一条 WEBHOOK 渠道配置并返回主键。
func (e *ruleSvcEnv) createWebhookChannel(t *testing.T, name, url string, enabled bool) uint32 {
	t.Helper()

	id, err := e.channel.Create(e.ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:       trans.Ptr("rule-svc-wh-" + name),
			Type:       notificationChannelV1.NotificationChannel_WEBHOOK.Enum(),
			WebhookUrl: trans.Ptr(url),
			Enabled:    trans.Ptr(enabled),
		},
	}, 88)
	require.NoError(t, err)
	return id
}

// setEnabled 改一条渠道配置的启用状态（走渠道仓储的同一条掩码写路径）。
func (e *ruleSvcEnv) setEnabled(t *testing.T, id uint32, enabled bool) {
	t.Helper()

	require.NoError(t, e.channel.Update(e.ctx, &notificationChannelV1.UpdateNotificationChannelRequest{
		Id:         id,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"enabled"}},
		Data:       &notificationChannelV1.NotificationChannel{Enabled: trans.Ptr(enabled)},
	}, 88))
}

// TestNotificationRuleServiceSqlite_InitSeedsDefaults 播种：空表时按 constants 写满四行、
// 二次启动不重复；删掉一行后再启动不复活（count==0 守卫的既定语义，也是"管理员的删除是决定"
// 这条设计取舍的可执行记录）。
func TestNotificationRuleServiceSqlite_InitSeedsDefaults(t *testing.T) {
	e := newRuleSvcEnv(t)

	cnt, err := e.repo.Count(e.ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "起点必须是空表")

	e.seedViaInit(t)

	list, err := e.repo.List(e.ctx, &paginationV1.PagingRequest{
		Page:     trans.Ptr(uint32(1)),
		PageSize: trans.Ptr(uint32(20)),
	})
	require.NoError(t, err)
	require.Equal(t, uint64(len(constants.DefaultNotificationRules)), list.GetTotal(),
		"播种行数必须与 constants 的行数一致（只有一份来源，不允许漂移）")

	byEvent := map[notificationV1.EventType]*notificationV1.NotificationRule{}
	for _, it := range list.GetItems() {
		byEvent[it.GetEventType()] = it
	}
	for _, want := range constants.DefaultNotificationRules {
		got, ok := byEvent[want.GetEventType()]
		require.True(t, ok, "播种应覆盖事件 %s", want.GetEventType().String())
		require.Equal(t, want.GetChannel(), got.GetChannel(), "渠道要与 constants 一致")
		require.Equal(t, want.GetIsAsync(), got.GetIsAsync(), "is_async 要与 constants 一致")
		require.True(t, got.GetIsEnabled(), "播种行一律启用，否则路由等于没配")
	}

	// 二次启动：count>0 直接返回，不重复插行（重复会被 event_type 唯一索引挡下并刷一屏错误日志）
	e.seedViaInit(t)
	cnt, err = e.repo.Count(e.ctx)
	require.NoError(t, err)
	require.Equal(t, len(constants.DefaultNotificationRules), cnt, "第二次播种不该新增行")

	// 删掉 INTERNAL_MESSAGE 一行 = "这个事件不再通知"的决定，重启不该复活它。
	// 代价同步钉住：新增事件类型时已部署实例不会自动多出这一行，只能手工加或重播空表。
	rule, err := e.repo.GetByEventType(e.ctx, notificationV1.EventType_INTERNAL_MESSAGE)
	require.NoError(t, err)
	require.NotNil(t, rule)
	require.NoError(t, e.repo.Delete(e.ctx, rule.GetId()))

	e.seedViaInit(t)
	cnt, err = e.repo.Count(e.ctx)
	require.NoError(t, err)
	require.Equal(t, len(constants.DefaultNotificationRules)-1, cnt, "被删掉的规则行不该在下次启动复活")
	miss, err := e.repo.GetByEventType(e.ctx, notificationV1.EventType_INTERNAL_MESSAGE)
	require.NoError(t, err)
	require.Nil(t, miss)
}

// TestNotificationRuleServiceSqlite_Crud 覆盖服务层的增删改查转发与入参守卫：
// 创建走 auth 里的操作人、回读整个 DTO；重复事件类型被拒；掩码外字段保持原值。
func TestNotificationRuleServiceSqlite_Crud(t *testing.T) {
	e := newRuleSvcEnv(t)

	created, err := e.svc.CreateNotificationRule(e.opCtx, &notificationV1.CreateNotificationRuleRequest{
		Data: &notificationV1.NotificationRule{
			EventType: notificationV1.EventType_PASSWORD_RESET_CODE.Enum(),
			Channel:   notificationV1.Channel_EMAIL.Enum(),
			IsAsync:   trans.Ptr(true),
			IsEnabled: trans.Ptr(true),
			Remark:    trans.Ptr("找回密码"),
		},
	})
	require.NoError(t, err, "创建规则应回带整行 DTO")
	require.NotZero(t, created.GetId())
	require.Equal(t, notificationV1.EventType_PASSWORD_RESET_CODE, created.GetEventType())
	require.Equal(t, notificationV1.Channel_EMAIL, created.GetChannel())
	require.True(t, created.GetIsAsync())
	require.Equal(t, "找回密码", created.GetRemark())

	// 入参守卫
	_, err = e.svc.CreateNotificationRule(e.opCtx, nil)
	require.Error(t, err, "nil 请求应被拒绝")
	_, err = e.svc.CreateNotificationRule(e.opCtx, &notificationV1.CreateNotificationRuleRequest{})
	require.Error(t, err, "nil Data 应被拒绝")

	_, err = e.svc.CreateNotificationRule(e.opCtx, &notificationV1.CreateNotificationRuleRequest{
		Data: &notificationV1.NotificationRule{
			EventType: notificationV1.EventType_PASSWORD_RESET_CODE.Enum(),
			Channel:   notificationV1.Channel_WEBHOOK.Enum(),
		},
	})
	require.Error(t, err, "同一事件的第二行应被拒")
	require.Contains(t, err.Error(), "already has a routing rule")

	// 没有操作人上下文的写入必须失败（created_by 落不出一个存在的主人）。
	// 这里现造一个裸 viewer ctx：env.ctx 现在是平台管理员 ctx（服务层 requirePlatformAdmin 要过它）。
	_, err = e.svc.CreateNotificationRule(enttest.NewSystemViewerCtx(context.Background()),
		&notificationV1.CreateNotificationRuleRequest{
			Data: &notificationV1.NotificationRule{
				EventType: notificationV1.EventType_CONTACT_BIND_CODE.Enum(),
				Channel:   notificationV1.Channel_EMAIL.Enum(),
			},
		})
	require.Error(t, err, "缺鉴权上下文应拒绝写入")

	// Get
	got, err := e.svc.GetNotificationRule(e.ctx, &notificationV1.GetNotificationRuleRequest{Id: created.GetId()})
	require.NoError(t, err)
	require.Equal(t, created.GetId(), got.GetId())
	_, err = e.svc.GetNotificationRule(e.ctx, &notificationV1.GetNotificationRuleRequest{Id: 0})
	require.Error(t, err, "id=0 应被拒绝")
	_, err = e.svc.GetNotificationRule(e.ctx, nil)
	require.Error(t, err, "nil 请求应被拒绝")
	_, err = e.svc.GetNotificationRule(e.ctx, &notificationV1.GetNotificationRuleRequest{Id: 987654})
	require.Error(t, err, "不存在的 id 应返回 NotFound")

	// Update：只改派发方式，remark 保持原值
	_, err = e.svc.UpdateNotificationRule(e.opCtx, &notificationV1.UpdateNotificationRuleRequest{
		Id:         created.GetId(),
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"is_async"}},
		Data: &notificationV1.NotificationRule{
			IsAsync: trans.Ptr(false),
			Remark:  trans.Ptr("masked-out"),
		},
	})
	require.NoError(t, err)
	got, err = e.svc.GetNotificationRule(e.ctx, &notificationV1.GetNotificationRuleRequest{Id: created.GetId()})
	require.NoError(t, err)
	require.False(t, got.GetIsAsync(), "掩码内的 is_async 应更新")
	require.Equal(t, "找回密码", got.GetRemark(), "掩码外的 remark 应保持原值")
	require.NotNil(t, got.GetUpdatedAt(), "更新应写入 updated_at")

	_, err = e.svc.UpdateNotificationRule(e.opCtx, &notificationV1.UpdateNotificationRuleRequest{Id: created.GetId()})
	require.Error(t, err, "nil Data 的更新应被拒绝")

	// List
	list, err := e.svc.ListNotificationRule(e.ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(1), list.GetTotal())

	// Delete
	_, err = e.svc.DeleteNotificationRule(e.opCtx, &notificationV1.DeleteNotificationRuleRequest{Id: created.GetId()})
	require.NoError(t, err)
	list, err = e.svc.ListNotificationRule(e.ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Zero(t, list.GetTotal(), "删除后不该剩行")

	_, err = e.svc.DeleteNotificationRule(e.opCtx, &notificationV1.DeleteNotificationRuleRequest{Id: 0})
	require.Error(t, err, "id=0 的删除应被拒绝")
	_, err = e.svc.DeleteNotificationRule(e.opCtx, nil)
	require.Error(t, err, "nil 请求的删除应被拒绝")
}

// TestNotificationRuleServiceSqlite_TestDispatchDelegatesToRule 测试投递把决定权交回规则行：
// 载荷带 EventType 与目标，唯独不带 Channel；异步规则原样回 SENDING + 台账 ID
// （刻意不"顺手改成同步来测"，那样测的就不是这条规则了）。
func TestNotificationRuleServiceSqlite_TestDispatchDelegatesToRule(t *testing.T) {
	e := newRuleSvcEnv(t)
	e.seedViaInit(t)

	rule, err := e.repo.GetByEventType(e.ctx, notificationV1.EventType_PASSWORD_RESET_CODE)
	require.NoError(t, err)

	resp, err := e.svc.TestDispatchNotification(e.opCtx, &notificationV1.TestDispatchNotificationRequest{
		Id:      rule.GetId(),
		Target:  trans.Ptr("bob@example.com"),
		Title:   trans.Ptr("手工标题"),
		Content: trans.Ptr("手工正文"),
	})
	require.NoError(t, err)
	require.Equal(t, uint32(777), resp.GetDeliveryId(), "台账 ID 要回传：报错时靠它去投递记录页定位")
	require.Equal(t, notificationV1.DeliveryStatus_SENT, resp.GetStatus())

	require.Len(t, e.notifier.calls, 1)
	sent := e.notifier.calls[0]
	require.Equal(t, notificationV1.EventType_PASSWORD_RESET_CODE, sent.GetEventType())
	require.Nil(t, sent.Channel, "渠道由规则行决定，测试投递不许覆盖")
	require.Nil(t, sent.ChannelId, "没点名渠道配置时交由渠道自选")
	require.Equal(t, "bob@example.com", sent.GetTarget())
	require.Equal(t, "手工标题", sent.GetTitle())
	require.Equal(t, "手工正文", sent.GetContent())
	require.NotNil(t, sent.OperatorUserId)
	require.Equal(t, uint32(88), *sent.OperatorUserId, "台账要能回答是谁点的这个按钮")

	// 标题/正文留空：服务端按事件类型渲染测试文案（渲染本体见 pkg/mailtext 的测试）
	_, err = e.svc.TestDispatchNotification(e.opCtx, &notificationV1.TestDispatchNotificationRequest{
		Id:     rule.GetId(),
		Target: trans.Ptr("bob@example.com"),
	})
	require.NoError(t, err)
	require.NotEmpty(t, e.notifier.calls[1].GetTitle(), "缺省应渲染测试文案而不是发空邮件")
	require.NotEmpty(t, e.notifier.calls[1].GetContent())

	// 异步规则（PASSWORD_RESET_CODE 播种即 is_async=true）：状态原样透传，不强制同步
	e.notifier.resp = &notificationV1.SendNotificationResponse{
		DeliveryId: 778,
		Status:     notificationV1.DeliveryStatus_SENDING,
	}
	resp, err = e.svc.TestDispatchNotification(e.opCtx, &notificationV1.TestDispatchNotificationRequest{
		Id:     rule.GetId(),
		Target: trans.Ptr("bob@example.com"),
	})
	require.NoError(t, err)
	require.True(t, rule.GetIsAsync(), "这条种子规则本就该是异步")
	require.Equal(t, notificationV1.DeliveryStatus_SENDING, resp.GetStatus(),
		"异步规则测出来就是 SENDING：管理员要看的是真投递会发生什么")
}

// TestNotificationRuleServiceSqlite_TestDispatchWebhookTarget 目标留空只有 WEBHOOK 兜得出默认值：
// 取那条渠道登记的 webhook_url，并把 channel_id 钉成同一行——否则 sender 会另选一条启用的
// WEBHOOK 配置，管理员在 A 行点测试、实际打到 B 行。
func TestNotificationRuleServiceSqlite_TestDispatchWebhookTarget(t *testing.T) {
	e := newRuleSvcEnv(t)

	ruleID, err := e.repo.Create(e.ctx, &notificationV1.NotificationRule{
		EventType: notificationV1.EventType_CHANNEL_TEST_EMAIL.Enum(),
		Channel:   notificationV1.Channel_WEBHOOK.Enum(),
		IsEnabled: trans.Ptr(true),
	}, 88)
	require.NoError(t, err)

	// 没有任何 WEBHOOK 渠道：报错要指回缺配置这件事
	_, err = e.svc.TestDispatchNotification(e.opCtx, &notificationV1.TestDispatchNotificationRequest{Id: ruleID})
	require.Error(t, err, "一条 WEBHOOK 渠道都没配时不能凭空定出目标")

	// 有渠道但没填地址：兜底目标定不出来，报错要指回那一格没填
	blank := e.createWebhookChannel(t, "blank", "", true)
	_, err = e.svc.TestDispatchNotification(e.opCtx, &notificationV1.TestDispatchNotificationRequest{Id: ruleID})
	require.Error(t, err, "渠道没填 webhook_url 时不能凭空定出目标")
	require.Contains(t, err.Error(), "channel ["+strconv.FormatUint(uint64(blank), 10)+"] has no webhook_url configured",
		"错误要点出是哪一条配置，否则管理员会去查所有渠道")

	// 把这条停用，再放一条"启用 + 有地址"的行：自选只认启用的那条
	e.setEnabled(t, blank, false)
	idOn := e.createWebhookChannel(t, "live", "https://live.example.test/hook", true)

	_, err = e.svc.TestDispatchNotification(e.opCtx, &notificationV1.TestDispatchNotificationRequest{Id: ruleID})
	require.NoError(t, err)
	call := e.notifier.calls[len(e.notifier.calls)-1]
	require.Equal(t, "https://live.example.test/hook", call.GetTarget(), "留空的目标应取渠道登记的 webhook_url")
	require.NotNil(t, call.ChannelId, "兜底出来的目标必须同时钉住渠道配置行")
	require.Equal(t, idOn, call.GetChannelId(), "钉住的必须是兜底用的那一行，而不是 sender 另选的启用行")

	// 显式填了目标：以它为准，不越权钉渠道（"我要发到这个地址"是管理员的决定）
	const explicit = "https://other.example.test/hook"
	_, err = e.svc.TestDispatchNotification(e.opCtx, &notificationV1.TestDispatchNotificationRequest{
		Id:     ruleID,
		Target: trans.Ptr(explicit),
	})
	require.NoError(t, err)
	call = e.notifier.calls[len(e.notifier.calls)-1]
	require.Equal(t, explicit, call.GetTarget())
	require.Nil(t, call.ChannelId, "管理员自己填目标时不该替他选渠道配置")
}

// TestNotificationRuleServiceSqlite_TestDispatchGuards 覆盖按钮的拒绝分支：
// EMAIL 规则留空目标、INTERNAL 规则（需要真实消息本体）、id 缺失与不存在的规则、
// 以及投递失败时原始错误必须回给前端（这个按钮唯一有用的输出就是"为什么没发出去"）。
func TestNotificationRuleServiceSqlite_TestDispatchGuards(t *testing.T) {
	e := newRuleSvcEnv(t)
	e.seedViaInit(t)

	emailRule, err := e.repo.GetByEventType(e.ctx, notificationV1.EventType_CHANNEL_TEST_EMAIL)
	require.NoError(t, err)
	_, err = e.svc.TestDispatchNotification(e.opCtx, &notificationV1.TestDispatchNotificationRequest{
		Id: emailRule.GetId(),
	})
	require.Error(t, err, "EMAIL 渠道兜不出收件地址，必须要求填写")
	require.Contains(t, err.Error(), "target is required")

	internalRule, err := e.repo.GetByEventType(e.ctx, notificationV1.EventType_INTERNAL_MESSAGE)
	require.NoError(t, err)
	_, err = e.svc.TestDispatchNotification(e.opCtx, &notificationV1.TestDispatchNotificationRequest{
		Id:     internalRule.GetId(),
		Target: trans.Ptr("1024"),
	})
	require.Error(t, err, "站内信需要真实消息本体，填个收件人 ID 测不出东西")
	require.Contains(t, err.Error(), "internal-message page")
	require.Empty(t, e.notifier.calls, "被拒的三次调用一次都不该走到投递")

	_, err = e.svc.TestDispatchNotification(e.opCtx, &notificationV1.TestDispatchNotificationRequest{Id: 0})
	require.Error(t, err, "id=0 应被拒绝")
	_, err = e.svc.TestDispatchNotification(e.opCtx, nil)
	require.Error(t, err, "nil 请求应被拒绝")
	_, err = e.svc.TestDispatchNotification(e.opCtx, &notificationV1.TestDispatchNotificationRequest{
		Id: 987654, Target: trans.Ptr("bob@example.com"),
	})
	require.Error(t, err, "不存在的规则应返回 NotFound")

	// 投递失败：原始错误原文回出去，不能压成一句"操作失败"
	e.notifier.err = errors.New("send mail via channel [3] failed: smtp host/port is not configured")
	resp, err := e.svc.TestDispatchNotification(e.opCtx, &notificationV1.TestDispatchNotificationRequest{
		Id:     emailRule.GetId(),
		Target: trans.Ptr("bob@example.com"),
	})
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "smtp host/port is not configured")
	require.Contains(t, err.Error(), "channel [3]", "错误里的渠道号要留着，否则管理员不知道该去改哪一条")

	// 缺鉴权上下文：拿不到操作人就不投（台账的 created_by 会落空）
	_, err = e.svc.TestDispatchNotification(e.ctx, &notificationV1.TestDispatchNotificationRequest{
		Id:     emailRule.GetId(),
		Target: trans.Ptr("bob@example.com"),
	})
	require.Error(t, err, "无操作人上下文应拒绝")
}
