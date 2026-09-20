package channel

import (
	"context"
	"errors"
	"fmt"

	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
	notificationChannelV1 "go-wind-admin/api/gen/go/notification_channel/service/v1"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/pkg/mailer"
)

// EmailSender 邮件（SMTP）渠道。
//
// 渠道选择策略收在这一处：显式指定 channel_id 则用该条（配置页的测试邮件），
// 否则取第一个启用的 EMAIL 渠道（系统发起的验证码邮件）。
// 此前这条策略在两个 caller 各写了一遍。
//
// 不持有 logger：发送失败由服务层统一记账并写日志（脱敏后的目标 + 台账 ID），
// 在这里再记一遍只会漏掉台账关联。
type EmailSender struct {
	channel *data.NotificationChannelRepo
}

var _ Prechecker = (*EmailSender)(nil)

func NewEmailSender(channelRepo *data.NotificationChannelRepo) *EmailSender {
	return &EmailSender{channel: channelRepo}
}

func (s *EmailSender) Channel() notificationV1.Channel {
	return notificationV1.Channel_EMAIL
}

// Send 投递一封邮件。返回的 error 一律是"投递失败"，包括渠道不可用——
// 由服务层用 ErrChannelNotConfigured 判别是否记 SKIPPED。
func (s *EmailSender) Send(ctx context.Context, req *SendRequest) (*SendReceipt, error) {
	if req == nil || req.Target == "" {
		return nil, errors.New("email target is empty")
	}

	account, err := s.pickAccount(ctx, req.ChannelID)
	if err != nil {
		return nil, err
	}

	if err = mailer.SendMail(ctx, mailer.SmtpConfig{
		Host:     account.Host,
		Port:     account.Port,
		Username: account.Username,
		Password: account.Password,
		From:     account.From,
		TlsMode:  account.TlsMode,
	}, []string{req.Target}, req.Title, req.Content); err != nil {
		return nil, fmt.Errorf("send mail via channel [%d] failed: %w", account.ID, err)
	}

	return &SendReceipt{ChannelID: account.ID}, nil
}

// Precheck 只解析 SMTP 配置并自检可用性，不拨号。见 channel.Prechecker。
func (s *EmailSender) Precheck(ctx context.Context, channelID uint32) error {
	_, err := s.pickAccount(ctx, channelID)
	return err
}

// pickAccount 解析实际使用的 SMTP 账号：显式 ID 优先，并校验类型与启用状态。
//
// 一切"渠道用不了"（没配、没启用、类型不对、查不到）都包成 ErrChannelNotConfigured
// 返回——服务层据此把台账记 SKIPPED（没尝试过投递）而非 FAILED（投递报错），
// 原始原因留在 error 文本里，配置页据此还能看到 SMTP 报错详情。
//
// 返回的账号自带渠道 ID：自选分支下这是台账里 channel_id 的唯一来源，
// 早先只回一个 0，事件路由的发信在账上就答不出走的哪个 SMTP 账号。
func (s *EmailSender) pickAccount(ctx context.Context, channelID uint32) (*data.SmtpAccount, error) {
	account, err := s.resolveAccount(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if err = checkUsable(account); err != nil {
		return nil, err
	}
	return account, nil
}

// checkUsable 已解析配置的可用性自检。
//
// 加密方式是字符串列，库里存着枚举外的值（演示数据留下的 SSL_TLS）时原本要一路走到
// 拨号才报错，而"没拨过号"和"拨号被拒"在台账上是两种结论（SKIPPED / FAILED）：在这里拦掉。
func checkUsable(account *data.SmtpAccount) error {
	if !mailer.IsSupportedTlsMode(account.TlsMode) {
		return fmt.Errorf("%w: channel [%d] smtp_tls %q is not one of NONE/START_TLS/SSL",
			ErrChannelNotConfigured, account.ID, account.TlsMode)
	}
	return nil
}

func (s *EmailSender) resolveAccount(ctx context.Context, channelID uint32) (*data.SmtpAccount, error) {
	if channelID == 0 {
		account, err := s.channel.GetFirstEnabledEmailChannel(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrChannelNotConfigured, err)
		}
		return account, nil
	}

	// 先走脱敏的 Get 校验类型/启用：GetDecryptedSmtpAccount 不区分渠道类型，
	// 直接取会把一条 WEBHOOK 配置当 SMTP 用。
	dto, err := s.channel.Get(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("%w: channel [%d] not found: %v", ErrChannelNotConfigured, channelID, err)
	}
	if dto.GetType() != notificationChannelV1.NotificationChannel_EMAIL {
		return nil, fmt.Errorf("%w: channel [%d] is not an EMAIL channel", ErrChannelNotConfigured, channelID)
	}
	if !dto.GetEnabled() {
		return nil, fmt.Errorf("%w: channel [%d] is disabled", ErrChannelNotConfigured, channelID)
	}

	account, err := s.channel.GetDecryptedSmtpAccount(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("%w: channel [%d] credentials: %v", ErrChannelNotConfigured, channelID, err)
	}

	return account, nil
}
