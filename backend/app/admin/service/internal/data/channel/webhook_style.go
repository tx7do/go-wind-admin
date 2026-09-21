package channel

import (
	"bytes"
	"crypto/hmac"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go-wind-admin/app/admin/service/internal/data"
)

// webhookStyle 一条 WEBHOOK 渠道的"出站风格"：签名怎么算、签名放哪、用什么判对端收下了没有。
//
// 值与 sys_notification_channels.webhook_sign_style 列、proto 的 SignStyle 成员名逐字相同：
// repo 的 EnumTypeConverter 按名字字符串配对，任何一侧加前缀都会让那一列静默变 nil。
type webhookStyle string

const (
	styleCustom   webhookStyle = "CUSTOM"
	styleNone     webhookStyle = "NONE"
	styleDingtalk webhookStyle = "DINGTALK"
	styleFeishu   webhookStyle = "FEISHU"
	styleWecom    webhookStyle = "WECOM"
)

// builtinPayloadTemplates 各风格的内置默认正文，管理员没填 webhook_payload_template 时用。
//
// 只有 text 一种消息形状：markdown 与卡片的结构三家各不同，且都要额外的排版决策，
// 那是"模板"下一期的事；text 足够让一条通知进群被人看见。
// 企微 text 正文上限 2048 字节、钉钉 20000 —— 这里不截断：按字节数腰斩一条通知，
// 不如让对端的报错说清楚"太长了"。
var builtinPayloadTemplates = map[webhookStyle]string{
	styleDingtalk: `{"msgtype":"text","text":{"content":"{{title}}\n{{content}}"}}`,
	styleFeishu:   `{"msg_type":"text","content":{"text":"{{title}}\n{{content}}"}}`,
	styleWecom:    `{"msgtype":"text","text":{"content":"{{title}}\n{{content}}"}}`,
}

// webhookOutbound 一次出站请求的渲染结果：发什么、签名放哪、用什么判成败。
type webhookOutbound struct {
	body []byte

	// timestampHeader / signatureHeader 放进请求头的签名（CUSTOM）。空则不带。
	timestampHeader string
	signatureHeader string

	// query 要拼进 URL 的签名参数（钉钉）。nil 表示不改地址。
	query url.Values

	// verdictField 判"对端真的收下了"要看的 body 字段名；空表示只看 HTTP 状态码。
	//
	// 为什么非得看 body：三家都会在 HTTP 200 里返回失败（钉钉被关键词/安全策略拦下时
	// 给的就是 200 + errcode 310000），只看状态码会把"对方没收"记成 DELIVERED。
	verdictField string
}

// resolveWebhookStyle 风格列 → 风格。
//
// 空串按 CUSTOM：这一列可空，NULL 的语义就是"加列之前只有 CUSTOM 一种行为"。
// （本机实测：PG 加列带 DEFAULT，存量行被回填成 'CUSTOM' 而非 NULL，所以空串这一档主要兜直接写 SQL 置 NULL 的行。）
// 认不出的值报错而不是兜成 CUSTOM：把 DINGTALK 拼成 DINGTAKL 后静默按自有方案发出去，
// 对端只表现为"签名不对"，排查方向整个是反的。
func resolveWebhookStyle(raw string) (webhookStyle, error) {
	switch webhookStyle(raw) {
	case "":
		return styleCustom, nil
	case styleCustom, styleNone, styleDingtalk, styleFeishu, styleWecom:
		return webhookStyle(raw), nil
	default:
		return "", fmt.Errorf("unknown webhook sign style %q (expected CUSTOM / NONE / DINGTALK / FEISHU / WECOM)", raw)
	}
}

// verdictFieldOf 该风格用哪个 body 字段判"对端真的收下了"。
func verdictFieldOf(style webhookStyle) string {
	switch style {
	case styleDingtalk, styleWecom:
		return "errcode"
	case styleFeishu:
		return "code"
	default:
		return ""
	}
}

