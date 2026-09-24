// 本文件为 smtp.go 的单元测试，包含三部分：
//
//  1. buildMessage 的白盒测试：逐项断言 From/To（含多收件人）/Subject/正文/
//     MIME 头的拼装结果，并与完整字节序列做精确比对；
//  2. SendMail 的本地校验错误（host 为空、port 为 0、收件人为空、TlsMode 不认识），
//     这些分支在拨号之前就返回，不依赖任何网络；
//  3. 借助一个脚本化的明文 SMTP 假服务器（fakeSMTPServer）验证连接失败
//     （端口关闭、坏问候语）、STARTTLS 广告但不做 TLS、明文发送的完整会话
//     （含 AUTH 接受）以及各会话阶段被服务器拒绝的错误分支。
//     假服务器应答与 net/smtp 客户端的期望码一一对应（EHLO/MAIL/终止点=250、
//     RCPT=2xx 前缀、DATA=354、QUIT=221、AUTH=235/535、STARTTLS=220）。
package mailer

import (
	"bufio"
	"context"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// 脚本化 SMTP 假服务器
// ---------------------------------------------------------------------------

// fakeServerOptions 控制假服务器在各协议节点的脚本化行为。
type fakeServerOptions struct {
	badGreeting       bool   // 用 421 问候并断开：驱动 smtp.NewClient 读问候失败
	advertiseStartTLS bool   // EHLO 应答广告 STARTTLS 扩展，但从不真正做 TLS
	authReply         string // AUTH 命令的应答（如 "235 ok\r\n" 表示接受）；空串=以 502 拒绝
	rejectCommand     string // 需要以 5xx 拒绝的会话命令（AUTH/MAIL/RCPT/DATA/TERM）；空串=全部接受
}

// fakeSMTPServer 是一个只服务于单元测试的明文 SMTP 假服务器：
// 单线程 accept，按脚本应答命令，并记录 DATA 阶段收到的原始字节，
// 供测试断言 SendMail 实际发出的报文。
type fakeSMTPServer struct {
	fakeServerOptions

	ln net.Listener

	mu      sync.Mutex
	payload []byte // DATA 阶段收到的原始字节（不含终止点行）
}

// startFakeSMTPServer 在回环地址上启动假服务器，并在测试结束时随 t.Cleanup 关闭。
func startFakeSMTPServer(t *testing.T, opts fakeServerOptions) *fakeSMTPServer {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	s := &fakeSMTPServer{fakeServerOptions: opts, ln: ln}
	go s.serve()
	t.Cleanup(func() {
		s.ln.Close()
	})
	return s
}

// port 返回假服务器监听的端口。
func (s *fakeSMTPServer) port() uint32 {
	return uint32(s.ln.Addr().(*net.TCPAddr).Port)
}

// receivedPayload 返回 DATA 阶段记录到的原始字节流。
// 记录动作严格发生在服务器应答终止点之前，因此当 SendMail 返回时字节流必已完整。
func (s *fakeSMTPServer) receivedPayload() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.payload
}

// serve 单线程接受连接：每个单元测试只会建立一条 SMTP 会话。
func (s *fakeSMTPServer) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		s.handleConn(conn)
	}
}

// writeLine 向客户端写一行应答；写失败返回 false，调用方应放弃该会话。
func writeLine(conn net.Conn, s string) bool {
	_, err := conn.Write([]byte(s))
	return err == nil
}

// smtpVerb 归类一行协议文本的命令动词（DATA 阶段外）。
func smtpVerb(line string) string {
	s := strings.ToUpper(strings.TrimSpace(line))
	switch {
	case s == "QUIT":
		return "QUIT"
	case s == "STARTTLS":
		return "STARTTLS"
	case s == "DATA":
		return "DATA"
	case strings.HasPrefix(s, "AUTH"):
		return "AUTH"
	case strings.HasPrefix(s, "EHLO"):
		return "EHLO"
	case strings.HasPrefix(s, "HELO"):
		return "HELO"
	case strings.HasPrefix(s, "MAIL"):
		return "MAIL"
	case strings.HasPrefix(s, "RCPT"):
		return "RCPT"
	default:
		return "OTHER"
	}
}

