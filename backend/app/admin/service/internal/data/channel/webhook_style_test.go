// WEBHOOK 出站风格与载荷模板的纯单元测试（不落库、不发 HTTP）。
//
// 这一层的全部风险都在于"算错了也看不出来"：签名 key/data 分工记反、模板少转义一个引号、
// 风格值拼错被静默兜成 CUSTOM，三种错误的对外表现都只是"对端说签名不对/正文不是 JSON"，
// 排查方向被引到对端身上。所以这里的断言一律钉**具体值**：
//   - 两家机器人的签名向量各钉一条，且用独立实现（node:crypto createHmac）交叉算出，
//     不拿本文件的函数自证；
//   - 正文形状钉字节序列（CUSTOM 那份是存量对端已经在验的形状）；
//   - 模板的拒绝路径逐条点名（不支持的占位符 / 渲染出非 JSON / {{sign}} 在 CUSTOM 下循环）。
//
// 真发一次 HTTP 的部分（query 拼接后打到对端、200 但其实 errcode!=0）在
// webhook_sender_sqlite_test.go，与 SSRF 防线一起测。
package channel

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"go-wind-admin/app/admin/service/internal/data"

	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
)

const (
	// styleTestSecret 两家机器人官方文档示例里用的密钥，向量是照它算的。
	styleTestSecret = "xxx"
	// styleTestTimestamp 官方示例时间戳（2022-08-30）。
	styleTestTimestamp = "1661860880"
	// wantFeishuSign 飞书 GenSign("xxx", 1661860880) 的期望值：
	// base64(hmac_sha256(key="1661860880\nxxx", data=空))，取自飞书开放平台自定义机器人签名校验文档。
	wantFeishuSign = "QnWVTSBe6FmQDE0bG6X0mURbI+DnvVyu1h+j5dHOjrU="
	// wantDingtalkSign 钉钉同输入下的期望值：
	// base64(hmac_sha256(key="xxx", data="1661860880\nxxx"))。文档只给了 Java 片段与另一组输入，
	// 所以这一条按文档公式由独立实现（node:crypto）算出，不是本文件跑出来的自证值。
	wantDingtalkSign = "dIUMcehSFkPlWOGqwoP5EFcartyN67QSY9wbVW+hTbE="
)

func styleAccount(style, template, secret string) *data.WebhookAccount {
	return &data.WebhookAccount{
		ID:              7,
		URL:             "https://example.com/hook",
		Secret:          secret,
		SignStyle:       style,
		PayloadTemplate: template,
	}
}

func styleReq() *SendRequest {
	return &SendRequest{
		Title:           "重置验证码",
		Content:         "您的验证码是 123456",
		EventType:       notificationV1.EventType_PASSWORD_RESET_CODE,
		RecipientUserID: 1024,
		RelatedID:       7,
	}
}

// TestWebhookSignVectors 两个签名函数的黄金向量。
//
// 两条一起看才看得出这条测试的意义：钉钉是 key=secret / data="<ts>\n<secret>"，
// 飞书是 key="<ts>\n<secret>" / data=空 —— 分工记反时对端只回一句 "sign not match"，
// 而两个函数互换后产出的值恰好互不相等，所以断言里再钉一次"彼此不等"。
func TestWebhookSignVectors(t *testing.T) {
	require.Equal(t, wantDingtalkSign, signDingtalk(styleTestSecret, styleTestTimestamp),
		"钉钉：base64(hmac_sha256(key=secret, data=\"<ms>\\n<secret>\"))")
	require.Equal(t, wantFeishuSign, signFeishu(styleTestSecret, styleTestTimestamp),
		"飞书：base64(hmac_sha256(key=\"<s>\\n<secret>\", data=空))")

	require.NotEqual(t, signDingtalk(styleTestSecret, styleTestTimestamp), signFeishu(styleTestSecret, styleTestTimestamp),
		"两个算法在同样输入下必须产出不同值：相等说明有一侧的 key/data 分工写反成了另一侧")
}

