package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/id"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/hibiken/asynq"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/channel"
	"go-wind-admin/pkg/task"
)

// NotificationService 通知投递：把"某个业务事件要通知某人"落成一次真实投递 + 一条台账。
//
// 与站内信（InternalMessageService）的分工：站内信是"消息本体 + 收件箱"的内容域，
// 本服务是"某事件经某渠道投递出去的结果"的投递域。站内信自己也是一种渠道
// （InternalMessageSender 注册为 Channel_INTERNAL），生产点经 Notifier 缝走这里。
type NotificationService struct {
	adminV1.NotificationServiceHTTPServer

	log          *bLogger.Helper
	deliveryRepo *data.NotificationDeliveryRepo
	channels     *channel.Registry

	// taskEnqueuer 异步派发能力。默认 nil（asynq 未配置）＝全部事件走同步投递；
	// 配置后由 asynq_server.go 的 RegisterTaskEnqueuer 注入。
	taskEnqueuer TaskEnqueuer
}

func NewNotificationService(
	ctx *bootstrap.Context,
	deliveryRepo *data.NotificationDeliveryRepo,
	channelRegistry *channel.Registry,
) *NotificationService {
	return &NotificationService{
		log:          ctx.NewLoggerHelper("notification/service/admin-service"),
		deliveryRepo: deliveryRepo,
		channels:     channelRegistry,
	}
}

// RegisterTaskEnqueuer 注入 asynq 任务入队能力，与 InternalMessageService 同模式：
// 构造期留 nil（＝纯同步投递），装配期由 NewAsynqServer 覆盖。
func (s *NotificationService) RegisterTaskEnqueuer(enqueuer TaskEnqueuer) {
	s.taskEnqueuer = enqueuer
}

// eventChannels 是"事件 → 渠道"路由表。
//
// 为什么先写在 Go 里：一期事件种类个位数，DB 规则表带来的"不发版改路由"抵不上
// 多一张表 + 一套 BFF + 三端页面的成本。P2 建 notification_rules 时整表平移进 DB，
// 本变量换成查表，调用方签名不变。
//
// INTERNAL_MESSAGE 一条看起来多余（站内信本来就走 INTERNAL 渠道），但它是这条路由的
// 唯一声明处：删掉它，站内信的生产点就会退回"自己 new 收件行、自己推 SSE、不留台账"。
var eventChannels = map[notificationV1.EventType]notificationV1.Channel{
	notificationV1.EventType_PASSWORD_RESET_CODE: notificationV1.Channel_EMAIL,
	notificationV1.EventType_CONTACT_BIND_CODE:   notificationV1.Channel_EMAIL,
	notificationV1.EventType_CHANNEL_TEST_EMAIL:  notificationV1.Channel_EMAIL,
	notificationV1.EventType_INTERNAL_MESSAGE:    notificationV1.Channel_INTERNAL,
}

// asyncDispatchEvents 是"事件 → 是否异步派发"的第二张表。
//
// 判据只有一条：**调用方需不需要这次投递的结论**。
//
//   - 找回密码 / 换绑验证码：请求只要"已受理"，SMTP 往返的 3~30 秒不该占住 HTTP，
//     失败反正由台账兜着 → 异步。
//   - CHANNEL_TEST_EMAIL：这个接口的产物就是报错原文（"535 authentication failed"），
//     异步化会让配置页点完测试只看到"已入队"，什么也测不出来 → 同步。
//   - INTERNAL_MESSAGE：站内信渠道要在同一趟里返回收件行主键，SSE 载荷与前端乐观插入
//     都依赖它 → 同步。
//
// 与 eventChannels 一样先写在 Go 里，P2 建 notification_rules 时一并平移进 DB。
var asyncDispatchEvents = map[notificationV1.EventType]bool{
	notificationV1.EventType_PASSWORD_RESET_CODE: true,
	notificationV1.EventType_CONTACT_BIND_CODE:   true,
}

const (
	// notificationDispatchMaxRetry 异步派发的重试上限。
	// 短（3 次）是因为载荷带一次性验证码：重试到第 10 次时验证码早过期了，
	// 多出来的尝试只是把明文 OTP 在 Redis 里多留几轮。
	notificationDispatchMaxRetry = 3

	// notificationDispatchMaxAttempts 一次派发的总尝试次数 = 首次 + 重试额度。
	// asynq 的 MaxRetry 只数"重试"，台账的 attempts 数"尝试（含首次）"，
	// 差这一行换算写在常量里，别让 handler 拿 attempts 去比 MaxRetry。
	notificationDispatchMaxAttempts = notificationDispatchMaxRetry + 1

	// notificationDispatchTimeout 单次尝试的超时。
	// mailer 用 DialContext 拨号，故该超时能真正掐断 SMTP 握手（否则 asynq 的超时只杀任务、不杀连接）。
	notificationDispatchTimeout = 30 * time.Second
)