// handleConn 按脚本执行一次完整会话：
// 坏问候变体直接 421 断开；STARTTLS 变体在广告扩展后对 STARTTLS 命令
// 假装就绪随即断开（客户端 TLS 握手必然失败）；常规会话逐命令应答，
// DATA 阶段逐行记录字节直到终止点 "." 为止。
func (s *fakeSMTPServer) handleConn(conn net.Conn) {
	defer conn.Close()

	if s.badGreeting {
		writeLine(conn, "421 busy\r\n")
		return
	}
	if !writeLine(conn, "220 test\r\n") {
		return
	}

	rd := bufio.NewReader(conn)
	inData := false
	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			return
		}

		if inData {
			if line == ".\r\n" {
				inData = false
				reply := "250 ok\r\n"
				if s.rejectCommand == "TERM" {
					reply = "550 rejected\r\n"
				}
				if !writeLine(conn, reply) {
					return
				}
				continue
			}
			s.mu.Lock()
			s.payload = append(s.payload, line...)
			s.mu.Unlock()
			continue
		}

		switch smtpVerb(line) {
		case "QUIT":
			writeLine(conn, "221 bye\r\n")
			return
		case "EHLO", "HELO":
			// 广告 STARTTLS 的多行 EHLO 应答（250- 续行、250 终行），
			// 客户端会据此进入 StartTLS 分支
			if s.advertiseStartTLS {
				if !writeLine(conn, "250-x\r\n250 STARTTLS\r\n") {
					return
				}
			} else if !writeLine(conn, "250 ok\r\n") {
				return
			}
		case "STARTTLS":
			// 假装可以升级随即断开：对面并不跑 TLS，客户端握手必然失败
			writeLine(conn, "220 go\r\n")
			return
		case "AUTH":
			reply := "502 unsupported\r\n"
			if s.rejectCommand == "AUTH" {
				reply = "535 auth failed\r\n"
			} else if s.authReply != "" {
				reply = s.authReply
			}
			if !writeLine(conn, reply) {
				return
			}
		case "MAIL", "RCPT":
			reply := "250 ok\r\n"
			if s.rejectCommand == smtpVerb(line) {
				reply = "550 rejected\r\n"
			}
			if !writeLine(conn, reply) {
				return
			}
		case "DATA":
			if s.rejectCommand == "DATA" {
				if !writeLine(conn, "550 rejected\r\n") {
					return
				}
				continue
			}
			if !writeLine(conn, "354 go\r\n") {
				return
			}
			inData = true
		default:
			// 其余命令（含客户端中断 AUTH 时的 "*" 行）一律 250，保证会话不悬死
			if !writeLine(conn, "250 ok\r\n") {
				return
			}
		}
	}
}

// closedPort 打开一个回环监听后立即关闭，返回一个保证无人监听的端口，
// 用于驱动拨号失败分支。
func closedPort(t *testing.T) (host string, port uint32) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()

	addr, ok := ln.Addr().(*net.TCPAddr)
	require.True(t, ok, "回环监听地址必须是 *net.TCPAddr")
	return "127.0.0.1", uint32(addr.Port)
}

// ---------------------------------------------------------------------------
// buildMessage 白盒测试
// ---------------------------------------------------------------------------