// TestResolveWebhookStyle 空串按 CUSTOM（存量行必为 NULL），认不出的值报错。
func TestResolveWebhookStyle(t *testing.T) {
	got, err := resolveWebhookStyle("")
	require.NoError(t, err)
	require.Equal(t, styleCustom, got, "这两列是后加的：存量行 NULL 的语义就是\"当时只有 CUSTOM 一种行为\"")

	for _, raw := range []string{"CUSTOM", "NONE", "DINGTALK", "FEISHU", "WECOM"} {
		got, err = resolveWebhookStyle(raw)
		require.NoError(t, err, raw)
		require.Equal(t, webhookStyle(raw), got)
	}

	// 大小写敏感：钉钉拼成小写也不能认，那会让"库里写的是什么"与"发出去的是什么"分家。
	for _, raw := range []string{"custom", "DINGTAKL", "DingTalk", "dingtalk", "  FEISHU", "SMS"} {
		got, err = resolveWebhookStyle(raw)
		require.Error(t, err, "%q 不该被认出来", raw)
		require.Empty(t, got)
		require.Contains(t, err.Error(), raw, "报错要带上原值，否则排障时看不出库里此刻写着什么")
		require.Contains(t, err.Error(), "CUSTOM / NONE", "要说出可取值清单")
	}
}

// TestVerdictFieldOf 各风格"对端真的收下了"看哪个 body 字段。
func TestVerdictFieldOf(t *testing.T) {
	require.Equal(t, "errcode", verdictFieldOf(styleDingtalk))
	require.Equal(t, "errcode", verdictFieldOf(styleWecom))
	require.Equal(t, "code", verdictFieldOf(styleFeishu), "飞书的结论字段是 code/msg，不是 errcode")
	require.Empty(t, verdictFieldOf(styleCustom), "自有方案没有约定字段：只看状态码")
	require.Empty(t, verdictFieldOf(styleNone))
}

// TestBuildWebhookOutboundDingtalk 钉钉：签名进 URL query（毫秒时间戳），不进头、不进正文。
func TestBuildWebhookOutboundDingtalk(t *testing.T) {
	out, err := buildWebhookOutbound(styleDingtalk, styleAccount("DINGTALK", "", styleTestSecret), styleReq())
	require.NoError(t, err)

	require.Empty(t, out.timestampHeader)
	require.Empty(t, out.signatureHeader, "钉钉不签 body（签名只盖时间戳与密钥），所以没有签名头")
	require.Equal(t, "errcode", out.verdictField)

	require.NotNil(t, out.query)
	require.Len(t, out.query["timestamp"], 1)
	require.Len(t, out.query["sign"], 1)

	// 毫秒：钉钉服务端按 ±1 小时窗口判过期，用秒会直接报"timestamp outside validity period"。
	ms, parseErr := strconv.ParseInt(out.query["timestamp"][0], 10, 64)
	require.NoError(t, parseErr)
	require.Greater(t, ms, int64(1e12), "必须是 13 位毫秒，10 位秒会被当成 1970 年")
	require.Less(t, ms, int64(1e14))

	require.Equal(t, signDingtalk(styleTestSecret, out.query["timestamp"][0]), out.query["sign"][0],
		"query 里的 timestamp 与 sign 必须成对：签名用的时间戳和拼出去的时间戳差一位就对端验不过")

	// 内置正文是 text 形状，标题与正文之间是 JSON 转义过的 \n。
	var body struct {
		MsgType string `json:"msgtype"`
		Text    struct {
			Content string `json:"content"`
		} `json:"text"`
	}
	require.NoError(t, json.Unmarshal(out.body, &body))
	require.Equal(t, "text", body.MsgType)
	require.Equal(t, "重置验证码\n您的验证码是 123456", body.Text.Content)
	require.NotContains(t, string(out.body), "sign", "钉钉的签名不该出现在正文里")
}

// TestBuildWebhookOutboundDingtalkUnsigned 没配密钥时不动地址：钉钉可以只按关键词/IP 放白。
func TestBuildWebhookOutboundDingtalkUnsigned(t *testing.T) {
	out, err := buildWebhookOutbound(styleDingtalk, styleAccount("DINGTALK", "", ""), styleReq())
	require.NoError(t, err)
	require.Nil(t, out.query, "签名一栏留空时不该往 URL 上拼 timestamp=&sign=，那对端只看得到一个空签名")

	// 不签名不等于没有正文：形状仍是钉钉的 text。
	require.JSONEq(t,
		`{"msgtype":"text","text":{"content":"重置验证码\n您的验证码是 123456"}}`,
		string(out.body))
	require.Equal(t, "errcode", out.verdictField, "判据是风格带的，跟签不签名无关")
}

