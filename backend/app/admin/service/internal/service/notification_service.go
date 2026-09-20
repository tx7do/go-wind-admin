package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/channel"
)

// NotificationService 通知投递：把"某个业务事件要通知某人"落成一次真实投递 + 一条台账。
//
// 与站内信（InternalMessageService）的分工：站内信是"消息本体 + 收件箱"的内容域，
// 本服务是"某事件经某渠道投递出去的结果"的投递域。一期三个事件全是站外邮件，
// 站内信尚未注册为 Sender（见 docs/notification_domain_design.md §4 P1 修订）。
type NotificationService struct {
	adminV1.NotificationServiceHTTPServer

	log          *bLogger.Helper
	deliveryRepo *data.NotificationDeliveryRepo
	channels     *channel.Registry
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

	receipt, sendErr := sender.Send(ctx, &channel.SendRequest{
		Target:          req.GetTarget(),
		Title:           req.GetTitle(),
		Content:         req.GetContent(),
		ChannelID:       req.GetChannelId(),
		RecipientUserID: req.GetRecipientUserId(),
		OperatorUserID:  req.GetOperatorUserId(),
		RelatedID:       req.GetRelatedId(),
	})
	if sendErr != nil {
		// 渠道用不了（没配/没启用）＝一条都没尝试发 → SKIPPED；渠道报了错 → FAILED。
		status := notificationV1.DeliveryStatus_FAILED
		if errors.Is(sendErr, channel.ErrChannelNotConfigured) {
			status = notificationV1.DeliveryStatus_SKIPPED
		}
		s.markResult(ctx, deliveryId, &data.DeliveryOutcome{Status: status, LastError: sendErr.Error()})
		s.log.Errorf(ctx, "notify [%s] via %s to [%s] failed: %s",
			req.GetEventType().String(), ch.String(), maskTarget(ch, req.GetTarget()), sendErr.Error())

		return &notificationV1.SendNotificationResponse{
			DeliveryId: deliveryId,
			Status:     status,
		}, sendErr
	}

	sentAt := time.Now()
	s.markResult(ctx, deliveryId, &data.DeliveryOutcome{
		Status: notificationV1.DeliveryStatus_SENT,
		SentAt: &sentAt,
		// 自选渠道时 Create 阶段还没有 channel_id（策略在 Sender 内部），发完补上。
		ChannelID: pickedChannelId(receipt),
	})

	return &notificationV1.SendNotificationResponse{
		DeliveryId: deliveryId,
		Status:     notificationV1.DeliveryStatus_SENT,
	}, nil
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
