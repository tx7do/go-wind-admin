package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// HTTP egress 安全默认值。
// 白名单为空 = 全部出站被拒（fail-closed）；管理员经
// SCRIPT_HTTP_ALLOWED_DOMAINS（逗号分隔，如 "hooks.slack.com,oapi.dingtalk.com"）放行域名。
const (
	defaultHTTPTimeout  = 10 * time.Second
	maxHTTPResponseBody = 1 << 20 // 1MB
	maxHTTPRequestBody  = 1 << 20 // 1MB
	defaultUserAgent    = "go-wind-admin-script/1.0"
	maxRedirects        = 3
)

// HTTPOptions 约束 http 模块的出站行为（跨语言共享，Lua/JS 同一套护栏）。
type HTTPOptions struct {
	// AllowedDomains 域名白名单（精确匹配子域需显式列出；空 = 全部拒绝）。
	AllowedDomains []string
	// Timeout 单请求超时（默认 10s，上限 30s）。
	Timeout time.Duration
	// MaxResponseBody 响应体上限字节（默认/上限 1MB）。
	MaxResponseBody int64
	// DenyLoopback 环回地址硬禁开关（默认 true）。
	// 即使白名单显式列出 localhost/127.0.0.1 也拒绝，防 SSRF 基线；
	// 仅测试环境（httptest 监听 127.0.0.1）显式置 false 放行。
	DenyLoopback bool
}

// normalizeHTTPOptions 填充默认值并收敛越界配置。
func normalizeHTTPOptions(o HTTPOptions) HTTPOptions {
	if o.Timeout <= 0 || o.Timeout > 30*time.Second {
		o.Timeout = defaultHTTPTimeout
	}
	if o.MaxResponseBody <= 0 || o.MaxResponseBody > maxHTTPResponseBody {
		o.MaxResponseBody = maxHTTPResponseBody
	}
	// trim 空项并小写化
	domains := make([]string, 0, len(o.AllowedDomains))
	for _, d := range o.AllowedDomains {
		d = strings.ToLower(strings.TrimSpace(d))
		if d != "" {
			domains = append(domains, d)
		}
	}
	o.AllowedDomains = domains
	return o
}

// checkAllowedURL 校验 URL：仅 http(s)、host 必须精确命中白名单
// （支持 "*.example.com" 通配一级子域；裸 "example.com" 不含子域）。
func checkAllowedURL(rawURL string, allowed []string, denyLoopback bool) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("only http/https schemes are allowed")
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return errors.New("url host is empty")
	}
	// 环回/元数据地址硬禁（防 SSRF 基线）；DenyLoopback=false 仅为测试逃生门
	if denyLoopback {
		for _, banned := range []string{"localhost", "127.0.0.1", "0.0.0.0", "::1", "169.254.169.254"} {
			if host == banned {
				return fmt.Errorf("host %q is not allowed", host)
			}
		}
	}

	for _, pattern := range allowed {
		if pattern == host {
			return nil
		}
		if strings.HasPrefix(pattern, "*.") {
			suffix := strings.TrimPrefix(pattern, "*")
			// 仅匹配一级子域：a.example.com 命中 *.example.com；a.b.example.com 不命中
			if strings.HasSuffix(host, suffix) && !strings.Contains(strings.TrimSuffix(host, suffix), ".") {
				return nil
			}
		}
	}
	return fmt.Errorf("host %q is not in the allowed domains list", host)
}

// httpRequest 执行受限的出站请求（核心实现，Lua/JS 适配层共用）。
// options: method(GET默认)、headers(map)、body(string)。
// 返回: {status, body, headers(map, 全小写键)}。
func httpRequest(ctx context.Context, client *http.Client, rawURL, method string, headers map[string]string, body string, maxBody int64) (map[string]any, error) {
	var bodyReader io.Reader
	if body != "" {
		if int64(len(body)) > maxHTTPRequestBody {
			return nil, errors.New("request body exceeds 1MB limit")
		}
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, rawURL, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	for k, v := range headers {
		// 不允许脚本伪装/覆盖防护头
		lk := strings.ToLower(k)
		if lk == "host" || lk == "user-agent" || lk == "content-length" {
			continue
		}
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return nil, err
	}
	if int64(len(respBody)) > maxBody {
		return nil, errors.New("response body exceeds size limit")
	}

	respHeaders := make(map[string]string, len(resp.Header))
	for k, vs := range resp.Header {
		if len(vs) > 0 {
			respHeaders[strings.ToLower(k)] = vs[0]
		}
	}

	return map[string]any{
		"status":  resp.StatusCode,
		"body":    string(respBody),
		"headers": respHeaders,
	}, nil
}

// buildHTTPClient 构造带重定向限制的出站客户端。
// 重定向每次跳转都会重新校验白名单（CheckRedirect 内），防止白名单域名 302 到内网。
func buildHTTPClient(opts HTTPOptions) *http.Client {
	return &http.Client{
		Timeout: opts.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return errors.New("too many redirects")
			}
			return checkAllowedURL(req.URL.String(), opts.AllowedDomains, opts.DenyLoopback)
		},
	}
}

