package netutil

// 本文件覆盖 ssrf.go 中 LookupAndCheckHost 的解析分支：
//  1. localhost（本机 hosts/OS 解析，无外网依赖）解析到回环必须被拦；
//  2. 带端口与方括号的 IPv6 字面量必须先剥壳再判禁；
//  3. 域名走 DNS 解析的两条分支（解析成功返回 IP 列表 / 解析出禁投
//     网段地址报错）——通过本地 UDP 假 DNS 服务器 + 临时替换
//     net.DefaultResolver 离线覆盖，全程不发外网请求。
//
// 依赖 golang.org/x/net/dns/dnsmessage（仓库既有依赖，无新增三方库）。

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/net/dns/dnsmessage"
)

// TestLookupAndCheckHost_LocalhostBlocked localhost 由本机解析为回环地址，
// 属于 SSRF 高危目标，必须被拦（无外网请求）。
func TestLookupAndCheckHost_LocalhostBlocked(t *testing.T) {
	_, err := LookupAndCheckHost(context.Background(), "localhost")
	assert.Error(t, err, "localhost 解析到回环地址，必须被拦截")
}

// TestLookupAndCheckHost_BracketedIPv6LiteralWithPort 带端口与方括号的
// IPv6 字面量：SplitHostPort 剥壳后得到 ::1，必须按禁投地址拦截。
func TestLookupAndCheckHost_BracketedIPv6LiteralWithPort(t *testing.T) {
	_, err := LookupAndCheckHost(context.Background(), "[::1]:8080")
	assert.Error(t, err, "回环 IPv6 字面量（带端口）必须被拦截")
}

// TestLookupAndCheckHost_LoopbackLiteral 裸回环字面量同样必须被拦。
func TestLookupAndCheckHost_LoopbackLiteral(t *testing.T) {
	_, err := LookupAndCheckHost(context.Background(), "127.0.0.1")
	assert.Error(t, err)
}

// fakeDNSServer 本地 UDP 假 DNS 服务器：对 A 查询回指定记录（按 FQDN
// 键匹配，含结尾点），对 AAAA 及未登记名回空答案（NOERROR/NODATA），
// 其余类型一律空答案。仅供测试内离线驱动域名解析分支。
type fakeDNSServer struct {
	conn     *net.UDPConn
	aRecords map[string]string
}

// startFakeDNSServer 启动假 DNS 服务器并注册清理（关闭连接、停 goroutine）。
func startFakeDNSServer(t *testing.T, aRecords map[string]string) *fakeDNSServer {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	udp, ok := pc.(*net.UDPConn)
	require.True(t, ok, "ListenPacket(udp) 应返回 *net.UDPConn")
	s := &fakeDNSServer{conn: udp, aRecords: aRecords}
	go s.serve()
	t.Cleanup(func() {
		_ = pc.Close()
	})
	return s
}

// serve 循环读取查询数据报并回写构造好的响应。
func (s *fakeDNSServer) serve() {
	for {
		buf := make([]byte, 512)
		n, addr, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		resp := s.buildResponse(buf[:n])
		if resp == nil {
			continue
		}
		_, _ = s.conn.WriteToUDP(resp, addr)
	}
}

// buildResponse 构造 DNS 响应：回显查询 ID 与问题段；仅对登记过的
// A 查询回一条 A 记录，其余回零答案。构造失败返回 nil（丢弃该查询）。
func (s *fakeDNSServer) buildResponse(query []byte) []byte {
	var p dnsmessage.Parser
	hdr, err := p.Start(query)
	if err != nil {
		return nil
	}
	q, err := p.Question()
	if err != nil {
		return nil
	}

	var answerIP [4]byte
	hasAnswer := false
	if q.Type == dnsmessage.TypeA && q.Class == dnsmessage.ClassINET {
		if ipStr, ok := s.aRecords[q.Name.String()]; ok {
			if ip := net.ParseIP(ipStr).To4(); ip != nil {
				copy(answerIP[:], ip)
				hasAnswer = true
			}
		}
	}

	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{
		ID:       hdr.ID,
		Response: true,
		RCode:    dnsmessage.RCodeSuccess,
	})
	if err := b.StartQuestions(); err != nil {
		return nil
	}
	if err := b.Question(q); err != nil {
		return nil
	}
	if err := b.StartAnswers(); err != nil {
		return nil
	}
	if hasAnswer {
		rh := dnsmessage.ResourceHeader{
			Name:  q.Name,
			Type:  dnsmessage.TypeA,
			Class: dnsmessage.ClassINET,
			TTL:   60,
		}
		if err := b.AResource(rh, dnsmessage.AResource{A: answerIP}); err != nil {
			return nil
		}
	}
	resp, err := b.Finish()
	if err != nil {
		return nil
	}
	return resp
}

// installFakeResolver 用假 DNS 服务器替换 net.DefaultResolver，
// 并在用例结束时还原。替换后的解析器仅将查询定向到本地假服务器，
// 不会发起任何外网请求。
func installFakeResolver(t *testing.T, server *fakeDNSServer) {
	t.Helper()
	serverAddr := server.conn.LocalAddr().String()
	old := net.DefaultResolver
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, network, serverAddr)
		},
	}
	t.Cleanup(func() {
		net.DefaultResolver = old
	})
}

// TestLookupAndCheckHost_ResolvesToBlockedAddress 域名解析出私有网段地址
// 时必须报错（SSRF 借域名绕过 IP 直判的典型路径，如内网域名指向
// 10.0.0.5）。全程离线。
func TestLookupAndCheckHost_ResolvesToBlockedAddress(t *testing.T) {
	server := startFakeDNSServer(t, map[string]string{
		"internal.example.": "10.0.0.5",
	})
	installFakeResolver(t, server)

	_, err := LookupAndCheckHost(context.Background(), "internal.example")
	require.Error(t, err, "解析到 10.0.0.5（私有 A 段）必须被拦截")
}

// TestLookupAndCheckHost_PublicHostResolutions 域名解析为公网地址时的
// 通过路径：返回解析到的 IP 列表（带端口形态应先剥端口再解析）。
func TestLookupAndCheckHost_PublicHostResolutions(t *testing.T) {
	server := startFakeDNSServer(t, map[string]string{
		"public.example.": "93.184.216.34",
	})
	installFakeResolver(t, server)

	t.Run("bare host", func(t *testing.T) {
		ips, err := LookupAndCheckHost(context.Background(), "public.example")
		require.NoError(t, err)
		require.Len(t, ips, 1)
		assert.Equal(t, "93.184.216.34", ips[0].String(),
			"应返回假 DNS 登记的公网 IP")
	})
	t.Run("host with port", func(t *testing.T) {
		ips, err := LookupAndCheckHost(context.Background(), "public.example:8080")
		require.NoError(t, err)
		require.Len(t, ips, 1)
		assert.Equal(t, "93.184.216.34", ips[0].String(),
			"端口应被剥离后按主机名解析")
	})
}

// TestLookupAndCheckHost_UnmappedHostErrors 假 DNS 对未登记名回空答案，
// 解析侧应得到错误而非空列表通过（空结果集走错误路径）。
func TestLookupAndCheckHost_UnmappedHostErrors(t *testing.T) {
	server := startFakeDNSServer(t, map[string]string{})
	installFakeResolver(t, server)

	_, err := LookupAndCheckHost(context.Background(), "unmapped.example")
	assert.Error(t, err, "无解析结果的域名必须报错")
}
