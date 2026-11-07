package service

import (
	"fmt"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/app/admin/service/cmd/server/assets"
)

func TestMenuListToQueryString(t *testing.T) {
	type args struct {
		menus []uint32
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Single menu ID",
			args: args{menus: []uint32{1}},
			want: `{"id__in":"[\"1\"]","status":"ON","type__not":"BUTTON"}`,
		},
		{
			name: "Multiple menu IDs",
			args: args{menus: []uint32{1, 2, 3}},
			want: `{"id__in":"[\"1\", \"2\", \"3\"]","status":"ON","type__not":"BUTTON"}`,
		},
		{
			name: "No menu IDs",
			args: args{menus: []uint32{}},
			want: `{"id__in":"[]","status":"ON","type__not":"BUTTON"}`,
		},
		{
			name: "Large menu IDs",
			args: args{menus: []uint32{1234567890, 987654321}},
			want: `{"id__in":"[\"1234567890\", \"987654321\"]","status":"ON","type__not":"BUTTON"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AdminPortalService{}
			if got := s.menuListToQueryString(tt.args.menus, false); got != tt.want {
				t.Errorf("menuListToQueryString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOpenAPI(t *testing.T) {
	// 加载 OpenAPI V3 文档
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(assets.OpenApiData)
	if err != nil {
		t.Fatalf("加载 OpenAPI 文档失败: %v", err)
	}

	if doc == nil {
		t.Fatal("OpenAPI 文档为空")
	}
	if doc.Paths == nil {
		t.Fatal("OpenAPI 文档的路径为空")
	}

	// 遍历所有路径和操作
	for path, pathItem := range doc.Paths.Map() {
		fmt.Printf("路径: %s\n", path)
		for method, operation := range pathItem.Operations() {
			fmt.Printf("  方法: %s\n", method)
			if operation.Summary != "" {
				fmt.Printf("    摘要: %s\n", operation.Summary)
			}
			if operation.Description != "" {
				fmt.Printf("    描述: %s\n", operation.Description)
			}
			if len(operation.Tags) > 0 {
				tag := doc.Tags.Get(operation.Tags[0])
				fmt.Printf("    服务: %s %s\n", tag.Name, tag.Description)
			}
		}
	}
}

// TestFilterMenuTree 钉住套餐白名单的递归过滤语义（欠账 2 后半回归）：
// 历史缺陷是只遍历顶层平铺 for——sys_menus 根节点全不带 module、带 module 的
// 全是叶子，白名单实际过滤面为 0；递归接上后叶子才真正按套餐判定。
func TestFilterMenuTree(t *testing.T) {
	mod := func(m identityV1.Module) *identityV1.Module { return &m }
	allowed := map[identityV1.Module]bool{
		identityV1.Module_OPM:  true,
		identityV1.Module_TASK: true,
	}

	leaf := func(name string, m *identityV1.Module) *permissionV1.Menu {
		return &permissionV1.Menu{Name: trans.Ptr(name), Module: m}
	}

	t.Run("叶子在白名单内保留、不在则剔除", func(t *testing.T) {
		assert.NotNil(t, filterMenuTree(leaf("用户", mod(identityV1.Module_OPM)), allowed))
		assert.Nil(t, filterMenuTree(leaf("字典", mod(identityV1.Module_DICT)), allowed))
	})

	t.Run("未归类叶子放行（通知等平台页刻意不登记 module）", func(t *testing.T) {
		assert.NotNil(t, filterMenuTree(leaf("通知", nil), allowed))
	})

	t.Run("nil 输入剔除", func(t *testing.T) {
		assert.Nil(t, filterMenuTree(nil, allowed))
	})

	t.Run("容器保留白名单子树剔除其余", func(t *testing.T) {
		root := &permissionV1.Menu{
			Name: trans.Ptr("运营管理"),
			Children: []*permissionV1.Menu{
				leaf("用户", mod(identityV1.Module_OPM)),
				leaf("任务", mod(identityV1.Module_TASK)),
				leaf("字典", mod(identityV1.Module_DICT)),
			},
		}
		kept := filterMenuTree(root, allowed)
		require.NotNil(t, kept, "仍有白名单子项，容器应保留")
		require.Len(t, kept.Children, 2)
		names := []string{*kept.Children[0].Name, *kept.Children[1].Name}
		assert.ElementsMatch(t, []string{"用户", "任务"}, names)
	})

	t.Run("容器子项全被剔除时一并丢弃不留空目录", func(t *testing.T) {
		root := &permissionV1.Menu{
			Name: trans.Ptr("字典"),
			Children: []*permissionV1.Menu{
				leaf("字典类型", mod(identityV1.Module_DICT)),
				leaf("字典项", mod(identityV1.Module_DICT)),
			},
		}
		assert.Nil(t, filterMenuTree(root, allowed), "全部子项被剔，容器应整枝剔除")
	})

	t.Run("深层嵌套逐层收缩", func(t *testing.T) {
		root := &permissionV1.Menu{
			Name: trans.Ptr("根"),
			Children: []*permissionV1.Menu{
				{
					Name: trans.Ptr("中间层"),
					Children: []*permissionV1.Menu{
						leaf("任务列表", mod(identityV1.Module_TASK)),
						leaf("日志", mod(identityV1.Module_LOG)),
					},
				},
				leaf("AI", mod(identityV1.Module_AI)),
			},
		}
		kept := filterMenuTree(root, allowed)
		require.NotNil(t, kept)
		require.Len(t, kept.Children, 1, "AI 叶子被剔")
		mid := kept.Children[0]
		require.Len(t, mid.Children, 1, "中间层只剩任务子树")
		assert.Equal(t, "任务列表", *mid.Children[0].Name)
	})
}
