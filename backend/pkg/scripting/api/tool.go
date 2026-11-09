package api

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	lua "github.com/yuin/gopher-lua"

	"github.com/tx7do/go-scripts/lua/convert"
)

// scriptToolMaxTimeoutSecs 脚本工具 timeout_secs 上限（与任务处理器同一把全局
// execMu 锁，长工具会占锁冻结全部实体写路径）。注册与调用两处都钳制。
const scriptToolMaxTimeoutSecs = 300

// ToolHandlerRegistry stores script-registered LLM tools（工具市场脚本侧注册表）。
// 生命周期与任务处理器同构：脚本经 tool.register 自注册 → Resync 取代际号 →
// PruneStaleToolHandlers 清旧代际；消费侧经 InvokeToolHandler 执行。
type ToolHandlerRegistry struct {
	handlers map[string]*LuaToolHandler
	mu       sync.RWMutex
	logger   *bLogger.Helper
	engine   VMManager
}

// LuaToolHandler represents an LLM tool registered from a script
type LuaToolHandler struct {
	Name        string
	Description string
	Function    *lua.LFunction
	L           *lua.LState
	Required    []string               // 必填参数名（供 LLM parameters schema）
	Optional    map[string]interface{} // 可选参数名 → 类型描述（"string"/"number"/"boolean"）
	TimeoutSecs int
	Generation  int64
}

var globalToolRegistry = &ToolHandlerRegistry{
	handlers: make(map[string]*LuaToolHandler),
}

// toolHandlerGeneration 注册代际计数器（与任务处理器同构）。
var toolHandlerGeneration int64

// NextToolGeneration 取下一个工具代际号（Resync 前调用）。
func NextToolGeneration() int64 {
	return atomic.AddInt64(&toolHandlerGeneration, 1)
}

// PruneStaleToolHandlers 清理代际早于 generation 的脚本工具
// （所属脚本在本轮全量重同步中未被重新注册，即已禁用或已删除）。
func PruneStaleToolHandlers(generation int64) int {
	globalToolRegistry.mu.Lock()
	defer globalToolRegistry.mu.Unlock()

	removed := 0
	for name, h := range globalToolRegistry.handlers {
		if h.Generation < generation {
			delete(globalToolRegistry.handlers, name)
			removed++
		}
	}
	if removed > 0 && globalToolRegistry.logger != nil {
		globalToolRegistry.logger.Infof(context.Background(), "Pruned %d stale script tools (generation < %d)", removed, generation)
	}
	return removed
}

// RegisterTool registers the tool API for Lua scripts (SetGlobal style).
func RegisterTool(L *lua.LState, engine VMManager, logger *bLogger.Helper) {
	globalToolRegistry.logger = logger
	globalToolRegistry.engine = engine

	toolModule := L.NewTable()
	toolModule.RawSetString("register", L.NewFunction(registerToolHandler))

	L.SetGlobal("tool", toolModule)
	L.PreloadModule("tool", func(L *lua.LState) int {
		L.Push(toolModule)
		return 1
	})

	logger.Info(context.Background(), "✅ Tool API registered, tool.register() is now available")
}

// LoaderTool 返回 tool 模块的 loader，供 go-scripts 引擎 RegisterModule 使用。
func LoaderTool(engine VMManager, logger *bLogger.Helper) lua.LGFunction {
	globalToolRegistry.logger = logger
	globalToolRegistry.engine = engine

	return func(L *lua.LState) int {
		L.Push(buildToolModule(L))
		return 1
	}
}

// buildToolModule 构建 tool 模块并压入栈顶。
func buildToolModule(L *lua.LState) *lua.LTable {
	toolModule := L.NewTable()
	toolModule.RawSetString("register", L.NewFunction(registerToolHandler))
	return toolModule
}

