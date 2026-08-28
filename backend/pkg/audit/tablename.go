package audit

import "strings"

// 数据分类码，与前端 i18n（enum.dataAccessAuditLog.dataCategory.*）一一对应。
const (
	CategoryUserData      = "USER_DATA"
	CategoryOrgData       = "ORG_DATA"
	CategoryAccessControl = "ACCESS_CONTROL"
	CategoryTenantData    = "TENANT_DATA"
	CategoryMessage       = "MESSAGE_DATA"
	CategoryAuditLog      = "AUDIT_LOG"
	CategorySystemConfig  = "SYSTEM_CONFIG"
	CategoryUnknown       = "UNKNOWN"
)

// ExtractTables 从（已脱敏的）SQL 文本提取被访问的数据表名。
// 覆盖 FROM / INSERT INTO / UPDATE / DELETE FROM / JOIN 及 FROM 的逗号多表；
// 跳过子查询括号（内层 FROM 会在主扫描中被再次捕获）；按首个出现顺序去重。
// 依赖 sqlmask.go 的词法扫描器（字符串/注释不拆词、双引号标识符整体成词）。
func ExtractTables(sql string) []string {
	toks := lexSQL(sql)
	var out []string
	seen := map[string]bool{}
	add := func(name string) {
		lower := strings.ToLower(name)
		if lower == "" || seen[lower] {
			return
		}
		seen[lower] = true
		out = append(out, name)
	}
	i := 0
	for i < len(toks) {
		t := toks[i]
		if t.kind != tokIdent || t.quoted {
			i++
			continue
		}
		var multi bool
		switch strings.ToUpper(t.text) {
		case "FROM", "INTO":
			multi = true
		case "UPDATE", "JOIN":
			multi = false
		default:
			i++
			continue
		}
		i = parseTables(toks, i+1, multi, add)
	}
	return out
}

// ClassifyTable 按表名映射数据分类码；未匹配返回 UNKNOWN。
func ClassifyTable(table string) string {
	t := strings.ToLower(strings.Trim(table, `"`))
	switch {
	case strings.HasSuffix(t, "_audit_logs"), t == "sys_policy_evaluation_logs":
		return CategoryAuditLog
	case t == "sys_users", strings.HasPrefix(t, "sys_user_"):
		return CategoryUserData
	case strings.HasPrefix(t, "sys_org_units"),
		strings.HasPrefix(t, "sys_positions"),
		strings.HasPrefix(t, "sys_membership"):
		return CategoryOrgData
	case t == "sys_tenants", t == "sys_plans", strings.HasPrefix(t, "sys_plan_"):
		return CategoryTenantData
	case strings.HasPrefix(t, "sys_internal_message"):
		return CategoryMessage
	case t == "sys_apis", t == "sys_menus",
		strings.HasPrefix(t, "sys_roles"),
		strings.HasPrefix(t, "sys_role_"),
		strings.HasPrefix(t, "sys_permission"),
		strings.HasPrefix(t, "sys_login_polic"):
		return CategoryAccessControl
	case strings.HasPrefix(t, "sys_dict"),
		t == "sys_languages",
		t == "sys_files",
		strings.HasPrefix(t, "sys_tasks"):
		return CategorySystemConfig
	default:
		return CategoryUnknown
	}
}

// ---- 简化词法器：ident（裸词/双引号）与单字节标点 ----

type tokKind int

const (
	tokIdent tokKind = iota
	tokPunct
)

type sqlToken struct {
	text   string
	kind   tokKind
	quoted bool
}

