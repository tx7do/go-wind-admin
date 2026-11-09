package channel

import (
	"context"
	"errors"
	"fmt"

	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
	notificationChannelV1 "go-wind-admin/api/gen/go/notification_channel/service/v1"

	"github.com/tx7do/go-utils/notify"
	"github.com/tx7do/go-utils/notify/sms"

	"go-wind-admin/app/admin/service/internal/data"
)

// SmsSender 短信渠道（阿里云 / 腾讯云）。
//
// 库层负责"怎么发"（服务商协议/签名/答案判读/手机号归一），这里只做两件事——
// 从 sys_notification_channels 解析出用哪条账号配置，以及守住本平台的业务策略：
// 只发验证码事件（params["code"]），正文一律出自服务商侧审核过的模板。
//
// 短信与邮件/webhook 的本质差异：**系统侧没有自由文本通道**。三家服务商都要求
// 正文出自审核通过的模板，验证码作为模板占位值传入——所以 Send 只接受 params["code"]，
// 没有它（比如告警事件被路由到 SMS）直接失败并把原因说清楚，而不是发一条没有验证码的短信。
// 出站地址是服务商的固定域名（不是渠道行里配的 URL），SSRF 拦网在这里没有对象。
type SmsSender struct {
	channel *data.NotificationChannelRepo
}

var _ Prechecker = (*SmsSender)(nil)

func NewSmsSender(channelRepo *data.NotificationChannelRepo) *SmsSender {
	return &SmsSender{channel: channelRepo}
}

// Channel 渠道标识。
func (s *SmsSender) Channel() notificationV1.Channel {
	return notificationV1.Channel_SMS
}

// Precheck 只解析渠道配置并自检必填项，不拨号。见 channel.Prechecker。
//
// 与 WebhookSender.Precheck 同一条理由：一次性验证码躺在队列里等三次重试才失败，
// 和压根没配渠道一样是没发出去，但台账会记成 FAILED 且验证码已经过期。
func (s *SmsSender) Precheck(ctx context.Context, channelID uint64) error {
	account, err := s.pickAccount(ctx, channelID)
	if err != nil {
		return err
	}

	svc, err := sms.New(nil, smsConfigOf(account))
	if err != nil {
		return fmt.Errorf("%w: channel [%d] %v", ErrChannelNotConfigured, account.ID, err)
	}
	if err = svc.Precheck(); err != nil {
		return fmt.Errorf("%w: channel [%d] %v", ErrChannelNotConfigured, account.ID, stripNotifySentinel(err))
	}
	return nil
}

// Send 投递一条验证码短信。返回 error 一律是"投递失败"（含渠道不可用），
// 由服务层用 ErrChannelNotConfigured 判 SKIPPED / FAILED。
func (s *SmsSender) Send(ctx context.Context, req *SendRequest) (*SendReceipt, error) {
	if req == nil || req.Target == "" {
		return nil, errors.New("sms target is empty")
	}

	// 没带 code 的投递在选账号之前就拒绝：这是路由/调用方的问题，不是渠道配置问题，
	// 记 FAILED（下一次路由改对了就好了），不该混进 SKIPPED 让管理员白查一遍配置。
	code := req.Params["code"]
	if code == "" {
		return nil, fmt.Errorf(`sms channel [%d] requires params["code"]: SMS only carries verification-code events, the body is rendered by the provider-side approved template`, req.ChannelID)
	}

	account, err := s.pickAccount(ctx, req.ChannelID)
	if err != nil {
		return nil, err
	}

	svc, err := sms.New(nil, smsConfigOf(account))
	if err != nil {
		return &SendReceipt{ChannelID: account.ID}, fmt.Errorf("send sms via channel [%d] failed: %w", account.ID, err)
	}

	receipt, sendErr := svc.Send(ctx, &notify.Message{
		Target: req.Target,
		Params: map[string]string{"code": code},
	})
	if sendErr != nil {
		// 账号已选出：失败也必须把"是哪条渠道没发出去"带回给台账（见 Sender 的契约）。
		return &SendReceipt{ChannelID: account.ID}, s.wrapSendFailure(account.ID, sendErr)
	}

	// 服务商侧消息 ID（阿里云 RequestId、腾讯云 SerialNo）带回落台账
	providerID := ""
	if receipt != nil {
		providerID = receipt.ProviderID
	}
	return &SendReceipt{ChannelID: account.ID, ProviderID: providerID}, nil
}

// wrapSendFailure 库错误 → 台账文案：目标不合法（手机号归一失败）等库层配置类错误
// 转译成本包哨兵（SKIPPED：一次都没碰对端）；其余按"拨过号被拒"（FAILED）带出渠道 ID。
func (s *SmsSender) wrapSendFailure(accountID uint64, err error) error {
	if errors.Is(err, notify.ErrNotConfigured) {
		return fmt.Errorf("%w: channel [%d] %v", ErrChannelNotConfigured, accountID, stripNotifySentinel(err))
	}
	return fmt.Errorf("send sms via channel [%d] failed: %w", accountID, err)
}

// smsConfigOf 账号行 → 库层配置。字段一一对应：repo 的 SmsAccount 本来就是库层 Config 的镜像。
func smsConfigOf(account *data.SmsAccount) sms.Config {
	return sms.Config{
		Provider:     account.Provider,
		AccessKeyID:  account.AccessKeyID,
		AccessSecret: account.AccessSecret,
		SignName:     account.SignName,
		TemplateCode: account.TemplateCode,
		AppID:        account.AppID,
		Region:       account.Region,
	}
}

// pickAccount 解析实际使用的短信账号：显式 ID 优先，并校验类型与启用状态。
// 与 EmailSender.pickAccount 同一条判据：一切"渠道用不了"都包成 ErrChannelNotConfigured，
// 服务层据此把台账记 SKIPPED（没拨过号）而不是 FAILED（拨过号被拒）。
func (s *SmsSender) pickAccount(ctx context.Context, channelID uint64) (*data.SmsAccount, error) {
	if channelID == 0 {
		account, err := s.channel.GetFirstEnabledSmsChannel(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrChannelNotConfigured, err)
		}
		return account, nil
	}

	// 先走脱敏的 Get 校验类型/启用：GetDecryptedSmsAccount 守类型但不看 status，
	// 直接取会把一条停用的渠道当可用配置用。
	dto, err := s.channel.Get(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("%w: channel [%d] not found: %v", ErrChannelNotConfigured, channelID, err)
	}
	if dto.GetType() != notificationChannelV1.NotificationChannel_SMS {
		return nil, fmt.Errorf("%w: channel [%d] is not an SMS channel", ErrChannelNotConfigured, channelID)
	}
	if !dto.GetEnabled() {
		return nil, fmt.Errorf("%w: channel [%d] is disabled", ErrChannelNotConfigured, channelID)
	}

	account, err := s.channel.GetDecryptedSmsAccount(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("%w: channel [%d] credentials: %v", ErrChannelNotConfigured, channelID, err)
	}

	return account, nil
}
