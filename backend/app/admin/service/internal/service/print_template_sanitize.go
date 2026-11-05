package service

import (
	"bytes"
	"fmt"
	"net"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// 打印渲染（headless Chrome）的 SSRF/脚本注入防线（P1-11）。
//
// RenderPdf 接受任意用户 HTML 直送 Chromium（about:blank + document.write）：
//   - <iframe src="http://169.254.169.254/latest/meta-data/"> 的响应会被渲染进
//     PDF 返还给攻击者——一个"可读的 SSRF 原语"，云元数据/内网 HTTP 服务一锅端；
//   - <script> 在无 CSP 的 about:blank 里照常执行（Chromium n-day + --no-sandbox
//     时即容器 root RCE 的前置面）。
//
// 处置：送渲染前经 HTML 清洗——剥离脚本/嵌入类元素与事件属性，拒收指向
// 私网/回环/链路本地/元数据地址的资源 URL；另在服务层加渲染并发信号量
//（见 RenderPdf），每请求独立超时由 printrender.Options.Timeout 承担。

// blockedURLAttributeErr 清洗失败（发现被禁资源）的哨兵错误内容。
type printHTMLBlockedError struct{ detail string }

func (e *printHTMLBlockedError) Error() string { return e.detail }

// printURLAttributes 需要做 URL 校验的元素/属性对（其余元素的 src/href 视具体标签）。
var printURLAttributes = map[string]map[string]bool{
	"a":      {"href": true},
	"img":    {"src": true},
	"area":   {"href": true},
	"video":  {"src": true, "poster": true},
	"audio":  {"src": true},
	"source": {"src": true},
	"input":  {"src": true},
	"form":   {"action": true},
	"td":     {"background": true},
	"table":  {"background": true},
	"body":   {"background": true},
}

// printStripElements 渲染前整体剥除的元素：脚本执行与子资源加载面。
// iframe/object/embed/link/meta 整体移除（iframe 是 SSRF 回读主通道；
// link/meta 的 stylesheet/CSP 覆盖在打印场景非必需）。
var printStripElements = map[string]bool{
	"script": true, "iframe": true, "frame": true, "frameset": true,
	"object": true, "embed": true, "applet": true, "link": true, "meta": true,
}

// sanitizePrintHTML 清洗用户 HTML 后返回可安全送 Chromium 的版本。
// 返回 error 表示发现被禁内容（调用方应 400 拒绝而不是静默改写）。
func sanitizePrintHTML(input string) (string, error) {
	doc, err := html.Parse(strings.NewReader(input))
	if err != nil {
		return "", err
	}

	var blocked *printHTMLBlockedError
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if blocked != nil {
			return
		}
		for c := n.FirstChild; c != nil && blocked == nil; {
			next := c.NextSibling
			if c.Type == html.ElementNode {
				tag := strings.ToLower(c.Data)
				if printStripElements[tag] || tag == "base" {
					n.RemoveChild(c)
					c = next
					continue
				}
				kept := c.Attr[:0]
				for _, a := range c.Attr {
					an := strings.ToLower(a.Key)
					switch {
					case strings.HasPrefix(an, "on"):
						continue // 事件处理器一律剥除
					case an == "srcset":
						continue // 复杂 URL 列表，打印场景非必需，整体剥除
					case printURLAttributes[tag] != nil && printURLAttributes[tag][an]:
						if verr := checkPrintResourceURL(a.Val); verr != nil {
							blocked = &printHTMLBlockedError{detail: verr.Error()}
							return
						}
					}
					kept = append(kept, a)
				}
				c.Attr = kept
			}
			walk(c)
			c = next
		}
	}
	walk(doc)
	if blocked != nil {
		return "", blocked
	}

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// checkPrintResourceURL 校验资源 URL：
//   - 非 http(s) 的绝对 scheme（javascript:/data:/vbscript:/file: 等）一律拒绝；
//   - 相对 URL 在 about:blank 源下不可解析（本就打不开），放行；
//   - http(s)：主机为回环/私网/链路本地/元数据地址，或解析到私网段 → 拒绝。
func checkPrintResourceURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid resource url %q", raw)
	}
	scheme := strings.ToLower(u.Scheme)
	if u.Host == "" {
		switch scheme {
		case "", "http", "https":
			return nil // 相对 URL：about:blank 源不可解析，无害
		}
		return fmt.Errorf("blocked url scheme %q", scheme)
	}
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("blocked url scheme %q", scheme)
	}

	host := u.Hostname()
	if isBlockedPrintHost(host) {
		return fmt.Errorf("blocked resource host %q (private/loopback/metadata)", host)
	}
	// 域名形式：解析后逐 IP 复核（阻断"公网域名→内网 IP"与云元数据域名两条路）
	if net.ParseIP(host) == nil {
		ips, lerr := net.LookupIP(host)
		if lerr != nil || len(ips) == 0 {
			return fmt.Errorf("cannot resolve resource host %q", host)
		}
		for _, ip := range ips {
			if isBlockedPrintIP(ip) {
				return fmt.Errorf("blocked resource host %q (resolves to private address)", host)
			}
		}
	}
	return nil
}

// isBlockedPrintHost 主机字符串层判定（含 IP 字面量）。
func isBlockedPrintHost(host string) bool {
	lh := strings.ToLower(strings.TrimSuffix(host, "."))
	if lh == "localhost" || strings.HasSuffix(lh, ".localhost") ||
		strings.HasSuffix(lh, ".local") || strings.HasSuffix(lh, ".internal") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return isBlockedPrintIP(ip)
	}
	return false
}

// isBlockedPrintIP IP 层判定：回环/私网/链路本地/未指定/组播 + 云元数据端点。
func isBlockedPrintIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	// CGNAT（100.64.0.0/10）：部分云元数据/内部服务落在该段
	if v4 := ip.To4(); v4 != nil && v4[0] == 100 && v4[1] >= 64 && v4[1] < 128 {
		return true
	}
	return false
}
