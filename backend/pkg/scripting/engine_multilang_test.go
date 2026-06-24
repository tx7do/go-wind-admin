package scripting

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/log"

	gsEngine "github.com/tx7do/go-scripts"
	lua "github.com/yuin/gopher-lua"
)

// newTestEngine 创建一个带默认 Lua 引擎的编排器（禁用自动加载，避免依赖脚本目录）。
func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	cfg := DefaultConfig()
	cfg.ScriptDir = "" // 测试不自动加载
	e := NewEngine(cfg, log.DefaultLogger)
	t.Cleanup(func() { e.Close() })
	return e
}

// TestEngine_DefaultLuaType 验证默认工厂创建的是 Lua 引擎类型。
// 这是多语言统一抽象的基础：未来 NewScriptEngine(JavaScriptType) 即可切换。
func TestEngine_DefaultLuaType(t *testing.T) {
	e := newTestEngine(t)

	se := e.ScriptEngine()
	if se == nil {
		t.Fatal("default factory should create an engine")
	}
	if se.GetType() != gsEngine.LuaType {
		t.Errorf("expected lua type, got %s", se.GetType())
	}
	t.Log("✓ Default engine is go-scripts/lua")
}

// TestEngine_ExecuteString 验证编排器执行内联脚本。
func TestEngine_ExecuteString(t *testing.T) {
	e := newTestEngine(t)

	err := e.LoadScriptString(context.Background(), "test", `local x = 1 + 1`)
	if err != nil {
		t.Fatalf("LoadScriptString failed: %v", err)
	}
}

// TestEngine_RuntimeHook_BusinessModule 验证 RuntimeHook 注入的业务模块可在脚本中 require。
// 这是切换到 go-scripts/lua 后的核心：业务 API 经 RuntimeHook 注入，脚本可调用。
func TestEngine_RuntimeHook_BusinessModule(t *testing.T) {
	e := newTestEngine(t)

	// logger 模块经 RuntimeHook 注入，脚本可 require 并调用
	err := e.LoadScriptString(context.Background(), "use_logger", `
		local log = require "kratos_logger"
		log.info("hello from lua")
	`)
	if err != nil {
		t.Fatalf("require kratos_logger failed: %v", err)
	}
}

// TestEngine_RuntimeHook_HookRegister 验证 hook.register 反向回调注册。
// 脚本通过 require "kratos_hook" 调用 register，回调经适配器转发到编排器。
func TestEngine_RuntimeHook_HookRegister(t *testing.T) {
	e := newTestEngine(t)

	// 脚本注册 hook 回调
	err := e.LoadScriptString(context.Background(), "reg_hook", `
		local hook = require "kratos_hook"
		hook.register("test.event", "test callback", function(ctx)
			return true
		end)
	`)
	if err != nil {
		t.Fatalf("hook.register failed: %v", err)
	}

	// 验证回调已注册到编排器
	e.callbacksMu.RLock()
	cbs := e.callbacks["test.event"]
	e.callbacksMu.RUnlock()

	if len(cbs) != 1 {
		t.Fatalf("expected 1 callback for test.event, got %d", len(cbs))
	}

	// 执行 hook（应触发已注册的回调）
	err = e.ExecuteHook(context.Background(), "test.event", NewContext("test.event"))
	if err != nil {
		t.Fatalf("ExecuteHook failed: %v", err)
	}
}

// TestEngine_RuntimeHook_ContextFunctions 验证执行上下文函数注入。
// 脚本通过 __get_ctx() 获取上下文表，调用 ctx.set/get。
func TestEngine_RuntimeHook_ContextFunctions(t *testing.T) {
	e := newTestEngine(t)

	// 预注册一个 hook 脚本，使用 __get_ctx().set
	err := e.AddScript("ctx.hook", &Script{
		Name: "ctx_script",
		Hook: "ctx.hook",
		Source: `
		function execute()
			local ctx = __get_ctx()
			ctx.set("result", "from-lua")
			return true
		end
		`,
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("AddScript failed: %v", err)
	}

	ctx := NewContext("ctx.hook")
	err = e.ExecuteHook(context.Background(), "ctx.hook", ctx)
	if err != nil {
		t.Fatalf("ExecuteHook failed: %v", err)
	}

	val, ok := ctx.Data["result"]
	if !ok {
		t.Fatal("expected ctx.Data['result'] to be set by script")
	}
	if val != "from-lua" {
		t.Errorf("expected 'from-lua', got %v", val)
	}
}

// TestEngineWithFactory_CustomEngine 验证通过工厂注入自定义引擎实现。
// 这是多语言切换的入口：替换工厂即可切换底层脚本语言。
func TestEngineWithFactory_CustomEngine(t *testing.T) {
	injected := false
	customFactory := func(config *Config, _ log.Logger) (gsEngine.Engine, error) {
		injected = true
		// 返回标准 Lua 引擎（验证工厂机制本身）
		return gsEngine.NewScriptEngine(config.EngineType)
	}

	cfg := DefaultConfig()
	cfg.ScriptDir = ""
	e := NewEngineWithFactory(cfg, log.DefaultLogger, customFactory)
	defer e.Close()

	if !injected {
		t.Fatal("custom factory was not invoked")
	}

	if e.ScriptEngine() == nil {
		t.Fatal("script engine should not be nil")
	}
	t.Log("✓ Custom engine factory injection works")
}

// TestEngine_RegisterModule 验证宿主模块注册（require 可用）。
// go-scripts/lua 的 RegisterModule 调用 loader 时传入模块名，
// loader 须执行 PreloadModule（preload 风格），而非直接构建模块。
func TestEngine_RegisterModule(t *testing.T) {
	e := newTestEngine(t)
	eng := e.ScriptEngine()

	// builder：构建模块并 push（require 时调用）
	var builder lua.LGFunction = func(L *lua.LState) int {
		mod := L.NewTable()
		var ping lua.LGFunction = func(L *lua.LState) int {
			L.Push(lua.LString("pong"))
			return 1
		}
		mod.RawSetString("ping", L.NewFunction(ping))
		L.Push(mod)
		return 1
	}

	// preload 风格 loader：读 name 参数，注册 PreloadModule
	var preloadLoader lua.LGFunction = func(L *lua.LState) int {
		name := L.CheckString(1)
		L.PreloadModule(name, builder)
		return 0
	}

	if err := eng.RegisterModule("mymod", preloadLoader); err != nil {
		t.Fatalf("RegisterModule failed: %v", err)
	}

	// 在脚本中 require 并调用
	err := e.LoadScriptString(context.Background(), "use_mod", `
		local m = require "mymod"
		result = m.ping()
	`)
	if err != nil {
		t.Fatalf("script execution failed: %v", err)
	}

	// 验证模块返回值（通过 GetGlobal 读取脚本设置的全局）
	val, err := eng.GetGlobal("result")
	if err != nil {
		t.Fatalf("GetGlobal failed: %v", err)
	}
	if s, ok := val.(string); !ok || s != "pong" {
		t.Errorf("expected 'pong', got %v (%T)", val, val)
	}
}