// RegisterHTTP 注册 http 模块（Lua，preload 风格）。
func RegisterHTTP(L *lua.LState, opts HTTPOptions) {
	L.PreloadModule("kratos_http", LoaderHTTP(opts))
}

// LoaderHTTP 返回 http 模块（kratos_http）的 loader，供 go-scripts 引擎 RegisterModule 使用。
// 白名单为空时模块函数仍在，但所有请求都会被拒（fail-closed，报错信息指明白名单）。
func LoaderHTTP(opts HTTPOptions) lua.LGFunction {
	normalized := normalizeHTTPOptions(opts)
	client := buildHTTPClient(normalized)

	return func(L *lua.LState) int {
		httpModule := L.NewTable()

		// http.request(url, [options]) → {status, body, headers}
		// options: {method="POST", headers={k=v}, body="..."}
		httpModule.RawSetString("request", L.NewFunction(func(L *lua.LState) int {
			rawURL := L.CheckString(1)
			if err := checkAllowedURL(rawURL, normalized.AllowedDomains, normalized.DenyLoopback); err != nil {
				L.RaiseError("http blocked: %v", err)
				return 0
			}

			method := "GET"
			headerMap := map[string]string{}
			body := ""
			if options := L.OptTable(2, nil); options != nil {
				if m := options.RawGetString("method"); m != lua.LNil {
					method = strings.ToUpper(m.String())
				}
				if h, ok := options.RawGetString("headers").(*lua.LTable); ok {
					h.ForEach(func(k, v lua.LValue) {
						headerMap[k.String()] = v.String()
					})
				}
				if b := options.RawGetString("body"); b != lua.LNil {
					body = b.String()
				}
			}

			result, err := httpRequest(context.Background(), client, rawURL, method, headerMap, body, normalized.MaxResponseBody)
			if err != nil {
				L.RaiseError("http request failed: %v", err)
				return 0
			}
			L.Push(luaTableFromMap(L, result))
			return 1
		}))

		// http.get(url) → {status, body, headers}
		httpModule.RawSetString("get", L.NewFunction(func(L *lua.LState) int {
			rawURL := L.CheckString(1)
			if err := checkAllowedURL(rawURL, normalized.AllowedDomains, normalized.DenyLoopback); err != nil {
				L.RaiseError("http blocked: %v", err)
				return 0
			}
			result, err := httpRequest(context.Background(), client, rawURL, "GET", nil, "", normalized.MaxResponseBody)
			if err != nil {
				L.RaiseError("http request failed: %v", err)
				return 0
			}
			L.Push(luaTableFromMap(L, result))
			return 1
		}))

		// http.post(url, body, [contentType]) → {status, body, headers}
		httpModule.RawSetString("post", L.NewFunction(func(L *lua.LState) int {
			rawURL := L.CheckString(1)
			body := L.CheckString(2)
			contentType := L.OptString(3, "application/json")
			if err := checkAllowedURL(rawURL, normalized.AllowedDomains, normalized.DenyLoopback); err != nil {
				L.RaiseError("http blocked: %v", err)
				return 0
			}
			result, err := httpRequest(context.Background(), client, rawURL, "POST",
				map[string]string{"Content-Type": contentType}, body, normalized.MaxResponseBody)
			if err != nil {
				L.RaiseError("http request failed: %v", err)
				return 0
			}
			L.Push(luaTableFromMap(L, result))
			return 1
		}))

		L.Push(httpModule)
		return 1
	}
}

// httpError 构造脚本侧 HTTP 错误。
func httpError(msg string) error { return errors.New(msg) }

// luaTableFromMap 把 map[string]any 转为 Lua 表（嵌套 map[string]string 亦处理）。
func luaTableFromMap(L *lua.LState, m map[string]any) *lua.LTable {
	t := L.NewTable()
	for k, v := range m {
		switch val := v.(type) {
		case string:
			t.RawSetString(k, lua.LString(val))
		case int:
			t.RawSetString(k, lua.LNumber(val))
		case int64:
			t.RawSetString(k, lua.LNumber(val))
		case map[string]string:
			t.RawSetString(k, luaTableFromStrings(L, val))
		default:
			t.RawSetString(k, lua.LString(fmt.Sprintf("%v", val)))
		}
	}
	return t
}

func luaTableFromStrings(L *lua.LState, m map[string]string) *lua.LTable {
	t := L.NewTable()
	for k, v := range m {
		t.RawSetString(k, lua.LString(v))
	}
	return t
}
