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

// scriptTaskMaxTimeoutSecs 脚本任务 timeout_secs 上限。
// 任务执行全程持引擎全局 execMu（与实体钩子共用一把锁），脚本自报
// timeout_secs=86400 即可让单任务占锁一天，冻结平台全部实体写路径
//（before 钩子同步跑在业务写事务前）。注册与消费两处都钳制。
const scriptTaskMaxTimeoutSecs = 300

// TaskHandlerRegistry stores Lua-based task handlers
type TaskHandlerRegistry struct {
	handlers map[string]*LuaTaskHandler
	mu       sync.RWMutex
	logger   *bLogger.Helper
	engine   VMManager
}

// VMManager provides VM management operations
type VMManager interface {
	MarkVMDedicated(L *lua.LState)
}

// LuaTaskHandler represents a task handler registered from Lua
type LuaTaskHandler struct {
	Name        string
	Description string
	Function    *lua.LFunction
	L           *lua.LState
	Required    []string
	Optional    map[string]interface{}
	TimeoutSecs int // Timeout in seconds (default: 30)
	MaxRetries  int // Max retry attempts (default: 2)
	Priority    int // Task priority (default: 5 = normal)
	Generation  int64
}

var globalTaskRegistry = &TaskHandlerRegistry{
	handlers: make(map[string]*LuaTaskHandler),
}

// taskHandlerGeneration 注册代际计数器：Resync 前取号，重载完成后清理旧代际的处理器，
// 使已禁用/删除脚本的处理器不残留。
var taskHandlerGeneration int64

// NextTaskGeneration 取下一个代际号。
func NextTaskGeneration() int64 {
	return atomic.AddInt64(&taskHandlerGeneration, 1)
}

// PruneStaleTaskHandlers 清理代际早于 generation 的任务处理器
// （它们所属的脚本在本轮全量重同步中未被重新注册，即已禁用或已删除）。
func PruneStaleTaskHandlers(generation int64) int {
	globalTaskRegistry.mu.Lock()
	defer globalTaskRegistry.mu.Unlock()

	removed := 0
	for name, h := range globalTaskRegistry.handlers {
		if h.Generation < generation {
			delete(globalTaskRegistry.handlers, name)
			removed++
		}
	}
	if removed > 0 && globalTaskRegistry.logger != nil {
		globalTaskRegistry.logger.Infof(context.Background(), "Pruned %d stale task handlers (generation < %d)", removed, generation)
	}
	return removed
}

// RegisterTask registers the task API for Lua scripts
// RegisterTask registers the task API for Lua scripts (direct LState style).
func RegisterTask(L *lua.LState, engine VMManager, logger *bLogger.Helper) {
	globalTaskRegistry.logger = logger
	globalTaskRegistry.engine = engine

	loader := LoaderTask(engine, logger)
	taskModule := buildTaskModule(L, loader)

	// Register module
	L.SetGlobal("task", taskModule)

	// Also make it available via require('task')
	L.PreloadModule("task", func(L *lua.LState) int {
		L.Push(taskModule)
		return 1
	})

	logger.Info(context.Background(), "✅ Task API registered, task.register_handler() is now available")
}

// LoaderTask 返回 task 模块的 loader，供 go-scripts 引擎 RegisterModule 使用。
// engine 为 nil 时返回空模块。
//
// 注意：loader 必须把模块表压栈并返回 1——此前漏 Push 导致 require "task"
// 拿到栈上残留的模块名字符串（"attempt to call a non-function object"）。
func LoaderTask(engine VMManager, logger *bLogger.Helper) lua.LGFunction {
	globalTaskRegistry.logger = logger
	globalTaskRegistry.engine = engine

	return func(L *lua.LState) int {
		if engine == nil {
			L.Push(L.NewTable())
			return 1
		}
		L.Push(buildTaskModule(L, nil))
		return 1
	}
}

// buildTaskModule 构建 task 模块并压入栈顶。
// secondLoader 为 nil 时仅构建模块；非 nil 时用于 require 注册（兼容旧 SetGlobal 路径）。
func buildTaskModule(L *lua.LState, secondLoader lua.LGFunction) *lua.LTable {
	taskModule := L.NewTable()
	taskModule.RawSetString("register_handler", L.NewFunction(registerTaskHandler))
	return taskModule
}

