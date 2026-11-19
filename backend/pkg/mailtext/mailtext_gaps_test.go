// 本文件补 mailtext 既有测试（mailtext_test.go）未覆盖的分支：
//   - WithLocale 显式注入的优先级路径（覆盖请求头判定）——找回密码免鉴权流程
//     在拿到用户后按其偏好渲染，这条路径此前零覆盖；
//   - LocaleOfTag（exported，此前无覆盖）：BCP-47 标签归一与不识别回落；
//   - SMS 专属三对文案（SmsPasswordResetCode / PhoneBindCode / ChannelTestSms）
//     与 EXT 渠道的 ChannelTestExt 的中英双语渲染；
//   - 完整性断言补 SMS 专属字段（TestTablesAreComplete 只查 email 8 字段）。
package mailtext

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolve_ExplicitLocaleOverridesHeader(t *testing.T) {
	// 显式 locale 优先：请求头是英文，注入中文偏好 → 中文。
	ctx := WithLocale(requestCtx(http.Header{"Accept-Language": []string{"en-US"}}), LocaleZhCN)
	require.Equal(t, LocaleZhCN, Resolve(ctx))

	// 反向同理。
	ctx2 := WithLocale(requestCtx(http.Header{"Accept-Language": []string{"zh-CN"}}), LocaleEnUS)
	require.Equal(t, LocaleEnUS, Resolve(ctx2))

	// 无请求头的后台任务上下文：显式 locale 照常生效。
	require.Equal(t, LocaleEnUS, Resolve(WithLocale(defaultTestCtx(), LocaleEnUS)))
}

func defaultTestCtx() context.Context { return testCtxNoHeader() }

func testCtxNoHeader() context.Context { return requestCtx(http.Header{}) }

func TestLocaleOfTag(t *testing.T) {
	cases := []struct {
		tag     string
		want    Locale
		wantOK  bool
	}{
		{"zh-CN", LocaleZhCN, true},
		{"zh", LocaleZhCN, true},
		{"ZH-Hant-TW", LocaleZhCN, true},
		{" en ", LocaleEnUS, true},
		{"en-US", LocaleEnUS, true},
		{"ja-JP", LocaleZhCN, false}, // 不识别：ok=false，返回值回落默认
		{"", LocaleZhCN, false},
		{"zh_", LocaleZhCN, false}, // 下划线不是分隔符，主子标签是 "zh_"，不命中
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("[%q]", c.tag), func(t *testing.T) {
			loc, ok := LocaleOfTag(c.tag)
			require.Equal(t, c.wantOK, ok)
			require.Equal(t, c.want, loc)
		})
	}
}

// TestSmsRenderFunctions SMS 专属文案双语渲染：title 必须点明"短信"（不能冒充邮件），
// 正文含验证码占位。
func TestSmsRenderFunctions(t *testing.T) {
	// zh-CN（默认回落）
	title, content := SmsPasswordResetCode(defaultTestCtx(), "654321")
	require.Equal(t, "GoWind Admin 密码重置验证码（短信）", title)
	require.Contains(t, content, "654321")
	require.Contains(t, content, "短信")

	title, content = PhoneBindCode(defaultTestCtx(), "112233")
	require.Equal(t, "GoWind Admin 手机号绑定验证码（短信）", title)
	require.Contains(t, content, "112233")

	title, content = ChannelTestSms(defaultTestCtx(), 7, 42)
	require.Equal(t, "GoWind Admin 通知渠道测试短信", title)
	require.Contains(t, content, "[7]")
	require.Contains(t, content, "42")

	// en-US
	enCtx := WithLocale(testCtxNoHeader(), LocaleEnUS)
	title, content = SmsPasswordResetCode(enCtx, "654321")
	require.Equal(t, "GoWind Admin password reset code (SMS)", title)
	require.Contains(t, content, "654321")

	title, content = PhoneBindCode(enCtx, "112233")
	require.Equal(t, "GoWind Admin phone binding code (SMS)", title)

	title, content = ChannelTestSms(enCtx, 7, 42)
	require.Equal(t, "GoWind Admin notification channel test SMS", title)
	require.Contains(t, content, "[7]")
}

// TestChannelTestExt EXT 渠道测试消息：不走文案表的内联双语分支。
func TestChannelTestExt(t *testing.T) {
	title, content := ChannelTestExt(defaultTestCtx(), 99)
	require.Equal(t, "GoWind Admin 通知渠道测试消息", title)
	require.Contains(t, content, "99")
	require.Contains(t, content, "扩展渠道")

	title, content = ChannelTestExt(WithLocale(testCtxNoHeader(), LocaleEnUS), 99)
	require.Equal(t, "GoWind Admin notification channel test message", title)
	require.Contains(t, content, "Operator user ID: 99")
}

// TestSMSTablesAreComplete 完整性断言：SMS 专属 6 个文案字段在两种语言下
// 均非空、占位符数量匹配（mailtext_test.go 的 TestTablesAreComplete 只查 email 8 字段）。
func TestSMSTablesAreComplete(t *testing.T) {
	for locale, table := range tables {
		t.Run(string(locale), func(t *testing.T) {
			require.NotEmpty(t, table.smsPasswordResetTitle)
			require.Equal(t, 1, strings.Count(table.smsPasswordResetBody, "%s"))
			require.NotEmpty(t, table.phoneBindTitle)
			require.Equal(t, 1, strings.Count(table.phoneBindBody, "%s"))
			require.NotEmpty(t, table.channelTestSmsTitle)
			require.Equal(t, 2, strings.Count(table.channelTestSmsBody, "%d"))
		})
	}
}

// --- 基准：文案渲染（每次通知投递调用） ---

func BenchmarkSmsRender(b *testing.B) {
	ctx := requestCtx(http.Header{"Accept-Language": []string{"zh-CN"}})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = SmsPasswordResetCode(ctx, "654321")
	}
}

func BenchmarkChannelTestEmailRender(b *testing.B) {
	ctx := requestCtx(http.Header{"Accept-Language": []string{"en-US"}})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ChannelTestEmail(ctx, 7, 42)
	}
}