// TestBuildWebhookOutboundFeishuEnvelope 飞书：签名当正文字段，时间戳是秒。
func TestBuildWebhookOutboundFeishuEnvelope(t *testing.T) {
	out, err := buildWebhookOutbound(styleFeishu, styleAccount("FEISHU", "", styleTestSecret), styleReq())
	require.NoError(t, err)

	require.Nil(t, out.query)
	require.Empty(t, out.signatureHeader)
	require.Equal(t, "code", out.verdictField)

	var envelope map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(out.body, &envelope))

	require.Contains(t, envelope, "timestamp")
	require.Contains(t, envelope, "sign", "飞书把签名放正文顶层两个字段，不是请求头")

	var ts, sign string
	require.NoError(t, json.Unmarshal(envelope["timestamp"], &ts))
	require.NoError(t, json.Unmarshal(envelope["sign"], &sign))

	// 秒：飞书文档的示例是 10 位。
	require.Len(t, ts, 10, "飞书要 10 位秒时间戳")
	require.Equal(t, signFeishu(styleTestSecret, ts), sign)

	// 并信封不许丢掉模板渲染出来的正文。
	require.Contains(t, envelope, "msg_type")
	require.Contains(t, envelope, "content")

	var body struct {
		MsgType string `json:"msg_type"`
		Content struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	require.NoError(t, json.Unmarshal(out.body, &body))
	require.Equal(t, "text", body.MsgType)
	require.Equal(t, "重置验证码\n您的验证码是 123456", body.Content.Text)

	// 并信封不得把 < > & 转义成 \u003c：飞书/钉钉的 @ 人语法就是 <at> 标签，
	// 转义一次等于群里只看得见一串字面量。本机实测里飞书正文中的中文尖括号就是这么被吃掉的。
	escaped, err := buildWebhookOutbound(styleFeishu,
		styleAccount("FEISHU", "", styleTestSecret),
		&SendRequest{Title: "<尖括号> & 与号", Content: "a > b", EventType: notificationV1.EventType_PASSWORD_RESET_CODE})
	require.NoError(t, err)
	require.NotContains(t, string(escaped.body), `\u003c`, "并完信封不许再 HTML 转义一次")
	require.Contains(t, string(escaped.body), `<尖括号> & 与号\na > b`)
}

// TestBuildWebhookOutboundWecomUnsigned 企微：key 在 URL 上，签名一列留空即可。
func TestBuildWebhookOutboundWecomUnsigned(t *testing.T) {
	out, err := buildWebhookOutbound(styleWecom, styleAccount("WECOM", "", styleTestSecret), styleReq())
	require.NoError(t, err)

	require.Nil(t, out.query)
	require.Empty(t, out.signatureHeader)
	require.Equal(t, "errcode", out.verdictField)
	require.JSONEq(t,
		`{"msgtype":"text","text":{"content":"重置验证码\n您的验证码是 123456"}}`,
		string(out.body))
}

// TestBuildWebhookOutboundCustomKeepsLegacyBytes CUSTOM 且不填模板时必须维持既有字节形状与顺序。
func TestBuildWebhookOutboundCustomKeepsLegacyBytes(t *testing.T) {
	account := styleAccount("CUSTOM", "", styleTestSecret)

	out, err := buildWebhookOutbound(styleCustom, account, styleReq())
	require.NoError(t, err)

	require.Equal(t, "", out.verdictField, "自有方案没约定判据字段：只看状态码")

	body := string(out.body)
	require.True(t, strings.HasPrefix(body,
		`{"event_type":"PASSWORD_RESET_CODE","title":"重置验证码","content":"您的验证码是 123456","recipient_user_id":1024,"related_id":7,"delivered_at":"`),
		"字段名与顺序都是存量对端已在验的东西，加两列不该把它换成等价但不同形的 JSON：\n%s", body)
	require.True(t, strings.HasSuffix(body, `"}`))
	require.NotContains(t, body, "\n")

	// 时间戳进头、签名覆盖"时间戳.正文"：头里的 ts 必须就是签名里那个。
	require.Len(t, out.timestampHeader, 10, "CUSTOM 用秒时间戳")
	require.NotEmpty(t, out.signatureHeader)
	mac := hmac.New(sha256.New, []byte(styleTestSecret))
	mac.Write([]byte(out.timestampHeader))
	mac.Write([]byte("."))
	mac.Write(out.body)
	require.Equal(t, "sha256="+hex.EncodeToString(mac.Sum(nil)), out.signatureHeader,
		"签名必须在正文定型之后算：先签再渲染会让对端复算永远对不上")
}

