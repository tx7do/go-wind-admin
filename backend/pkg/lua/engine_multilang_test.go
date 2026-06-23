package lua

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/log"

	gsEngine "github.com/tx7do/go-scripts"
	lua "github.com/yuin/gopher-lua"
)

// TestLuaEngine_ImplementsGSEngine 验证 LuaEngine 实现 go-scripts.Engine 接口。
// 这是多语言统一抽象的基础：未来 JSEngine/PythonEngine 实现同一接口即可替换。
func TestLuaEngine_ImplementsGSEngine(t *testing.T) {
	var _ gsEngine.Engine = (*LuaEngine)(nil)
	t.Log("✓ LuaEngine satisfies gsEngine.Engine interface")
}

// TestLuaEngine_Type 验证引擎类型标识。
func TestLuaEngine_Type(t *testing.T) {
	le := NewLuaEngine(DefaultConfig(), log.DefaultLogger)
	if le.GetType() != gsEngine.LuaType {
		t.Errorf("expected %s, got %s", gsEngine.LuaType, le.GetType())
	}
}

// TestLuaEngine_Lifecycle 验证 Init/Close 生命周期。
func TestLuaEngine_Lifecycle(t *testing.T) {
	le := NewLuaEngine(DefaultConfig(), log.DefaultLogger)

	if le.IsInitialized() {
		t.Fatal("engine should not be initialized before Init")
	}

	if err := le.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if !le.IsInitialized() {
		t.Fatal("engine should be initialized after Init")
	}

	// 重复 Init 应返回错误
	if err := le.Init(context.Background()); err == nil {
		t.Fatal("expected error on double Init")
	}

	if err := le.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if le.IsInitialized() {
		t.Fatal("engine should not be initialized after Close")
	}
}

// TestLuaEngine_ExecuteString 验证内联脚本执行。
func TestLuaEngine_ExecuteString(t *testing.T) {
	le := NewLuaEngine(DefaultConfig(), log.DefaultLogger)
	if err := le.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer le.Close()

	_, err := le.ExecuteString(context.Background(), "test", `local x = 1 + 1`)
	if err != nil {
		t.Fatalf("ExecuteString failed: %v", err)
	}
}

// TestLuaEngine_CallFunction 验证调用脚本定义的函数。
//
// 注意：池化引擎下，脚本通过 ExecuteString 定义的函数是 per-VM 的瞬时态，
// 跨 VM 不可见（与 go-scripts/lua 的单脚本语义一致）。
// 因此本测试使用 ExecuteWithResult 在同一 VM 内定义并调用函数。
func TestLuaEngine_CallFunction(t *testing.T) {
	le := NewLuaEngine(DefaultConfig(), log.DefaultLogger)
	if err := le.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer le.Close()

	// 在单次执行中定义并调用（同一 VM）
	result, err := le.ExecuteWithResult(context.Background(), "add_test", `
		function add(a, b) return a + b end
		function execute()
			return add(3, 4)
		end
	`, nil)
	if err != nil {
		t.Fatalf("ExecuteWithResult failed: %v", err)
	}

	// Lua 数字统一以 float64 返回（convert.ToGoValue 语义）
	if n, ok := result.(float64); !ok || n != 7 {
		t.Errorf("expected 7 (float64), got %v (%T)", result, result)
	}
}

// TestLuaEngine_RegisterGlobalAndGet 验证全局变量注册与读取。
func TestLuaEngine_RegisterGlobalAndGet(t *testing.T) {
	le := NewLuaEngine(DefaultConfig(), log.DefaultLogger)
	if err := le.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer le.Close()

	if err := le.RegisterGlobal("my_var", "hello"); err != nil {
		t.Fatalf("RegisterGlobal failed: %v", err)
	}

	val, err := le.GetGlobal("my_var")
	if err != nil {
		t.Fatalf("GetGlobal failed: %v", err)
	}

	if s, ok := val.(string); !ok || s != "hello" {
		t.Errorf("expected 'hello', got %v", val)
	}
}

