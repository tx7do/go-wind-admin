// Package mailtext 收口一期邮件文案（主题 + 正文），按收件请求的语言渲染。
//
// 为什么文案在 Go 里而不在模板表：模板管理与渲染排在 P3（见
// docs/notification_domain_design.md §4），一期只有三个事件，带类型的函数比引一套
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

	"go-wind-admin/pkg/netutil"
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

// mailCopy 三个事件在某一种语言下的文案。正文里的占位符由对应的 Render 函数填充，
// 顺序固定：验证码 %s；渠道测试邮件 %d（渠道 ID）、%d（操作人用户 ID）。
type mailCopy struct {
	passwordResetTitle string
	passwordResetBody  string
	contactBindTitle   string
	contactBindBody    string
	channelTestTitle   string
	channelTestBody    string
}

var tables = map[Locale]mailCopy{
	LocaleZhCN: {
		passwordResetTitle: "GoWind Admin 密码重置验证码",
		passwordResetBody:  "您的密码重置验证码是：%s\n\n10 分钟内有效。若非本人操作请忽略本邮件。\n",
		contactBindTitle:   "GoWind Admin 邮箱绑定验证码",
		contactBindBody:    "您的邮箱绑定验证码是：%s\n\n10 分钟内有效。若非本人操作请忽略本邮件。\n",
		channelTestTitle:   "GoWind Admin 通知渠道测试邮件",
		channelTestBody:    "这是一封来自 GoWind Admin 的测试邮件。\n如果您收到了它，说明渠道 [%d] 配置可用。\n操作人用户 ID: %d\n",
	},
	LocaleEnUS: {
		passwordResetTitle: "GoWind Admin password reset code",
		passwordResetBody:  "Your password reset code is %s.\n\nIt expires in 10 minutes. Ignore this email if you did not request it.\n",
		contactBindTitle:   "GoWind Admin email binding code",
		contactBindBody:    "Your email binding code is %s.\n\nIt expires in 10 minutes. Ignore this email if you did not request it.\n",
		channelTestTitle:   "GoWind Admin notification channel test email",
		channelTestBody:    "This is a test email from GoWind Admin.\nReceiving it means channel [%d] is configured correctly.\nOperator user ID: %d\n",
	},
}

// Resolve 取 ctx 所在请求的邮件语言。
func Resolve(ctx context.Context) Locale {
	header := netutil.HeaderFromContext(ctx)
	if header == nil {
		return defaultLocale
	}
	return localeOf(header.Get(headerAcceptLanguage))
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
func ChannelTestEmail(ctx context.Context, channelID, operatorID uint32) (title, content string) {
	t := tableOf(ctx)
	return t.channelTestTitle, fmt.Sprintf(t.channelTestBody, channelID, operatorID)
}