// registerTaskHandler is the Lua API function to register a task handler
// Usage:
//
//	task.register_handler("my_handler", "Description", function(ctx)
//	  -- handler logic
//	  return true
//	end, {
//	  required = {"field1", "field2"},
//	  optional = {field3 = "default", field4 = 123}
//	})
func registerTaskHandler(L *lua.LState) int {
	// Get arguments
	name := L.CheckString(1)
	description := L.CheckString(2)
	handlerFunc := L.CheckFunction(3)
	options := L.OptTable(4, L.NewTable())

	// Extract required fields
	var required []string
	if reqTable := options.RawGetString("required"); reqTable.Type() == lua.LTTable {
		reqTable.(*lua.LTable).ForEach(func(k, v lua.LValue) {
			if v.Type() == lua.LTString {
				required = append(required, v.String())
			}
		})
	}

	// Extract optional fields
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

	// Extract execution configuration
	timeoutSecs := 30 // Default: 30 seconds
	if timeout := options.RawGetString("timeout_secs"); timeout.Type() == lua.LTNumber {
		timeoutSecs = int(timeout.(lua.LNumber))
	}
	// P1 修复：上限钳制——execMu 全局锁被长任务占住会冻结全部实体写路径
	if timeoutSecs > scriptTaskMaxTimeoutSecs {
		timeoutSecs = scriptTaskMaxTimeoutSecs
	}

	maxRetries := 2 // Default: 2 retries
	if retries := options.RawGetString("max_retries"); retries.Type() == lua.LTNumber {
		maxRetries = int(retries.(lua.LNumber))
	}

	priority := 5 // Default: 5 (normal priority)
	if prio := options.RawGetString("priority"); prio.Type() == lua.LTNumber {
		priority = int(prio.(lua.LNumber))
	}

	// Create handler
	handler := &LuaTaskHandler{
		Name:        name,
		Description: description,
		Function:    handlerFunc,
		L:           L,
		Required:    required,
		Optional:    optional,
		TimeoutSecs: timeoutSecs,
		MaxRetries:  maxRetries,
		Priority:    priority,
		Generation:  atomic.AddInt64(&taskHandlerGeneration, 1),
	}

	// Register globally
	globalTaskRegistry.mu.Lock()
	globalTaskRegistry.handlers[name] = handler
	globalTaskRegistry.mu.Unlock()

	// Mark the VM as dedicated so it won't be returned to the pool
	// This ensures the handler function remains available for execution
	if globalTaskRegistry.engine != nil {
		globalTaskRegistry.engine.MarkVMDedicated(L)
		if globalTaskRegistry.logger != nil {
			globalTaskRegistry.logger.Debugf(context.Background(), "VM marked as dedicated for task handler: %s", name)
		}
	}

	if globalTaskRegistry.logger != nil {
		globalTaskRegistry.logger.Infof(context.Background(), "📝 Registered Lua task handler: %s (timeout: %ds, retries: %d, priority: %d)",
			name, timeoutSecs, maxRetries, priority)
	}

	return 0
}

// GetRegisteredHandlers returns all registered Lua task handlers
func GetRegisteredHandlers() map[string]*LuaTaskHandler {
	globalTaskRegistry.mu.RLock()
	defer globalTaskRegistry.mu.RUnlock()

	handlers := make(map[string]*LuaTaskHandler, len(globalTaskRegistry.handlers))
	for name, h := range globalTaskRegistry.handlers {
		handlers[name] = h
	}
	return handlers
}

// GetHandler returns a specific Lua task handler
func GetHandler(name string) (*LuaTaskHandler, bool) {
	globalTaskRegistry.mu.RLock()
	defer globalTaskRegistry.mu.RUnlock()

	handler, exists := globalTaskRegistry.handlers[name]
	return handler, exists
}

// InvokeHandler 执行指定的任务处理器：合并 params（可选参数取默认值、校验必填项），
// 在处理器声明的超时内以 params 表为唯一参数调用脚本函数。
// 调用方须自行保证与该处理器所属引擎的执行互斥（编排器的 ExecuteTaskHandler 已加锁）。
func InvokeHandler(ctx context.Context, name string, params map[string]any) error {
	handler, exists := GetHandler(name)
	if !exists {
		return fmt.Errorf("task handler not found: %s", name)
	}

	// 合并可选参数默认值 + 校验必填
	merged := make(map[string]any, len(params)+len(handler.Optional))
	for k, v := range handler.Optional {
		merged[k] = v
	}
	var missing []string
	for k, v := range params {
		merged[k] = v
	}
	for _, req := range handler.Required {
		if _, ok := merged[req]; !ok {
			missing = append(missing, req)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("task handler %s missing required params: %v", name, missing)
	}

	timeoutSecs := handler.TimeoutSecs
	if timeoutSecs <= 0 {
		timeoutSecs = 30
	}
	// P1 修复：消费侧同样钳制（直接改库/旧注册行可能带着超限值）
	if timeoutSecs > scriptTaskMaxTimeoutSecs {
		timeoutSecs = scriptTaskMaxTimeoutSecs
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSecs)*time.Second)
	defer cancel()

	L := handler.L
	L.SetContext(timeoutCtx)
	defer L.SetContext(context.Background())

	L.Push(handler.Function)
	L.Push(convert.ToLuaValue(L, merged))

	if err := L.PCall(1, 0, nil); err != nil {
		return fmt.Errorf("task handler %s execution error: %w", name, err)
	}
	return nil
}
