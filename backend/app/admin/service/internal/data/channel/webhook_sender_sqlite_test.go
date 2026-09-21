// WebhookSender 的 SQLite 内存库 + 本地 httptest 测试。
//
// 覆盖只有真发一次 HTTP 才暴露得出的几件事：
//   - 出站 JSON 的形状与签名头（对端唯一的判据：模板渲染排在 P3，此刻正文里没有事件信息）；
//   - 签名风格两列在真实一次出站里的落点：钉钉的签名进 URL query、飞书的进 body 顶层，
//     以及"HTTP 200 但 body 报失败"必须判成投递失败（见 webhook_style.go）；
//   - SSRF 防线：拨号前按**解析后的 IP** 拒绝内网，并且归成 SKIPPED 语义（没联系过对端）；
//   - 非 2xx 与 3xx 都是"拨过号被拒"（FAILED），且报错带出对端响应片段；
//   - 渠道选择与 SMTP 那条路同判据：停用/类型不对/没配 url 一律 ErrChannelNotConfigured。
//
// 本机 httptest 的地址是 127.0.0.1，正好落在拦网表里，因此"发得出去"的几个用例必须
// t.Setenv 打开 NOTIFICATION_WEBHOOK_ALLOW_PRIVATE（这也是这个开关唯一的存在理由）；
// 而"拦得住"的用例把它置空，走的是生产默认的 guarded 路径。
package channel

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/enttest"

	notificationChannelV1 "go-wind-admin/api/gen/go/notification_channel/service/v1"
	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
)

// webhookRowSeq 只给测试内的渠道名用，理由见 createWebhook。
var webhookRowSeq uint64

type webhookSenderEnv struct {
	sender *WebhookSender
	repo   *data.NotificationChannelRepo
	ctx    context.Context

	// 对端侧记录：断言"到底有没有拨号"全靠它（records 为空 = 一次都没到）。
	records []*http.Request
	bodies  []string
}

// newWebhookSenderEnv 构造 sender。allowPrivate 决定 SSRF 防线开不开——
// 两个方向都必须显式设置：环境变量若在宿主 shell 里残留，防线会静默失效。
func newWebhookSenderEnv(t *testing.T, allowPrivate bool) *webhookSenderEnv {
	t.Helper()

	if allowPrivate {
		t.Setenv(EnvAllowPrivateWebhook, "1")
	} else {
		t.Setenv(EnvAllowPrivateWebhook, "")
	}

	entClient := enttest.NewEntClientForTest(t)
	repo := data.NewNotificationChannelRepoForTest(entClient)

	return &webhookSenderEnv{
		sender: NewWebhookSender(repo),
		repo:   repo,
		ctx:    enttest.NewSystemViewerCtx(context.Background()),
	}
}

// createWebhook 落一条 WEBHOOK 渠道配置，返回主键。url/secret 传空串表示该列不写。
func (e *webhookSenderEnv) createWebhook(t *testing.T, url, secret string, enabled bool) uint32 {
	t.Helper()

	// name 上有唯一索引，而 Windows 的 time.Now 粒度足以让同一测试里的两次调用取到同一个
	// UnixNano —— 计数器保证同库内不撞名（撞了报错是一句 500，指不到"重名"这件事）。
	id, err := e.repo.Create(e.ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:       trans.Ptr("webhook-" + strconv.FormatUint(atomic.AddUint64(&webhookRowSeq, 1), 10)),
			Type:       notificationChannelV1.NotificationChannel_WEBHOOK.Enum(),
			WebhookUrl: trans.Ptr(url),
			Enabled:    trans.Ptr(enabled),
		},
		WebhookSecret: trans.Ptr(secret),
	}, 1)
	require.NoError(t, err)

	return id
}