// resolveChannel 决定本次投递的渠道：请求显式指定优先，否则查事件路由表。
func resolveChannel(req *notificationV1.SendDirectNotificationRequest) (notificationV1.Channel, error) {
	if req.GetChannel() != notificationV1.Channel_CHANNEL_UNSPECIFIED {
		return req.GetChannel(), nil
	}

	ch, ok := eventChannels[req.GetEventType()]
	if !ok {
		return notificationV1.Channel_CHANNEL_UNSPECIFIED,
			fmt.Errorf("no channel routing rule for event type %s", req.GetEventType().String())
	}
	return ch, nil
}

// SendDirect 投递一条通知并落台账。实现 Notifier 接口。
//
// 台账先落 SENDING 再投递：进程在 SMTP 握手中途被杀时，留下一条永远停在 SENDING 的行，
// 这就是"发了一半"的唯一线索（若改成发完再写，这种情况连痕迹都没有）。
//
// 命中 asyncDispatchEvents 的事件不在这里等投递结果：入队即返回 SENDING，
// 结论由 AsyncNotificationDispatch 回写同一行台账（见 dispatchAsync 的三种出口）。
//
// 不读 auth.FromContext：找回密码等场景没有操作人（免鉴权），系统任务同理。
// 操作人由调用方显式传入 OperatorUserId，只用于台账归类。
func (s *NotificationService) SendDirect(ctx context.Context, req *notificationV1.SendDirectNotificationRequest) (*notificationV1.SendNotificationResponse, error) {
	if req == nil || req.GetTarget() == "" {
		return nil, errors.New("notification target is required")
	}
	if req.GetTitle() == "" || req.GetContent() == "" {
		return nil, errors.New("notification title and content are required")
	}

	ch, err := resolveChannel(req)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	delivery, err := s.deliveryRepo.Create(ctx, &notificationV1.NotificationDelivery{
		EventType:       trans.Ptr(req.GetEventType()),
		Channel:         trans.Ptr(ch),
		ChannelId:       req.ChannelId,
		RecipientUserId: req.RecipientUserId,
		RelatedId:       req.RelatedId,
		RequestId:       trans.Ptr(resolveRequestId(req)),
		Target:          trans.Ptr(maskTarget(ch, req.GetTarget())),
		Status:          trans.Ptr(notificationV1.DeliveryStatus_SENDING),
		CreatedBy:       req.OperatorUserId,
		CreatedAt:       timeutil.TimeToTimestamppb(&now),
	})
	if err != nil {
		return nil, err
	}
	deliveryId := delivery.GetId()

	sender, registered := s.channels.Sender(ch)
	if !registered {
		// 路由表指向了没有实现的渠道 = 代码 bug，不是配置问题 → FAILED。
		reason := fmt.Sprintf("channel %s is not registered", ch.String())
		s.markResult(ctx, deliveryId, &data.DeliveryOutcome{
			Status:    notificationV1.DeliveryStatus_FAILED,
			LastError: reason,
		})
		return &notificationV1.SendNotificationResponse{
			DeliveryId: deliveryId,
			Status:     notificationV1.DeliveryStatus_FAILED,
		}, errors.New(reason)
	}

	if asyncDispatchEvents[req.GetEventType()] {
		if resp, dispatched, dispatchErr := s.dispatchAsync(ctx, deliveryId, req, sender); dispatched {
			return resp, dispatchErr
		}
	}

	return s.deliver(ctx, deliveryId, req, sender)
}

// resolveRequestId 台账的幂等锚：调用方给了就用，没给则服务端生成。
//
// 生成而非留空：这一列是排障时"一次调用 → 若干条投递"的唯一抓手，三个生产点今天都不传，
// 留空等于让台账里最该串起来的那条线全是 NULL。
func resolveRequestId(req *notificationV1.SendDirectNotificationRequest) string {
	if v := strings.TrimSpace(req.GetRequestId()); v != "" {
		return v
	}
	return id.NewGUIDv4(false)
}