// TestLuaEngine_RegisterFunction 验证宿主函数注册。
// 注册的函数持久化到所有 VM（经 replay 机制），可在任意 VM 中调用。
func TestLuaEngine_RegisterFunction(t *testing.T) {
	le := NewLuaEngine(DefaultConfig(), log.DefaultLogger)
	if err := le.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer le.Close()

	// 注册一个宿主函数（类型必须为 lua.LGFunction，即命名类型）
	var hostFn lua.LGFunction = func(L *lua.LState) int {
		L.Push(lua.LString("from-go"))
		return 1
	}
	if err := le.RegisterFunction("greet", hostFn); err != nil {
		t.Fatalf("RegisterFunction failed: %v", err)
	}

	// 通过 ExecuteWithResult 在脚本中调用已注册的宿主函数
	result, err := le.ExecuteWithResult(context.Background(), "call_greet", `
		function execute()
			return greet()
		end
	`, nil)
	if err != nil {
		t.Fatalf("ExecuteWithResult failed: %v", err)
	}

	if s, ok := result.(string); !ok || s != "from-go" {
		t.Errorf("expected 'from-go', got %v (%T)", result, result)
	}
}

// TestEngineWithFactory_CustomEngine 验证通过工厂注入自定义引擎实现。
// 这是多语言切换的入口：替换工厂即可切换底层脚本语言。
func TestEngineWithFactory_CustomEngine(t *testing.T) {
	injected := false
	customFactory := func(config *Config, logger log.Logger) (gsEngine.Engine, error) {
		injected = true
		// 返回标准 Lua 引擎（验证工厂机制本身）
		return NewLuaEngine(config, logger), nil
	}

	e := NewEngineWithFactory(DefaultConfig(), log.DefaultLogger, customFactory)
	defer e.Close()

	if !injected {
		t.Fatal("custom factory was not invoked")
	}

	if e.ScriptEngine() == nil {
		t.Fatal("script engine should not be nil")
	}
	t.Log("✓ Custom engine factory injection works")
}

// TestEngineWithFactory_DefaultLua 验证默认工厂创建 Lua 引擎。
func TestEngineWithFactory_DefaultLua(t *testing.T) {
	e := NewEngineWithFactory(DefaultConfig(), log.DefaultLogger, nil)
	defer e.Close()

	se := e.ScriptEngine()
	if se == nil {
		t.Fatal("default factory should create an engine")
	}
	if se.GetType() != gsEngine.LuaType {
		t.Errorf("expected lua type, got %s", se.GetType())
	}
}

// TestLuaEngine_ModuleRegistrar 验证模块注册（require 可用）。
// 注册的模块持久化到所有 VM（经 replay 机制），可在脚本中 require。
func TestLuaEngine_ModuleRegistrar(t *testing.T) {
	le := NewLuaEngine(DefaultConfig(), log.DefaultLogger)
	if err := le.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer le.Close()

	// 注册一个模块 loader（类型必须为 lua.LGFunction）
	var modLoader lua.LGFunction = func(L *lua.LState) int {
		mod := L.NewTable()
		var ping lua.LGFunction = func(L *lua.LState) int {
			L.Push(lua.LString("pong"))
			return 1
		}
		mod.RawSetString("ping", L.NewFunction(ping))
		L.Push(mod)
		return 1
	}

	if err := le.RegisterModule("mymod", modLoader); err != nil {
		t.Fatalf("RegisterModule failed: %v", err)
	}

	// 在脚本中 require 并调用
	result, err := le.ExecuteWithResult(context.Background(), "use_mod", `
		function execute()
			local m = require "mymod"
			return m.ping()
		end
	`, nil)
	if err != nil {
		t.Fatalf("ExecuteWithResult failed: %v", err)
	}

	if s, ok := result.(string); !ok || s != "pong" {
		t.Errorf("expected 'pong', got %v (%T)", result, result)
	}
}
