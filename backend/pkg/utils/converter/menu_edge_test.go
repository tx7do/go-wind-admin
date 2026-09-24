// 本文件补 menu.go 的 ConvertCode 早退/过滤臂与 buttonAction 关键词全臂。
package converter

import (
	"testing"

	"github.com/stretchr/testify/require"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

// TestMenuConvertCodeEdges ConvertCode 的空路径/全过滤段早退与
// 冒号前缀段过滤（管道内路径段先 TrimSpace、单数化，再按规则过滤）。
func TestMenuConvertCodeEdges(t *testing.T) {
	c := NewMenuPermissionConverter()
	require.Equal(t, "", c.ConvertCode("", "t", permissionV1.Menu_BUTTON))
	require.Equal(t, "", c.ConvertCode("   ", "t", permissionV1.Menu_BUTTON))
	require.Equal(t, "", c.ConvertCode("/", "t", permissionV1.Menu_BUTTON))
	// 多段路径先丢弃首段，剩余冒号前缀段全部被过滤 → 主体为空 + 默认动作
	require.Equal(t, ":act", c.ConvertCode("a/:b/:c", "无关词", permissionV1.Menu_BUTTON))
}

// TestButtonActionArms buttonAction 关键词全臂与默认臂。
func TestButtonActionArms(t *testing.T) {
	c := NewMenuPermissionConverter()
	for title, want := range map[string]string{
		"   ": "act",
		"新增":  "create",
		"添加":  "create",
		"保存":  "edit",
		"更新":  "edit",
		"删除":  "delete",
		"移除":  "delete",
		"导入":  "import",
		"导出":  "export",
		"下载":  "export",
		"无关词": "act",
	} {
		require.Equal(t, want, c.buttonAction(title), "buttonAction(%q)", title)
	}
}
