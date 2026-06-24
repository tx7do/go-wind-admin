package scripting

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/log"

	gsEngine "github.com/tx7do/go-scripts"
)

// newTestJSEngine 创建一个 JS 引擎编排器（禁用自动加载）。
func newTestJSEngine(t *testing.T) *Engine {
	t.Helper()
	cfg := DefaultConfig()
	cfg.ScriptDir = ""
	cfg.EngineType = gsEngine.JavaScriptType
	e := NewEngine(cfg, log.DefaultLogger)
	t.Cleanup(func() { e.Close() })
	return e
}

// TestJSEngine_Type 验证 JS 引擎类型。
func TestJSEngine_Type(t *testing.T) {
	e := newTestJSEngine(t)
	se := e.ScriptEngine()
	if se == nil {
		t.Fatal("JS engine should be created")
	}
	if se.GetType() != gsEngine.JavaScriptType {
		t.Errorf("expected javascript type, got %s", se.GetType())
	}
	t.Log("✓ JS engine created via go-scripts/javascript")
}

// TestJSEngine_ExecuteString 验证 JS 脚本执行。
func TestJSEngine_ExecuteString(t *testing.T) {
	e := newTestJSEngine(t)
	err := e.LoadScriptString(context.Background(), "test", `var x = 1 + 1`)
	if err != nil {
		t.Fatalf("ExecuteString failed: %v", err)
	}
}

// TestJSEngine_RuntimeHook_Logger 验证 RuntimeHook 注入的 logger 模块可在 JS 中调用。
func TestJSEngine_RuntimeHook_Logger(t *testing.T) {
	e := newTestJSEngine(t)
	err := e.LoadScriptString(context.Background(), "log_test", `log.info("hello from JS")`)
	if err != nil {
		t.Fatalf("log.info failed: %v", err)
	}
}

// TestJSEngine_RuntimeHook_Crypto 验证 crypto 模块在 JS 中可用。
func TestJSEngine_RuntimeHook_Crypto(t *testing.T) {
	e := newTestJSEngine(t)
	err := e.LoadScriptString(context.Background(), "crypto_test", `
		var h = crypto.hash_sha256("test")
		if (h.length != 64) throw "hash length wrong: " + h.length
	`)
	if err != nil {
		t.Fatalf("crypto test failed: %v", err)
	}
}

// TestJSEngine_ExecuteWithContext 验证 execute() 入口 + __get_ctx 上下文。
func TestJSEngine_ExecuteWithContext(t *testing.T) {
	e := newTestJSEngine(t)

	err := e.AddScript("js.ctx", &Script{
		Name: "ctx_script",
		Hook: "js.ctx",
		Source: `
			function execute() {
				var ctx = __get_ctx();
				ctx.set("result", "from-js");
				return true;
			}
		`,
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("AddScript failed: %v", err)
	}

	ctx := NewContext("js.ctx")
	err = e.ExecuteHook(context.Background(), "js.ctx", ctx)
	if err != nil {
		t.Fatalf("ExecuteHook failed: %v", err)
	}

	val, ok := ctx.Data["result"]
	if !ok {
		t.Fatal("expected ctx.Data['result'] set by JS script")
	}
	if val != "from-js" {
		t.Errorf("expected 'from-js', got %v", val)
	}
	t.Log("✓ JS execute() + __get_ctx works")
}

// TestJSEngine_HookRegister 验证 JS hook.register 反向回调注册。
func TestJSEngine_HookRegister(t *testing.T) {
	e := newTestJSEngine(t)

	// 脚本注册 hook 回调
	err := e.LoadScriptString(context.Background(), "reg_hook", `
		hook.register("js.event", "JS callback", function(ctx) {
			ctx.set("js_callback", true);
			return true;
		});
	`)
	if err != nil {
		t.Fatalf("hook.register failed: %v", err)
	}

	// 验证回调已注册
	e.callbacksMu.RLock()
	cbs := e.callbacks["js.event"]
	e.callbacksMu.RUnlock()
	if len(cbs) != 1 {
		t.Fatalf("expected 1 callback for js.event, got %d", len(cbs))
	}

	// 执行 hook
	ctx := NewContext("js.event")
	err = e.ExecuteHook(context.Background(), "js.event", ctx)
	if err != nil {
		t.Fatalf("ExecuteHook failed: %v", err)
	}

	if v, ok := ctx.Data["js_callback"].(bool); !ok || !v {
		t.Errorf("expected js_callback=true, got %v", ctx.Data["js_callback"])
	}
	t.Log("✓ JS hook.register callback works")
}

// TestJSEngine_ScriptAbort 验证 JS execute 返回 false 中止。
func TestJSEngine_ScriptAbort(t *testing.T) {
	e := newTestJSEngine(t)

	err := e.AddScript("js.abort", &Script{
		Name: "abort_script",
		Hook: "js.abort",
		Source: `
			function execute() {
				return false;
			}
		`,
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("AddScript failed: %v", err)
	}

	ctx := NewContext("js.abort")
	err = e.ExecuteHook(context.Background(), "js.abort", ctx)
	if err == nil {
		t.Fatal("expected error when JS script returns false")
	}
	t.Logf("✓ JS abort works: %v", err)
}

// TestMultiLanguageSwitch 验证同一进程内 Lua 和 JS 引擎可共存。
func TestMultiLanguageSwitch(t *testing.T) {
	// Lua 引擎
	luaCfg := DefaultConfig()
	luaCfg.ScriptDir = ""
	luaEng := NewEngine(luaCfg, log.DefaultLogger)
	defer luaEng.Close()
	if luaEng.ScriptEngine().GetType() != gsEngine.LuaType {
		t.Fatal("expected lua engine")
	}

	// JS 引擎
	jsCfg := DefaultConfig()
	jsCfg.ScriptDir = ""
	jsCfg.EngineType = gsEngine.JavaScriptType
	jsEng := NewEngine(jsCfg, log.DefaultLogger)
	defer jsEng.Close()
	if jsEng.ScriptEngine().GetType() != gsEngine.JavaScriptType {
		t.Fatal("expected javascript engine")
	}

	// 两者都能执行各自的脚本
	if err := luaEng.LoadScriptString(context.Background(), "lua", `local x = 1`); err != nil {
		t.Fatalf("Lua exec failed: %v", err)
	}
	if err := jsEng.LoadScriptString(context.Background(), "js", `var x = 1`); err != nil {
		t.Fatalf("JS exec failed: %v", err)
	}

	t.Log("✓ Lua and JS engines coexist in same process")
}
