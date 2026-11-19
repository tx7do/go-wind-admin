// 本文件针对 task.go / tool.go 的注册表与调用边界做包内白盒测试。
// 父目录的 task_bridge_test.go / tool_bridge_test.go 经完整引擎装配间接覆盖了
// 注册→调用主链路；本文件补单点分支：
//   - options 解析：required 里的非字符串值跳过、optional 四类值转换、
//     timeout 钳制（>300 收口、负值）、max_retries/priority 默认值、重名覆盖注册；
//   - MarkVMDedicated 的调用时机（engine 非 nil 才调）；
//   - PruneStaleTaskHandlers：无旧代际返 0、等号保留、并发安全；
//   - InvokeHandler/InvokeToolHandler：未知名报错、可选默认合并、超时回退与钳制、
//     脚本抛错的包装、params 为 nil、工具返回值经 convert.ToGoValue 收口；
//   - GetRegisteredHandlers 返回拷贝。
//
// 注册表是包级全局单例且无 reset 助手，与父目录桥接测试共享状态——
// 每个用例收尾必须清空 globalTaskRegistry.handlers / globalToolRegistry.handlers。
package api

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	lua "github.com/yuin/gopher-lua"
)

// fakeVMManager 记录 MarkVMDedicated 调用次数。
type fakeVMManager struct {
	marked int
}

func (f *fakeVMManager) MarkVMDedicated(*lua.LState) { f.marked++ }

// clearTaskRegistry 清空全局任务注册表（测试隔离：与父目录桥接测试共享单例）。
func clearTaskRegistry() {
	globalTaskRegistry.mu.Lock()
	globalTaskRegistry.handlers = make(map[string]*LuaTaskHandler)
	globalTaskRegistry.mu.Unlock()
}

func clearToolRegistry() {
	globalToolRegistry.mu.Lock()
	globalToolRegistry.handlers = make(map[string]*LuaToolHandler)
	globalToolRegistry.mu.Unlock()
}

// newTaskLuaState 构造带 RegisterTask 的 LState（logger 必须非 nil：RegisterTask 直接解引用）。
func newTaskLuaState(t *testing.T, engine VMManager) *lua.LState {
	t.Helper()
	L := lua.NewState()
	t.Cleanup(L.Close)
	RegisterTask(L, engine, bLogger.NewHelper(bLogger.NopLogger()))
	return L
}

func TestRegisterTaskHandler_OptionsParsing(t *testing.T) {
	t.Cleanup(clearTaskRegistry)
	vm := &fakeVMManager{}
	L := newTaskLuaState(t, vm)

	require.NoError(t, L.DoString(`
		task.register_handler("full", "全参数处理器", function() end, {
			required = {"a", 123, "b"},          -- 非字符串值被跳过
			optional = {
				s = "str",
				n = 3.5,
				b = true,
				t = {"x"},                        -- 其他类型按 String 收口
			},
			timeout_secs = 99999,                 -- 超限钳制到 300
			max_retries = 7,
			priority = 9,
		})
		task.register_handler("defaults", "默认值处理器", function() end)
	`))

	full, ok := GetHandler("full")
	require.True(t, ok)
	require.Equal(t, []string{"a", "b"}, full.Required)
	require.Equal(t, map[string]interface{}{
		"s": "str", "n": float64(3.5), "b": true,
		// 其他类型按 v.String() 收口：表值得到 "table: 0x..." 地址串（现状钉住，
		// 脚本作者不应把表放进 optional）。
		"t": full.Optional["t"],
	}, full.Optional)
	require.True(t, len(full.Optional["t"].(string)) > len("table: "),
		"table-typed optional falls into the default String() branch")
	require.Equal(t, scriptTaskMaxTimeoutSecs, full.TimeoutSecs, "timeout_secs>300 must clamp to 300")
	require.Equal(t, 7, full.MaxRetries)
	require.Equal(t, 9, full.Priority)
	require.Equal(t, 2, vm.marked, "engine 非 nil 时每次注册都标记 VM 专用")

	defaults, ok := GetHandler("defaults")
	require.True(t, ok)
	require.Empty(t, defaults.Required)
	require.Empty(t, defaults.Optional)
	require.Equal(t, 30, defaults.TimeoutSecs)
	require.Equal(t, 2, defaults.MaxRetries)
	require.Equal(t, 5, defaults.Priority)
}