// TestBuildWebhookOutboundCustomUnsigned 无密钥 → 不发签名头。
func TestBuildWebhookOutboundCustomUnsigned(t *testing.T) {
	out, err := buildWebhookOutbound(styleCustom, styleAccount("CUSTOM", "", ""), styleReq())
	require.NoError(t, err)
	require.Empty(t, out.signatureHeader, "没有密钥却发一个空签名头，对端会拿它和自己的 HMAC 比")
}

// TestBuildWebhookOutboundNoneUsesDefaultBody NONE 只管"不签名"，正文仍要有默认形状。
//
// 这一条钉的是加列时的一个真实缺口：早期实现里内置模板表只登记了三家机器人，
// NONE 不填模板会渲染出空正文，报出来的是"不是合法 JSON"，看不出真正的原因是没默认形状。
func TestBuildWebhookOutboundNoneUsesDefaultBody(t *testing.T) {
	out, err := buildWebhookOutbound(styleNone, styleAccount("NONE", "", styleTestSecret), styleReq())
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(out.body, &parsed), "NONE 不填模板也要出合法 JSON")
	require.Equal(t, "PASSWORD_RESET_CODE", parsed["event_type"])
	require.Equal(t, "重置验证码", parsed["title"])
	require.Equal(t, float64(1024), parsed["recipient_user_id"])

	require.Nil(t, out.query)
	require.Empty(t, out.signatureHeader, "NONE 配了密钥也不签：这一列的语义就是\"什么都不带\"")
	require.Empty(t, out.verdictField)
}

// TestTemplateRendersEachValueAsJsonStringContent 模板变量按 JSON 字符串内容转义、引号归作者。
func TestTemplateRendersEachValueAsJsonStringContent(t *testing.T) {
	req := styleReq()
	req.Title = `带"引号"、反斜杠 \、换行` + "\n" + `与 <尖括号> & 和 中文`
	req.Content = "正文里有 \t 制表符" + "\"结束\""

	const template = `{"t":"{{title}}","c":"{{content}}","uid":{{recipient_user_id}},"rid":{{related_id}},"e":"{{event_type}}","n":"{{nonce}}"}`

	out, err := buildWebhookOutbound(styleNone, styleAccount("NONE", template, ""), req)
	require.NoError(t, err)

	raw := string(out.body)
	require.Contains(t, raw, "<尖括号>", "SetEscapeHTML(false)：中文正文里的 < > & 不该变成 \\u003c，否则群里看到的是转义串")
	require.Contains(t, raw, "&", "同上")
	require.Contains(t, raw, `"uid":1024`, "ID 类变量给的是裸数字，作者不必再处理引号")
	require.Contains(t, raw, `"rid":7`)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(out.body, &parsed), "含引号与换行的正文仍要出合法 JSON")
	require.Equal(t, req.Title, parsed["t"])
	require.Equal(t, req.Content, parsed["c"])
	require.Equal(t, "PASSWORD_RESET_CODE", parsed["e"])
	require.Equal(t, float64(1024), parsed["uid"])

	// 每次投递换一个 nonce：对端据此判"同一时间戳同正文"的重放。
	out2, err := buildWebhookOutbound(styleNone, styleAccount("NONE", template, ""), req)
	require.NoError(t, err)
	require.NotEqual(t, raw, string(out2.body))
}

