package scripting

import (
	"context"
	"testing"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"go-wind-admin/pkg/scripting/api"
)

// newToolBridgeEngine 构造测试用脚本引擎（ScriptDir 清空 = 源码经 LoadScriptString 注入）。
func newToolBridgeEngine(t *testing.T) *Engine {
	t.Helper()
	cfg := DefaultConfig()
	cfg.ScriptDir = ""
	e := NewEngine(cfg, bLogger.NopLogger())
	t.Cleanup(func() { _ = e.Close() })
	return e
}

// TestToolBridge_RegisterAndInvoke 闭环验证：脚本经 tool.register 注册 LLM 工具，
// InvokeToolHandler 合并参数默认值/校验必填并执行，产出经 convert.ToGoValue 回收。
func TestToolBridge_RegisterAndInvoke(t *testing.T) {
	e := newToolBridgeEngine(t)

	source := `
local tool = require "tool"
tool.register("query_greeting", "返回问候语", function(args)
  local name = args.name or "world"
  return { greeting = "Hello, " .. name }
end, {
  required = {"name"},
  optional = { suffix = "!" },
  timeout_secs = 10
})
`
	if err := e.LoadScriptString(context.Background(), "tool_smoke", source); err != nil {
		t.Fatalf("register script failed: %v", err)
	}

	h, ok := api.GetTool("query_greeting")
	if !ok {
		t.Fatal("tool query_greeting not registered")
	}
	if h.Description != "返回问候语" {
		t.Fatalf("description = %q", h.Description)
	}

	// 经执行上下文参数回收执行结果
	execCtx := NewContext("tool_invoke")
	prev := e.execCtx.set(execCtx)

	result, err := api.InvokeToolHandler(context.Background(), "query_greeting",
		map[string]any{"name": "world"})
	e.execCtx.reset(prev)
	if err != nil {
		t.Fatalf("InvokeToolHandler failed: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("result type = %T, want map[string]any", result)
	}
	if m["greeting"] != "Hello, world" {
		t.Fatalf("greeting = %v", m["greeting"])
	}
}

// TestToolBridge_MissingRequired 工具缺必填参数时执行应报错（不静默给空产出）。
func TestToolBridge_MissingRequired(t *testing.T) {
	e := newToolBridgeEngine(t)

	source := `
local tool = require "tool"
tool.register("strict_tool", "必填参数工具", function(args)
  return { ok = true }
end, { required = {"mandatory"} })
`
	if err := e.LoadScriptString(context.Background(), "tool_strict", source); err != nil {
		t.Fatalf("register script failed: %v", err)
	}

	_, err := api.InvokeToolHandler(context.Background(), "strict_tool", map[string]any{})
	if err == nil {
		t.Fatal("missing required param should fail")
	}
	if !contains(err.Error(), "missing required param") {
		t.Fatalf("err = %v, want missing required param", err)
	}
}

// TestToolBridge_GenerationPrune 工具代际清理：Resync 后未被重新注册的工具被清除。
func TestToolBridge_GenerationPrune(t *testing.T) {
	e := newToolBridgeEngine(t)

	source := `
local tool = require "tool"
tool.register("gen_tool", "代际测试", function(args) return {} end)
`
	if err := e.LoadScriptString(context.Background(), "gen_tool_script", source); err != nil {
		t.Fatalf("register script failed: %v", err)
	}

	if _, ok := api.GetTool("gen_tool"); !ok {
		t.Fatal("tool should be registered")
	}

	generation := api.NextToolGeneration()
	pruned := api.PruneStaleToolHandlers(generation)
	if pruned < 1 {
		t.Fatalf("pruned = %d, want >= 1", pruned)
	}
	if _, ok := api.GetTool("gen_tool"); ok {
		t.Fatal("tool should be pruned after generation cleanup")
	}
}

func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && stringContains(s, substr)
}

func stringContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