// TestBuildMessage 逐项断言 buildMessage 的产物：From、To（单/多收件人、
// 空收件人列表）、Subject、MIME 头、头部与正文的空行分隔、正文及其行尾，
// 最后与完整字节序列做精确比对，锁定任何一处拼装偏差。
func TestBuildMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		from         string
		to           []string
		subject      string
		body         string
		wantContains []string
		wantFull     string
	}{
		{
			name:    "single recipient",
			from:    "sender@example.test",
			to:      []string{"rcpt1@example.test"},
			subject: "单元测试主题",
			body:    "正文第一行\r\n正文第二行",
			wantContains: []string{
				"From: sender@example.test\r\n",
				"To: rcpt1@example.test\r\n",
				"Subject: 单元测试主题\r\n",
				"MIME-Version: 1.0\r\n",
				"Content-Type: text/plain; charset=UTF-8\r\n",
				"正文第一行\r\n正文第二行\r\n",
			},
			wantFull: "From: sender@example.test\r\n" +
				"To: rcpt1@example.test\r\n" +
				"Subject: 单元测试主题\r\n" +
				"MIME-Version: 1.0\r\n" +
				"Content-Type: text/plain; charset=UTF-8\r\n" +
				"\r\n" +
				"正文第一行\r\n正文第二行\r\n",
		},
		{
			name:    "multiple recipients joined by comma",
			from:    "noreply@example.test",
			to:      []string{"rcpt1@example.test", "rcpt2@example.test", "rcpt3@example.test"},
			subject: "multi subject",
			body:    "multi body",
			wantContains: []string{
				"From: noreply@example.test\r\n",
				"To: rcpt1@example.test, rcpt2@example.test, rcpt3@example.test\r\n",
				"Subject: multi subject\r\n",
				"MIME-Version: 1.0\r\n",
				"Content-Type: text/plain; charset=UTF-8\r\n",
				"multi body\r\n",
			},
			wantFull: "From: noreply@example.test\r\n" +
				"To: rcpt1@example.test, rcpt2@example.test, rcpt3@example.test\r\n" +
				"Subject: multi subject\r\n" +
				"MIME-Version: 1.0\r\n" +
				"Content-Type: text/plain; charset=UTF-8\r\n" +
				"\r\n" +
				"multi body\r\n",
		},
		{
			name:    "empty fields still emit every header line",
			from:    "",
			to:      []string{},
			subject: "",
			body:    "",
			wantContains: []string{
				"From: \r\n",
				"To: \r\n",
				"Subject: \r\n",
				"MIME-Version: 1.0\r\n",
				"Content-Type: text/plain; charset=UTF-8\r\n",
				"\r\n\r\n",
			},
			wantFull: "From: \r\n" +
				"To: \r\n" +
				"Subject: \r\n" +
				"MIME-Version: 1.0\r\n" +
				"Content-Type: text/plain; charset=UTF-8\r\n" +
				"\r\n" +
				"\r\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := buildMessage(tc.from, tc.to, tc.subject, tc.body)

			for _, want := range tc.wantContains {
				require.Contains(t, string(got), want, "缺少或错误拼装了某个报文元素")
			}
			require.Equal(t, tc.wantFull, string(got),
				"完整报文必须与逐字节期望值精确一致")
		})
	}
}

// ---------------------------------------------------------------------------
// SendMail 本地校验错误（不依赖网络）
// ---------------------------------------------------------------------------

// TestSendMail_InvalidConfig 逐项验证 SendMail 的本地守卫：
// host 为空、port 为 0、收件人列表为空、TlsMode 不认识，必须各自返回确定错误。
// 这些分支在拨号之前返回，因此传入的地址永远不会被访问。
func TestSendMail_InvalidConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cfg       SmtpConfig
		to        []string
		wantError string
	}{
		{
			name:      "empty host",
			cfg:       SmtpConfig{Host: "", Port: 25, TlsMode: "NONE"},
			to:        []string{"rcpt@example.test"},
			wantError: "smtp host/port is not configured",
		},
		{
			name:      "zero port",
			cfg:       SmtpConfig{Host: "127.0.0.1", Port: 0, TlsMode: "NONE"},
			to:        []string{"rcpt@example.test"},
			wantError: "smtp host/port is not configured",
		},
		{
			name:      "empty recipient list",
			cfg:       SmtpConfig{Host: "127.0.0.1", Port: 25, TlsMode: "NONE"},
			to:        nil,
			wantError: "recipient is empty",
		},
		{
			name:      "unknown tls mode",
			cfg:       SmtpConfig{Host: "127.0.0.1", Port: 25, TlsMode: "QUANTUM"},
			to:        []string{"rcpt@example.test"},
			wantError: "unsupported tls mode: QUANTUM",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := SendMail(context.Background(), tc.cfg, tc.to, "subject", "body")
			require.EqualError(t, err, tc.wantError)
		})
	}
}

