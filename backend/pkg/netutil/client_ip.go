// Package netutil 提供绑定 kratos transport 的网络辅助函数。
//
// 客户端真实 IP 解析（P2 修复）：X-Forwarded-For 只能信任来自可信代理的
// 追加——通过 GOWIND_TRUSTED_PROXIES（逗号分隔 CIDR/IP）配置可信代理链；
// 未配置时 XFF 一律不信任，直接返回 RemoteAddr。此前委托 go-utils 取 XFF
// 最左值，直连部署下客户端可任意伪造，污染五类审计/登录限流/AI 告警。
package netutil

import (
	"context"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/go-kratos/kratos/v2/transport"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// 可信代理链（进程级，启动期经 SetTrustedProxiesFromEnv 装载一次）。
var (
	trustedMu      sync.RWMutex
	trustedProxies []*net.IPNet
)

// SetTrustedProxies 配置可信代理网段。传 nil 即清空（回到"不信任任何 XFF"）。
func SetTrustedProxies(cidrs []*net.IPNet) {
	trustedMu.Lock()
	defer trustedMu.Unlock()
	trustedProxies = cidrs
}

// SetTrustedProxiesFromEnv 从 GOWIND_TRUSTED_PROXIES 解析并生效。
// 条目支持 CIDR 或裸 IP；解析失败的条目跳过。
func SetTrustedProxiesFromEnv() {
	raw := strings.TrimSpace(os.Getenv("GOWIND_TRUSTED_PROXIES"))
	if raw == "" {
		SetTrustedProxies(nil)
		return
	}
	var cidrs []*net.IPNet
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !strings.Contains(part, "/") {
			part += "/32"
		}
		if _, ipnet, err := net.ParseCIDR(part); err == nil {
			cidrs = append(cidrs, ipnet)
		}
	}
	SetTrustedProxies(cidrs)
}

func isTrustedProxy(ip net.IP) bool {
	trustedMu.RLock()
	defer trustedMu.RUnlock()
	for _, cidr := range trustedProxies {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// remoteIP 从 RemoteAddr 剥端口取 IP 字符串（兼容 IPv6 [::1] 形态）。
func remoteIP(r *http.Request) string {
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	if ip == nil {
		return host
	}
	return ip.String()
}

// clientIPFromRequest 可信代理链语义的客户端真实 IP 解析。
//
// 未配置可信代理：直接返回 RemoteAddr。
// 已配置：仅当 RemoteAddr 本身是可信代理时才解析 XFF（XFF 的条目只能由
// 可信代理追加——直连客户端自带的 XFF 任何条目都不可信，必须整体忽略）；
// 沿 XFF 由右向左遍历，第一个不属于可信代理的条目即客户端；
// 全部可信（XFF 耗尽）取最左条目。
func clientIPFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	remote := remoteIP(r)
	if len(trustedProxies) == 0 {
		return remote
	}
	if !isTrustedProxy(net.ParseIP(remote)) {
		return remote
	}

	xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if xff == "" {
		return remote
	}

	parts := strings.Split(xff, ",")
	cur := remote
	for i := len(parts) - 1; i >= 0; i-- {
		ip := net.ParseIP(strings.TrimSpace(parts[i]))
		if ip == nil {
			return cur
		}
		if !isTrustedProxy(ip) {
			return ip.String()
		}
		cur = ip.String()
	}
	return cur
}

// ClientRealIPFromRequest 审计中间件用的真实 IP 提取（可信代理链语义）。
func ClientRealIPFromRequest(r *http.Request) string {
	return clientIPFromRequest(r)
}

// ClientIPFromContext 从 kratos context 中提取客户端真实 IP。
// service 层通过此函数即可取得 IP，无需直接依赖中间件包。
// 若上下文非 HTTP 传输或取不到 request，返回空串。
func ClientIPFromContext(ctx context.Context) string {
	return clientIPFromRequest(requestFromContext(ctx))
}

// HeaderFromContext 从 kratos context 中提取 HTTP 请求头。
// 用于 service 层读取通过 header 传递的参数（如验证码 id/value）。
// 若上下文非 HTTP 传输或取不到 request，返回 nil。
func HeaderFromContext(ctx context.Context) http.Header {
	req := requestFromContext(ctx)
	if req == nil {
		return nil
	}
	return req.Header
}

// CookieFromContext 从 kratos context 中提取具名 HTTP cookie 值。
// 用于 refresh token 接口从 HttpOnly cookie 中读取刷新令牌。
// 若上下文非 HTTP 传输、取不到 request 或无此 cookie，返回空串。
func CookieFromContext(ctx context.Context, name string) string {
	req := requestFromContext(ctx)
	if req == nil {
		return ""
	}
	c, err := req.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}

// requestFromContext 从 kratos server context 中取出 *http.Request。
func requestFromContext(ctx context.Context) *http.Request {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return nil
	}
	htr, ok := tr.(khttp.Transporter)
	if !ok {
		return nil
	}
	return htr.Request()
}