// receiver 起一个记录请求的本地服务，响应状态与正文由参数定。
func (e *webhookSenderEnv) receiver(t *testing.T, status int, responseBody string) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		e.records = append(e.records, r)
		e.bodies = append(e.bodies, string(body))
		w.WriteHeader(status)
		_, _ = io.WriteString(w, responseBody)
	}))
	t.Cleanup(srv.Close)

	return srv
}

func webhookReq(target string, channelID uint32) *SendRequest {
	return &SendRequest{
		Target:          target,
		Title:           "重置验证码",
		Content:         "您的验证码是 123456",
		EventType:       notificationV1.EventType_PASSWORD_RESET_CODE,
		ChannelID:       channelID,
		RecipientUserID: 1024,
		RelatedID:       7,
	}
}

// TestWebhookSenderSqlite_DeliversSignedJson 成功投递：出站体是约定形状，签名按
// `sha256=<hex(hmac("<timestamp>.", body))>` 可被对端复算。
func TestWebhookSenderSqlite_DeliversSignedJson(t *testing.T) {
	const secret = "s3cr3t-key"

	e := newWebhookSenderEnv(t, true)
	srv := e.receiver(t, http.StatusNoContent, "")
	id := e.createWebhook(t, srv.URL, secret, true)

	receipt, err := e.sender.Send(e.ctx, webhookReq(srv.URL, id))
	require.NoError(t, err)
	require.Equal(t, id, receipt.ChannelID, "回执要带真实命中的渠道 ID，台账的 channel_id 全靠它")

	require.Len(t, e.records, 1)
	rec := e.records[0]
	require.Equal(t, http.MethodPost, rec.Method)
	require.Equal(t, "application/json; charset=utf-8", rec.Header.Get("Content-Type"))

	var got webhookPayload
	require.NoError(t, json.Unmarshal([]byte(e.bodies[0]), &got))
	require.Equal(t, "PASSWORD_RESET_CODE", got.EventType,
		"对端只能靠 event_type 判断这是什么事件（正文里没有别的信息）")
	require.Equal(t, "重置验证码", got.Title)
	require.Equal(t, "您的验证码是 123456", got.Content)
	require.Equal(t, uint32(1024), got.RecipientUserID)
	require.Equal(t, uint32(7), got.RelatedID)
	_, parseErr := time.Parse(time.RFC3339, got.DeliveredAt)
	require.NoError(t, parseErr, "delivered_at 必须是 RFC3339，对端据此判重放")

	ts := rec.Header.Get(headerTimestamp)
	require.NotEmpty(t, ts)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts))
	mac.Write([]byte("."))
	mac.Write([]byte(e.bodies[0]))
	require.Equal(t, "sha256="+hex.EncodeToString(mac.Sum(nil)), rec.Header.Get(headerSignature),
		"签名必须覆盖时间戳：只签 body 的话重放是逐字节免费的")
}

// TestWebhookSenderSqlite_NoSecretHeaderWhenUnconfigured 没配密钥就不发签名头：
// 对端据此知道"这条渠道不需要验签"，而不是收到一个空值去和自己的 HMAC 比对。
func TestWebhookSenderSqlite_NoSecretHeaderWhenUnconfigured(t *testing.T) {
	e := newWebhookSenderEnv(t, true)
	srv := e.receiver(t, http.StatusOK, `{"ok":true}`)
	id := e.createWebhook(t, srv.URL, "", true)

	_, err := e.sender.Send(e.ctx, webhookReq(srv.URL, id))
	require.NoError(t, err)
	require.Empty(t, e.records[0].Header.Get(headerSignature))
	require.Empty(t, e.records[0].Header.Get(headerTimestamp))
}

