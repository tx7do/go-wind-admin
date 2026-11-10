// Package mailtext 收口一期邮件文案（主题 + 正文），按收件请求的语言渲染。
//
// 为什么文案在 Go 里而不在模板表：模板管理与渲染排在 P3（见
// docs/notification_domain_design.md §4），一期只有四个事件，带类型的函数比引一套
// 模板引擎便宜，同时仍守住"邮件文案只有一个出口"这条约束——新增事件必须改这里，
// 调用点不再各自拼中文字符串。
//
// 语言取自 HTTP 请求头 Accept-Language：三端 request-client 的请求拦截器都已注入该头，
// 而找回密码 / 绑定邮箱走免鉴权白名单，上下文里没有 token 级的 locale 可读。
// 非 HTTP 上下文（后台任务、测试）拿不到请求头，一律回落 defaultLocale。
package mailtext

import (
	"context"
	"fmt"
	"strings"

	"github.com/tx7do/go-utils/netutil"
)

// Locale 邮件语言。取值与前端 locale 同名，便于排障时按语言直接对上文案。
type Locale string

const (
	LocaleZhCN Locale = "zh-CN"
	LocaleEnUS Locale = "en-US"
)

// defaultLocale 回落语言：读不到请求头，或请求语言不受支持时使用。
const defaultLocale = LocaleZhCN

// headerAcceptLanguage Kratos 侧读取的头名（大小写由 net/http 规范化，此处按线格式写）。
const headerAcceptLanguage = "Accept-Language"

// mailCopy 四个事件在某一种语言下的文案。正文里的占位符由对应的 Render 函数填充，
// 顺序固定：验证码 %s；渠道测试邮件 %d（渠道 ID）、%d（操作人用户 ID）；
// 路由测试投递 %d（规则 ID）、%s（被测事件）。
// 带 sms 后缀的三对是 SMS 渠道专用的台账文案：短信正文由服务商侧的审核模板出，
// 这里的 title/content 只落投递台账——但措辞不能冒充"邮件"，否则台账读不出走的哪个渠道。
type mailCopy struct {
	passwordResetTitle string
	passwordResetBody  string
	contactBindTitle   string
	contactBindBody    string
	channelTestTitle   string
	channelTestBody    string
	ruleTestTitle      string
	ruleTestBody       string

	smsPasswordResetTitle string
	smsPasswordResetBody  string
	phoneBindTitle        string
	phoneBindBody         string
	channelTestSmsTitle   string
	channelTestSmsBody    string
}