// TestIsSupportedTlsMode 加密方式自检的判定集，必须与 SendMail 的分拨号 switch 一致。
// 尤其 SSL_TLS：演示数据/手工改库留下的值，判错会让配置问题伪装成投递失败。
func TestIsSupportedTlsMode(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{"", "NONE", "none", "START_TLS", "SSL", "ssl"} {
		require.True(t, IsSupportedTlsMode(mode), "%q 应被识别", mode)
	}
	for _, mode := range []string{"SSL_TLS", "TLS", "STARTTLS", "QUANTUM"} {
		require.False(t, IsSupportedTlsMode(mode), "%q 不该被识别", mode)
	}
}

// ---------------------------------------------------------------------------
// 连接失败分支
// ---------------------------------------------------------------------------

// TestSendMail_ConnectFailure_ClosedPort 验证对已关闭端口的拨号失败：
// NONE/空/START_TLS（同走明文拨号）与 SSL（走 TLS 拨号）四种模式
// 都必须包装为 "connect smtp server failed" 返回。
func TestSendMail_ConnectFailure_ClosedPort(t *testing.T) {
	t.Parallel()

	host, port := closedPort(t)

	modes := []struct {
		name string
		mode string
	}{
		{name: "NONE mode", mode: "NONE"},
		{name: "empty mode defaults to plaintext dial", mode: ""},
		{name: "START_TLS mode", mode: "START_TLS"},
		{name: "SSL mode", mode: "SSL"},
	}

	for _, m := range modes {
		t.Run(m.name, func(t *testing.T) {
			t.Parallel()

			cfg := SmtpConfig{Host: host, Port: port, From: "sender@example.test", TlsMode: m.mode}
			err := SendMail(context.Background(), cfg, []string{"rcpt@example.test"}, "subject", "body")
			require.ErrorContains(t, err, "connect smtp server failed")
		})
	}
}

// TestSendMail_ConnectFailure_BadGreeting 验证服务器以 421 问候时，
// smtp.NewClient 读取 220 失败，同样包装为 "connect smtp server failed"。
func TestSendMail_ConnectFailure_BadGreeting(t *testing.T) {
	t.Parallel()

	srv := startFakeSMTPServer(t, fakeServerOptions{badGreeting: true})
	cfg := SmtpConfig{Host: "127.0.0.1", Port: srv.port(), From: "sender@example.test", TlsMode: "NONE"}

	err := SendMail(context.Background(), cfg, []string{"rcpt@example.test"}, "subject", "body")
	require.ErrorContains(t, err, "connect smtp server failed")
}

// TestSendMail_StartTLSAdvertisedButPlaintext 验证服务器广告 STARTTLS
// 但不做真正 TLS 握手时，客户端 StartTLS 必然失败，
// SendMail 报出内层 "STARTTLS failed" 与外层 "connect smtp server failed" 包装。
func TestSendMail_StartTLSAdvertisedButPlaintext(t *testing.T) {
	t.Parallel()

	srv := startFakeSMTPServer(t, fakeServerOptions{advertiseStartTLS: true})
	cfg := SmtpConfig{Host: "127.0.0.1", Port: srv.port(), From: "sender@example.test", TlsMode: "NONE"}

	err := SendMail(context.Background(), cfg, []string{"rcpt@example.test"}, "subject", "body")
	require.ErrorContains(t, err, "STARTTLS failed")
	require.ErrorContains(t, err, "connect smtp server failed")
}

// ---------------------------------------------------------------------------
// 明文会话 happy path 与 AUTH
// ---------------------------------------------------------------------------

