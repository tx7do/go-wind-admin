// Package printrender 服务端打印渲染：把前端渲染管线（go-wind-print 内核）产出的
// 最终 HTML 用本机 Chrome/Edge（chromedp 驱动，自动探测安装路径）打印成 PDF。
//
// 分工边界：分页/毫米布局/字体全部由前端内核完成（两遍渲染：测量→分页→最终 HTML），
// 本包只做 HTML → PDF 这一步（PrintToPDF），纸张宽高与方向由调用方按模板元数据传入——
// 保证"预览即所得"（同一 Chromium 引擎），后端不重排、不理解模板。
//
// 渲染器留接口（Renderer）：默认 chromedp 进程内实现；部署侧如偏好伴生容器
// （worm 生态 print-render 微服务，Playwright + Chromium 镜像），实现 Renderer 换入即可。
package printrender

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

// Options 一次 PDF 打印的参数。
type Options struct {
	// HTML 前端内核产出的最终文档（已完成分页），应为完整 <html> 文档。
	HTML string
	// PaperWidthMm / PaperHeightMm 纸张尺寸（毫米），前端按模板 paperSize 解析后传入。
	PaperWidthMm  float64
	PaperHeightMm float64
	// Landscape 横向（前端按模板方向交换宽高后传 false，避免二次旋转）。
	Landscape bool
	// Background 打印背景色/背景图（PrintToPDF 默认不打印背景）。
	Background bool
	// Timeout 单次渲染超时，默认 60s。
	Timeout time.Duration
}

// Renderer HTML → PDF 渲染器接口（可换远程 print-render 服务实现）。
type Renderer interface {
	RenderPDF(ctx context.Context, opts Options) ([]byte, error)
}

var (
	mu       sync.Mutex
	instance Renderer
)

// SetRenderer 替换默认渲染器。
func SetRenderer(r Renderer) {
	mu.Lock()
	defer mu.Unlock()
	instance = r
}

// RenderPDF 用默认渲染器打印 PDF（未设置时用本机 Chrome 的 ChromeRenderer）。
func RenderPDF(ctx context.Context, opts Options) ([]byte, error) {
	mu.Lock()
	r := instance
	mu.Unlock()
	if r == nil {
		r = &ChromeRenderer{}
	}
	return r.RenderPDF(ctx, opts)
}

// ChromeRenderer 基于本机 Chrome/Edge 的渲染器（浏览器实例进程级复用）。
type ChromeRenderer struct {
	mu       sync.Mutex
	allocCtx context.Context
	cancel   context.CancelFunc
}

// findBrowser 浏览器查找：CHROME_PATH 环境变量 → chromedp 默认探测 → Edge 兜底。
func findBrowser() (string, error) {
	if p := os.Getenv("CHROME_PATH"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	if path, err := exec.LookPath("chrome"); err == nil {
		return path, nil
	}
	for _, p := range []string{
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
	} {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	// Edge 兜底（Windows 服务器常无 Chrome；Edge 同 Chromium 协议）
	for _, p := range []string{
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		"/usr/bin/microsoft-edge",
	} {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", errors.New("printrender: no Chrome/Edge found (set CHROME_PATH or install chromium in the image)")
}

func (c *ChromeRenderer) allocator(ctx context.Context) (context.Context, context.CancelFunc, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.allocCtx != nil {
		return c.allocCtx, c.cancel, nil
	}
	path, err := findBrowser()
	if err != nil {
		return nil, nil, err
	}
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(path),
		// 打印服务无网络/扩展需求：收窄攻击面
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
	)
	// Linux 容器常以 root 运行：Chrome 无 --no-sandbox 时直接拒启
	// （"Running as root without --no-sandbox is not supported"）。
	// PRINTRENDER_NO_SANDBOX=0 可强制关闭该行为。
	if runtime.GOOS == "linux" && runningAsRoot() && os.Getenv("PRINTRENDER_NO_SANDBOX") != "0" {
		opts = append(opts, chromedp.Flag("no-sandbox", true))
	}
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	c.allocCtx, c.cancel = allocCtx, cancel
	return allocCtx, cancel, nil
}

func (c *ChromeRenderer) RenderPDF(ctx context.Context, opts Options) ([]byte, error) {
	if opts.HTML == "" {
		return nil, errors.New("printrender: empty html")
	}
	if opts.PaperWidthMm <= 0 || opts.PaperHeightMm <= 0 {
		return nil, errors.New("printrender: paper size required (mm)")
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	allocCtx, cancelAlloc, err := c.allocator(ctx)
	if err != nil {
		return nil, err
	}
	defer cancelAlloc()

	taskCtx, cancelTask := chromedp.NewContext(allocCtx)
	defer cancelTask()
	taskCtx, cancelTimeout := context.WithTimeout(taskCtx, timeout)
	defer cancelTimeout()

	// html 经 JSON 编码嵌入表达式（json.Marshal 默认转义 <>& 与 U+2028/29，JS 字符串安全）
	htmlJSON, err := json.Marshal(opts.HTML)
	if err != nil {
		return nil, err
	}

	var pdf []byte
	pdfOpts := []chromedp.PDFOption{
		chromedp.PDFPaper(chromedp.PaperSize{
			Width:  opts.PaperWidthMm / 25.4,
			Height: opts.PaperHeightMm / 25.4,
		}),
		chromedp.PDFMargin(0),
	}
	if opts.Landscape {
		pdfOpts = append(pdfOpts, chromedp.PDFLandscape())
	}
	if opts.Background {
		pdfOpts = append(pdfOpts, chromedp.PDFPrintBackground())
	}
	if err := chromedp.Do(taskCtx,
		chromedp.Navigate("about:blank"),
		// html 经 JSON 编码内嵌表达式（json.Marshal 默认转义 <>& 与 U+2028/29，JS 字符串安全）
		chromedp.Evaluate[chromedp.Void](
			`document.open(); document.write(`+string(htmlJSON)+`); document.close();`,
		),
		chromedp.WaitReady("body"),
		chromedp.Sleep(300*time.Millisecond),
	); err != nil {
		return nil, fmt.Errorf("printrender: %w", err)
	}
	pdf, err = chromedp.Run(taskCtx, chromedp.PrintToPDF(pdfOpts...))
	if err != nil {
		return nil, fmt.Errorf("printrender: %w", err)
	}
	if len(pdf) == 0 {
		return nil, errors.New("printrender: empty pdf")
	}
	return pdf, nil
}

// runningAsRoot Linux 下检测当前进程是否 root（容器默认用户）。
func runningAsRoot() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	return os.Geteuid() == 0
}