var tables = map[Locale]mailCopy{
	LocaleZhCN: {
		passwordResetTitle: "GoWind Admin 密码重置验证码",
		passwordResetBody:  "您的密码重置验证码是：%s\n\n10 分钟内有效。若非本人操作请忽略本邮件。\n",
		contactBindTitle:   "GoWind Admin 邮箱绑定验证码",
		contactBindBody:    "您的邮箱绑定验证码是：%s\n\n10 分钟内有效。若非本人操作请忽略本邮件。\n",
		channelTestTitle:   "GoWind Admin 通知渠道测试邮件",
		channelTestBody:    "这是一封来自 GoWind Admin 的测试邮件。\n如果您收到了它，说明渠道 [%d] 配置可用。\n操作人用户 ID: %d\n",
		ruleTestTitle:      "GoWind Admin 通知路由测试投递",
		ruleTestBody:       "这是规则 [%d] 的一次测试投递，不含任何需要您处理的内容。\n被测事件: %s\n如果您收到了它，说明该事件到渠道的整条投递链是通的。\n",

		smsPasswordResetTitle: "GoWind Admin 密码重置验证码（短信）",
		smsPasswordResetBody:  "已向手机号发送密码重置验证码（服务商模板短信）。\n验证码：%s，10 分钟内有效。若非本人操作请忽略。\n",
		phoneBindTitle:        "GoWind Admin 手机号绑定验证码（短信）",
		phoneBindBody:         "已向本手机号发送绑定验证码（服务商模板短信）。\n验证码：%s，10 分钟内有效。若非本人操作请忽略。\n",
		channelTestSmsTitle:   "GoWind Admin 通知渠道测试短信",
		channelTestSmsBody:    "这是一条来自 GoWind Admin 的测试短信（验证码）。\n收到即说明渠道 [%d] 配置可用。\n操作人用户 ID: %d\n",
	},
	LocaleEnUS: {
		passwordResetTitle: "GoWind Admin password reset code",
		passwordResetBody:  "Your password reset code is %s.\n\nIt expires in 10 minutes. Ignore this email if you did not request it.\n",
		contactBindTitle:   "GoWind Admin email binding code",
		contactBindBody:    "Your email binding code is %s.\n\nIt expires in 10 minutes. Ignore this email if you did not request it.\n",
		channelTestTitle:   "GoWind Admin notification channel test email",
		channelTestBody:    "This is a test email from GoWind Admin.\nReceiving it means channel [%d] is configured correctly.\nOperator user ID: %d\n",
		ruleTestTitle:      "GoWind Admin notification routing test delivery",
		ruleTestBody:       "This is a test delivery for routing rule [%d]; nothing here needs your action.\nEvent under test: %s\nReceiving it means the whole delivery chain for that event reaches this channel.\n",

		smsPasswordResetTitle: "GoWind Admin password reset code (SMS)",
		smsPasswordResetBody:  "A password reset code has been sent to this phone number (provider template SMS).\nCode: %s. It expires in 10 minutes. Ignore this if you did not request it.\n",
		phoneBindTitle:        "GoWind Admin phone binding code (SMS)",
		phoneBindBody:         "A phone binding code has been sent to this number (provider template SMS).\nCode: %s. It expires in 10 minutes. Ignore this if you did not request it.\n",
		channelTestSmsTitle:   "GoWind Admin notification channel test SMS",
		channelTestSmsBody:    "This is a test SMS (verification code) from GoWind Admin.\nReceiving it means channel [%d] is configured correctly.\nOperator user ID: %d\n",
	},
}

// Resolve 取邮件语言：显式 locale（WithLocale 注入，来自收件用户的偏好设置）
// 优先；未设置时回落请求 Accept-Language 头；均无则默认 zh-CN。
func Resolve(ctx context.Context) Locale {
	if v := ctx.Value(localeContextKey{}); v != nil {
		if loc, ok := v.(Locale); ok {
			return loc
		}
	}
	header := netutil.HeaderFromContext(ctx)
	if header == nil {
		return defaultLocale
	}
	return localeOf(header.Get(headerAcceptLanguage))
}

// localeContextKey WithLocale 注入的显式语言键。
type localeContextKey struct{}

// WithLocale 显式指定邮件语言（如收件用户的偏好设置），覆盖请求头判定。
// 找回密码这类免鉴权流程在拿到用户后调用，验证码邮件即按用户偏好渲染。
func WithLocale(ctx context.Context, locale Locale) context.Context {
	return context.WithValue(ctx, localeContextKey{}, locale)
}

// LocaleOfTag 把 BCP-47 风格标签（zh-CN/en-US/zh/en）归一为受支持的 Locale；
// 不识别返回 ok=false（调用方保持默认）。
func LocaleOfTag(tag string) (Locale, bool) {
	base, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(tag)), "-")
	switch base {
	case "zh":
		return LocaleZhCN, true
	case "en":
		return LocaleEnUS, true
	}
	return defaultLocale, false
}

// localeOf 按列表顺序取第一个受支持的语言，只比主语言子标签（zh / en），不解析 q 权重：
// 浏览器发出的列表本身已按偏好排序，加权解析在这个规模上是多余的复杂度。
func localeOf(acceptLanguage string) Locale {
	for _, item := range strings.Split(acceptLanguage, ",") {
		tag, _, _ := strings.Cut(item, ";")
		base, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(tag)), "-")
		switch base {
		case "zh":
			return LocaleZhCN
		case "en":
			return LocaleEnUS
		}
	}
	return defaultLocale
}

