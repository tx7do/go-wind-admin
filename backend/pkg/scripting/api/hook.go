package api

import (
	"github.com/go-kratos/kratos/v2/log"
	lua "github.com/yuin/gopher-lua"
)

// HookEngine defines the interface for hook management
// This avoids circular dependency with the main lua package
type HookEngine interface {
	RegisterHook(name, description string) error
	AddScript(hookName string, script interface{}) error
	ListHooks() []string
	RegisterCallback(hookName string, L *lua.LState, fn *lua.LFunction)
}

// Script represents a Lua script for the hook engine
type Script struct {
	ID          string
	Name        string
	Hook        string
	Source      string
	Enabled     bool
	Priority    int
	Description string
	Version     string
	Author      string
	Critical    bool
}

// RegisterHookAPI registers the hook management API for Lua as a requireable module
// This allows Lua scripts to register themselves to hooks dynamically
func RegisterHookAPI(L *lua.LState, engine HookEngine, logger *log.Helper) {
	// Register in package.preload so it can be required
	L.PreloadModule("kratos_hook", LoaderHook(engine, logger))
}

// LoaderHook 返回 hook 模块（kratos_hook）的 loader，供 go-scripts 引擎 RegisterModule 使用。
// engine 为 nil 时返回空模块。
func LoaderHook(engine HookEngine, logger *log.Helper) lua.LGFunction {
	return func(L *lua.LState) int {
		if engine == nil {
			L.Push(L.NewTable())
			return 1
		}
		// Create hook module
		hookModule := L.NewTable()

		// hook.register(hook_name, description, callback_function)
		// Register a new hook point and optionally attach a callback
		hookModule.RawSetString("register", L.NewFunction(func(L *lua.LState) int {
			hookName := L.CheckString(1)
			description := L.OptString(2, "")
			callback := L.Get(3) // Optional callback function

			// Register the hook (ignore error if already registered)
			err := engine.RegisterHook(hookName, description)
			if err != nil {
				// Only log error, don't return - we still want to register the callback
				logger.Debugf("hook.register: %v (continuing to register callback)", err)
			} else {
				logger.Debugf("Registered hook: %s", hookName)
			}

			// If callback provided, register it directly with the engine
			if callback != lua.LNil {
				if fn, ok := callback.(*lua.LFunction); ok {
					// Register the callback function with the engine
					// The engine will store the LState and function for later execution
					engine.RegisterCallback(hookName, L, fn)
					logger.Debugf("Registered callback for hook: %s", hookName)
				} else {
					logger.Warnf("Third argument is not a function, ignoring callback")
				}
			}

			L.Push(lua.LBool(true))
			return 1
		}))

		// hook.add_script(hook_name, script_table)
		// Add a script to a hook dynamically
		// script_table: {name, source, enabled, priority, description}
		hookModule.RawSetString("add_script", L.NewFunction(func(L *lua.LState) int {
			hookName := L.CheckString(1)
			scriptTable := L.CheckTable(2)

			// Extract script properties
			name := scriptTable.RawGetString("name").String()
			source := scriptTable.RawGetString("source").String()

			enabled := true
			if enabledVal := scriptTable.RawGetString("enabled"); enabledVal != lua.LNil {
				enabled = lua.LVAsBool(enabledVal)
			}

			priority := 0
			if priorityVal := scriptTable.RawGetString("priority"); priorityVal != lua.LNil {
				if num, ok := priorityVal.(lua.LNumber); ok {
					priority = int(num)
				}
			}

			description := ""
			if descVal := scriptTable.RawGetString("description"); descVal != lua.LNil {
				description = descVal.String()
			}

			// Create script
			script := Script{
				Name:        name,
				Hook:        hookName,
				Source:      source,
				Enabled:     enabled,
				Priority:    priority,
				Description: description,
			}

			err := engine.AddScript(hookName, script)
			if err != nil {
				logger.Errorf("hook.add_script error: %v", err)
				L.Push(lua.LBool(false))
				L.Push(lua.LString(err.Error()))
				return 2
			}

			logger.Debugf("Added script '%s' to hook: %s", name, hookName)
			L.Push(lua.LBool(true))
			return 1
		}))

		// hook.list()
		// Get list of all registered hooks
		hookModule.RawSetString("list", L.NewFunction(func(L *lua.LState) int {
			hooks := engine.ListHooks()

			// ConvertCode to Lua table (array)
			table := L.NewTable()
			for i, hookName := range hooks {
				table.RawSetInt(i+1, lua.LString(hookName))
			}

			L.Push(table)
			return 1
		}))

		L.Push(hookModule)
		return 1
	}
}