// lexSQL 把 SQL 切成 token：字符串字面量与注释整体跳过（复用 sqlmask 的扫描器），
// 双引号标识符去引号成 ident（quoted=true，关键字判断只认裸词）。
func lexSQL(sql string) []sqlToken {
	var toks []sqlToken
	i, n := 0, len(sql)
	for i < n {
		c := sql[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '\'':
			i = scanString(sql, i+1)
		case c == '-' && i+1 < n && sql[i+1] == '-':
			i = scanLineComment(sql, i)
		case c == '/' && i+1 < n && sql[i+1] == '*':
			i = scanBlockComment(sql, i)
		case c == '"':
			j := scanQuotedIdent(sql, i+1)
			toks = append(toks, sqlToken{kind: tokIdent, quoted: true, text: sql[i+1 : j-1]})
			i = j
		case isIdentChar(c):
			j := i
			for j < n && isIdentChar(sql[j]) {
				j++
			}
			toks = append(toks, sqlToken{kind: tokIdent, text: sql[i:j]})
			i = j
		default:
			toks = append(toks, sqlToken{kind: tokPunct, text: sql[i : i+1]})
			i++
		}
	}
	return toks
}

// tableStopWords 是表引用位置的裸词黑名单：出现在这里说明不是表名
// （如 ON CONFLICT ... DO UPDATE SET 中的 SET——UPDATE 关键字误触发时兜底）。
var tableStopWords = map[string]bool{
	"AS": true, "SET": true, "WHERE": true, "SELECT": true, "VALUES": true,
	"ON": true, "DO": true, "NOTHING": true, "RETURNING": true, "GROUP": true,
	"ORDER": true, "LIMIT": true, "OFFSET": true, "JOIN": true, "LEFT": true,
	"RIGHT": true, "INNER": true, "OUTER": true, "CROSS": true, "FULL": true,
	"USING": true, "UNION": true, "EXCEPT": true, "INTERSECT": true,
	"EXCLUDED": true, "DEFAULT": true, "WITH": true, "FOR": true, "TO": true,
	"WHEN": true, "THEN": true, "ELSE": true, "END": true, "AND": true,
	"OR": true, "NOT": true, "IS": true, "NULL": true, "IN": true,
	"EXISTS": true, "BETWEEN": true, "LIKE": true, "ILIKE": true,
	"HAVING": true, "DISTINCT": true, "ALL": true, "CASE": true,
	"ASC": true, "DESC": true, "ONLY": true, "NOWAIT": true, "SKIP": true,
}

// parseTables 从 toks[i] 起解析表引用：[schema.]table [AS alias]，multi 时
// 支持逗号连接多个。遇到子查询 '('、停用词或其它非标识 token 即止，
// 返回停止处的 token 下标（供主扫描继续找后续关键字，如 JOIN）。
func parseTables(toks []sqlToken, i int, multi bool, add func(string)) int {
	for i < len(toks) {
		name, next, ok := parseIdentRef(toks, i)
		if !ok {
			return i
		}
		add(name)
		i = next
		// 可选 AS 别名（别名可为裸词或引号标识符，不计入表名）
		if i < len(toks) && toks[i].kind == tokIdent && !toks[i].quoted &&
			strings.EqualFold(toks[i].text, "AS") {
			i++
			if i < len(toks) && toks[i].kind == tokIdent {
				i++
			}
		}
		if multi && i < len(toks) && toks[i].kind == tokPunct && toks[i].text == "," {
			i++
			continue
		}
		return i
	}
	return i
}

// parseIdentRef 解析单个（可 schema 限定的）表引用，返回完整名与下一 token 下标。
// 裸词且命中停用词黑名单时视为非表名（ok=false）。
func parseIdentRef(toks []sqlToken, i int) (string, int, bool) {
	if i >= len(toks) || toks[i].kind != tokIdent {
		return "", i, false
	}
	first := toks[i]
	if !first.quoted && tableStopWords[strings.ToUpper(first.text)] {
		return "", i, false
	}
	parts := []string{first.text}
	i++
	// schema.table：'.' 后再接一个标识符
	if i+1 < len(toks) && toks[i].kind == tokPunct && toks[i].text == "." &&
		toks[i+1].kind == tokIdent {
		parts = append(parts, toks[i+1].text)
		i += 2
	}
	return strings.Join(parts, "."), i, true
}