// deliver 当场投递并回写台账结论。同步事件、以及入队失败的兜底共用这一条路。
func (s *NotificationService) deliver(
	ctx context.Context,
	deliveryId uint32,
	req *notificationV1.SendDirectNotificationRequest,
	sender channel.Sender,
) (*notificationV1.SendNotificationResponse, error) {
	outcome, sendErr := s.sendOnce(ctx, req, sender)
	// 当场投递就是一次尝试：不让 attempts 只在异步路径有意义，否则"SENT 但 attempts=0"
	// 这种行会逼着每个读台账的人先回忆哪条路写过这一列。
	outcome.Attempts = trans.Ptr(uint32(1))

	s.markResult(ctx, deliveryId, outcome)

	return &notificationV1.SendNotificationResponse{
		DeliveryId: deliveryId,
		Status:     outcome.Status,
	}, sendErr
}

// sendOnce 拨一次 + 给出该写进台账的结论，但不落库（写不写、什么时候写由调用方决定）。
//
// 分类只有一条判据：渠道有没有"真的试过"。配置类错误（没配/没启用/类型不对）＝一条都没发出去
// → SKIPPED；其余（认证被拒、连接超时、对方 5xx）＝拨过号了 → FAILED。
// 找回密码的回话文案就靠这个区分，异步路径还靠它决定"值不值得重试"（见 AsyncNotificationDispatch）。
func (s *NotificationService) sendOnce(
	ctx context.Context,
	req *notificationV1.SendDirectNotificationRequest,
	sender channel.Sender,
) (*data.DeliveryOutcome, error) {
	receipt, sendErr := sender.Send(ctx, &channel.SendRequest{
		Target:          req.GetTarget(),
		Title:           req.GetTitle(),
		Content:         req.GetContent(),
		ChannelID:       req.GetChannelId(),
		RecipientUserID: req.GetRecipientUserId(),
		OperatorUserID:  req.GetOperatorUserId(),
		RelatedID:       req.GetRelatedId(),
	})
	if sendErr == nil {
		sentAt := time.Now()
		return &data.DeliveryOutcome{
			Status: notificationV1.DeliveryStatus_SENT,
			SentAt: &sentAt,
			// 自选渠道时 Create 阶段还没有 channel_id（策略在 Sender 内部），发完补上。
			ChannelID: pickedChannelId(receipt),
		}, nil
	}

	status := notificationV1.DeliveryStatus_FAILED
	if errors.Is(sendErr, channel.ErrChannelNotConfigured) {
		status = notificationV1.DeliveryStatus_SKIPPED
	}
	s.log.Errorf(ctx, "notify [%s] via %s to [%s] failed: %s",
		req.GetEventType().String(), sender.Channel().String(),
		maskTarget(sender.Channel(), req.GetTarget()), sendErr.Error())

	return &data.DeliveryOutcome{Status: status, LastError: sendErr.Error()}, sendErr
}

// dispatchAsync 把投递交给 asynq。三种出口：
//
//  1. 预检判定渠道用不了 → 当场落 SKIPPED/FAILED 并返回错误（dispatched=true）；
//  2. 入队成功 → 返回 SENDING，本次请求到此为止（dispatched=true）；
//  3. 队列不可用（未配 asynq / Redis 报错）→ dispatched=false，调用方走同步投递。
//
// 出口 1 存在的理由就是"载荷带正文"的代价：验证码一旦进队列，就要等它被消费或过期，
// 而"SMTP 压根没配"这种错误在同步请求里一次查库就能判定。找回密码的调用方把 error 原文
// 回显给用户（"未配置邮件服务，请联系管理员"），异步化会让它先收到"已发送"再石沉大海。
//
// 载荷只带 target/title/content：渠道、事件、收件人、操作人都是台账已有的事实，
// 再抄一份进 Redis 只是多一个会漂移的副本。
func (s *NotificationService) dispatchAsync(
	ctx context.Context,
	deliveryId uint32,
	req *notificationV1.SendDirectNotificationRequest,
	sender channel.Sender,
) (*notificationV1.SendNotificationResponse, bool, error) {
	if s.taskEnqueuer == nil {
		return nil, false, nil
	}

	if prechecker, ok := sender.(channel.Prechecker); ok {
		if err := prechecker.Precheck(ctx, req.GetChannelId()); err != nil {
			status := notificationV1.DeliveryStatus_FAILED
			if errors.Is(err, channel.ErrChannelNotConfigured) {
				status = notificationV1.DeliveryStatus_SKIPPED
			}
			s.markResult(ctx, deliveryId, &data.DeliveryOutcome{Status: status, LastError: err.Error()})
			s.log.Errorf(ctx, "precheck notify [%s] via %s failed: %s",
				req.GetEventType().String(), sender.Channel().String(), err.Error())

			return &notificationV1.SendNotificationResponse{
				DeliveryId: deliveryId,
				Status:     status,
			}, true, err
		}
	}

	if err := s.taskEnqueuer.NewTask(
		task.NotificationDispatchTaskType,
		&task.NotificationDispatchTaskData{
			DeliveryId: deliveryId,
			Target:     req.GetTarget(),
			Title:      req.GetTitle(),
			Content:    req.GetContent(),
		},
		asynq.MaxRetry(notificationDispatchMaxRetry),
		asynq.Timeout(notificationDispatchTimeout),
	); err != nil {
		// 队列挂了不能顺带把通知丢掉：退回同步投递，结论照旧当场给出。
		s.log.Errorf(ctx, "enqueue dispatch for delivery [%d] failed, falling back to inline send: %s",
			deliveryId, err.Error())
		return nil, false, nil
	}

	return &notificationV1.SendNotificationResponse{
		DeliveryId: deliveryId,
		Status:     notificationV1.DeliveryStatus_SENDING,
	}, true, nil
}

