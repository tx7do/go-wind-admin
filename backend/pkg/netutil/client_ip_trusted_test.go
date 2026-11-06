package netutil

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func trustedReq(remote, xff string) *http.Request {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = remote
	if xff != "" {
		r.Header.Set("X-Forwarded-For", xff)
	}
	return r
}

// TestClientIP_TrustedProxyChain 可信代理链语义（P2 修复）：
// 未配置可信代理时 XFF 一律不信任（此前取最左值可任意伪造）；
// 配置后沿 XFF 右向左取第一个非可信条目。
func TestClientIP_TrustedProxyChain(t *testing.T) {
	reset := func() { SetTrustedProxies(nil) }
	defer reset()

	t.Run("无可信配置时 XFF 不信任", func(t *testing.T) {
		reset()
		require.Equal(t, "203.0.113.9",
			clientIPFromRequest(trustedReq("203.0.113.9:5555", "8.8.8.8")),
			"伪造 XFF 不应生效")
	})

	t.Run("可信链内直取 XFF 客户端", func(t *testing.T) {
		_, cidr, _ := net.ParseCIDR("10.0.0.0/8")
		SetTrustedProxies([]*net.IPNet{cidr})
		require.Equal(t, "8.8.8.8",
			clientIPFromRequest(trustedReq("10.1.2.3:5555", "8.8.8.8")),
			"RemoteAddr 可信时取 XFF 条目")
		require.Equal(t, "1.2.3.4",
			clientIPFromRequest(trustedReq("10.1.2.3:5555", "1.2.3.4, 10.1.2.3")),
			"多级代理取第一个非可信条目")
	})

	t.Run("不可信来源直连忽略 XFF", func(t *testing.T) {
		_, cidr, _ := net.ParseCIDR("10.0.0.0/8")
		SetTrustedProxies([]*net.IPNet{cidr})
		require.Equal(t, "203.0.113.9",
			clientIPFromRequest(trustedReq("203.0.113.9:5555", "8.8.8.8")),
			"直连客户端伪造 XFF 不应生效")
	})

	t.Run("全可信条目耗尽取最左", func(t *testing.T) {
		_, cidr, _ := net.ParseCIDR("10.0.0.0/8")
		SetTrustedProxies([]*net.IPNet{cidr})
		require.Equal(t, "10.9.9.9",
			clientIPFromRequest(trustedReq("10.1.2.3:5555", "10.9.9.9, 10.1.2.3")))
	})

	t.Run("SetTrustedProxiesFromEnv 解析", func(t *testing.T) {
		t.Setenv("GOWIND_TRUSTED_PROXIES", "192.168.1.1, 172.16.0.0/12")
		SetTrustedProxiesFromEnv()
		require.Equal(t, "8.8.8.8",
			clientIPFromRequest(trustedReq("192.168.1.1:5555", "8.8.8.8")),
			"裸 IP 条目应生效")
	})
}
