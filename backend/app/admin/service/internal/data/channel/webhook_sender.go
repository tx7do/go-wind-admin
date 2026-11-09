package channel

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"

	"github.com/tx7do/go-utils/notify"
	"github.com/tx7do/go-utils/notify/webhook"

	"go-wind-admin/app/admin/service/internal/data"
)

// EnvAllowPrivateWebhook 放行内网/环回地址的开关，仅供本机联调。
// 值 "1" 时整条 SSRF 防线关闭（连环回都能发），生产部署不得设置。
// 库层有自己的通用开关名（NOTIFY_ALLOW_PRIVATE_TARGET），这里保留旧名：
// 已部署环境的运维脚本认的是它。
const EnvAllowPrivateWebhook = "NOTIFICATION_WEBHOOK_ALLOW_PRIVATE"

// webhookPayload CUSTOM/NONE 风格且未配模板时的默认出站 JSON 形状。
// 字段名与顺序都是存量对端已在验的东西——这是业务侧与对端的契约，
// 所以默认正文的形状留在本仓（经 webhook.Config.DefaultPayload 注入库层），
// 不跟着库的通用默认（title/content/delivered_at）走。
type webhookPayload struct {
	EventType       string `json:"event_type"`
	Title           string `json:"title"`
	Content         string `json:"content"`
	RecipientUserID uint64 `json:"recipient_user_id,omitempty"`
	RelatedID       uint64 `json:"related_id,omitempty"`
	DeliveredAt     string `json:"delivered_at"`
}

// WebhookSender HTTP 回调渠道：库层负责"怎么发"（签名风格/模板渲染/SSRF 防线/verdict），
// 这里只做两件事——从 sys_notification_channels 解析出用哪条账号配置，
// 以及把业务侧的关联字段（事件类型/收件人/业务对象）垫进默认出站体。
//
// 地址取自 SendRequest.Target（不是渠道行的 webhook_url）：台账的 target 列回答
// "这条到底发去了哪里"，让 sender 私自解析地址等于把这一列的来源挪到一个不回写的地方。
// webhook_url 的作用域因此是"这条渠道该发到哪"的登记值 + 预检门槛，
// 并由测试投递入口在管理员留空时兜出来（见 service/notification_rule_service.go）。
type WebhookSender struct {
	channel *data.NotificationChannelRepo
	client  *http.Client
}

var _ Prechecker = (*WebhookSender)(nil)

func NewWebhookSender(channelRepo *data.NotificationChannelRepo) *WebhookSender {
	return &WebhookSender{
		channel: channelRepo,
		client:  notify.NewGuardedClient(allowPrivateWebhook()),
	}
}

// allowPrivateWebhook 读环境变量决定要不要关掉 SSRF 防线。
// 只认精确的 "1"：真值判断（strconv.ParseBool）会让 "false"、"0" 之外的垃圾值也放行，
// 而这一条开关放行的内容是"本进程可向内网任意地址发请求"。
func allowPrivateWebhook() bool {
	return os.Getenv(EnvAllowPrivateWebhook) == "1"
}

// Channel 渠道标识。
func (s *WebhookSender) Channel() notificationV1.Channel {
	return notificationV1.Channel_WEBHOOK
}

// Precheck 只解析渠道配置并自检可用性，不拨号。见 channel.Prechecker。
//
// 自检包含"这条渠道的模板渲染得出合法 JSON"：一次性验证码躺在队列里等四次重试
// 才失败，和压根没配渠道一样是没发出去，但台账会记成 FAILED 且验证码已经过期。
func (s *WebhookSender) Precheck(ctx context.Context, channelID uint64) error {
	account, err := s.pickAccount(ctx, channelID)
	if err != nil {
		return err
	}
	if err = s.service(account, nil).Precheck(); err != nil {
		return fmt.Errorf("%w: channel [%d] %v", ErrChannelNotConfigured, account.ID, stripNotifySentinel(err))
	}
	return nil
}