// AsyncNotificationDispatch 消费 notification_dispatch 任务：读回台账 → 记一次尝试 → 投递 → 回写结论。
//
// 幂等靠台账状态而不是任务 ID：status 不再是 SENDING 就说明这一行已经有了结论（ SENT/FAILED/SKIPPED），
// asynq 的重投或归档后手工重跑都在这里止步，不会再发第二封。
// 由此推出重试的写法：还有额度的失败**必须**留在 SENDING，否则重投一进门就被这道幂等门挡掉，
// 重试额度等于没有（下面逐条注明）。
// 这是"至少一次投递 + 结果行幂等"，不是精确一次：进程在"邮件已发出、结论未回写"之间被杀，
// 重试仍会再发一封 —— ent 给不了条件更新（CAS）的影响行数，本就不该假装能做到。
//
// 不注入 viewer：台账与渠道配置两张表都没有租户列，这条链路上的读写本来就不经租户谓词
// （站内信广播 handler 必须注入，是因为它按受众拉用户表）。
func (s *NotificationService) AsyncNotificationDispatch(ctx context.Context, _ string, payload *task.NotificationDispatchTaskData) error {
	if payload == nil || payload.DeliveryId == 0 {
		return errors.New("notification dispatch payload is empty")
	}
	deliveryId := payload.DeliveryId

	row, err := s.deliveryRepo.Get(ctx, deliveryId)
	if err != nil {
		// 台账行是入队之前单独提交的，这里读不到＝DB 侧临时故障（或行被手工删了）→ 交给重试。
		return err
	}
	if row.GetStatus() != notificationV1.DeliveryStatus_SENDING {
		s.log.Infof(ctx, "delivery [%d] already settled as %s, skip dispatch", deliveryId, row.GetStatus().String())
		return nil
	}

	sender, registered := s.channels.Sender(row.GetChannel())
	if !registered {
		// 装配期之后渠道不会消失，走到这里只能是代码 bug：重试不会改变结论。
		reason := fmt.Sprintf("channel %s is not registered", row.GetChannel().String())
		s.markResult(ctx, deliveryId, &data.DeliveryOutcome{
			Status:    notificationV1.DeliveryStatus_FAILED,
			LastError: reason,
		})
		return fmt.Errorf("%s: %w", reason, asynq.SkipRetry)
	}

	// 先记账再拨号：attempts 是"这一次真的试过"的痕迹，若和结论一起写，
	// 进程死在握手中途就只剩下一个孤零零的 SENDING，看不出试到第几次。
	attempts := row.GetAttempts() + 1
	if err = s.deliveryRepo.MarkAttempted(ctx, deliveryId, attempts); err != nil {
		return err
	}

	outcome, sendErr := s.sendOnce(ctx, dispatchRequest(row, payload), sender)
	if sendErr == nil {
		s.markResult(ctx, deliveryId, outcome)
		return nil
	}

	// 配置类错误（渠道在这之间被关掉/删了）重试不会改变结论，而载荷里是明文验证码：
	// 立刻定案 SKIPPED 并跳过重试，别把 OTP 留在归档队列里等过期。
	if outcome.Status == notificationV1.DeliveryStatus_SKIPPED {
		s.markResult(ctx, deliveryId, outcome)
		return fmt.Errorf("%w: %v", asynq.SkipRetry, sendErr)
	}

	// 还有重试额度时**不**定案：状态留在 SENDING，只把这次的报错写进去。
	// 若在这里直接写 FAILED，asynq 重投时会撞上上面那道"status 已非 SENDING"的幂等门，
	// 于是重试额度形同虚设 —— 而 450 忙线、连接超时这类恰恰是最该重试的。
	if attempts < notificationDispatchMaxAttempts {
		s.markResult(ctx, deliveryId, &data.DeliveryOutcome{
			Status:    notificationV1.DeliveryStatus_SENDING,
			LastError: outcome.LastError,
		})
		return sendErr
	}

	// 额度用尽：最后一次尝试写 FAILED，台账就此闭环（asynq 那边同时进归档）。
	s.markResult(ctx, deliveryId, outcome)
	return sendErr
}