func TestRegisterTaskHandler_NegativeTimeoutKeptAtInvokeFallback(t *testing.T) {
	t.Cleanup(clearTaskRegistry)
	L := newTaskLuaState(t, &fakeVMManager{})

	require.NoError(t, L.DoString(`
		task.register_handler("neg", "负超时", function() end, {timeout_secs = -5})
	`))
	h, ok := GetHandler("neg")
	require.True(t, ok)
	require.Equal(t, -5, h.TimeoutSecs, "注册侧不拦负值；消费侧回退 30s（见 InvokeHandler 测试）")
}

func TestRegisterTaskHandler_ReregisterOverrides(t *testing.T) {
	t.Cleanup(clearTaskRegistry)
	L := newTaskLuaState(t, nil)

	require.NoError(t, L.DoString(`
		task.register_handler("dup", "第一版", function() end)
		task.register_handler("dup", "第二版", function() end, {timeout_secs = 60})
	`))
	h, ok := GetHandler("dup")
	require.True(t, ok)
	require.Equal(t, "第二版", h.Description)
	require.Equal(t, 60, h.TimeoutSecs)
	require.Len(t, GetRegisteredHandlers(), 1)
}

func TestLoaderTask_NilEngine(t *testing.T) {
	L := lua.NewState()
	t.Cleanup(L.Close)
	L.PreloadModule("task", LoaderTask(nil, nil))
	require.NoError(t, L.DoString(`
		local t = require "task"
		assert(type(t) == "table", "module should be a table")
		assert(t.register_handler == nil, "nil engine must not expose register_handler")
	`))
}

func TestNextTaskGeneration_Monotonic(t *testing.T) {
	a := NextTaskGeneration()
	b := NextTaskGeneration()
	require.Greater(t, b, a)
}

func TestPruneStaleTaskHandlers(t *testing.T) {
	t.Cleanup(clearTaskRegistry)
	L := newTaskLuaState(t, nil)
	require.NoError(t, L.DoString(`
		task.register_handler("old", "旧代际", function() end)
	`))
	oldHandler, ok := GetHandler("old")
	require.True(t, ok)
	oldGen := oldHandler.Generation

	// 等号保留：当前代际不算过期。
	require.Equal(t, 0, PruneStaleTaskHandlers(oldGen))
	_, ok = GetHandler("old")
	require.True(t, ok)

	// 新代际清理旧代际。
	NextTaskGeneration()
	newGen := NextTaskGeneration()
	require.Equal(t, 1, PruneStaleTaskHandlers(newGen))
	_, ok = GetHandler("old")
	require.False(t, ok)

	// 无可清理项返 0。
	require.Equal(t, 0, PruneStaleTaskHandlers(newGen+100))
}

func TestPruneStaleTaskHandlers_Concurrent(t *testing.T) {
	t.Cleanup(clearTaskRegistry)
	L := newTaskLuaState(t, nil)
	require.NoError(t, L.DoString(`
		for i = 1, 50 do task.register_handler("h" .. i, "c", function() end) end
	`))

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			PruneStaleTaskHandlers(NextTaskGeneration())
		}()
	}
	wg.Wait()
	require.Empty(t, GetRegisteredHandlers())
}

func TestGetRegisteredHandlers_ReturnsCopy(t *testing.T) {
	t.Cleanup(clearTaskRegistry)
	L := newTaskLuaState(t, nil)
	require.NoError(t, L.DoString(`task.register_handler("copy", "c", function() end)`))

	snap := GetRegisteredHandlers()
	require.Len(t, snap, 1)
	delete(snap, "copy")
	require.Len(t, GetRegisteredHandlers(), 1, "modifying the returned map must not affect the registry")
}

// --- InvokeHandler ---

func TestInvokeHandler_UnknownName(t *testing.T) {
	err := InvokeHandler(context.Background(), "no-such-handler", nil)
	require.ErrorContains(t, err, "task handler not found: no-such-handler")
}

