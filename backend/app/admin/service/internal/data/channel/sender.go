// Package channel 通知投递渠道：Sender 接口 + 按渠道分发的 Registry。
//
// 这里是"怎么发"，sys_notification_channels 表是"用什么账号发"。
// 一个渠道一个实现，路由（事件 → 渠道）不在本包，见 service/notification_service.go。
package channel

import (
	"context"
	"fmt"

	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
)

// SendRequest 一次投递意图。
//
// 不含模板/参数：一期正文由调用方直接给出（渲染与模板属 P3）。
type SendRequest struct {
	// Target 投递目标：EMAIL 为收件地址，WEBHOOK 为 URL，INTERNAL 为收件用户 ID 的十进制字符串。
	Target string
	// Title 通知标题（邮件为 Subject）。
	Title string
	// Content 通知正文（邮件为纯文本正文）。
	Content string
	// ChannelID 显式指定渠道配置（sys_notification_channels.id）；0 表示由渠道策略自选。
	// 渠道测试必须走显式指定：否则"给渠道 3 发测试邮件"会退化成"给第一个启用的发"。
	ChannelID uint32
	// RecipientUserID 收件用户 ID，0 表示直发模式（只有 Target，不知道是谁）。
	// INTERNAL 渠道必填：站内信没有"地址"这种独立目标，Target 只是它的字符串形式。
	RecipientUserID uint32
	// OperatorUserID 触发这次投递的操作人，0 表示系统发起。
	// 台账的 created_by 由 NotificationService 直接取请求字段，这里带进来是给
	// "渠道自己还要落业务行"的实现用的（站内信要把发送人写进收件记录的 created_by）。
	OperatorUserID uint32
	// RelatedID 产生本次投递的业务对象主键，含义由事件类型决定
	// （INTERNAL_MESSAGE → sys_internal_messages.id）。渠道自己用不上，
	// 但站内信必须靠它知道往哪个消息本体上挂收件行。
	RelatedID uint32
	// EventType 这次投递属于哪个业务事件。SMTP 用不上它（主题与正文已经是这个事件的），
	// WEBHOOK 必须带上：出站 JSON 是对端唯一的判据，而模板渲染排在 P3，
	// 正文里此刻没有任何东西说明"这是找回密码还是换绑验证码"。
	EventType notificationV1.EventType
}

// SendReceipt 渠道回执。
type SendReceipt struct {
	// ChannelID 实际使用的渠道配置 ID——台账要能回答"这条到底走的哪个 SMTP 账号"。
	ChannelID uint32
}

// Sender 一种投递渠道的实现。
//
// Send 返回 error 即视为投递失败（台账记 FAILED）；返回 err 为 nil 视为渠道已接受。
// 实现不应把"渠道未配置/未启用"降级成 nil——那是 SKIPPED 的语义，由调用方区分。
type Sender interface {
	Channel() notificationV1.Channel
	Send(ctx context.Context, req *SendRequest) (*SendReceipt, error)
}

// ErrChannelNotConfigured 没有可用的渠道配置（区别于"渠道报错"）。
// 服务层据此把台账记为 SKIPPED 而非 FAILED：管理员配错与压根没配是两件事。
var ErrChannelNotConfigured = fmt.Errorf("no enabled notification channel configured")

// Prechecker 渠道的"配置可用性"自检，由异步派发在入队之前调用。
//
// 为什么不干脆先 Send 一次看结果：异步载荷带着一次性验证码，能在这次请求里就判定
// "这条渠道根本用不了"，就不该把 OTP 塞进队列、等它躺在归档里过期。
// 只查配置不拨号——拨号是投递本身，归 Send。
//
// 不实现本接口的渠道（站内信）由服务层视作"没有预检可用"，直接入队。
type Prechecker interface {
	// Precheck 校验 channelID 指向的配置可用；0 表示由渠道策略自选。
	// 返回的 error 语义与 Send 一致：配置类错误必须包成 ErrChannelNotConfigured。
	Precheck(ctx context.Context, channelID uint32) error
}

// Registry 渠道注册表。路由指向未注册的渠道时 Notifier 记 FAILED 并报错（代码 bug，不是配置问题）。
//
// Register 覆盖同渠道的旧实现：装配期一次性写入，运行期只读。
type Registry struct {
	senders map[notificationV1.Channel]Sender
}

func NewRegistry() *Registry {
	return &Registry{senders: make(map[notificationV1.Channel]Sender)}
}

func (r *Registry) Register(sender Sender) {
	r.senders[sender.Channel()] = sender
}

// Sender 取渠道实现；未注册返回 ok=false。
func (r *Registry) Sender(channel notificationV1.Channel) (Sender, bool) {
	sender, ok := r.senders[channel]
	return sender, ok
}
