package netutil

// 本文件覆盖 client_ip.go 的三个 context 提取函数：
// ClientIPFromContext / HeaderFromContext / CookieFromContext。
//
// 通过实现 kratos transport 接口的桩（Transporter / khttp.Transporter）
// 构造带 HTTP 传输层的 server context，覆盖：非 server context、
// 非 HTTP 传输（gRPC 形态）、HTTP 传输但 request 为 nil、
// 以及正常 HTTP request 的完整提取路径。
//
// 这些函数是 service 层读取客户端 IP / 请求头 / HttpOnly cookie
// （如刷新令牌接口读 refresh_token cookie）的唯一入口，
// 对非 HTTP 形态必须返回零值而非 panic。

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-kratos/kratos/v2/transport"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// stubHeader 实现 transport.Header 的空实现。
type stubHeader struct{}

func (stubHeader) Get(string) string      { return "" }
func (stubHeader) Set(string, string)     {}
func (stubHeader) Add(string, string)     {}
func (stubHeader) Keys() []string         { return nil }
func (stubHeader) Values(string) []string { return nil }

// stubNonHTTPTransporter 以 gRPC 形态实现 transport.Transporter，
// 用于覆盖"传输层非 HTTP"的分支。
type stubNonHTTPTransporter struct{}

func (stubNonHTTPTransporter) Kind() transport.Kind            { return transport.KindGRPC }
func (stubNonHTTPTransporter) Endpoint() string                { return "" }
func (stubNonHTTPTransporter) Operation() string               { return "" }
func (stubNonHTTPTransporter) RequestHeader() transport.Header { return stubHeader{} }
func (stubNonHTTPTransporter) ReplyHeader() transport.Header   { return stubHeader{} }

// stubHTTPTransporter 实现 khttp.Transporter（含 Request/PathTemplate），
// 用于覆盖 HTTP 传输层的分支；req 可为 nil 以覆盖空 request 分支。
type stubHTTPTransporter struct {
	req *http.Request
}

func (*stubHTTPTransporter) Kind() transport.Kind            { return transport.KindHTTP }
func (*stubHTTPTransporter) Endpoint() string                { return "" }
func (*stubHTTPTransporter) Operation() string               { return "" }
func (*stubHTTPTransporter) RequestHeader() transport.Header { return stubHeader{} }
func (*stubHTTPTransporter) ReplyHeader() transport.Header   { return stubHeader{} }
func (s *stubHTTPTransporter) Request() *http.Request        { return s.req }
func (*stubHTTPTransporter) PathTemplate() string            { return "" }

// 编译期断言：桩确实实现了目标接口（否则测试失真）。
var _ khttp.Transporter = (*stubHTTPTransporter)(nil)
var _ transport.Transporter = stubNonHTTPTransporter{}

// TestClientIPFromContext_NonHTTPPaths 非 server context、非 HTTP 传输、
// 空 request 三种形态必须返回空串。
func TestClientIPFromContext_NonHTTPPaths(t *testing.T) {
	t.Run("plain context", func(t *testing.T) {
		assert.Empty(t, ClientIPFromContext(context.Background()))
	})
	t.Run("non-http transport", func(t *testing.T) {
		ctx := transport.NewServerContext(context.Background(), stubNonHTTPTransporter{})
		assert.Empty(t, ClientIPFromContext(ctx))
	})
	t.Run("nil request", func(t *testing.T) {
		ctx := transport.NewServerContext(context.Background(), &stubHTTPTransporter{req: nil})
		assert.Empty(t, ClientIPFromContext(ctx))
	})
}

// TestClientIPFromContext_HTTPRequest 正常 HTTP 传输时等价于
// ClientIPFromRequest：此处对端为非可信代理，返回对端 IP 且忽略 XFF。
func TestClientIPFromContext_HTTPRequest(t *testing.T) {
	SetTrustedProxies(trustedV4CIDRs())
	req := buildReq("8.8.8.8:12345", map[string]string{
		"X-Forwarded-For": "203.0.113.7",
	})
	ctx := transport.NewServerContext(context.Background(), &stubHTTPTransporter{req: req})
	assert.Equal(t, "8.8.8.8", ClientIPFromContext(ctx),
		"非可信代理对端必须忽略 XFF 并返回对端 IP")
}

// TestHeaderFromContext_HeaderExtraction 请求头提取的三种零值形态
// 与正常提取形态。
func TestHeaderFromContext_HeaderExtraction(t *testing.T) {
	t.Run("plain context", func(t *testing.T) {
		assert.Nil(t, HeaderFromContext(context.Background()))
	})
	t.Run("non-http transport", func(t *testing.T) {
		ctx := transport.NewServerContext(context.Background(), stubNonHTTPTransporter{})
		assert.Nil(t, HeaderFromContext(ctx))
	})
	t.Run("nil request", func(t *testing.T) {
		ctx := transport.NewServerContext(context.Background(), &stubHTTPTransporter{req: nil})
		assert.Nil(t, HeaderFromContext(ctx))
	})
	t.Run("http request header returned", func(t *testing.T) {
		req := buildReq("8.8.8.8:12345", map[string]string{"X-Marker": "present"})
		ctx := transport.NewServerContext(context.Background(), &stubHTTPTransporter{req: req})
		header := HeaderFromContext(ctx)
		require.NotNil(t, header, "HTTP 传输必须返回请求头")
		assert.Equal(t, "present", header.Get("X-Marker"))
	})
}

// TestCookieFromContext_CookieExtraction cookie 提取：存在时返回值，
// 不存在或非 HTTP 形态返回空串。刷新令牌接口依赖该函数读取
// HttpOnly refresh_token cookie，空串路径不得误报为有值。
func TestCookieFromContext_CookieExtraction(t *testing.T) {
	t.Run("plain context", func(t *testing.T) {
		assert.Empty(t, CookieFromContext(context.Background(), "refresh_token"))
	})
	t.Run("non-http transport", func(t *testing.T) {
		ctx := transport.NewServerContext(context.Background(), stubNonHTTPTransporter{})
		assert.Empty(t, CookieFromContext(ctx, "refresh_token"))
	})
	t.Run("nil request", func(t *testing.T) {
		ctx := transport.NewServerContext(context.Background(), &stubHTTPTransporter{req: nil})
		assert.Empty(t, CookieFromContext(ctx, "refresh_token"))
	})
	t.Run("present cookie", func(t *testing.T) {
		req := buildReq("8.8.8.8:12345", nil)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "rt-value"})
		ctx := transport.NewServerContext(context.Background(), &stubHTTPTransporter{req: req})
		assert.Equal(t, "rt-value", CookieFromContext(ctx, "refresh_token"))
	})
	t.Run("absent cookie", func(t *testing.T) {
		req := buildReq("8.8.8.8:12345", nil)
		ctx := transport.NewServerContext(context.Background(), &stubHTTPTransporter{req: req})
		assert.Empty(t, CookieFromContext(ctx, "refresh_token"))
	})
}