// TestSendMail_PlaintextSessionSucceeds 验证 TlsMode=NONE（服务器不广告 STARTTLS，
// 全程明文）的完整发送流程：SendMail 返回 nil，且假服务器在 DATA 阶段
// 收到的字节流与 buildMessage 的产物逐字节一致（From/To/Subject、
// MIME 头与正文全部核对）。两个用例分别覆盖显式 From 与
// From/Username 皆空时回退为空 From 的分支，收件人为多地址。
func TestSendMail_PlaintextSessionSucceeds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		from     string
		fromLine string
	}{
		{
			name:     "explicit from",
			from:     "sender@example.test",
			fromLine: "From: sender@example.test\r\n",
		},
		{
			name:     "empty from and username fallback",
			from:     "",
			fromLine: "From: \r\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := startFakeSMTPServer(t, fakeServerOptions{})
			cfg := SmtpConfig{
				Host:    "127.0.0.1",
				Port:    srv.port(),
				From:    tc.from,
				TlsMode: "NONE",
			}
			to := []string{"rcpt1@example.test", "rcpt2@example.test"}

			err := SendMail(context.Background(), cfg, to, "plain subject", "plain body line 1\r\nplain body line 2")
			require.NoError(t, err, "明文会话必须完整走通并返回 nil")

			want := tc.fromLine +
				"To: rcpt1@example.test, rcpt2@example.test\r\n" +
				"Subject: plain subject\r\n" +
				"MIME-Version: 1.0\r\n" +
				"Content-Type: text/plain; charset=UTF-8\r\n" +
				"\r\n" +
				"plain body line 1\r\nplain body line 2\r\n"
			require.Equal(t, want, string(srv.receivedPayload()),
				"服务器收到的 DATA 字节流必须与 buildMessage 的产物逐字节一致")
		})
	}
}

// TestSendMail_AuthAcceptedThenSendSucceeds 验证 AUTH 分支：
// 服务器接受 PLAIN 认证（235）后 SendMail 继续完成整个发送流程并返回 nil。
// 明文连接之所以能用 PLAIN AUTH，是因为目标地址是回环地址 127.0.0.1
// （标准库 smtp.PlainAuth 对 localhost 的豁免规则）。
func TestSendMail_AuthAcceptedThenSendSucceeds(t *testing.T) {
	t.Parallel()

	srv := startFakeSMTPServer(t, fakeServerOptions{authReply: "235 ok\r\n"})
	cfg := SmtpConfig{
		Host:     "127.0.0.1",
		Port:     srv.port(),
		Username: "user@example.test",
		Password: "auth-token",
		From:     "sender@example.test",
		TlsMode:  "NONE",
	}

	err := SendMail(context.Background(), cfg, []string{"rcpt1@example.test"}, "auth subject", "auth body")
	require.NoError(t, err, "AUTH 被接受后必须完成发送并返回 nil")

	want := "From: sender@example.test\r\n" +
		"To: rcpt1@example.test\r\n" +
		"Subject: auth subject\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		"auth body\r\n"
	require.Equal(t, want, string(srv.receivedPayload()))
}

// ---------------------------------------------------------------------------
// 会话阶段被服务器拒绝的分支
// ---------------------------------------------------------------------------

// TestSendMail_SessionCommandFailures 表驱动验证服务器在各会话阶段
// 以 5xx 拒绝时，SendMail 必须返回对应的包装错误。
// 所有拒绝都是确定性的状态码应答，不依赖任何时序。
func TestSendMail_SessionCommandFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		rejectAt     string
		useAuth      bool
		wantContains []string
	}{
		{
			name:         "auth rejected",
			rejectAt:     "AUTH",
			useAuth:      true,
			wantContains: []string{"smtp auth failed"},
		},
		{
			name:         "mail command rejected",
			rejectAt:     "MAIL",
			useAuth:      false,
			wantContains: []string{"smtp MAIL failed"},
		},
		{
			name:         "rcpt command rejected",
			rejectAt:     "RCPT",
			useAuth:      false,
			wantContains: []string{"smtp RCPT failed for"},
		},
		{
			name:         "data command rejected",
			rejectAt:     "DATA",
			useAuth:      false,
			wantContains: []string{"smtp DATA failed"},
		},
		{
			name:         "message terminator rejected",
			rejectAt:     "TERM",
			useAuth:      false,
			wantContains: []string{"close message failed"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := startFakeSMTPServer(t, fakeServerOptions{rejectCommand: tc.rejectAt})
			cfg := SmtpConfig{
				Host:    "127.0.0.1",
				Port:    srv.port(),
				From:    "sender@example.test",
				TlsMode: "NONE",
			}
			if tc.useAuth {
				cfg.Username = "user@example.test"
				cfg.Password = "pw"
			}

			err := SendMail(context.Background(), cfg, []string{"rcpt1@example.test"}, "subject", "body")
			for _, want := range tc.wantContains {
				require.ErrorContains(t, err, want)
			}
		})
	}
}
