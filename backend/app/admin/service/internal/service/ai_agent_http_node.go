package service

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Agent 编排 http 节点（docs/ai_agent_orchestration.md §2.1 节点集扩展，M2+）：
// 渲染 URL/headers/body → SSRF 校验（与 RenderPdf 同款私网/回环/元数据防线）→
// 出站请求 → status/body/contentType 写入节点上下文。
//
// data 键：url(必填,模板)、method(默认 GET，白名单 GET/POST/PUT/DELETE/PATCH/HEAD)、
// headers(map[string]string，键值均可模板)、body(模板，POST/PUT/PATCH 有意义)、
// timeoutSeconds(默认 15，钳 [1,60])。

const (
	// agentHTTPMaxBody 响应体读取上限（防巨型响应撑爆内存与上下文）。
	agentHTTPMaxBody = 64 << 10

	// agentHTTPMaxContextBody 写入节点上下文的响应体上限——上下文会进 LLM 提示词
	// 模板，超大 body 对下游模型是负担；超限截断并标注。
	agentHTTPMaxContextBody = 8 << 10

	agentHTTPDefaultTimeout = 15 * time.Second
)

// agentHTTPAllowedMethods 出站方法白名单：编排图的 HTTP 节点是"取数/回调"语义，
// 不放行 CONNECT/TRACE/OPTIONS 等不用于此语义的方法。
var agentHTTPAllowedMethods = map[string]bool{
	http.MethodGet: true, http.MethodPost: true, http.MethodPut: true,
	http.MethodDelete: true, http.MethodPatch: true, http.MethodHead: true,
}

// validateAgentHTTPNode Run 前静态校验（模板在执行期才渲染，这里只校验字面结构）。
func validateAgentHTTPNode(n *agentGraphNode) error {
	if strings.TrimSpace(agentDataString(n.Data, "url")) == "" {
		return fmt.Errorf("http node %q missing url", n.ID)
	}
	method := agentHTTPMethod(n)
	if !agentHTTPAllowedMethods[method] {
		return fmt.Errorf("http node %q has disallowed method %q", n.ID, method)
	}
	if _, err := agentHTTPTimeout(n); err != nil {
		return fmt.Errorf("http node %q: %w", n.ID, err)
	}
	return nil
}

func agentHTTPMethod(n *agentGraphNode) string {
	m := strings.ToUpper(strings.TrimSpace(agentDataString(n.Data, "method")))
	if m == "" {
		return http.MethodGet
	}
	return m
}

func agentHTTPTimeout(n *agentGraphNode) (time.Duration, error) {
	raw := strings.TrimSpace(agentDataString(n.Data, "timeoutSeconds"))
	if raw == "" {
		return agentHTTPDefaultTimeout, nil
	}
	secs, err := strconv.Atoi(raw)
	if err != nil || secs < 1 || secs > 60 {
		return 0, fmt.Errorf("timeoutSeconds must be an integer in [1,60], got %q", raw)
	}
	return time.Duration(secs) * time.Second, nil
}

// runAgentHTTPNode 渲染 → 校验 → 出站 → 上下文写入。
// SSRF 防线与 RenderPdf 同源（isBlockedPrintHost/isBlockedPrintIP）：scheme 白名单 +
// 主机字面量/后缀判定 + 解析后逐 IP 复核。与那条线同样存在解析与请求两次解析之间的
// TOCTOU 窗口（DNS 重绑定），强度一致、接受同一权衡。
func runAgentHTTPNode(ctx context.Context, node *agentGraphNode, vars map[string]map[string]string) (status, body, contentType string, err error) {
	rawURL, rerr := renderAgentTemplate(agentDataString(node.Data, "url"), vars)
	if rerr != nil {
		return "", "", "", rerr
	}
	rawURL = strings.TrimSpace(rawURL)
	if err = validateAgentOutboundURL(rawURL); err != nil {
		return "", "", "", err
	}

	method := agentHTTPMethod(node)
	timeout, terr := agentHTTPTimeout(node)
	if terr != nil {
		return "", "", "", terr
	}

	var bodyReader io.Reader
	payload := ""
	if method != http.MethodGet && method != http.MethodHead {
		payload, rerr = renderAgentTemplate(agentDataString(node.Data, "body"), vars)
		if rerr != nil {
			return "", "", "", rerr
		}
		if payload != "" {
			bodyReader = strings.NewReader(payload)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, rawURL, bodyReader)
	if err != nil {
		return "", "", "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "go-wind-agent-node/1.0")
	if payload != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	// headers：键值都可模板；键值直 set（Host/Authorization 等受限头交给调用者自查，
	// net/http 会拦 Host 的显式 set）。画布存 [{key,value}] 数组，DSL 手写可能是 map——两种形态都收。
	switch hs := node.Data["headers"].(type) {
	case map[string]any:
		for k, v := range hs {
			val, rerr := renderAgentTemplate(fmt.Sprintf("%v", v), vars)
			if rerr != nil {
				return "", "", "", rerr
			}
			req.Header.Set(strings.TrimSpace(k), val)
		}
	case []any:
		for _, item := range hs {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			k, _ := m["key"].(string)
			if strings.TrimSpace(k) == "" {
				continue
			}
			val, rerr := renderAgentTemplate(agentDataString(m, "value"), vars)
			if rerr != nil {
				return "", "", "", rerr
			}
			req.Header.Set(strings.TrimSpace(k), val)
		}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 自定义 Transport 复用默认连接池参数，仅显式关闭自动解压差异与代理继承之外的改动。
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req.WithContext(timeoutCtx))
	if err != nil {
		return "", "", "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, agentHTTPMaxBody+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return "", "", "", fmt.Errorf("read response: %w", err)
	}
	respBody := string(raw)
	if len(raw) > agentHTTPMaxBody {
		respBody = respBody[:agentHTTPMaxBody] + "…(truncated at 64KB)"
	}
	if len(respBody) > agentHTTPMaxContextBody {
		respBody = respBody[:agentHTTPMaxContextBody] + "…(truncated, full length " + strconv.Itoa(len(raw)) + ")"
	}

	return strconv.Itoa(resp.StatusCode), respBody, resp.Header.Get("Content-Type"), nil
}

// validateAgentOutboundURL http 节点出站 URL 校验（fail-closed）。
func validateAgentOutboundURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Errorf("invalid outbound url %q", raw)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("blocked outbound url scheme %q", scheme)
	}
	host := u.Hostname()
	if isBlockedPrintHost(host) {
		return fmt.Errorf("blocked outbound host %q (private/loopback/metadata)", host)
	}
	if net.ParseIP(host) == nil {
		ips, lerr := net.LookupIP(host)
		if lerr != nil || len(ips) == 0 {
			return fmt.Errorf("cannot resolve outbound host %q", host)
		}
		for _, ip := range ips {
			if isBlockedPrintIP(ip) {
				return fmt.Errorf("blocked outbound host %q (resolves to private address)", host)
			}
		}
	}
	return nil
}