// registerToolHandler is the Lua API function to register an LLM tool.
// Usage:
//
//	tool.register("query_orders", "按条件查询订单", function(args)
//	  -- args 是 LLM 给出的参数表
//	  return "结果文本"   -- 返回值（字符串/表）会作为工具结果回给模型
//	end, {
//	  required = {"orderNo"},
//	  optional = {status = "string"},
//	  timeout_secs = 30
//	})
func registerToolHandler(L *lua.LState) int {
	name := L.CheckString(1)
	description := L.CheckString(2)
	handlerFunc := L.CheckFunction(3)
	options := L.OptTable(4, L.NewTable())

	var required []string
	if reqTable := options.RawGetString("required"); reqTable.Type() == lua.LTTable {
		reqTable.(*lua.LTable).ForEach(func(k, v lua.LValue) {
			if v.Type() == lua.LTString {
				required = append(required, v.String())
			}
		})
	}

	optional := make(map[string]interface{})
	if optTable := options.RawGetString("optional"); optTable.Type() == lua.LTTable {
		optTable.(*lua.LTable).ForEach(func(k, v lua.LValue) {
			key := k.String()
			switch v.Type() {
			case lua.LTString:
				optional[key] = v.String()
			case lua.LTNumber:
				optional[key] = float64(v.(lua.LNumber))
			case lua.LTBool:
				optional[key] = bool(v.(lua.LBool))
			default:
				optional[key] = v.String()
			}
		})
	}

	timeoutSecs := 30
	if timeout := options.RawGetString("timeout_secs"); timeout.Type() == lua.LTNumber {
		timeoutSecs = int(timeout.(lua.LNumber))
	}
	if timeoutSecs > scriptToolMaxTimeoutSecs {
		timeoutSecs = scriptToolMaxTimeoutSecs
	}

	handler := &LuaToolHandler{
		Name:        name,
		Description: description,
		Function:    handlerFunc,
		L:           L,
		Required:    required,
		Optional:    optional,
		TimeoutSecs: timeoutSecs,
		Generation:  atomic.AddInt64(&toolHandlerGeneration, 1),
	}

	globalToolRegistry.mu.Lock()
	globalToolRegistry.handlers[name] = handler
	globalToolRegistry.mu.Unlock()

	if globalToolRegistry.engine != nil {
		globalToolRegistry.engine.MarkVMDedicated(L)
	}

	if globalToolRegistry.logger != nil {
		globalToolRegistry.logger.Infof(context.Background(),
			"🔧 Registered script tool: %s (required: %v, optional: %v, timeout: %ds)",
			name, required, optional, timeoutSecs)
	}

	return 0
}

// GetRegisteredTools returns all registered script tools
func GetRegisteredTools() map[string]*LuaToolHandler {
	globalToolRegistry.mu.RLock()
	defer globalToolRegistry.mu.RUnlock()

	handlers := make(map[string]*LuaToolHandler, len(globalToolRegistry.handlers))
	for name, h := range globalToolRegistry.handlers {
		handlers[name] = h
	}
	return handlers
}

// GetTool returns a specific script tool
func GetTool(name string) (*LuaToolHandler, bool) {
	globalToolRegistry.mu.RLock()
	defer globalToolRegistry.mu.RUnlock()

	handler, exists := globalToolRegistry.handlers[name]
	return handler, exists
}

// InvokeToolHandler 执行指定的脚本工具：args 表为唯一参数，函数返回值
// （经 convert.ToGoValue 收口）作为工具结果回给调用方（LLM 工具循环）。
// 调用方须自行保证与该工具所属引擎的执行互斥（engine.ExecuteToolHandler 已加锁）。
func InvokeToolHandler(ctx context.Context, name string, args map[string]any) (any, error) {
	handler, exists := GetTool(name)
	if !exists {
		return nil, fmt.Errorf("script tool not found: %s", name)
	}

	merged := make(map[string]any, len(args)+len(handler.Optional))
	for k, v := range handler.Optional {
		merged[k] = v
	}
	for k, v := range args {
		merged[k] = v
	}
	for _, req := range handler.Required {
		if _, ok := merged[req]; !ok {
			return nil, fmt.Errorf("script tool %s missing required param: %s", name, req)
		}
	}

	timeoutSecs := handler.TimeoutSecs
	if timeoutSecs <= 0 {
		timeoutSecs = 30
	}
	if timeoutSecs > scriptToolMaxTimeoutSecs {
		timeoutSecs = scriptToolMaxTimeoutSecs
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSecs)*time.Second)
	defer cancel()

	L := handler.L
	L.SetContext(timeoutCtx)
	defer L.SetContext(context.Background())

	L.Push(handler.Function)
	L.Push(convert.ToLuaValue(L, merged))

	if err := L.PCall(1, 1, nil); err != nil {
		return nil, fmt.Errorf("script tool %s execution error: %w", name, err)
	}

	ret := L.Get(-1)
	L.Pop(1)
	return convert.ToGoValue(ret), nil
}
