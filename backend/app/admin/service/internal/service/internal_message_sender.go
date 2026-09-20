package service

import (
	"context"
	"errors"
	"time"

	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"

	"go-wind-admin/app/admin/service/internal/data/channel"
)

// InternalMessageSender 站内信（INTERNAL）渠道。
//
// 为什么它住在 internal/service 而不是 data/channel/：站内信的实现就是同包里的
// InternalMessageService（收件行落库 + SSE 推送），而 data 包 import service 包会成环。
// 于是本适配器由 wiring_ent.go 在装配期注册进 data/channel 的 Registry——
// 这是全仓唯一一条"渠道实现反向依赖服务层"的边，见 docs/notification_domain_design.md §3.1。
//
// 它持有的是 *InternalMessageService 本身而不是那两个协作者（收件仓储 + publisher）：
// publisher 在 SSE 服务起来之后才由 RegisterInternalMessagePublisher 注入，
// 构造期快照会把它永久钉在 noop 上——站内信落库成功但没人收到实时通知，且不报错。
type InternalMessageSender struct {
	messages *InternalMessageService
}

var _ channel.Sender = (*InternalMessageSender)(nil)

func NewInternalMessageSender(messages *InternalMessageService) *InternalMessageSender {
	return &InternalMessageSender{messages: messages}
}

func (s *InternalMessageSender) Channel() notificationV1.Channel {
	return notificationV1.Channel_INTERNAL
}

// Send 把一个收件人挂到一条已存在的消息本体上：落收件行 + 尽力推送。
//
// 回执不带 ChannelID：站内信不吃 sys_notification_channels 花名册（没有"用哪个账号发"的
// 问题），台账该列留空即"这条投递不来自任何渠道配置"。
//
// SSE 推送失败不算投递失败——publishNotification 自己只记日志、不返回错误，
// 因为站内信以落库为准，用户离线时重连后仍能从收件箱补取。
func (s *InternalMessageSender) Send(ctx context.Context, req *channel.SendRequest) (*channel.SendReceipt, error) {
	if req.RelatedID == 0 {
		return nil, errors.New("internal message delivery requires related_id: the sys_internal_messages row to attach this recipient to")
	}
	if req.RecipientUserID == 0 {
		return nil, errors.New("internal message delivery requires recipient_user_id")
	}

	now := time.Now()
	if err := s.messages.sendNotification(ctx, req.RelatedID, req.RecipientUserID, req.OperatorUserID, &now, req.Title, req.Content); err != nil {
		return nil, err
	}

	return &channel.SendReceipt{}, nil
}
