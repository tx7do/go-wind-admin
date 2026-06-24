package api

import (
	"github.com/go-kratos/kratos/v2/log"
	lua "github.com/yuin/gopher-lua"
)

// TaskHandlerRegistry stores Lua-based task handlers
type TaskHandlerRegistry struct {
	handlers map[string]*LuaTaskHandler
	logger   *log.Helper
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
}

var globalTaskRegistry = &TaskHandlerRegistry{
	handlers: make(map[string]*LuaTaskHandler),
}

// RegisterTask registers the task API for Lua scripts
// RegisterTask registers the task API for Lua scripts (direct LState style).
func RegisterTask(L *lua.LState, engine VMManager, logger *log.Helper) {
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

	logger.Info("✅ Task API registered, task.register_handler() is now available")
}

// LoaderTask 返回 task 模块的 loader，供 go-scripts 引擎 RegisterModule 使用。
// engine 为 nil 时返回空模块。
func LoaderTask(engine VMManager, logger *log.Helper) lua.LGFunction {
	globalTaskRegistry.logger = logger
	globalTaskRegistry.engine = engine

	return func(L *lua.LState) int {
		if engine == nil {
			L.Push(L.NewTable())
			return 1
		}
		buildTaskModule(L, nil)
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
	}

	// Register globally
	globalTaskRegistry.handlers[name] = handler

	// Mark the VM as dedicated so it won't be returned to the pool
	// This ensures the handler function remains available for execution
	if globalTaskRegistry.engine != nil {
		globalTaskRegistry.engine.MarkVMDedicated(L)
		if globalTaskRegistry.logger != nil {
			globalTaskRegistry.logger.Debugf("VM marked as dedicated for task handler: %s", name)
		}
	}

	if globalTaskRegistry.logger != nil {
		globalTaskRegistry.logger.Infof("📝 Registered Lua task handler: %s (timeout: %ds, retries: %d, priority: %d)",
			name, timeoutSecs, maxRetries, priority)
	}

	return 0
}

// GetRegisteredHandlers returns all registered Lua task handlers
func GetRegisteredHandlers() map[string]*LuaTaskHandler {
	if globalTaskRegistry.logger != nil {
		globalTaskRegistry.logger.Infof("📋 GetRegisteredHandlers called: %d handlers available", len(globalTaskRegistry.handlers))
		for name := range globalTaskRegistry.handlers {
			globalTaskRegistry.logger.Infof("  - %s", name)
		}
	}
	return globalTaskRegistry.handlers
}

// GetHandler returns a specific Lua task handler
func GetHandler(name string) (*LuaTaskHandler, bool) {
	handler, exists := globalTaskRegistry.handlers[name]
	return handler, exists
}
