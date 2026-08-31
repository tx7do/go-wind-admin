// register 将新 CRUD 模块的登记代码一次性注入手写装配文件,替代原 Wire 时代的
// "改 wire set + make wire" 自动化。注入点为两个文件中的 register:* 锚点注释:
//
//	cmd/server/wiring_ent.go     仓储构造行 / 服务构造行 / NewRestServer 实参
//	internal/server/rest_server.go  服务形参 / 路由注册调用
//
// 仅覆盖 CRUD 生成器的标准形态(New<Ent>Repo(ctx, entClient) /
// New<Ent>Service(ctx, <ent>Repo));依赖更多的模块请手工调整注入行。
//
// 用法: go run ./tools/register -entity product   (亦接受 DictEntry / dict_entry 风格)
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"unicode"
)

type patch struct {
	path    string   // 目标文件
	anchors []string // 锚点行(按去空白后的全文匹配),插入到首个命中的锚点之后
	lines   []string // 插入的行(不含行尾符)
	skipIf  string   // 文件已包含该片段时跳过(幂等)
}

func main() {
	entity := flag.String("entity", "", "模块名:product / DictEntry / dict_entry 均可")
	appDir := flag.String("dir", "app/admin/service", "服务应用目录")
	flag.Parse()

	if *entity == "" {
		flag.Usage()
		os.Exit(2)
	}

	typ := pascal(*entity)
	name := lowerFirst(typ)
	wiringPath := *appDir + "/cmd/server/wiring_ent.go"
	serverPath := *appDir + "/internal/server/rest_server.go"
	if err := apply(
		// 仓储构造行:注入到仓储层小节末尾
		patch{
			path:   wiringPath,
			skipIf: "data.New" + typ + "Repo(",
			anchors: []string{
				"// ── register:repo ── 新模块仓储在此行后注册(make register 工具锚点,勿删)",
			},
			lines: []string{"\t" + name + "Repo := data.New" + typ + "Repo(ctx, entClient)"},
		},
		// 服务构造行:注入到服务层小节末尾
		patch{
			path:   wiringPath,
			skipIf: "service.New" + typ + "Service(",
			anchors: []string{
				"// ── register:service ── 新模块服务在此行后注册(make register 工具锚点,勿删)",
			},
			lines: []string{"\t" + name + "Service := service.New" + typ + "Service(ctx, " + name + "Repo)"},
		},
		// NewRestServer 实参
		patch{
			path:   wiringPath,
			skipIf: "\t\t" + name + "Service,",
			anchors: []string{
				"// register:rest-arg ── 新模块服务实参在此行后追加(make register 工具锚点,勿删)",
			},
			lines: []string{"\t\t" + name + "Service,"},
		},
		// rest_server.go 形参
		patch{
			path:   serverPath,
			skipIf: name + "Service *service." + typ + "Service,",
			anchors: []string{
				"// register:param ── 新模块服务形参在此行后注册(make register 工具锚点,勿删)",
			},
			lines: []string{"\t" + name + "Service *service." + typ + "Service,"},
		},
		// rest_server.go 路由注册
		patch{
			path:   serverPath,
			skipIf: "Register" + typ + "ServiceHTTPServer(srv, " + name + "Service)",
			anchors: []string{
				"// register:route ── 新模块路由在此行后注册(make register 工具锚点,勿删)",
			},
			lines: []string{"\tadminV1.Register" + typ + "ServiceHTTPServer(srv, " + name + "Service)"},
		},
	); err != nil {
		fmt.Fprintln(os.Stderr, "register:", err)
		os.Exit(1)
	}

	fmt.Printf("已登记 %s:\n"+
		"  %s (register:repo / register:service / register:rest-arg)\n"+
		"  internal/server/rest_server.go (register:param / register:route)\n"+
		"下一步: 实现 data/%s_repo.go 与 internal/service/%s_service.go,然后 go build。\n",
		typ, wiringPath, lowerFirst(typ), lowerFirst(typ))
}

// apply 依次应用各 patch;同文件多个 patch 顺序生效,幂等由 skipIf 保证。
func apply(patches ...patch) error {
	for _, p := range patches {
		if err := applyOne(p); err != nil {
			return err
		}
	}
	return nil
}

func applyOne(p patch) error {
	raw, err := os.ReadFile(p.path)
	if err != nil {
		return err
	}
	content := string(raw)
	if strings.Contains(content, p.skipIf) {
		return nil // 已登记,幂等跳过
	}

	eol := "\n"
	if strings.Contains(content, "\r\n") {
		eol = "\r\n"
	}
	lines := strings.Split(content, eol)

	for _, anchor := range p.anchors {
		at := indexOfAnchor(lines, anchor)
		if at < 0 {
			continue
		}
		out := make([]string, 0, len(lines)+len(p.lines))
		out = append(out, lines[:at+1]...)
		out = append(out, p.lines...)
		out = append(out, lines[at+1:]...)
		return os.WriteFile(p.path, []byte(strings.Join(out, eol)), 0o644)
	}
	return fmt.Errorf("%s: 未找到注册锚点 %q,请检查锚点注释是否被移动或删除", p.path, p.anchors[0])
}

func indexOfAnchor(lines []string, anchor string) int {
	for i, line := range lines {
		if strings.TrimSpace(line) == anchor {
			return i
		}
	}
	return -1
}

// pascal 把 product / dict_entry / dictEntry 统一为 PascalCase 的实体名。
func pascal(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	var b strings.Builder
	for _, p := range parts {
		r := []rune(p)
		b.WriteRune(unicode.ToUpper(r[0]))
		b.WriteString(string(r[1:]))
	}
	out := b.String()
	if out == "" {
		return s
	}
	return out
}

func lowerFirst(s string) string {
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}