// TestTemplateIsNotRecursive 正文里的花括号不能被当模板再渲染一次。
func TestTemplateIsNotRecursive(t *testing.T) {
	req := styleReq()
	req.Title = "{{content}}"
	req.Content = "真正文"

	out, err := buildWebhookOutbound(styleNone, styleAccount("NONE", `{"t":"{{title}}"}`, ""), req)
	require.NoError(t, err)

	var parsed map[string]string
	require.NoError(t, json.Unmarshal(out.body, &parsed))
	require.Equal(t, "{{content}}", parsed["t"],
		"逐变量 ReplaceAll 会让 {{title}} 的值里的 {{content}} 被后一轮替换掉：单趟扫描每个值只出现一次")
}

// TestTemplateRejections 模板层的拒绝路径逐条点名。
func TestTemplateRejections(t *testing.T) {
	cases := []struct {
		name      string
		style     string
		template  string
		wantCause string
	}{
		{
			name:      "unsupported placeholder",
			style:     "CUSTOM",
			template:  `{"t":"{{contnet}}"}`,
			wantCause: "unsupported placeholder {{contnet}}",
		},
		{
			name:      "sign under custom is circular",
			style:     "CUSTOM",
			template:  `{"t":"{{title}}","s":"{{sign}}"}`,
			wantCause: "unsupported placeholder {{sign}}",
		},
		{
			name:      "renders not json",
			style:     "FEISHU",
			template:  `通知：{{title}}`,
			wantCause: "rendered invalid JSON",
		},
		{
			name:      "dangling quote",
			style:     "WECOM",
			template:  `{"t":"{{title}}}`,
			wantCause: "rendered invalid JSON",
		},
	}

	for _, c := range cases {
		_, err := buildWebhookOutbound(webhookStyle(c.style), styleAccount(c.style, c.template, ""), styleReq())
		require.Error(t, err, c.name)
		require.Contains(t, err.Error(), c.wantCause, c.name)
		require.Contains(t, err.Error(), "channel [7]", "报错要带渠道 ID：库里可能有好几条")
		// 前缀只垫一次：内层消息也写 "webhook:" 的话，台账 last_error 就念成
		// "webhook: channel [7] webhook: payload template …"（本机实测撞出来的原话）。
		require.Equal(t, 1, strings.Count(err.Error(), "webhook: "), c.name)
	}
}

// TestRenderWebhookTemplateUnclosedIsLiteral 没有闭合的花括号当正文尾巴。
func TestRenderWebhookTemplateUnclosedIsLiteral(t *testing.T) {
	// "{{{" 这类值出现在通知正文里不该让整条投递失败：它在 JSON 字符串里是合法内容。
	got, err := renderWebhookTemplate(`{"t":"{{title}}","x":"{{unclosed"}`, map[string]string{"title": "v"})
	require.NoError(t, err)
	require.Equal(t, `{"t":"v","x":"{{unclosed"}`, string(got))
}

// TestPrecheckWebhookAccount 入队自检：配置类失败一律包成 ErrChannelNotConfigured（台账记 SKIPPED）。
func TestPrecheckWebhookAccount(t *testing.T) {
	for _, ok := range []*data.WebhookAccount{
		// 存量行的默认：两列都空。
		styleAccount("", "", ""),
		styleAccount("CUSTOM", "", "k"),
		styleAccount("NONE", "", ""),
		styleAccount("DINGTALK", "", "k"),
		styleAccount("DINGTALK", "", ""),
		styleAccount("FEISHU", `{"msg_type":"text","content":{"text":"{{title}}"},"s":"{{sign}}"}`, "k"),
		styleAccount("WECOM", `{"msgtype":"text","text":{"content":"{{content}}"}}`, ""),
	} {
		require.NoError(t, precheckWebhookAccount(ok), "style=%q template=%q", ok.SignStyle, ok.PayloadTemplate)
	}

	broken := []*data.WebhookAccount{
		styleAccount("DINGTAKL", "", ""),
		styleAccount("CUSTOM", `{"t":{{title}}}`, ""),
		styleAccount("CUSTOM", `{"s":"{{sign}}"}`, "k"),
		styleAccount("FEISHU", `plain {{title}}`, "k"),
		// 飞书要往顶层并签名字段，模板渲染出数组就并不进去。
		styleAccount("FEISHU", `["{{title}}"]`, "k"),
	}

	for _, account := range broken {
		err := precheckWebhookAccount(account)
		require.Error(t, err, "style=%q template=%q", account.SignStyle, account.PayloadTemplate)
		require.True(t, errors.Is(err, ErrChannelNotConfigured),
			"配置错属 SKIPPED，不该让一次性验证码按 FAILED 重试四次")
		require.Contains(t, err.Error(), "channel [7]")
	}
}

