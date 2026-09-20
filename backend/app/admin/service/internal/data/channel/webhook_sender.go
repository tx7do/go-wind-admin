package channel

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"

	"go-wind-admin/app/admin/service/internal/data"
)

const (
	// webhookTimeout 单次回调的总超时（拨号 + 写请求 + 读响应）。
	// 10s 与脚本系统 http 模块的默认值同形：出站这件事不需要每个渠道各定一个数。
	webhookTimeout = 10 * time.Second

	// webhookDialTimeout 拨号阶段上限（掐在 TCP 握手，不影响慢但对端活着的响应）。
	webhookDialTimeout = 5 * time.Second

	// webhookMaxResponseBytes 失败时截入 last_error 的对端响应体上限。
	// 只用于排障（"对方 403 说了什么"），所以刻意小：对端可能返回一整页 HTML。
	webhookMaxResponseBytes = 512

	// EnvAllowPrivateWebhook 放行内网/环回地址的开关，仅供本机联调。
	// 值 "1" 时整条 SSRF 防线关闭（连环回都能发），生产部署不得设置。
	EnvAllowPrivateWebhook = "NOTIFICATION_WEBHOOK_ALLOW_PRIVATE"

	// 签名头：对端用 webhook_secret 按 `sha256=<hex(hmac("<timestamp>.", body))>` 复算比对。
	// 把时间戳折进签名而不是只签 body，是为了让"同一份 body 重放"必须同时重放时间戳，
	// 对端据此拒收旧消息；只签 body 的话重放是逐字节免费的。
	headerSignature = "X-Gw-Signature"
	headerTimestamp = "X-Gw-Timestamp"
)

// webhookBlockedNets SSRF 拦网的地址段。
//
// 为什么是一张显式 CIDR 表而不是 net.IP 的一组布尔判断：布尔法会漏掉 100.64.0.0/10
// （CGNAT，Go 的 IsPrivate 只认 RFC1918 三段，且其文档明写"不得用于访问控制"），
// 而这张表里每一项都是"本平台上真能打到东西"的段：环回、RFC1918、CGNAT、
// 链路本地（含 169.254.169.254 这块云元数据端点）、组播与保留段。
// 表还有一个好处：想知道防线是什么，读这一处就够，不必在脑内合成五个布尔判断的并集。
var webhookBlockedNets = mustParseNets(
	"0.0.0.0/8", // "本网络"：部分实现把 0.x 当本地
	"10.0.0.0/8",
	"100.64.0.0/10", // CGNAT：K8s 服务网段常落在这里
	"127.0.0.0/8",
	"169.254.0.0/16", // 链路本地 + 云厂商元数据端点
	"172.16.0.0/12",
	"192.0.0.0/24",
	"192.168.0.0/16",
	"198.18.0.0/15", // 基准测试段
	"224.0.0.0/4",
	"240.0.0.0/4",
	"::1/128",
	"fc00::/7", // IPv6 ULA
	"fe80::/10",
)

// WebhookSender HTTP 回调渠道：把通知正文 POST 到渠道配置的地址。
//
// 与 EmailSender 共用同一套渠道选择策略（显式 channel_id 优先，否则取第一个启用的同类型行），
// 因此"配了两条 webhook 会走哪条"这件事只有一个答案。
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
		client:  newWebhookClient(allowPrivateWebhook()),
	}
}

// allowPrivateWebhook 读环境变量决定要不要关掉 SSRF 防线。
// 只认精确的 "1"：真值判断（strconv.ParseBool）会让 "false"、"0" 之外的垃圾值也放行，
// 而这一条开关放行的内容是"本进程可向内网任意地址发请求"。
func allowPrivateWebhook() bool {
	return os.Getenv(EnvAllowPrivateWebhook) == "1"
}

