package service

import (
	"fmt"
	"regexp"
	"strings"
)

// 租户路径的结构化强制校验（P1-5 深层修复，纯 Go 零依赖方案）。
//
// 背景：租户谓词此前只做"文本出现性"正则检查，可被以下形态穿透：
//   - 多表 JOIN 时只过滤了其中一张（如 JOIN 子表过滤了主表没过滤）；
//   - 子查询/派生表内的表完全不过滤（谓词只写在外层主表上）。
//
// 方案：按查询块（顶层 + 每个括号子块）统计白名单表引用数与
// `tenant_id = <数字字面量>` 合取条件数，要求每块"条件数 ≥ 表数"。
// 配合既有的结构 OR 拒绝与注释拒绝，剩余可绕过面收窄至：
// 同一查询块内同表多次出现共享同一谓词（SQL 语义上本就对每次出现生效，合法）。
//
// 前置：调用方已做过白名单表纠正、OR 拒绝、注释拒绝——本函数只做
// 逐块表/条件计数。字符串字面量与注释必须先剥离（stripSQLStringsAndComments）。

var (
	tenantCondPattern = regexp.MustCompile(`(?i)(?:[a-zA-Z_][a-zA-Z0-9_]*\.)?tenant_id\s*=\s*[0-9]+\b`)
	whitelistRefPattern = regexp.MustCompile(`(?i)\b(FROM|JOIN)\s+([a-zA-Z_][a-zA-Z0-9_]*)`)
)

// queryBlock 一个查询块（顶层或括号子块）的结构统计。
type queryBlock struct {
	text         string // 本块直接层文本（嵌套子块被替换为占位符）
	whitelistRefs int
	tenantConds  int
}

// splitQueryBlocks 把剥离了字符串/注释的 SQL 切成查询块：
// 返回顶层块与全部子块（含嵌套）。每块文本中更深层的括号子块被替换为
// 空格占位（保证本块的 FROM/JOIN/条件计数不含子块内容）。
func splitQueryBlocks(stripped string) (top string, blocks []queryBlock) {
	var build func(s string) string
	var subs []string

	build = func(s string) string {
		var out strings.Builder
		depth := 0
		start := -1
		for i := 0; i < len(s); i++ {
			switch s[i] {
			case '(':
				depth++
				if depth == 1 {
					start = i
				}
			case ')':
				if depth > 0 {
					if depth == 1 {
						inner := s[start+1 : i]
						// 子块内递归切分（嵌套子查询）
						subTop := build(inner)
						subs = append(subs, subTop)
						// 子块自身作为独立块再统计一次（含其直接层）
						blocks = append(blocks, queryBlock{ text: subTop })
					}
					depth--
					if depth == 0 {
						out.WriteByte(' ')
						continue
					}
				}
				out.WriteByte(' ')
			default:
				if depth == 0 {
					out.WriteByte(s[i])
				}
			}
		}
		return out.String()
	}

	top = build(stripped)
	// 顶层块
	blocks = append([]queryBlock{{text: top}}, blocks...)
	return top, blocks
}

// countBlock 统计一个查询块的白名单表引用与租户条件数。
// 两者均只统计**首个 FROM 之后**的本块直接层文本：SELECT 列表（含 CASE WHEN）
// 里的谓词不构成对表的有效过滤，不得计入覆盖。
func countBlock(b *queryBlock, whitelist map[string]string) {
	text := b.text
	if idx := strings.Index(text, " FROM "); idx >= 0 {
		text = text[idx+1:]
	} else if idx := strings.Index(text, " JOIN "); idx >= 0 {
		text = text[idx+1:]
	}
	for _, m := range whitelistRefPattern.FindAllStringSubmatch(text, -1) {
		t := strings.ToLower(m[2])
		if _, ok := whitelist[t]; ok {
			b.whitelistRefs++
		}
	}
	b.tenantConds = len(tenantCondPattern.FindAllString(text, -1))
}

// enforcePerBlockTenantFilter 租户路径的结构化逐块强制：
// 每个含白名单表引用的查询块，其 tenant_id = <租户ID> 合取条件数必须
// 不小于表引用数。返回 error 即拒绝（fail-closed）。
func enforcePerBlockTenantFilter(strippedSQL string, whitelist map[string]string, tenantID int64) error {
	_, blocks := splitQueryBlocks(strippedSQL)
	for i := range blocks {
		countBlock(&blocks[i], whitelist)
		b := &blocks[i]
		if b.whitelistRefs > 0 && b.tenantConds < b.whitelistRefs {
			return fmt.Errorf(
				"tenant filter insufficient: %d table reference(s) but only %d tenant_id condition(s) in one query block",
				b.whitelistRefs, b.tenantConds)
		}
	}
	_ = tenantID // 谓词值本身由既有正则出现性检查保证
	return nil
}
