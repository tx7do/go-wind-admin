package audit

import "strings"

// maskToken 是字面量脱敏后的统一占位符。
const maskToken = "***"

// MaskingRules 记录当前脱敏策略标识，落库到 masking_rules 字段。
const MaskingRules = "sql_literals:v1"

// MaskSQL 对 SQL 文本做字面量级脱敏：字符串字面量（'…'、E'…'、$tag$…$tag$）
// 与数值字面量整体替换为 ***，保留 SQL 结构（关键字、双引号标识符、$n 占位符、
// 注释、::类型转换）。按 PostgreSQL 词法规则实现扫描器：
//   - 单引号串内 '' 与 \' 均按转义处理（宁多掩不漏掩：普通串在
//     standard_conforming_strings 下 '\' 是字面反斜杠，按转义处理只会
//     多吞字符、不会提前截断导致漏脱敏）
//   - 美元引用支持空标签 $$ 与 $tag$，可跨行，整体脱敏
//   - 块注释按 PG 语义支持嵌套；注释与双引号标识符中的引号不误判
//   - $n 占位符原样保留（参数值本就不在 SQL 文本中）
func MaskSQL(sql string) string {
	var b strings.Builder
	b.Grow(len(sql))
	i, n := 0, len(sql)
	for i < n {
		c := sql[i]
		switch {
		case c == '\'':
			b.WriteString(maskToken)
			i = scanString(sql, i+1)
		case c == '"':
			i = copyVerbatim(&b, sql, i, scanQuotedIdent(sql, i+1))
		case c == '-' && i+1 < n && sql[i+1] == '-':
			i = copyVerbatim(&b, sql, i, scanLineComment(sql, i))
		case c == '/' && i+1 < n && sql[i+1] == '*':
			i = copyVerbatim(&b, sql, i, scanBlockComment(sql, i))
		case c == '$':
			i = maskDollar(&b, sql, i)
		case (c == 'e' || c == 'E') && i+1 < n && sql[i+1] == '\'':
			// E'…' 转义串：与普通串同一套扫描（'\' 已按转义处理）
			b.WriteString(maskToken)
			i = scanString(sql, i+2)
		case isDigit(c) || (c == '.' && i+1 < n && isDigit(sql[i+1])):
			b.WriteString(maskToken)
			i = scanNumber(sql, i)
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

// scanString 扫描单引号串，i 为开引号后首字节，返回闭引号后一字节索引。
func scanString(sql string, i int) int {
	n := len(sql)
	for i < n {
		switch sql[i] {
		case '\\':
			i += 2 // \' 转义（E 串标准语义；普通串宁多掩不漏掩）
		case '\'':
			if i+1 < n && sql[i+1] == '\'' {
				i += 2 // '' 转义
				continue
			}
			return i + 1
		default:
			i++
		}
	}
	return n // 未闭合：脱敏到末尾
}

// scanQuotedIdent 扫描双引号标识符（"" 为转义引号），内容不脱敏。
func scanQuotedIdent(sql string, i int) int {
	n := len(sql)
	for i < n {
		switch sql[i] {
		case '"':
			if i+1 < n && sql[i+1] == '"' {
				i += 2
				continue
			}
			return i + 1
		default:
			i++
		}
	}
	return n
}

// scanLineComment 扫描行注释（不含换行符），内容原样保留。
func scanLineComment(sql string, i int) int {
	n := len(sql)
	for i < n && sql[i] != '\n' {
		i++
	}
	return i
}

// scanBlockComment 扫描块注释（PG 支持嵌套），内容原样保留。
func scanBlockComment(sql string, i int) int {
	n := len(sql)
	depth := 0
	for i < n {
		if sql[i] == '/' && i+1 < n && sql[i+1] == '*' {
			depth++
			i += 2
			continue
		}
		if sql[i] == '*' && i+1 < n && sql[i+1] == '/' {
			depth--
			i += 2
			if depth == 0 {
				return i
			}
			continue
		}
		i++
	}
	return n
}

// maskDollar 处理 '$'：$n 占位符原样保留；$tag$…$tag$ 美元引用整体脱敏；
// 其余（非法或结尾 '$'）原样保留单字节。
func maskDollar(b *strings.Builder, sql string, i int) int {
	n := len(sql)
	// $n 占位符：数字后跟非标识字符
	j := i + 1
	for j < n && isDigit(sql[j]) {
		j++
	}
	if j > i+1 && (j >= n || !isIdentChar(sql[j])) {
		copyVerbatim(b, sql, i, j)
		return j
	}
	// 美元引用：$tag$（tag 可为空，不以数字开头）
	k := i + 1
	if k < n && !isDigit(sql[k]) {
		for k < n && isIdentChar(sql[k]) {
			k++
		}
	}
	if k < n && sql[k] == '$' {
		tag := sql[i : k+1]
		if rest := strings.Index(sql[k+1:], tag); rest >= 0 {
			b.WriteString(maskToken)
			return k + 1 + rest + len(tag)
		}
	}
	b.WriteByte('$')
	return i + 1
}

// scanNumber 扫描数值字面量（整数/小数/科学计数/十六进制）。
func scanNumber(sql string, i int) int {
	n := len(sql)
	j := i
	for j < n {
		c := sql[j]
		if isDigit(c) || c == '.' || isIdentChar(c) {
			j++
			continue
		}
		if (c == '+' || c == '-') && j > i && (sql[j-1] == 'e' || sql[j-1] == 'E') && j+1 < n && isDigit(sql[j+1]) {
			j++ // 科学计数指数符号
			continue
		}
		break
	}
	return j
}

func copyVerbatim(b *strings.Builder, sql string, start, end int) int {
	b.WriteString(sql[start:end])
	return end
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isIdentChar(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}