// newWebhookClient 造一个"拨号前先把地址过一遍内网拦网"的 http.Client。
//
// 为什么防线在 dial 而不是在 url 校验：校验 hostname 只挡得住字面量（127.0.0.1、
// 169.254.169.254），挡不住 `internal.example.com → 10.0.0.5` 这种指到内网的记录。
// 这里自己解析、自己挑一个放行的 IP、并且**直接 dial 那个 IP**，
// 于是"校验的地址"与"连接的地址"是同一个，TOCTOU 的窗口只剩 DNS 重绑定这一种
// 需要在本进程解析之后再改记录的形式，而解析结果已经决定了连接目标。
func newWebhookClient(allowPrivate bool) *http.Client {
	dialer := &net.Dialer{Timeout: webhookDialTimeout, KeepAlive: 15 * time.Second}

	return &http.Client{
		Timeout: webhookTimeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				if allowPrivate {
					return dialer.DialContext(ctx, network, addr)
				}
				return dialGuarded(ctx, dialer, network, addr)
			},
		},
		// 不跟随重定向：Location 指到哪就等于绕过了"管理员配的这条地址"，
		// 台账里那行 target 也成了假话。3xx 一律当失败处理。
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// errBlockedTarget SSRF 防线主动拒绝拨号时垫底的哨兵。
//
// 为什么要单独一个错误而不是一句普通文本：拦网发生在拨号之前，一次都没有联系对端，
// 台账因此必须记 SKIPPED（配置/地址问题）而不是 FAILED（拨过号被拒），
// 异步侧也据此跳过重试——再拨四次也不会让这条 DNS 记录改口。
// 而 http.Client 会把 dial 错误包成 url.Error，只能靠 errors.Is 沿链找到它。
var errBlockedTarget = errors.New("webhook target blocked by the ssrf guard")

// dialGuarded 解析 addr 的 hostname，挑一个不在拦网表里的 IP 直连。
func dialGuarded(ctx context.Context, dialer *net.Dialer, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("webhook: bad dial address %q: %w", addr, err)
	}

	// "ip" 而非 network：这里要的是"两种族都查"，而 network 传的是 tcp/tcp4/tcp6 这类协议名。
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("webhook: resolve %q failed: %w", host, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("webhook: %q resolves to no address", host)
	}

	for _, ip := range ips {
		// 一个名字只要有一条记录指向内网就整条拒掉：允许"混着指"等于把
		// "这一次到底打到了哪"交给 DNS 的返回顺序。
		if blocked, reason := isBlockedWebhookTarget(ip.IP); blocked {
			return nil, fmt.Errorf("%w: target %q resolves to %s which is %s — off-host callbacks must point at a public address (%s=1 disables this check for local debugging)",
				errBlockedTarget, host, ip.IP.String(), reason, EnvAllowPrivateWebhook)
		}
		conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
		if dialErr != nil {
			// 这个 IP 试不通就接着试下一个：多 A 记录（v4/v6 双栈）是常态，
			// 拿第一条的连接失败当整次投递的失败会误判。
			err = dialErr
			continue
		}
		return conn, nil
	}

	return nil, fmt.Errorf("webhook: dial %q failed: %w", addr, err)
}

// isBlockedWebhookTarget 地址是否落在拦网表里。
//
// 返回的 reason 会进台账的 last_error，所以要能看懂"为什么这条被拦"。
func isBlockedWebhookTarget(ip net.IP) (bool, string) {
	if ip == nil {
		return true, "not an IP address"
	}
	for _, parsed := range webhookBlockedNets {
		if parsed.Contains(ip) {
			return true, "inside the blocked range " + parsed.String()
		}
	}
	return false, ""
}

// mustParseNets 解析拦网表；写错在启动期 panic，不留到第一次投递才报错。
func mustParseNets(cidrs ...string) []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, parsed, err := net.ParseCIDR(c)
		if err != nil {
			panic(fmt.Sprintf("channel: bad webhook blocked CIDR %q: %v", c, err))
		}
		nets = append(nets, parsed)
	}
	return nets
}