// TestWebhookSenderSqlite_PrivateTargetIsBlockedBeforeDial SSRF 防线：生产默认（不开环境变量）
// 下 127.0.0.1 一次都拨不出去，并且被判成"没尝试过投递"（SKIPPED 语义）。
func TestWebhookSenderSqlite_PrivateTargetIsBlockedBeforeDial(t *testing.T) {
	e := newWebhookSenderEnv(t, false)
	srv := e.receiver(t, http.StatusOK, "")
	id := e.createWebhook(t, srv.URL, "", true)

	receipt, err := e.sender.Send(e.ctx, webhookReq(srv.URL, id))
	require.Error(t, err)
	require.NotNil(t, receipt, "渠道配置已选出，失败的「是哪条」也要进台账")
	require.Equal(t, id, receipt.ChannelID)
	require.Empty(t, e.records, "对端压根没收到请求才算拦住了")
	require.True(t, errors.Is(err, ErrChannelNotConfigured),
		"拨号前就拒绝 = 一次都没联系对端，台账该记 SKIPPED 而不是 FAILED")
	require.Contains(t, err.Error(), "127.0.0.1")
	require.Contains(t, err.Error(), EnvAllowPrivateWebhook, "报错要说出放行开关，否则本机联调无从下手")
}

// TestIsBlockedWebhookTarget 拦网表逐点验证。
//
// 172.15 与 100.63 是刻意放的"紧邻边界外"取值：/12 与 /10 写错一位就会把它们一起拦掉，
// 而漏拦的代价（打内网）与误拦的代价（公网回调发不出去）都只能靠断言发现。
func TestIsBlockedWebhookTarget(t *testing.T) {
	cases := []struct {
		ip         string
		wantReject bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.5", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"100.64.0.1", true},      // CGNAT：Go 的 IsPrivate 认不出来
		{"169.254.169.254", true}, // 云厂商元数据端点
		{"0.0.0.0", true},
		{"::1", true},
		{"fd00::1", true}, // fc00::/7 下半段
		{"fe80::1", true},
		{"8.8.8.8", false},
		{"172.15.255.255", false},
		{"100.63.255.255", false},
		{"169.253.0.1", false},
	}

	for _, c := range cases {
		blocked, reason := isBlockedWebhookTarget(net.ParseIP(c.ip))
		require.Equal(t, c.wantReject, blocked, "%s", c.ip)
		if blocked {
			require.NotEmpty(t, reason, "拦下的必须给出原因，它要进台账 last_error")
		}
	}
}

// TestWebhookSenderSqlite_PeerErrorIsFailed 对端 5xx：拨过号了 → FAILED，且带响应片段。
func TestWebhookSenderSqlite_PeerErrorIsFailed(t *testing.T) {
	e := newWebhookSenderEnv(t, true)
	srv := e.receiver(t, http.StatusInternalServerError, "downstream queue is full")
	id := e.createWebhook(t, srv.URL, "", true)

	receipt, err := e.sender.Send(e.ctx, webhookReq(srv.URL, id))
	require.Error(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, id, receipt.ChannelID, "FAILED 也要答得出走的哪条渠道配置")
	require.False(t, errors.Is(err, ErrChannelNotConfigured), "发出去了但被拒：属投递失败")
	require.Contains(t, err.Error(), "500")
	require.Contains(t, err.Error(), "downstream queue is full")
	require.Contains(t, err.Error(), "channel ["+itoa(id)+"]")
}

// TestWebhookSenderSqlite_RedirectIsNotFollowed 3xx 一律当失败：跟随 Location 等于让对端
// 决定这一发打到哪，既绕过拦网也让台账的 target 变成假话。
func TestWebhookSenderSqlite_RedirectIsNotFollowed(t *testing.T) {
	e := newWebhookSenderEnv(t, true)
	hit := e.receiver(t, http.StatusOK, "")

	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, hit.URL, http.StatusFound)
	}))
	t.Cleanup(redirect.Close)

	id := e.createWebhook(t, redirect.URL, "", true)

	_, err := e.sender.Send(e.ctx, webhookReq(redirect.URL, id))
	require.Error(t, err)
	require.Empty(t, e.records, "重定向的目标一次都不该被命中")
	require.Contains(t, err.Error(), "302")
}