// Send 投递一次回调。返回 error 一律是"投递失败"（含渠道不可用），
// 由服务层用 ErrChannelNotConfigured 判 SKIPPED / FAILED。
func (s *WebhookSender) Send(ctx context.Context, req *SendRequest) (*SendReceipt, error) {
	if req == nil || req.Target == "" {
		return nil, errors.New("webhook target url is empty")
	}

	account, err := s.pickAccount(ctx, req.ChannelID)
	if err != nil {
		return nil, err
	}

	_, sendErr := s.service(account, req).Send(ctx, &notify.Message{
		Target:  req.Target,
		Title:   req.Title,
		Content: req.Content,
	})
	if sendErr != nil {
		// 账号已选出：失败也必须把"是哪条渠道没发出去"带回给台账（见 Sender 的契约）。
		return &SendReceipt{ChannelID: account.ID}, s.wrapSendFailure(account.ID, sendErr)
	}

	return &SendReceipt{ChannelID: account.ID}, nil
}

// service 按解析出的账号构造库层渠道。
// req 为 nil（预检）时默认出站体只带交付时间，业务字段留空。
func (s *WebhookSender) service(account *data.WebhookAccount, req *SendRequest) *webhook.Service {
	return webhook.New(s.client, webhook.Config{
		Ref:             strconv.FormatUint(uint64(account.ID), 10),
		Secret:          account.Secret,
		SignStyle:       account.SignStyle,
		PayloadTemplate: account.PayloadTemplate,
		DefaultPayload: func(*notify.Message) (any, error) {
			payload := webhookPayload{DeliveredAt: time.Now().UTC().Format(time.RFC3339)}
			if req != nil {
				payload.EventType = req.EventType.String()
				payload.Title = req.Title
				payload.Content = req.Content
				payload.RecipientUserID = req.RecipientUserID
				payload.RelatedID = req.RelatedID
			}
			return payload, nil
		},
	})
}

// wrapSendFailure 库错误 → 台账文案：配置类转译成本包哨兵（SKIPPED 语义），
// 其余按"拨过号被拒"（FAILED）带出渠道 ID。库层哨兵文本与本包同义，剥掉再说一次会结巴。
func (s *WebhookSender) wrapSendFailure(accountID uint64, err error) error {
	if errors.Is(err, notify.ErrNotConfigured) {
		text := stripNotifySentinel(err)
		if errors.Is(err, notify.ErrBlockedTarget) {
			text += fmt.Sprintf(" (%s=1 disables this check for local debugging)", EnvAllowPrivateWebhook)
		}
		return fmt.Errorf("%w: %v", ErrChannelNotConfigured, text)
	}
	return fmt.Errorf("send webhook via channel [%d] failed: %w", accountID, err)
}

// pickAccount 解析实际使用的渠道配置并自检可用性。
//
// 与 EmailSender.pickAccount 同一条判据：一切"渠道用不了"都包成 ErrChannelNotConfigured，
// 服务层据此把台账记 SKIPPED（没拨过号）而不是 FAILED（拨过号被拒）。
func (s *WebhookSender) pickAccount(ctx context.Context, channelID uint64) (*data.WebhookAccount, error) {
	account, err := s.resolveAccount(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if account.URL == "" {
		return nil, fmt.Errorf("%w: channel [%d] has no webhook_url configured", ErrChannelNotConfigured, account.ID)
	}
	return account, nil
}

// resolveAccount 显式 ID 优先，否则取第一个启用的 WEBHOOK 渠道。
func (s *WebhookSender) resolveAccount(ctx context.Context, channelID uint64) (*data.WebhookAccount, error) {
	if channelID == 0 {
		account, err := s.channel.GetFirstEnabledWebhookChannel(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrChannelNotConfigured, err)
		}
		return account, nil
	}

	// 先走脱敏的 Get 校验启用状态：GetDecryptedWebhookAccount 守类型但不看 status，
	// 直接取会把一条停用的渠道当可用配置用。
	dto, err := s.channel.Get(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("%w: channel [%d] not found: %v", ErrChannelNotConfigured, channelID, err)
	}
	if !dto.GetEnabled() {
		return nil, fmt.Errorf("%w: channel [%d] is disabled", ErrChannelNotConfigured, channelID)
	}

	account, err := s.channel.GetDecryptedWebhookAccount(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("%w: channel [%d] credentials: %v", ErrChannelNotConfigured, channelID, err)
	}

	return account, nil
}
