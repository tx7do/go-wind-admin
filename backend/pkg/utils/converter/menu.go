package converter

import (
	"strings"
	"unicode"

	"github.com/jinzhu/inflection"
	"github.com/tx7do/go-utils/trans"

	resourceV1 "go-wind-admin/api/gen/go/resource/service/v1"
)

type MenuPermissionConverter struct {
}

func NewMenuPermissionConverter() *MenuPermissionConverter {
	return &MenuPermissionConverter{}
}

// ConvertCode 将菜单的完整路径和类型转换为权限代码
func (c *MenuPermissionConverter) ConvertCode(path, title string, typ resourceV1.Menu_Type) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	// 移除掉路径前后的斜杠
	path = strings.Trim(path, "/")
	if path == "" {
		return ""
	}

	paths := strings.Split(path, "/")
	if len(paths) == 0 {
		return ""
	}

	if len(paths) > 1 {
		paths = paths[1:]
	}

	newPaths := paths[:0]
	for _, p := range paths {
		p = strings.TrimSpace(p)
		p = inflection.Singular(p)
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, ":") {
			continue
		}
		newPaths = append(newPaths, p)
	}
	paths = newPaths

	// 将路径段用 ':' 连接，作为权限主体
	permBase := strings.Join(paths, ":")

	// 根据菜单类型，决定是否添加动作后缀
	action := c.typeToAction(title, typ)
	if action == "" {
		return permBase
	}

	return permBase + ":" + action
}

// ComposeMenuPaths 递归拼接 menus 中每个菜单的 path 并写回菜单的 Path 字段（*string）。
// - menus: 待处理的菜单切片（会就地修改）
// 行为说明：
// 1. 使用 id->menu 映射快速查找父节点。
// 2. 用递归 + memoization 计算每个节点的完整 path（去除两端斜杠并用 '/' 连接）。
// 3. 若父节点 id 为 0 或父节点不存在，则视为根路径（仅使用自身 path 部分）。
// 4. 若出现自引用或循环，函数会将该节点视为只使用自身 path。
func (c *MenuPermissionConverter) ComposeMenuPaths(menus []*resourceV1.Menu) {
	// 建立 id -> menu 映射
	m := make(map[uint32]*resourceV1.Menu, len(menus))
	for _, mi := range menus {
		m[mi.GetId()] = mi
	}

	// 记忆计算结果：id -> fullPath
	memo := make(map[uint32]string, len(menus))

	var compute func(id uint32, seen map[uint32]bool) string
	compute = func(id uint32, seen map[uint32]bool) string {
		// 已计算
		if v, ok := memo[id]; ok {
			return v
		}
		// 循环检测
		if seen[id] {
			memo[id] = strings.Trim(m[id].GetPath(), "/")
			return memo[id]
		}
		menu, ok := m[id]
		if !ok {
			memo[id] = ""
			return ""
		}

		seen[id] = true
		defer delete(seen, id)

		part := menu.GetPath()
		parentId := menu.GetParentId()
		// 根节点或无父节点
		if parentId == 0 || parentId == id {
			memo[id] = part
			return memo[id]
		}
		parent, ok := m[parentId]
		if !ok {
			memo[id] = part
			return memo[id]
		}

		parentFull := compute(parent.GetId(), seen)
		var fullPath string
		switch {
		case parentFull == "":
			fullPath = part
		case part == "":
			fullPath = parentFull
		default:
			fullPath = parentFull + "/" + part
		}
		memo[id] = fullPath
		return fullPath
	}

	// 为每个菜单计算并写回 Path 字段
	for _, menu := range menus {
		id := menu.GetId()
		fullPath := compute(id, map[uint32]bool{})
		//log.Infof("Menu ID %d full path: %s", id, fullPath)
		// 写回为指针字符串
		menu.Path = trans.Ptr(fullPath)
	}
}

// typeToAction 将 Menu_Type 转换为 action 字符串
func (c *MenuPermissionConverter) typeToAction(title string, typ resourceV1.Menu_Type) string {

	switch typ {
	case resourceV1.Menu_CATALOG:
		return "dir"
	case resourceV1.Menu_MENU:
		return "view"
	case resourceV1.Menu_BUTTON:
		return c.buttonAction(title)
	case resourceV1.Menu_EMBEDDED:
		return "view"
	case resourceV1.Menu_LINK:
		return "jump"
	default:
		return ""
	}
}

// buttonAction 根据按钮标题和类型生成动作标识符
func (c *MenuPermissionConverter) buttonAction(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return "act"
	}

	title = strings.ToLower(title)

	addKeys := []string{
		"add",
		"addto",
		"add+",
		"create",
		"new",
		"plus",
		"append",
		"新增",
		"添加",
		"创建",
	}
	editKeys := []string{
		"edit",
		"update",
		"modify",
		"save",
		"patch",
		"保存",
		"修改",
		"更新",
		"编辑",
	}
	deleteKeys := []string{
		"delete",
		"del",
		"remove",
		"destroy",
		"drop",
		"discard",
		"trash",
		"删除",
		"移除",
		"弃用",
		"清除",
	}
	exportKeys := []string{
		"export",
		"download",
		"exportcsv",
		"exportexcel",
		"导出",
		"下载",
		"导出为",
	}
	importKeys := []string{
		"import",
		"importcsv",
		"importexcel",
		"导入",
		"导入为",
	}

	if matchAnyKeyword(title, addKeys) {
		return "create"
	}
	if matchAnyKeyword(title, editKeys) {
		return "edit"
	}
	if matchAnyKeyword(title, deleteKeys) {
		return "delete"
	}
	if matchAnyKeyword(title, importKeys) {
		return "import"
	}
	if matchAnyKeyword(title, exportKeys) {
		return "export"
	}

	return "act"
}

// matchAnyKeyword 先按 token 精确/前缀匹配，再回退到 substring 匹配
func matchAnyKeyword(title string, keys []string) bool {
	tokens := tokenize(title)

	for _, k := range keys {
		k = strings.ToLower(k)
		for _, tk := range tokens {
			if tk == k || strings.HasPrefix(tk, k) {
				return true
			}
		}
	}
	// 回退：整句包含关键词
	for _, k := range keys {
		if strings.Contains(title, strings.ToLower(k)) {
			return true
		}
	}
	return false
}

// tokenize 按非字母数字分隔并返回小写 tokens
func tokenize(s string) []string {
	var buf []rune
	var out []string
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			buf = append(buf, unicode.ToLower(r))
			continue
		}
		if len(buf) > 0 {
			out = append(out, string(buf))
			buf = buf[:0]
		}
	}
	if len(buf) > 0 {
		out = append(out, string(buf))
	}
	return out
}