// buildWebhookOutbound 按风格算出正文与签名。
//
// 渲染顺序就是这件事的难点：钉钉/飞书的签名只盖时间戳与密钥、不盖正文，所以签名能先算出来、
// 再当占位符用；CUSTOM 的签名盖住正文，于是正文里不可能出现 {{sign}} —— 引用它就在这儿报错，
// 而不是渲染出一个空串让对端验签失败。
func buildWebhookOutbound(style webhookStyle, account *data.WebhookAccount, req *SendRequest) (*webhookOutbound, error) {
	timestamp := webhookTimestamp(style)

	// bodySign 只在"签名不依赖正文"的风格里有值；有值才作为 {{sign}} 暴露。
	var bodySign string
	outbound := &webhookOutbound{verdictField: verdictFieldOf(style)}

	switch style {
	case styleDingtalk:
		if account.Secret != "" {
			bodySign = signDingtalk(account.Secret, timestamp)
			outbound.query = url.Values{"timestamp": {timestamp}, "sign": {bodySign}}
		}
	case styleFeishu:
		if account.Secret != "" {
			bodySign = signFeishu(account.Secret, timestamp)
		}
	case styleCustom:
		// 头里的时间戳先定下来：它不进正文，也就不可能形成循环。
		outbound.timestampHeader = timestamp
	case styleNone, styleWecom:
		// 无签名：企微的凭据在 URL 的 key 上，NONE 是什么都不带。
	}

	body, err := renderWebhookBody(style, account, req, timestamp, bodySign)
	if err != nil {
		return nil, err
	}
	outbound.body = body

	// 飞书把签名当正文字段，所以它必须在模板渲染之后并进顶层对象。
	if style == styleFeishu && account.Secret != "" {
		merged, mergeErr := mergeWebhookEnvelope(body, timestamp, bodySign)
		if mergeErr != nil {
			return nil, fmt.Errorf("webhook: channel [%d] %w", account.ID, mergeErr)
		}
		outbound.body = merged
	}

	if style == styleCustom && account.Secret != "" {
		outbound.signatureHeader = "sha256=" + signWebhookBody(account.Secret, timestamp, outbound.body)
	}

	return outbound, nil
}

// webhookTimestamp 该风格的时间戳精度：钉钉要毫秒，其余按秒。
func webhookTimestamp(style webhookStyle) string {
	now := time.Now()
	if style == styleDingtalk {
		return strconv.FormatInt(now.UnixMilli(), 10)
	}
	return strconv.FormatInt(now.Unix(), 10)
}

// renderWebhookBody 渲染出站正文。
//
// 风格这一列只管"签名怎么算、放哪、用什么判成败"，正文形状是另一件事：
// CUSTOM 与 NONE 的默认正文都是 webhookPayload 那份 JSON（两者只差在签不签名），
// 三家群机器人则各用 builtinPayloadTemplates 里的 text 形状。
//
// CUSTOM 且没配模板时走 json.Marshal(webhookPayload)：那是这条渠道在本方案里的既有字节形状，
// 已经照它配好验签的对端不该因为这次加列而收到一份字段顺序不同的等价 JSON。
func renderWebhookBody(style webhookStyle, account *data.WebhookAccount, req *SendRequest, timestamp, sign string) ([]byte, error) {
	template := account.PayloadTemplate
	if template == "" {
		if style == styleCustom || style == styleNone {
			body, err := json.Marshal(webhookPayload{
				EventType:       req.EventType.String(),
				Title:           req.Title,
				Content:         req.Content,
				RecipientUserID: req.RecipientUserID,
				RelatedID:       req.RelatedID,
				DeliveredAt:     time.Now().UTC().Format(time.RFC3339),
			})
			if err != nil {
				return nil, fmt.Errorf("webhook: encode payload failed: %w", err)
			}
			return body, nil
		}

		template = builtinPayloadTemplates[style]
	}

	rendered, err := renderWebhookTemplate(template, webhookVars(req, timestamp, sign))
	if err != nil {
		return nil, fmt.Errorf("webhook: channel [%d] %w", account.ID, err)
	}
	if !json.Valid(rendered) {
		return nil, fmt.Errorf("webhook: channel [%d] payload template rendered invalid JSON: %s", account.ID, truncateForError(rendered))
	}
	return rendered, nil
}