func tableOf(ctx context.Context) mailCopy {
	return tables[Resolve(ctx)]
}

// PasswordResetCode 找回密码验证码邮件，对应 EventType_PASSWORD_RESET_CODE。
func PasswordResetCode(ctx context.Context, code string) (title, content string) {
	t := tableOf(ctx)
	return t.passwordResetTitle, fmt.Sprintf(t.passwordResetBody, code)
}

// ContactBindCode 邮箱绑定验证码邮件，对应 EventType_CONTACT_BIND_CODE。
func ContactBindCode(ctx context.Context, code string) (title, content string) {
	t := tableOf(ctx)
	return t.contactBindTitle, fmt.Sprintf(t.contactBindBody, code)
}

// ChannelTestEmail 渠道测试邮件，对应 EventType_CHANNEL_TEST_EMAIL。
func ChannelTestEmail(ctx context.Context, channelID, operatorID int64) (title, content string) {
	t := tableOf(ctx)
	return t.channelTestTitle, fmt.Sprintf(t.channelTestBody, channelID, operatorID)
}

// RuleTestNotification 路由测试投递（规则管理页的"测试投递"按钮），
// 覆盖 EMAIL 与 WEBHOOK 两种渠道：出站正文只有一个出口。
//
// eventType 传的是**被测规则**的事件类型名（不是这条测试本身的事件类型）：
// 管理员要一眼看出"这一发验证的是哪条路由"，而台账的 event_type 列记的也是它。
// 用字符串而不是 proto 枚举，是为了不把 notification.service.v1 拽进文案包。
func RuleTestNotification(ctx context.Context, ruleID int64, eventType string) (title, content string) {
	t := tableOf(ctx)
	return t.ruleTestTitle, fmt.Sprintf(t.ruleTestBody, ruleID, eventType)
}

// SmsPasswordResetCode 找回密码验证码短信（SMS 渠道），对应 EventType_PASSWORD_RESET_CODE。
// 短信正文由服务商审核模板渲染，这里的文案只落投递台账。
func SmsPasswordResetCode(ctx context.Context, code string) (title, content string) {
	t := tableOf(ctx)
	return t.smsPasswordResetTitle, fmt.Sprintf(t.smsPasswordResetBody, code)
}

// PhoneBindCode 手机号绑定验证码短信（SMS 渠道），对应 EventType_PHONE_BIND_CODE。
// 文案同样只落台账；验证码本体经 params.code 交给渠道。
func PhoneBindCode(ctx context.Context, code string) (title, content string) {
	t := tableOf(ctx)
	return t.phoneBindTitle, fmt.Sprintf(t.phoneBindBody, code)
}

// ChannelTestSms 渠道测试短信（SMS 渠道），对应 EventType_CHANNEL_TEST_SMS。
func ChannelTestSms(ctx context.Context, channelID, operatorID int64) (title, content string) {
	t := tableOf(ctx)
	return t.channelTestSmsTitle, fmt.Sprintf(t.channelTestSmsBody, channelID, operatorID)
}

// ChannelTestExt 扩展渠道（EXT_*）测试消息，对应 EventType_CHANNEL_TEST_EXT。
// 出站正文是通用测试文案（各渠道无模板概念）；台账文案与出站正文同一段。
func ChannelTestExt(ctx context.Context, operatorID int64) (title, content string) {
	title = "GoWind Admin 通知渠道测试消息"
	if LocaleEnUS == Resolve(ctx) {
		title = "GoWind Admin notification channel test message"
	}
	content = "这是一条来自 GoWind Admin 的测试消息。\n" +
		"收到即说明扩展渠道配置可用。\n" +
		fmt.Sprintf("操作人用户 ID: %d\n", operatorID)
	if LocaleEnUS == Resolve(ctx) {
		content = "This is a test message from GoWind Admin.\n" +
			"Receiving it means the extension channel is configured correctly.\n" +
			fmt.Sprintf("Operator user ID: %d\n", operatorID)
	}
	return title, content
}
