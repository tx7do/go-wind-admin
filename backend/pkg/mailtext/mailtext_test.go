package mailtext

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	ktransport "github.com/go-kratos/kratos/v2/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// transporter 以接口桩实现 khttp.Transporter，让 netutil.HeaderFromContext 命中，
// 从而在测试里造出"带 Accept-Language 的 HTTP 请求上下文"（桩形状与
// authentication_service_sqlite_test.go 的 fakeHeaderTransporter 一致）。
type transporter struct{ req *http.Request }

func (f *transporter) Kind() ktransport.Kind            { return ktransport.KindHTTP }
func (f *transporter) Endpoint() string                 { return "" }
func (f *transporter) Operation() string                { return "" }
func (f *transporter) Request() *http.Request           { return f.req }
func (f *transporter) PathTemplate() string             { return "" }
func (f *transporter) RequestHeader() ktransport.Header { return nil }
func (f *transporter) ReplyHeader() ktransport.Header   { return nil }

func requestCtx(header http.Header) context.Context {
	return ktransport.NewServerContext(context.Background(), &transporter{
		req: &http.Request{Header: header},
	})
}

func TestLocaleOf(t *testing.T) {
	cases := []struct {
		acceptLanguage string
		want           Locale
	}{
		{"zh-CN,zh;q=0.9,en;q=0.8", LocaleZhCN},
		{"en-US,en;q=0.9", LocaleEnUS},
		{"en-GB", LocaleEnUS},
		{"ZH-Hant", LocaleZhCN},
		{" ja, ko;q=0.8 ", LocaleZhCN}, // 不受支持的语言回落默认
		{"", LocaleZhCN},
		{"*", LocaleZhCN},
		// 列表里第一个不受支持时继续往后找，而不是直接回落
		{"fr-FR,fr;q=0.9,en-US;q=0.8", LocaleEnUS},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("[%q]", c.acceptLanguage), func(t *testing.T) {
			assert.Equal(t, c.want, localeOf(c.acceptLanguage))
		})
	}
}

func TestResolve(t *testing.T) {
	// 有 HTTP 传输上下文：按请求头选语言
	assert.Equal(t, LocaleEnUS, Resolve(requestCtx(http.Header{"Accept-Language": []string{"en"}})))
	assert.Equal(t, LocaleZhCN, Resolve(requestCtx(http.Header{"Accept-Language": []string{"zh-CN"}})))
	// 有传输上下文但没带头（例如直连 API、curl）
	assert.Equal(t, LocaleZhCN, Resolve(requestCtx(http.Header{})))
	// 完全没有传输上下文（后台任务、脚本发信）
	assert.Equal(t, LocaleZhCN, Resolve(context.Background()))
}

func TestRender(t *testing.T) {
	en := requestCtx(http.Header{"Accept-Language": []string{"en-US,en;q=0.9"}})
	zh := requestCtx(http.Header{"Accept-Language": []string{"zh-CN,zh;q=0.9"}})

	title, body := PasswordResetCode(zh, "123456")
	assert.Equal(t, "GoWind Admin 密码重置验证码", title)
	assert.Contains(t, body, "123456")

	title, body = PasswordResetCode(en, "123456")
	assert.Equal(t, "GoWind Admin password reset code", title)
	assert.Equal(t,
		"Your password reset code is 123456.\n\nIt expires in 10 minutes. Ignore this email if you did not request it.\n",
		body)

	title, body = ContactBindCode(en, "654321")
	assert.Equal(t, "GoWind Admin email binding code", title)
	assert.Contains(t, body, "654321")

	title, body = ChannelTestEmail(zh, 7, 1)
	assert.Equal(t, "GoWind Admin 通知渠道测试邮件", title)
	assert.Contains(t, body, "渠道 [7]")
	assert.Contains(t, body, "操作人用户 ID: 1")

	title, body = ChannelTestEmail(en, 7, 1)
	assert.Equal(t, "GoWind Admin notification channel test email", title)
	assert.Contains(t, body, "channel [7]")
	assert.Contains(t, body, "Operator user ID: 1")
}

// TestTablesAreComplete 守住"新增语言/新增事件不会静默漏文案"：
// 每个 locale 的六个字段都得有值，且正文渲染后不残留占位符（漏参数会原样带出 %s/%d）。
func TestTablesAreComplete(t *testing.T) {
	for locale, table := range tables {
		for name, field := range map[string]string{
			"passwordResetTitle": table.passwordResetTitle,
			"contactBindTitle":   table.contactBindTitle,
			"channelTestTitle":   table.channelTestTitle,
			"passwordResetBody":  table.passwordResetBody,
			"contactBindBody":    table.contactBindBody,
			"channelTestBody":    table.channelTestBody,
		} {
			require.NotEmpty(t, field, "locale %s 的 %s 为空", locale, name)
		}

		ctx := requestCtx(http.Header{"Accept-Language": []string{string(locale)}})
		rendered := map[string][]string{
			"password_reset": pair(PasswordResetCode(ctx, "123456")),
			"contact_bind":   pair(ContactBindCode(ctx, "123456")),
			"channel_test":   pair(ChannelTestEmail(ctx, 3, 9)),
		}
		for name, fields := range rendered {
			assert.NotEmpty(t, fields[0], "%s.%s 主题为空", locale, name)
			assert.NotContains(t, fields[0], "%", "%s.%s 主题残留占位符", locale, name)
			assert.False(t, strings.ContainsRune(fields[1], '%'), "%s.%s 正文渲染后仍有占位符: %q", locale, name, fields[1])
		}
	}
}

// pair 把 (title, content) 双返回值收成切片，便于放进 map 字面量批量断言。
func pair(title, content string) []string { return []string{title, content} }
