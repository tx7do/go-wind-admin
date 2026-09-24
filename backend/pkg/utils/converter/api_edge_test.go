// 本文件补 api.go 解析链的边角分支：
//
//	splitOperationID 的无 Service 后缀捕获组（m[2] 分支）与带尾随 name 段的
//	  operationID（nameRaw 非空分支）；
//	splitCamelAfterFirstWord 的空串与全小写（无大写边界）分支；
//	pathToResource 的空路径与首段空白分支；
//	stripVersionPrefix 的大小写不敏感 api 段剥离与路径中段 api 段过滤。
//
// 结构性不可达分支（记录在案）：splitOperationID 的 m[1]、m[2] 同时空串
// 早退分支——正则交替组要求其一非空才能整体匹配；stripVersionPrefix 的
// parts=nil else 分支——idx<=1 时 idx+1<=len(parts) 恒成立。
package converter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitOperationIDEdges(t *testing.T) {
	c := NewApiPermissionConverter()

	t.Run("无Service后缀走第二捕获组", func(t *testing.T) {
		svc, action, name, ok := c.splitOperationID("Foo_Bar")
		require.True(t, ok)
		require.Equal(t, "Foo", svc)
		require.Equal(t, "Bar", action)
		require.Empty(t, name)
	})

	t.Run("带尾随name段", func(t *testing.T) {
		// 捕获组不含 "Service" 后缀（正则按组定义剥离后缀）
		svc, action, name, ok := c.splitOperationID("TaskService_Update_TaskName")
		require.True(t, ok)
		require.Equal(t, "Task", svc)
		require.Equal(t, "Update", action)
		require.Equal(t, "_TaskName", name)
	})

	t.Run("不匹配形态返回false", func(t *testing.T) {
		for _, in := range []string{"", "Foo_", "Foo-Bar_Baz", "_List", "Service_"} {
			_, _, _, ok := c.splitOperationID(in)
			require.False(t, ok, "splitOperationID(%q) 不应匹配", in)
		}
	})
}

// TestConvertCodeByOperationIDNameArm 带 name 段与全小写 action 的完整组合：
// resource 追加 ":"+kebab(name)，小写 action 仍命中 view 归一臂。
func TestConvertCodeByOperationIDNameArm(t *testing.T) {
	c := NewApiPermissionConverter()
	got := c.ConvertCodeByOperationID("TaskService_Update_TaskName")
	require.Equal(t, "task:task-name:edit", got)
	got = c.ConvertCodeByOperationID("Foo_list")
	require.Equal(t, "foo:view", got)
}

func TestSplitCamelAfterFirstWordEdges(t *testing.T) {
	c := NewApiPermissionConverter()
	cases := []struct {
		in          string
		first, rest string
	}{
		{"", "", ""},
		{"lower", "lower", ""},
		{"List", "List", ""},
		{"ListFoo", "List", "Foo"},
	}
	for _, tc := range cases {
		f, r := c.splitCamelAfterFirstWord(tc.in)
		require.Equal(t, tc.first, f, "first(%q)", tc.in)
		require.Equal(t, tc.rest, r, "rest(%q)", tc.in)
	}
}

func TestPathToResourceEdges(t *testing.T) {
	c := NewApiPermissionConverter()
	// 注：首段空白的早退分支经管道不可达——stripVersionPrefix 已做 TrimSpace
	// 归一，空白段在进入 pathToResource 前即被清除。
	require.Equal(t, "", c.pathToResource(""))
	require.Equal(t, "", c.pathToResource("/"))
	// 资源段经 singularizeSegments 单数化（users→user）
	require.Equal(t, "user", c.pathToResource("/api/v1/users"))
}

func TestStripVersionPrefixEdgeCases(t *testing.T) {
	c := NewApiPermissionConverter()
	require.Equal(t, "x", c.stripVersionPrefix("/API/V1/x"), "api/vN 剥离大小写不敏感")
	require.Equal(t, "users", c.stripVersionPrefix("/v1/api/users"), "中段 api 段被过滤")
	require.Equal(t, "", c.stripVersionPrefix("/api"), "仅 api 段归空")
	require.Equal(t, "", c.stripVersionPrefix("   "), "空白归空")
}

// TestConvertCodeByOperationIDActionArms 动作归一化的 create/delete 臂
// （view/edit 臂已由上文与既有测试覆盖）。
func TestConvertCodeByOperationIDActionArms(t *testing.T) {
	c := NewApiPermissionConverter()
	for in, want := range map[string]string{
		"Foo_Create": "foo:create",
		"Foo_add":    "foo:create",
		"Foo_new":    "foo:create",
		"Foo_Delete": "foo:delete",
		"Foo_remove": "foo:delete",
		"Foo_del":    "foo:delete",
	} {
		require.Equal(t, want, c.ConvertCodeByOperationID(in), "ConvertCodeByOperationID(%q)", in)
	}
}