// TestCheckProviderResponse 200 但 body 报失败的判据。
func TestCheckProviderResponse(t *testing.T) {
	cases := []struct {
		name      string
		style     webhookStyle
		body      string
		wantError string
	}{
		{"dingtalk rejected", styleDingtalk, `{"errcode":310000,"errmsg":"keywords not in content"}`, "errcode=310000"},
		{"dingtalk ok", styleDingtalk, `{"errcode":0,"errmsg":"ok"}`, ""},
		{"dingtalk ignores feishu field", styleDingtalk, `{"code":19021}`, ""},
		{"wecom rejected", styleWecom, `{"errcode":93000,"errmsg":"invalid webhook k"}`, "invalid webhook k"},
		{"feishu rejected", styleFeishu, `{"code":19021,"msg":"sign match fail"}`, "code=19021"},
		{"feishu ok", styleFeishu, `{"code":0,"msg":"success"}`, ""},
		{"custom looks nowhere but status", styleCustom, `{"errcode":310000}`, ""},
		{"none same", styleNone, `{"errcode":310000}`, ""},
		{"html answer is not guessed", styleDingtalk, `<html>502 Bad Gateway</html>`, ""},
		{"empty body", styleDingtalk, ``, ""},
		{"json array", styleDingtalk, `[]`, ""},
		{"missing verdict field", styleFeishu, `{"msg":"failed"}`, ""},
	}

	for _, c := range cases {
		err := checkProviderResponse(c.style, []byte(c.body))
		if c.wantError == "" {
			require.NoError(t, err, c.name)
			continue
		}
		require.Error(t, err, c.name)
		require.Contains(t, err.Error(), "peer answered 200", c.name)
		require.Contains(t, err.Error(), c.wantError, c.name)
	}
}

// TestTruncateForError 进 last_error 的响应片段有上限（对端可能回一整页 HTML）。
func TestTruncateForError(t *testing.T) {
	require.Equal(t, "short", truncateForError([]byte("short")))

	long := strings.Repeat("a", webhookMaxResponseBytes+100)
	got := truncateForError([]byte(long))
	require.Len(t, got, webhookMaxResponseBytes+len("…"))
	require.True(t, strings.HasSuffix(got, "…"))
}

// TestJsonEscapeString 变量转义的边界：引号、反斜杠、控制字符与 HTML 三元组。
func TestJsonEscapeString(t *testing.T) {
	require.Equal(t, `a\"b`, jsonEscapeString(`a"b`))
	require.Equal(t, `a\\b`, jsonEscapeString(`a\b`))
	require.Equal(t, `a\nb`, jsonEscapeString("a\nb"), "真换行要写成 \\n 两字符，否则模板里的 JSON 断成两行")
	require.Equal(t, `a\tb`, jsonEscapeString("a\tb"))
	require.Equal(t, `a<b>&c`, jsonEscapeString(`a<b>&c`), "不转义 HTML：否则中文正文里的尖括号在群里变成 \\u003c")
	require.Equal(t, ``, jsonEscapeString(""))
	require.Equal(t, `中文`, jsonEscapeString("中文"))

	// 转义结果必须能原样嵌进引号里再解回来——这是"引号归模板作者"这一设计的成立前提。
	for _, raw := range []string{`带"引号"`, "换行\n与\t制表", `反斜杠\`, `<a href="x">&`, "中文"} {
		want, marshalErr := json.Marshal(map[string]string{"v": raw})
		require.NoError(t, marshalErr)
		require.JSONEq(t, string(want), `{"v":"`+jsonEscapeString(raw)+`"}`, "raw=%q", raw)
	}
}