// webhookVars 占位符表。值一律按 JSON 字符串内容转义、不加引号：
// 于是放在引号里的 {{title}} 与不放在引号里的 {{recipient_user_id}} 都能用，引号归模板作者。
func webhookVars(req *SendRequest, timestamp, sign string) map[string]string {
	vars := map[string]string{
		"title":             jsonEscapeString(req.Title),
		"content":           jsonEscapeString(req.Content),
		"event_type":        jsonEscapeString(req.EventType.String()),
		"delivered_at":      jsonEscapeString(time.Now().UTC().Format(time.RFC3339)),
		"timestamp":         jsonEscapeString(timestamp),
		"recipient_user_id": strconv.FormatUint(uint64(req.RecipientUserID), 10),
		"related_id":        strconv.FormatUint(uint64(req.RelatedID), 10),
		"nonce":             crand.Text(),
	}

	// 没有签名可暴露时把键删掉，而不是留一个空串：空串会让对端收到"sign":""，
	// 看上去像"签了个空签名"，而真实情况是这条渠道压根没配密钥。
	if sign == "" {
		delete(vars, "sign")
	} else {
		vars["sign"] = jsonEscapeString(sign)
	}

	return vars
}

// renderWebhookTemplate 单趟替换 {{name}}。
//
// 不用逐变量 strings.ReplaceAll：那样 {{content}} 的值里若含 "{{title}}"，
// 后一轮替换会把正文文本当模板再渲染一次。单趟扫描让每个值只出现一次，天然不递归。
// 未识别的占位符报错——拼错的 {{contnet}} 原样留在正文里，对端只会收到一串花括号。
func renderWebhookTemplate(template string, vars map[string]string) ([]byte, error) {
	var out strings.Builder

	rest := template
	for len(rest) > 0 {
		open := strings.Index(rest, "{{")
		if open < 0 {
			out.WriteString(rest)
			break
		}
		closing := strings.Index(rest[open+2:], "}}")
		if closing < 0 {
			// 没有闭合：当正文尾巴原样发出（"{{" 在 JSON 字符串里是合法内容，不值得为它失败）。
			out.WriteString(rest)
			break
		}
		name := rest[open+2 : open+2+closing]
		value, ok := vars[name]
		if !ok {
			// 不带 "webhook:" 前缀：调用方（renderWebhookBody）会统一垫一句
			// "webhook: channel [id] …"，实测里两处都写就把台账的 last_error 念成了结巴。
			return nil, fmt.Errorf("payload template uses unsupported placeholder {{%s}}", name)
		}
		out.WriteString(rest[:open])
		out.WriteString(value)
		rest = rest[open+2+closing+2:]
	}

	return []byte(out.String()), nil
}

