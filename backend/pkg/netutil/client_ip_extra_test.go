package netutil

// 本文件覆盖 client_ip.go 中既有测试未触达的分支：
// SetTrustedProxies 对 CIDR/裸 IP/空白/非法条目的解析规则（setTrustedProxiesLocked
// 的全部分支）、ipFromRemoteAddr 的各形态输入、以及 ClientIPFromRequest 中
// 非 IP 对端、XFF 全非法条目、X-Real-IP 非法/带空白等回退路径。
//
// 可信代理表是限流/审计取真实客户端 IP 的安全根基：解析规则漂移
// （如误信裸 IP 表达式、吞掉整段配置）会直接改变防伪造行为。

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// restoreTrustedProxies 恢复默认可信代理配置，避免本文件的专用配置
// 泄漏到后续用例。
func restoreTrustedProxies(t *testing.T) {
	t.Cleanup(func() {
		SetTrustedProxies(trustedV4CIDRs())
	})
}

// TestSetTrustedProxies_ParsingRules 可信代理配置解析规则表：
// 合法 CIDR 生效、首尾空白被裁剪、裸 IP 按单地址（/32 或 /128）生效、
// 空串/纯空白/非法 IP/非法 CIDR 条目被跳过且不影响其余条目。
func TestSetTrustedProxies_ParsingRules(t *testing.T) {
	cases := []struct {
		name      string
		config    []string
		trusted   []string
		untrusted []string
	}{
		{
			name:      "cidr with surrounding whitespace",
			config:    []string{"  172.16.0.0/12  "},
			trusted:   []string{"172.16.16.16"},
			untrusted: []string{"10.0.0.1", "192.168.1.1", "8.8.8.8"},
		},
		{
			name:      "bare IPv4 acts as /32",
			config:    []string{"8.8.8.8"},
			trusted:   []string{"8.8.8.8"},
			untrusted: []string{"8.8.8.9", "10.0.0.1"},
		},
		{
			name:      "bare IPv6 acts as /128",
			config:    []string{"::1"},
			trusted:   []string{"::1"},
			untrusted: []string{"::2", "fe80::1", "10.0.0.1"},
		},
		{
			name:      "invalid entries skipped, valid kept",
			config:    []string{"", "   ", "999.999.1.1", "not-an-ip", "10.0.0.0/33", "10.0.0.0/8"},
			trusted:   []string{"10.0.0.1"},
			untrusted: []string{"192.168.1.1", "172.16.0.1"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			restoreTrustedProxies(t)
			SetTrustedProxies(tc.config)
			for _, ipStr := range tc.trusted {
				t.Run("trusted/"+ipStr, func(t *testing.T) {
					ip := net.ParseIP(ipStr)
					require.NotNil(t, ip)
					assert.True(t, isTrustedProxy(ip),
						"%s 应按配置视为可信代理", ipStr)
				})
			}
			for _, ipStr := range tc.untrusted {
				t.Run("untrusted/"+ipStr, func(t *testing.T) {
					ip := net.ParseIP(ipStr)
					require.NotNil(t, ip)
					assert.False(t, isTrustedProxy(ip),
						"%s 不应视为可信代理", ipStr)
				})
			}
		})
	}
}

// TestSetTrustedProxies_AllInvalidYieldsEmptyTrust 全部条目非法时，
// 可信代理表为空，任何 IP 都不被信任（此后一律采用直连对端 IP）。
func TestSetTrustedProxies_AllInvalidYieldsEmptyTrust(t *testing.T) {
	restoreTrustedProxies(t)
	SetTrustedProxies([]string{"garbage", "1.2.3.4/99", "  "})
	for _, ipStr := range []string{"10.0.0.1", "127.0.0.1", "192.168.0.1", "8.8.8.8"} {
		ip := net.ParseIP(ipStr)
		require.NotNil(t, ip)
		assert.False(t, isTrustedProxy(ip),
			"全部条目非法时 %s 不应被信任", ipStr)
	}
}

// TestIPFromRemoteAddr RemoteAddr（host:port 形态）提取 host 的边界：
// 空串、无端口的裸 host、标准 host:port、非 IP 域名、无法拆分的 IPv6
// 字面量与多冒号串。
func TestIPFromRemoteAddr(t *testing.T) {
	cases := []struct {
		name       string
		remoteAddr string
		want       string
	}{
		{"empty", "", ""},
		{"bare host without port", "8.8.8.8", "8.8.8.8"},
		{"host port pair", "1.2.3.4:5678", "1.2.3.4"},
		{"domain host", "example.com:80", "example.com"},
		{"ipv6 literal without brackets", "2001:db8::1", "2001:db8::1"},
		{"too many colons", "1:2:3", "1:2:3"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ipFromRemoteAddr(tc.remoteAddr))
		})
	}
}

// TestClientIPFromRequest_NonIPRemoteAddr 直连对端不是 IP（如域名形态
// 的 RemoteAddr）时，对端解析失败，必须忽略一切转发头直接返回对端 host，
// 防伪造语义与"非可信代理"一致。
func TestClientIPFromRequest_NonIPRemoteAddr(t *testing.T) {
	SetTrustedProxies(trustedV4CIDRs())
	req := buildReq("example.com:80", map[string]string{
		"X-Forwarded-For": "203.0.113.7",
		"X-Real-IP":       "203.0.113.8",
	})
	assert.Equal(t, "example.com", ClientIPFromRequest(req),
		"非 IP 对端必须忽略转发头并返回对端 host")
}

// TestClientIPFromRequest_XFFAllUnparseable XFF 链全部条目不可解析时，
// 不得采信任何条目，回退对端 IP（可信代理本身）。
func TestClientIPFromRequest_XFFAllUnparseable(t *testing.T) {
	SetTrustedProxies(trustedV4CIDRs())
	req := buildReq("10.0.0.1:12345", map[string]string{
		"X-Forwarded-For": "not-an-ip, also-not-an-ip",
	})
	assert.Equal(t, "10.0.0.1", ClientIPFromRequest(req),
		"全非法 XFF 必须回退对端 IP")
}

// TestClientIPFromRequest_XRealIPInvalidOrPadded X-Real-IP 不可解析时
// 回退对端 IP；带首尾空白的有效 IP 被裁剪后采信。
func TestClientIPFromRequest_XRealIPInvalidOrPadded(t *testing.T) {
	SetTrustedProxies(trustedV4CIDRs())

	t.Run("unparseable x-real-ip falls back to peer", func(t *testing.T) {
		req := buildReq("10.0.0.1:12345", map[string]string{
			"X-Real-IP": "bogus-host-name",
		})
		assert.Equal(t, "10.0.0.1", ClientIPFromRequest(req))
	})

	t.Run("padded x-real-ip trimmed and trusted", func(t *testing.T) {
		req := buildReq("10.0.0.1:12345", map[string]string{
			"X-Real-IP": "  203.0.113.9  ",
		})
		assert.Equal(t, "203.0.113.9", ClientIPFromRequest(req))
	})
}