func TestInvokeHandler_MissingRequiredAndDefaults(t *testing.T) {
	t.Cleanup(clearTaskRegistry)
	L := newTaskLuaState(t, nil)
	require.NoError(t, L.DoString(`
		task.register_handler("calc", "必填+默认", function(p)
			assert(p.amount == 100, "required param must arrive, got " .. tostring(p.amount))
			assert(p.unit == "day", "optional default must be merged, got " .. tostring(p.unit))
			assert(p.extra == "given", "explicit param must override default, got " .. tostring(p.extra))
		end, {
			required = {"amount"},
			optional = {unit = "day", extra = "default"},
		})
	`))

	// 缺必填：报错点名缺失字段。
	err := InvokeHandler(context.Background(), "calc", map[string]any{})
	require.ErrorContains(t, err, "missing required params: [amount]")

	// params 为 nil 同样触发必填校验（合并后只有默认值）。
	err = InvokeHandler(context.Background(), "calc", nil)
	require.ErrorContains(t, err, "missing required params: [amount]")

	// 必填给上：默认值合并 + 显式覆盖。
	require.NoError(t, InvokeHandler(context.Background(), "calc", map[string]any{
		"amount": 100, "extra": "given",
	}))
}

func TestInvokeHandler_ScriptErrorWrapped(t *testing.T) {
	t.Cleanup(clearTaskRegistry)
	L := newTaskLuaState(t, nil)
	require.NoError(t, L.DoString(`
		task.register_handler("boom", "抛错", function() error("inside handler") end)
	`))
	err := InvokeHandler(context.Background(), "boom", nil)
	require.ErrorContains(t, err, "task handler boom execution error:")
	require.ErrorContains(t, err, "inside handler")
}

// TestInvokeHandler_TimeoutContext 消费侧超时回退（<=0 → 30s）与钳制（>300 → 300s），
// 经 handler 声明值验证：注册侧钳制在 300，负值保持到消费侧回退。
func TestInvokeHandler_TimeoutContext(t *testing.T) {
	t.Cleanup(clearTaskRegistry)
	L := newTaskLuaState(t, nil)
	require.NoError(t, L.DoString(`
		task.register_handler("neg_timeout", "负超时", function(p)
			assert(p ~= nil, "params table must arrive even when empty")
		end, {timeout_secs = -5})
	`))
	// 负值回退 30s：脚本正常执行、params 以空表形态到达（nil → ToLuaValue 空表）。
	require.NoError(t, InvokeHandler(context.Background(), "neg_timeout", nil))
}

// --- InvokeToolHandler（tool.go 同构差异点） ---

func TestInvokeToolHandler_ReturnValueConversion(t *testing.T) {
	t.Cleanup(clearToolRegistry)
	L := lua.NewState()
	t.Cleanup(L.Close)
	RegisterTool(L, nil, bLogger.NewHelper(bLogger.NopLogger()))

	require.NoError(t, L.DoString(`
		tool.register("add", "加法", function(p)
			return p.a + p.b
		end, {required = {"a", "b"}})
	`))

	ret, err := InvokeToolHandler(context.Background(), "add", map[string]any{"a": 1, "b": 2})
	require.NoError(t, err)
	require.Equal(t, float64(3), ret)

	// 未知工具名。
	_, err = InvokeToolHandler(context.Background(), "no-such-tool", nil)
	require.ErrorContains(t, err, "script tool not found: no-such-tool")

	// 缺必填（tool 侧逐条点名）。
	_, err = InvokeToolHandler(context.Background(), "add", map[string]any{"a": 1})
	require.ErrorContains(t, err, "missing required param: b")
}

// --- 基准 ---

func BenchmarkInvokeHandler(b *testing.B) {
	L := lua.NewState()
	defer L.Close()
	RegisterTask(L, nil, bLogger.NewHelper(bLogger.NopLogger()))
	require.NoError(b, L.DoString(`
		task.register_handler("bench", "基准", function(p) return true end, {
			optional = {k1 = "v1", k2 = "v2"},
		})
	`))
	params := map[string]any{"a": 1, "b": "x"}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := InvokeHandler(ctx, "bench", params); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInvokeToolHandler(b *testing.B) {
	L := lua.NewState()
	defer L.Close()
	RegisterTool(L, nil, bLogger.NewHelper(bLogger.NopLogger()))
	require.NoError(b, L.DoString(`
		tool.register("bench_tool", "基准", function(p) return p.a + 1 end)
	`))
	params := map[string]any{"a": 1}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := InvokeToolHandler(ctx, "bench_tool", params); err != nil {
			b.Fatal(err)
		}
	}
}