// mergeWebhookEnvelope 把 timestamp / sign 并进正文顶层对象（飞书要求的签名落点）。
//
// 只并顶层：飞书看的是 body 顶层那两个字段。模板作者把文字渲染进 content.text 里，
// 不该为此负责信封字段——所以渲染结果必须是 JSON 对象，否则这里报错。
func mergeWebhookEnvelope(body []byte, timestamp, sign string) ([]byte, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("feishu signature goes into the body, so the payload template must render a JSON object; got: %s", truncateForError(body))
	}

	rawTimestamp, err := json.Marshal(timestamp)
	if err != nil {
		return nil, fmt.Errorf("encode timestamp %q failed: %w", timestamp, err)
	}
	rawSign, err := json.Marshal(sign)
	if err != nil {
		return nil, fmt.Errorf("encode signature failed: %w", err)
	}

	envelope["timestamp"] = rawTimestamp
	envelope["sign"] = rawSign

	// 用 Encoder 而不是 json.Marshal：Marshal 会对 MarshalJSON 的产物再 HTML 转义一次，
	// 于是模板作者精心保住的 < > &（见 jsonEscapeString 的 SetEscapeHTML(false)）
	// 在这一步全变成 \u003c —— 群里 @ 人的 <at> 标签直接失效。本机实测抓到的就是这个。
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(envelope); err != nil {
		return nil, fmt.Errorf("merge envelope fields failed: %w", err)
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// precheckWebhookAccount 入队前的配置自检：风格认得出来、模板渲染得出合法 JSON。
//
// 用一份假正文渲染：模板能引用的变量全是这次投递的字段，而"能不能渲染成合法 JSON"
// 与内容无关 —— 所以不必等真实验证码躺在队列里失败四次。
// 失败一律包成 ErrChannelNotConfigured：这是配置问题，台账记 SKIPPED 而非 FAILED。
func precheckWebhookAccount(account *data.WebhookAccount) error {
	style, err := resolveWebhookStyle(account.SignStyle)
	if err != nil {
		return fmt.Errorf("%w: channel [%d] %v", ErrChannelNotConfigured, account.ID, err)
	}

	if _, err = buildWebhookOutbound(style, account, &SendRequest{
		Title:           "precheck",
		Content:         "precheck",
		RecipientUserID: 1,
	}); err != nil {
		// 不再补一次渠道 ID：正文与模板都是渠道行的事，build 出来的错已经带着了。
		return fmt.Errorf("%w: %v", ErrChannelNotConfigured, err)
	}

	return nil
}

// jsonEscapeString 转义成 JSON 字符串内容（不含首尾引号）。
//
// SetEscapeHTML(false)：中文正文里出现 < > & 时不该被写成 \u003c，那会让群里的字变成转义串。
func jsonEscapeString(s string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	// 目标是一个 bytes.Buffer，Encode 一个 string 不会失败；失败只可能是编程错误。
	if err := enc.Encode(s); err != nil {
		return strconv.Quote(s)[1 : len(strconv.Quote(s))-1]
	}
	encoded := buf.String()
	// Encode 写的是 `"…"\n`：去掉首尾引号与那个换行。
	return encoded[1 : len(encoded)-2]
}

// signDingtalk 钉钉：base64(hmac_sha256(key=secret, data="<ms>\\n<secret>"))。
//
// 密钥与消息的分工容易记反——它和飞书恰好相反（钉钉 key=secret，飞书 key=待签串、数据为空），
// 而两种写法的错误表现都是对端回 "sign not match"，所以向量测试各钉一条（见 webhook_style_test.go）。
func signDingtalk(secret, timestamp string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "\n" + secret))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// signFeishu 飞书：base64(hmac_sha256(key="<s>\\n<secret>", data=空))。
func signFeishu(secret, timestamp string) string {
	mac := hmac.New(sha256.New, []byte(timestamp+"\n"+secret))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// checkProviderResponse 该风格下对端 body 是否报失败。
//
// 只在 2xx 之后调用：非 2xx 由 Send 的失败分支带状态码与响应片段报出去。
// body 不是 JSON 时放行——对端已经用 2xx 表了态，猜它的意思比承认"看不出"更糟。
func checkProviderResponse(style webhookStyle, body []byte) error {
	verdict := verdictFieldOf(style)
	if verdict == "" {
		return nil
	}

	var answer struct {
		ErrCode *int64 `json:"errcode"`
		Code    *int64 `json:"code"`
		ErrMsg  string `json:"errmsg"`
		Msg     string `json:"msg"`
	}
	if err := json.Unmarshal(body, &answer); err != nil {
		return nil
	}

	code := answer.ErrCode
	if verdict == "code" {
		code = answer.Code
	}
	if code == nil || *code == 0 {
		return nil
	}

	reason := answer.ErrMsg
	if reason == "" {
		reason = answer.Msg
	}
	if reason == "" {
		reason = "no message"
	}
	return fmt.Errorf("peer answered 200 but %s=%d: %s", verdict, *code, reason)
}

// truncateForError 进 last_error 的响应片段截断：对端可能回一整页 HTML。
func truncateForError(body []byte) string {
	if len(body) <= webhookMaxResponseBytes {
		return string(body)
	}
	return string(body[:webhookMaxResponseBytes]) + "…"
}