// webhookPayload 出站 JSON 的形状。字段名与站内信/台账用词一致，对端不必查表对照。
type webhookPayload struct {
	EventType       string `json:"event_type"`
	Title           string `json:"title"`
	Content         string `json:"content"`
	RecipientUserID uint32 `json:"recipient_user_id,omitempty"`
	RelatedID       uint32 `json:"related_id,omitempty"`
	DeliveredAt     string `json:"delivered_at"`
}

// Channel 渠道标识。
func (s *WebhookSender) Channel() notificationV1.Channel {
	return notificationV1.Channel_WEBHOOK
}

// Precheck 只解析渠道配置并自检可用性，不拨号。见 channel.Prechecker。
func (s *WebhookSender) Precheck(ctx context.Context, channelID uint32) error {
	_, err := s.pickAccount(ctx, channelID)
	return err
}

// Send 投递一次回调。返回 error 一律是"投递失败"（含渠道不可用），
// 由服务层用 ErrChannelNotConfigured 判别 SKIPPED / FAILED。
func (s *WebhookSender) Send(ctx context.Context, req *SendRequest) (*SendReceipt, error) {
	if req == nil || req.Target == "" {
		return nil, errors.New("webhook target url is empty")
	}

	account, err := s.pickAccount(ctx, req.ChannelID)
	if err != nil {
		return nil, err
	}

	endpoint, err := normalizeWebhookURL(req.Target)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrChannelNotConfigured, err)
	}

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

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("webhook: build request for channel [%d] failed: %w", account.ID, err)
	}
	httpReq.Header.Set("Content-Type", "application/json; charset=utf-8")
	httpReq.Header.Set("User-Agent", "go-wind-admin-notification/1.0")
	if account.Secret != "" {
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		httpReq.Header.Set(headerTimestamp, ts)
		httpReq.Header.Set(headerSignature, "sha256="+signWebhookBody(account.Secret, ts, body))
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		if errors.Is(err, errBlockedTarget) {
			// 一次都没联系对端 = 地址不合法，属"这条投递没法发"（SKIPPED），不是"发了但被拒"。
			return nil, fmt.Errorf("%w: %v", ErrChannelNotConfigured, err)
		}

		return nil, fmt.Errorf("send webhook via channel [%d] failed: %w", account.ID, err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, webhookMaxResponseBytes))
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, webhookMaxResponseBytes))
		return nil, fmt.Errorf("send webhook via channel [%d] failed: peer answered %s: %s",
			account.ID, resp.Status, string(snippet))
	}

	return &SendReceipt{ChannelID: account.ID}, nil
}

// pickAccount 解析实际使用的渠道配置并自检可用性。
//
// 与 EmailSender.pickAccount 同一条判据：一切"渠道用不了"都包成 ErrChannelNotConfigured，
// 服务层据此把台账记 SKIPPED（没拨过号）而不是 FAILED（拨过号被拒）。
func (s *WebhookSender) pickAccount(ctx context.Context, channelID uint32) (*data.WebhookAccount, error) {
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
func (s *WebhookSender) resolveAccount(ctx context.Context, channelID uint32) (*data.WebhookAccount, error) {
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

// normalizeWebhookURL 只放行绝对 http(s) 地址。
//
// 相对地址（"//10.0.0.5/x"、"/y"）会在这一步被挡掉：它们到了 http.Client 那里
// 会报一个不含原址的错，排障时看不出"配在库里的值本来就不是绝对地址"。
func normalizeWebhookURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("webhook url %q is invalid: %w", raw, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("webhook url %q must be http(s)", raw)
	}
	if u.Host == "" {
		return "", fmt.Errorf("webhook url %q has no host", raw)
	}
	return u.String(), nil
}

// signWebhookBody HMAC-SHA256("<timestamp>.", body) 的十六进制。
func signWebhookBody(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
