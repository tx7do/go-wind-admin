// register 工具的纯逻辑测试。
//
// 工具以"锚点注释后插行"的方式给 wiring_ent.go / rest_server.go 注入新模块登记
// 代码，插错位置或破坏行尾符都会让生成代码直接编译失败，所以这里钉住四件事：
//   - 名字规整（pascal/lowerFirst）对三种输入风格（product/dict_entry/dictEntry）收敛一致；
//   - applyOne 在锚点行后插入、保留文件原有行尾符（CRLF 文件不能被写成混合行尾）；
//   - skipIf 幂等：重复运行不产生第二份注入；
//   - 锚点丢失必须报错（生成器挪了锚点注释时宁可失败也不能插错地方）。
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPascal(t *testing.T) {
	cases := map[string]string{
		"product":    "Product",
		"DictEntry":  "DictEntry",
		"dict_entry": "DictEntry",
		"dictEntry":  "DictEntry",
		"user_mfa":   "UserMfa",
	}
	for in, want := range cases {
		require.Equal(t, want, pascal(in), "pascal(%q) 应收敛为 PascalCase 实体名", in)
	}
}

func TestLowerFirst(t *testing.T) {
	require.Equal(t, "product", lowerFirst("Product"))
	require.Equal(t, "dictEntry", lowerFirst("DictEntry"))
}

const anchorLine = "// ── register:repo ── 新模块仓储在此行后注册(make register 工具锚点,勿删)"

// writeTempFile 落一个临时目标文件（内容按给定行尾符拼接），返回路径。
func writeTempFile(t *testing.T, eol string, lines ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "wiring_ent.go")
	require.NoError(t, os.WriteFile(path, []byte(strings.Join(lines, eol)), 0o644))
	return path
}

func TestApplyOne_InsertsAfterAnchorWithLF(t *testing.T) {
	path := writeTempFile(t, "\n",
		"package server",
		anchorLine,
		"// 下一节",
	)
	inject := "\tdictRepo := data.NewDictEntryRepo(ctx, entClient)"

	require.NoError(t, applyOne(patch{
		path:    path,
		anchors: []string{anchorLine},
		lines:   []string{inject},
		skipIf:  "data.NewDictEntryRepo(",
	}))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	out := string(raw)
	require.Contains(t, out, inject, "注入行应存在")
	// 注入行必须紧跟锚点行之后、且在原有下一行之前
	anchorIdx := strings.Index(out, anchorLine)
	injectIdx := strings.Index(out, inject)
	nextIdx := strings.Index(out, "// 下一节")
	require.Positive(t, injectIdx-anchorIdx, "注入行应在锚点之后")
	require.Less(t, injectIdx, nextIdx, "注入行应在锚点与原有下一行之间")
	require.NotContains(t, out, "\r\n", "LF 文件不应被写成 CRLF")
}

func TestApplyOne_PreservesCRLF(t *testing.T) {
	path := writeTempFile(t, "\r\n",
		"package server",
		anchorLine,
	)
	inject := "\tdictRepo := data.NewDictEntryRepo(ctx, entClient)"

	require.NoError(t, applyOne(patch{
		path:    path,
		anchors: []string{anchorLine},
		lines:   []string{inject},
	}))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	out := string(raw)
	require.Contains(t, out, inject)
	// CRLF 文件插入后必须保持纯 CRLF：混入裸 LF 会让 diff/工具链观感错乱
	require.NotContains(t, strings.TrimSuffix(out, "\r\n"), "\n\n",
		"注入处不应出现裸 LF 混排")
	require.Equal(t, 2, strings.Count(out, "\r\n"),
		"两行内容(锚点+注入)应恰好两处 CRLF 行尾")
}

func TestApplyOne_IdempotentSkipIf(t *testing.T) {
	original := strings.Join([]string{
		"package server",
		anchorLine,
		"\tdictRepo := data.NewDictEntryRepo(ctx, entClient)",
	}, "\n")
	path := filepath.Join(t.TempDir(), "wiring_ent.go")
	require.NoError(t, os.WriteFile(path, []byte(original), 0o644))

	// 已登记过（skipIf 命中）必须原样跳过：重复运行不能插出第二份
	require.NoError(t, applyOne(patch{
		path:    path,
		anchors: []string{anchorLine},
		lines:   []string{"\tdup := 1"},
		skipIf:  "data.NewDictEntryRepo(",
	}))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, original, string(raw), "幂等跳过时文件必须逐字节不变")
}

func TestApplyOne_MissingAnchorFails(t *testing.T) {
	path := writeTempFile(t, "\n",
		"package server",
		"// 锚点注释被挪走了",
	)

	err := applyOne(patch{
		path:    path,
		anchors: []string{anchorLine},
		lines:   []string{"\tshouldNotAppear := true"},
	})
	require.Error(t, err, "锚点丢失必须失败——宁可报错也不能把注入行插到错误位置")
	require.Contains(t, err.Error(), "未找到注册锚点", "错误要指向锚点检查")

	raw, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	require.NotContains(t, string(raw), "shouldNotAppear", "失败路径不应写入任何内容")
}

func TestApply_SequentialPatchesOnSameFile(t *testing.T) {
	path := writeTempFile(t, "\n",
		"package server",
		anchorLine,
		"// ── register:service ── 新模块服务在此行后注册(make register 工具锚点,勿删)",
	)
	serviceAnchor := "// ── register:service ── 新模块服务在此行后注册(make register 工具锚点,勿删)"

	require.NoError(t, apply(
		patch{path: path, anchors: []string{anchorLine}, lines: []string{"\trepoLine := true"}, skipIf: "NewProductRepo("},
		patch{path: path, anchors: []string{serviceAnchor}, lines: []string{"\tserviceLine := true"}, skipIf: "NewProductService("},
	))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	out := string(raw)
	require.Contains(t, out, "repoLine := true")
	require.Contains(t, out, "serviceLine := true", "同文件第二个 patch 应顺序生效")
	require.Less(t, strings.Index(out, "repoLine"), strings.Index(out, serviceAnchor),
		"repo 注入行应仍在 service 锚点之前（顺序插入不互相挤位）")
}