// TestWebhookSenderSqlite_ConfigIssuesAreSkipped 停用 / 没配 url / 类型不对：
// 三种"渠道用不了"都归 ErrChannelNotConfigured，与"发出去被拒"分开。
func TestWebhookSenderSqlite_ConfigIssuesAreSkipped(t *testing.T) {
	e := newWebhookSenderEnv(t, true)
	srv := e.receiver(t, http.StatusOK, "")

	disabled := e.createWebhook(t, srv.URL, "", false)
	noUrl := e.createWebhook(t, "", "", true)

	emailID, err := e.repo.Create(e.ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:    trans.Ptr("email-" + strconv.FormatUint(atomic.AddUint64(&webhookRowSeq, 1), 10)),
			Type:    notificationChannelV1.NotificationChannel_EMAIL.Enum(),
			Enabled: trans.Ptr(true),
		},
	}, 1)
	require.NoError(t, err)

	for _, c := range []struct {
		name string
		id   uint32
	}{
		{"disabled", disabled},
		{"no webhook_url", noUrl},
		{"wrong type", emailID},
	} {
		receipt, err := e.sender.Send(e.ctx, webhookReq(srv.URL, c.id))
		require.Error(t, err, c.name)
		require.Nil(t, receipt, "%s：账号没选出，channel_id 留空才是事实", c.name)
		require.True(t, errors.Is(err, ErrChannelNotConfigured), "%s 应归为配置不可用", c.name)
	}

	require.Empty(t, e.records, "三条都不该真的发出去")
}

// TestNormalizeWebhookUrl 只有绝对 http(s) 地址放行。
func TestNormalizeWebhookUrl(t *testing.T) {
	for _, bad := range []string{"", "/hooks/1", "//127.0.0.1/x", "ftp://example.com/x", "not a url"} {
		_, err := normalizeWebhookURL(bad)
		require.Error(t, err, "%q 不该放行", bad)
	}

	for _, ok := range []string{"https://example.com/hook", "http://example.com:8080/h?a=1"} {
		got, err := normalizeWebhookURL(ok)
		require.NoError(t, err, "%q 应放行", ok)
		require.Equal(t, ok, got)
	}
}

// TestWebhookSenderSqlite_AutoPickSkipsDisabled 自选分支取的是"第一个启用的" WEBHOOK 行：
// ID 更小的停用行不该被用。
func TestWebhookSenderSqlite_AutoPickSkipsDisabled(t *testing.T) {
	e := newWebhookSenderEnv(t, true)
	srv := e.receiver(t, http.StatusOK, "")

	off := e.createWebhook(t, "http://127.0.0.1:1/nope", "", false)
	on := e.createWebhook(t, srv.URL, "", true)
	require.Greater(t, on, off, "断言依赖启用那条的 ID 较大")

	receipt, err := e.sender.Send(e.ctx, webhookReq(srv.URL, 0))
	require.NoError(t, err)
	require.Equal(t, on, receipt.ChannelID)
}

// createWebhookStyle 落一条带签名风格/载荷模板的 WEBHOOK 渠道行（本块新加的两列）。
// style 传空串表示该列不写（等价于存量行的 NULL）。
func (e *webhookSenderEnv) createWebhookStyle(t *testing.T, url, secret, style, template string, enabled bool) uint32 {
	t.Helper()

	data := &notificationChannelV1.NotificationChannel{
		Name:       trans.Ptr("wh-" + strconv.FormatUint(atomic.AddUint64(&webhookRowSeq, 1), 10)),
		Type:       notificationChannelV1.NotificationChannel_WEBHOOK.Enum(),
		WebhookUrl: trans.Ptr(url),
		Enabled:    trans.Ptr(enabled),
	}
	if style != "" {
		// 按名字查表而不是写 SignStyle_DINGTALK 常量：这几行自己也依赖"成员名与列值逐字相同"，
		// 拼错时上面这句 require 会当场说清楚是哪一行的问题。
		parsed, ok := notificationChannelV1.SignStyle_value[style]
		require.True(t, ok, "测试里的风格名拼错了：%q", style)
		data.WebhookSignStyle = notificationChannelV1.SignStyle(parsed).Enum()
	}
	if template != "" {
		data.WebhookPayloadTemplate = trans.Ptr(template)
	}

	id, err := e.repo.Create(e.ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data:          data,
		WebhookSecret: trans.Ptr(secret),
	}, 1)
	require.NoError(t, err)

	return id
}