// dispatchRequest 从台账行 + 载荷还原一次投递意图。
//
// target/title/content 一律取载荷：台账的 target 是脱敏后的（只够排障、不够投递），
// 正文压根没有列。其余字段（渠道、事件、收件人、操作人、关联对象）都来自台账行，
// Redis 里不多留一份身份副本。
func dispatchRequest(row *notificationV1.NotificationDelivery, payload *task.NotificationDispatchTaskData) *notificationV1.SendDirectNotificationRequest {
	return &notificationV1.SendDirectNotificationRequest{
		Target:          payload.Target,
		Title:           payload.Title,
		Content:         payload.Content,
		EventType:       row.GetEventType(),
		Channel:         row.Channel,
		ChannelId:       row.ChannelId,
		RecipientUserId: row.RecipientUserId,
		OperatorUserId:  row.CreatedBy,
		RelatedId:       row.RelatedId,
		RequestId:       row.RequestId,
	}
}

// pickedChannelId 把回执里的渠道 ID 转成台账的可选列（0/无回执 → nil，表示不修改）。
func pickedChannelId(receipt *channel.SendReceipt) *uint32 {
	if receipt == nil || receipt.ChannelID == 0 {
		return nil
	}
	return trans.Ptr(receipt.ChannelID)
}

// markResult 回写台账结果，失败只记日志、不改投递结论。
//
// 邮件已经发出去了，台账写砸只是"可观测性缺陷"；把它上报成发送失败会诱导调用方重发，
// 那才是真事故。
func (s *NotificationService) markResult(ctx context.Context, deliveryId uint32, outcome *data.DeliveryOutcome) {
	if err := s.deliveryRepo.MarkResult(ctx, deliveryId, outcome); err != nil {
		s.log.Errorf(ctx, "mark delivery [%d] result [%s] failed: %s",
			deliveryId, outcome.Status.String(), err.Error())
	}
}

func (s *NotificationService) ListNotificationDelivery(ctx context.Context, req *paginationV1.PagingRequest) (*notificationV1.ListNotificationDeliveryResponse, error) {
	return s.deliveryRepo.List(ctx, req)
}

func (s *NotificationService) GetNotificationDelivery(ctx context.Context, req *notificationV1.GetNotificationDeliveryRequest) (*notificationV1.NotificationDelivery, error) {
	if req == nil || req.GetId() == 0 {
		return nil, adminV1.ErrorBadRequest("id is required")
	}
	return s.deliveryRepo.Get(ctx, req.GetId())
}

// maskTarget 投递目标的台账脱敏：邮箱留首字符与完整域名，其余一律只留末 4 位。
//
// 台账要能排障（"到底发给了谁"），但不能变成一张全平台明文邮箱表——
// 收件地址本身已由 sys_user_credentials 持有，脱敏后的形态足以定位到人。
//
// INTERNAL 渠道原样存：它的"地址"就是收件用户 ID，本平台内部主键，台账本就只对
// 平台管理员开放。掩成 `****1024` 只会把台账里唯一可读的字段变成噪音。
func maskTarget(ch notificationV1.Channel, target string) string {
	if ch == notificationV1.Channel_INTERNAL {
		return target
	}

	at := strings.LastIndex(target, "@")
	if at > 0 && at < len(target)-1 {
		local := target[:at]
		return local[:1] + "***" + target[at:]
	}

	if len(target) <= 4 {
		return "****"
	}
	return "****" + target[len(target)-4:]
}
