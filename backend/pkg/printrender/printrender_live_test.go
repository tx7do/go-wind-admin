package printrender

import (
	"context"
	"os"
	"testing"
)

// 本机 Chrome/Edge 冒烟：渲染一段含中文的简单文档并打印 PDF。
// 无浏览器环境跳过（CI 无头容器装了 chromium 则会执行）。
func TestRenderPDFLive(t *testing.T) {
	if _, err := findBrowser(); err != nil {
		t.Skip("no chrome/edge on this machine:", err.Error())
	}
	html := `<!doctype html><html><head><meta charset="utf-8"><style>
		body { margin: 20mm; font-family: "Microsoft YaHei", sans-serif; }
		h1 { font-size: 20pt; }
	</style></head><body><h1>GoWind 服务端 PDF 冒烟</h1><p>用户名：admin / 创建时间：2026-10-28</p></body></html>`
	pdf, err := RenderPDF(context.Background(), Options{
		HTML:         html,
		PaperWidthMm: 210, PaperHeightMm: 297,
		Background: true,
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if len(pdf) < 1000 {
		t.Fatalf("pdf too small: %d bytes", len(pdf))
	}
	out := os.Getenv("TEST_PRINT_PDF_OUT")
	if out != "" {
		if werr := os.WriteFile(out, pdf, 0o644); werr != nil {
			t.Fatalf("write %s: %v", out, werr)
		}
		t.Logf("pdf written to %s (%d bytes)", out, len(pdf))
	}
	t.Logf("pdf ok: %d bytes", len(pdf))
}
