package netutil

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/go-kratos/kratos/v2/transport"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// 客户端真实 IP 相关的请求头。
const (
	headerXForwardedFor = "X-Forwarded-For"
	headerXRealIP       = "X-Real-IP"
)

// ============================================================================
// 可信代理配置（H5：防 XFF 伪造绕过限流）
//
// 默认信任常见内网代理网段。当请求的直连对端（RemoteAddr）属于可信代理时，
// 才从 X-Forwarded-For 链中"从右向左"跳过可信跳，取第一个非可信 IP 作为客户端真实 IP；
// 否则一律使用 RemoteAddr，避免攻击者通过伪造 XFF 头绕过基于 IP 的限流/审计。
//
// 部署时通过 SetTrustedProxies 用实际的反向代理网段覆盖默认值。
// ============================================================================

var (
	trustedProxies     []*net.IPNet
	trustedProxiesOnce sync.Once
	trustedProxiesMu   sync.RWMutex
)

// defaultTrustedProxyCIDRs 默认可信代理网段（RFC1918 私有地址 + 回环）。
// 生产部署应通过 SetTrustedProxies 精确指定实际反向代理网段。
func defaultTrustedProxyCIDRs() []string {
	return []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"::1/128",
		"fc00::/7",
	}
}

// initTrustedProxies 线程安全地初始化默认可信代理列表（仅一次）。
func initTrustedProxies() {
	trustedProxiesOnce.Do(func() {
		setTrustedProxiesLocked(defaultTrustedProxyCIDRs())
	})
}

// SetTrustedProxies 设置可信代理 CIDR 列表（覆盖默认值）。
// 应在服务启动阶段（main）调用一次。传入空切片表示不信任任何代理（始终用 RemoteAddr）。
// 非法 CIDR 会被跳过并忽略。
func SetTrustedProxies(cidrs []string) {
	trustedProxiesMu.Lock()
	defer trustedProxiesMu.Unlock()
	setTrustedProxiesLocked(cidrs)
}

func setTrustedProxiesLocked(cidrs []string) {
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		// 允许不带掩码的单 IP（按 /32 或 /128 处理）
		if !strings.Contains(c, "/") {
			ip := net.ParseIP(c)
			if ip == nil {
				continue
			}
			if ip.To4() != nil {
				c += "/32"
			} else {
				c += "/128"
			}
		}
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			continue
		}
		nets = append(nets, n)
	}
	trustedProxies = nets
}

// isTrustedProxy 判断 IP 是否属于可信代理网段。
func isTrustedProxy(ip net.IP) bool {
	trustedProxiesMu.RLock()
	defer trustedProxiesMu.RUnlock()
	if len(trustedProxies) == 0 {
		return false
	}
	for _, n := range trustedProxies {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// ClientIPFromRequest 从 *http.Request 中解析客户端真实 IP。
//
// 解析策略（防 XFF 伪造）：
//  1. 取 RemoteAddr 的直连对端 IP。
//  2. 若对端 IP 不是可信代理 → 直接返回对端 IP，完全忽略 XFF（防止伪造）。
//  3. 若对端 IP 是可信代理 → 从 X-Forwarded-For 链"从右向左"跳过可信跳，
//     第一个非可信 IP 即为客户端真实 IP；若整条链都是可信跳，取最左侧 IP。
//
// 注意：X-Real-IP 仅在对端为可信代理时才采信，且优先级低于 XFF 链解析。
func ClientIPFromRequest(req *http.Request) string {
	if req == nil {
		return ""
	}

	initTrustedProxies()

	// 1. 直连对端 IP（TCP 连接的真实来源，无法伪造）
	peerIP := ipFromRemoteAddr(req.RemoteAddr)
	peer := net.ParseIP(peerIP)

	// 2. 对端非可信代理 → 不信任任何转发头，直接用对端 IP
	if peer == nil || !isTrustedProxy(peer) {
		return peerIP
	}

	// 3. 对端是可信代理 → 从 XFF 链从右向左解析
	if xff := req.Header.Get(headerXForwardedFor); xff != "" {
		ips := splitAndTrim(xff)
		// 从右向左跳过可信跳，第一个非可信 IP 即客户端
		for i := len(ips) - 1; i >= 0; i-- {
			ipStr := ips[i]
			ip := net.ParseIP(ipStr)
			if ip == nil {
				continue
			}
			if !isTrustedProxy(ip) {
				return ipStr
			}
			// 若整条链都是可信跳，循环到最后会返回最左侧（i==0 时即使可信也返回）
			if i == 0 {
				return ipStr
			}
		}
	}

	// 4. XFF 为空或全部非法 → 采信 X-Real-IP（仅在可信代理前提下）
	if xri := req.Header.Get(headerXRealIP); xri != "" {
		if ip := net.ParseIP(strings.TrimSpace(xri)); ip != nil {
			return strings.TrimSpace(xri)
		}
	}

	// 5. 兜底：对端 IP（可信代理本身，如直连 nginx）
	return peerIP
}

// splitAndTrim 按逗号切分并去空格，过滤空串。
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ipFromRemoteAddr 从 RemoteAddr（host:port）中提取 host。
func ipFromRemoteAddr(remoteAddr string) string {
	if remoteAddr == "" {
		return ""
	}
	// 处理可能无端口的情形
	if !strings.Contains(remoteAddr, ":") {
		return remoteAddr
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err != nil {
		return remoteAddr
	}
	if net.ParseIP(host) != nil {
		return host
	}
	return host
}

// ClientIPFromContext 从 kratos context 中提取客户端真实 IP。
// service 层通过此函数即可取得 IP，无需直接依赖中间件包。
// 若上下文非 HTTP 传输或取不到 request，返回空串。
func ClientIPFromContext(ctx context.Context) string {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return ""
	}
	htr, ok := tr.(khttp.Transporter)
	if !ok {
		return ""
	}
	req := htr.Request()
	if req == nil {
		return ""
	}
	return ClientIPFromRequest(req)
}

// HeaderFromContext 从 kratos context 中提取 HTTP 请求头。
// 用于 service 层读取通过 header 传递的参数（如验证码 id/value）。
// 若上下文非 HTTP 传输或取不到 request，返回 nil。
func HeaderFromContext(ctx context.Context) http.Header {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return nil
	}
	htr, ok := tr.(khttp.Transporter)
	if !ok {
		return nil
	}
	req := htr.Request()
	if req == nil {
		return nil
	}
	return req.Header
}

// CookieFromContext 从 kratos context 中提取具名 HTTP cookie 值。
// 用于 refresh token 接口从 HttpOnly cookie 中读取刷新令牌。
// 若上下文非 HTTP 传输、取不到 request 或无此 cookie，返回空串。
func CookieFromContext(ctx context.Context, name string) string {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return ""
	}
	htr, ok := tr.(khttp.Transporter)
	if !ok {
		return ""
	}
	req := htr.Request()
	if req == nil {
		return ""
	}
	c, err := req.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}
