// Package sseevent 的契约守护测试。
//
// SSE 事件名是跨语言、跨部署单元的线上契约（Go 写进帧的 event: 行，三端前端按
// 字面值注册回调，中间没有任何编译期检查）。本测试只有一个职责：把已发布事件的
// 常量值钉死。谁改了这里的常量值，测试就红——改名的正确姿势是"新增常量 + 三端
// 迁移 + 废弃旧值"，而不是原地改值（原地改值等于静默断流，见包注释）。
package sseevent

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// publishedEvents 是已发布到三端的事件名登记表。新增事件时在此补一行：
// 断言会自动覆盖到（非空、格式、唯一性）；键是 Go 常量名，值是线上契约字面值。
var publishedEvents = map[string]string{
	"Notification": Notification,
	"AIChatChunk":  AIChatChunk,
	"AIChatTool":   AIChatTool,
	"AgentRun":     AgentRun,
}

// eventRe 事件名格式：小写开头的 snake_case。帧里的 event: 行会经反代/日志/前端
// 常量表多处传递，收窄格式能让"手滑打成驼峰/带空格"在 CI 就被拦下。
var eventRe = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// TestPublishedEventContract 钉住已发布事件的字面值与格式。
func TestPublishedEventContract(t *testing.T) {
	require.NotEmpty(t, publishedEvents, "事件登记表不应为空")

	seen := map[string]string{}
	for goName, value := range publishedEvents {
		require.NotEmpty(t, value, "事件 %s 的常量值不应为空", goName)
		require.Regexp(t, eventRe, value,
			"事件 %s 的值 %q 应为 snake_case 字面值", goName, value)

		if prev, dup := seen[value]; dup {
			require.Fail(t, "事件名重复", "%s 与 %s 都注册了 %q——两个域撞同一个事件名会让前端回调收错域的消息", prev, goName, value)
		}
		seen[value] = goName
	}
}

// TestPublishedEventValues 钉死字面值本身：线上三端按这些字符串订阅，
// 任何一个值变化都是破坏性变更（断言消息里写明正确的演进路径）。
func TestPublishedEventValues(t *testing.T) {
	require.Equal(t, "notification", Notification,
		"Notification 的线上契约值已被三端订阅；改值=静默断流，正确做法是新增常量+三端迁移")
	require.Equal(t, "ai_chat_chunk", AIChatChunk,
		"AIChatChunk 的线上契约值已被三端订阅；改值=静默断流")
	require.Equal(t, "ai_chat_tool", AIChatTool,
		"AIChatTool 的线上契约值已被三端订阅；改值=静默断流")
	require.Equal(t, "ai_agent_run", AgentRun,
		"AgentRun 的线上契约值已被三端订阅；改值=静默断流")
}

// 三端 SSE 事件名常量表的仓库相对路径（自本包 backend/pkg/sseevent 起）。
// 前端按字面值注册回调，Go 侧新增事件而漏改某一端 = 该端用户静默收不到推送，
// 且没有任何编译期检查能兜住（前端类型带 `| string` 等于不约束）——
// docs/sse_architecture.md §8 那格"注册表是常量约定、不是运行时校验"由本测试收口。
var frontendEventTables = []string{
	"../../../frontend/admin/react/src/core/transport/sse/event.ts",
	"../../../frontend/admin/vue-element/src/core/transport/sse/event.ts",
	"../../../frontend/admin/vue-vben/src/transport/sse/event.ts",
}

// TestPublishedEventsReachAllFrontends 源扫描守护：publishedEvents 登记的每个
// 字面值必须出现在三端事件名常量表里，漏一端即失败。新增事件的操作顺序因此
// 被钉成"Go 常量 + publishedEvents 登记行 + 三端 event.ts 各一行"，少一步就红。
func TestPublishedEventsReachAllFrontends(t *testing.T) {
	for _, rel := range frontendEventTables {
		path := filepath.Join(thisDir(t), rel)
		src, err := os.ReadFile(path)
		require.NoError(t, err, "应能读取前端事件表 %s——若文件被移动，请同步更新 frontendEventTables", rel)
		for value, goName := range publishedEvents {
			require.Contains(t, string(src), value,
				"事件 %s（%q）未在前端事件表 %s 登记——该端用户将静默收不到此推送", goName, value, rel)
		}
	}
}

func thisDir(t testing.TB) string {
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller 应能取到本测试文件路径")
	return filepath.Dir(file)
}