// TestWebhookSenderSqlite_DingtalkSignsUrlQuery 钉钉风格在真发一次出站里的三个落点：
// 签名进 URL query（且不能吃掉地址原有的 access_token）、正文是钉钉的 text 形状、
// 自定义签名头一个都不发（钉钉只认 query）。
func TestWebhookSenderSqlite_DingtalkSignsUrlQuery(t *testing.T) {
	const secret = "ding-secret"

	e := newWebhookSenderEnv(t, true)
	srv := e.receiver(t, http.StatusOK, `{"errcode":0,"errmsg":"ok"}`)
	id := e.createWebhookStyle(t, srv.URL, secret, "DINGTALK", "", true)

	_, err := e.sender.Send(e.ctx, webhookReq(srv.URL+"/?access_token=abc123", id))
	require.NoError(t, err)
	require.Len(t, e.records, 1)

	q := e.records[0].URL.Query()
	require.Equal(t, "abc123", q.Get("access_token"), "拼签名必须保留地址原有的参数，否则钉钉先报 token 无效")

	ts := q.Get("timestamp")
	require.Len(t, ts, 13, "钉钉要 13 位毫秒，10 位秒会被判成 timestamp outside validity period")

	// 用 stdlib 独立复算，不调生产里的 signDingtalk：否则生产函数算错了这条断言也照样绿。
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "\n" + secret))
	require.Equal(t, base64.StdEncoding.EncodeToString(mac.Sum(nil)), q.Get("sign"))

	require.Empty(t, e.records[0].Header.Get(headerSignature))
	require.Empty(t, e.records[0].Header.Get(headerTimestamp))

	require.Contains(t, e.bodies[0], `"msgtype":"text"`)
	require.Contains(t, e.bodies[0], `重置验证码\n您的验证码是 123456`, "标题与正文之间是 JSON 转义后的 \\n")
	require.NotContains(t, e.bodies[0], "sign", "钉钉的签名不进正文")
}

// TestWebhookSenderSqlite_FeishuEnvelopeOnTheWire 飞书把签名当正文字段：顶层两个键真实到位、
// 且时间戳与签名成对可得（对端就是这么验的），模板渲染出来的正文不被信封并字段吃掉。
func TestWebhookSenderSqlite_FeishuEnvelopeOnTheWire(t *testing.T) {
	const secret = "feishu-secret"

	e := newWebhookSenderEnv(t, true)
	srv := e.receiver(t, http.StatusOK, `{"code":0,"msg":"success"}`)
	id := e.createWebhookStyle(t, srv.URL, secret, "FEISHU", "", true)

	_, err := e.sender.Send(e.ctx, webhookReq(srv.URL, id))
	require.NoError(t, err)

	var envelope struct {
		MsgType   string `json:"msg_type"`
		Content   struct {
			Text string `json:"text"`
		} `json:"content"`
		Timestamp string `json:"timestamp"`
		Sign      string `json:"sign"`
	}
	require.NoError(t, json.Unmarshal([]byte(e.bodies[0]), &envelope))

	require.Equal(t, "text", envelope.MsgType)
	require.Equal(t, "重置验证码\n您的验证码是 123456", envelope.Content.Text, "并信封字段不许丢正文")
	require.Len(t, envelope.Timestamp, 10, "飞书是 10 位秒时间戳")

	mac := hmac.New(sha256.New, []byte(envelope.Timestamp+"\n"+secret))
	require.Equal(t, base64.StdEncoding.EncodeToString(mac.Sum(nil)), envelope.Sign)

	require.Empty(t, e.records[0].Header.Get(headerSignature), "飞书不读自定义头")
}

