// 本文件针对 MenuPermissionConverter.ComposeMenuPaths 的全路径组装算法：
//
//	根节点（parentId=0）与自环/缺父节点保留自身路径；
//	父子链逐级以 "/" 拼接（父全路径为空则只取自身段、自身段为空则只取父全路径）；
//	成环（A→B→A）时环上先访问者经 seen 检测短路为去斜杠的自身路径，
//	后访问者仍按拼接语义组装——按实际行为钉死。
package converter

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

func menuNode(id, parent uint64, path string) *permissionV1.Menu {
	return &permissionV1.Menu{
		Id:       trans.Ptr(id),
		ParentId: trans.Ptr(parent),
		Path:     trans.Ptr(path),
	}
}

func TestComposeMenuPaths(t *testing.T) {
	t.Run("根节点保留自身路径", func(t *testing.T) {
		menus := []*permissionV1.Menu{menuNode(1, 0, "root")}
		NewMenuPermissionConverter().ComposeMenuPaths(menus)
		require.Equal(t, "root", *menus[0].Path)
	})

	t.Run("两级链拼接", func(t *testing.T) {
		menus := []*permissionV1.Menu{
			menuNode(1, 0, "root"),
			menuNode(2, 1, "child"),
		}
		NewMenuPermissionConverter().ComposeMenuPaths(menus)
		require.Equal(t, "root", *menus[0].Path)
		require.Equal(t, "root/child", *menus[1].Path)
	})

	t.Run("三级链拼接", func(t *testing.T) {
		menus := []*permissionV1.Menu{
			menuNode(1, 0, "a"),
			menuNode(2, 1, "b"),
			menuNode(3, 2, "c"),
		}
		NewMenuPermissionConverter().ComposeMenuPaths(menus)
		require.Equal(t, "a/b/c", *menus[2].Path)
	})

	t.Run("自环保留自身路径", func(t *testing.T) {
		menus := []*permissionV1.Menu{menuNode(1, 1, "self")}
		NewMenuPermissionConverter().ComposeMenuPaths(menus)
		require.Equal(t, "self", *menus[0].Path)
	})

	t.Run("缺父节点保留自身路径", func(t *testing.T) {
		menus := []*permissionV1.Menu{menuNode(1, 99, "orphan")}
		NewMenuPermissionConverter().ComposeMenuPaths(menus)
		require.Equal(t, "orphan", *menus[0].Path)
	})

	t.Run("父全路径为空时只取自身段", func(t *testing.T) {
		menus := []*permissionV1.Menu{
			menuNode(1, 0, ""),
			menuNode(2, 1, "child"),
		}
		NewMenuPermissionConverter().ComposeMenuPaths(menus)
		require.Equal(t, "child", *menus[1].Path)
	})

	t.Run("自身段为空时只取父全路径", func(t *testing.T) {
		menus := []*permissionV1.Menu{
			menuNode(1, 0, "root"),
			menuNode(2, 1, ""),
		}
		NewMenuPermissionConverter().ComposeMenuPaths(menus)
		require.Equal(t, "root", *menus[1].Path)
	})

	t.Run("成环时后访问者按拼接语义组装", func(t *testing.T) {
		menus := []*permissionV1.Menu{
			menuNode(1, 2, "a"),
			menuNode(2, 1, "b"),
		}
		NewMenuPermissionConverter().ComposeMenuPaths(menus)
		// 先访问的 A 命中环检测短路为去斜杠自身路径，后访问的 B 拼接 A 的短路结果
		require.Equal(t, "a/b/a", *menus[0].Path)
		require.Equal(t, "a/b", *menus[1].Path)
	})
}