// TestWebhookSenderSqlite_PeerRejectsWithin200IsFailed 三家都会在 HTTP 200 里报失败
// （钉钉被安全策略拦下时给的就是 200 + errcode 310000）。只看状态码会把"对方没收"记成
// DELIVERED，所以判据要落到 body 上；同时它仍是"拨过号被拒"= FAILED，不是 SKIPPED。
func TestWebhookSenderSqlite_PeerRejectsWithin200IsFailed(t *testing.T) {
	cases := []struct {
		name      string
		style     string
		answer    string
		wantCause string
	}{
		{"dingtalk", "DINGTALK", `{"errcode":310000,"errmsg":"keywords not in content"}`, "keywords not in content"},
		{"wecom", "WECOM", `{"errcode":93000,"errmsg":"invalid webhook k"}`, "invalid webhook k"},
		{"feishu", "FEISHU", `{"code":19021,"msg":"sign match fail"}`, "sign match fail"},
		// 判据字段各认各的：钉钉看到 code 不动作，飞书看到 errcode 不动作。
		{"field mismatch is not a verdict", "DINGTALK", `{"code":19021}`, ""},
	}

	for _, c := range cases {
		e := newWebhookSenderEnv(t, true)
		srv := e.receiver(t, http.StatusOK, c.answer)
		id := e.createWebhookStyle(t, srv.URL, "k", c.style, "", true)

		_, err := e.sender.Send(e.ctx, webhookReq(srv.URL, id))
		require.Len(t, e.records, 1, c.name)

		if c.wantCause == "" {
			require.NoError(t, err, c.name)
			continue
		}

		require.Error(t, err, c.name)
		require.Contains(t, err.Error(), c.wantCause, c.name)
		require.Contains(t, err.Error(), "peer answered 200", c.name)
		require.False(t, errors.Is(err, ErrChannelNotConfigured),
			"%s：拨过号且被对端拒 = FAILED；记成 SKIPPED 的话重投策略与台账颜色都会错", c.name)
	}
}

// TestWebhookSenderSqlite_BrokenTemplateNeverDials 模板渲染不出合法 JSON：错在出站之前，
// 一次都不拨号，归 SKIPPED 语义；入队自检要提前撞出同一个错，不等验证码躺在队列里失败四次。
func TestWebhookSenderSqlite_BrokenTemplateNeverDials(t *testing.T) {
	e := newWebhookSenderEnv(t, true)
	srv := e.receiver(t, http.StatusOK, "")

	broken := e.createWebhookStyle(t, srv.URL, "", "CUSTOM", `通知：{{title}}`, true)
	good := e.createWebhookStyle(t, srv.URL, "", "DINGTALK", "", true)

	receipt, err := e.sender.Send(e.ctx, webhookReq(srv.URL, broken))
	require.Error(t, err)
	require.NotNil(t, receipt, "渠道配置已选出，失败的「是哪条」要进台账")
	require.Equal(t, broken, receipt.ChannelID)
	require.True(t, errors.Is(err, ErrChannelNotConfigured), "配置错重试四次也不会变好")
	require.Contains(t, err.Error(), "channel ["+itoa(broken)+"]")
	require.Empty(t, e.records, "渲染失败发生在拨号之前")

	require.ErrorIs(t, e.sender.Precheck(e.ctx, broken), ErrChannelNotConfigured, "预检要提前拦住同一条")
	require.NoError(t, e.sender.Precheck(e.ctx, good), "内置模板的渠道预检该过")
}
