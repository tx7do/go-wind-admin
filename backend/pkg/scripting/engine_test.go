package scripting

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
)

// 测试脚本约定（适配 go-scripts/lua 架构）：
// execute() 不再接收 ctx 参数，而是通过 __get_ctx() 获取执行上下文。
// 上下文提供 ctx.get(k) / ctx.set(k,v) / ctx.stop(reason) 方法。

func TestEngine_BasicExecution(t *testing.T) {
	// Create engine
	config := DefaultConfig()
	config.PoolSize = 2
	config.ScriptDir = ""
	engine := NewEngine(config, log.DefaultLogger)
	defer engine.Close()

	// Register hook
	err := engine.RegisterHook("test_hook", "Test hook for basic execution")
	if err != nil {
		t.Fatalf("Failed to register hook: %v", err)
	}

	// Create script
	script := &Script{
		Name: "test_script",
		Hook: "test_hook",
		Source: `
local log = require "kratos_logger"

function execute()
    log.info("Hello from Lua!")
    local ctx = __get_ctx()
    ctx.set("result", "success")
    return true
end
`,
		Enabled:  true,
		Priority: 1,
	}

	// Add script to hook
	err = engine.AddScript("test_hook", script)
	if err != nil {
		t.Fatalf("Failed to add script: %v", err)
	}

	// Create execution context
	execCtx := NewContext("test_hook")
	execCtx.Set("input", "test")

	// Execute hook
	err = engine.ExecuteHook(context.Background(), "test_hook", execCtx)
	if err != nil {
		t.Fatalf("Hook execution failed: %v", err)
	}

	// Check result
	result := execCtx.GetString("result")
	if result != "success" {
		t.Errorf("Expected result='success', got '%s'", result)
	}

	t.Logf("✓ Basic execution test passed")
}

func TestEngine_ContextDataTransfer(t *testing.T) {
	config := DefaultConfig()
	config.ScriptDir = ""
	engine := NewEngine(config, log.DefaultLogger)
	defer engine.Close()

	engine.RegisterHook("data_transfer", "Test data transfer")

	script := &Script{
		Name: "data_script",
		Hook: "data_transfer",
		Source: `
function execute()
    local ctx = __get_ctx()
    local input = ctx.get("number")
    local doubled = input * 2
    ctx.set("output", doubled)
    return true
end
`,
		Enabled:  true,
		Priority: 1,
	}

	engine.AddScript("data_transfer", script)

	execCtx := NewContext("data_transfer")
	execCtx.Set("number", 21)

	err := engine.ExecuteHook(context.Background(), "data_transfer", execCtx)
	if err != nil {
		t.Fatalf("Hook execution failed: %v", err)
	}

	output := execCtx.GetInt("output")
	if output != 42 {
		t.Errorf("Expected output=42, got %d", output)
	}

	t.Logf("✓ Context data transfer test passed")
}

func TestEngine_ScriptAbort(t *testing.T) {
	config := DefaultConfig()
	config.ScriptDir = ""
	engine := NewEngine(config, log.DefaultLogger)
	defer engine.Close()

	engine.RegisterHook("abort_test", "Test script abort")

	script := &Script{
		Name: "abort_script",
		Hook: "abort_test",
		Source: `
local log = require "kratos_logger"

function execute()
    log.info("Script starting")
    return false  -- Return false to abort
end
`,
		Enabled:  true,
		Priority: 1,
	}

	engine.AddScript("abort_test", script)

	execCtx := NewContext("abort_test")

	err := engine.ExecuteHook(context.Background(), "abort_test", execCtx)
	if err == nil {
		t.Error("Expected error when script returns false")
	}

	t.Logf("✓ Script abort test passed: %v", err)
}

func TestEngine_MultipleScripts(t *testing.T) {
	config := DefaultConfig()
	config.ScriptDir = ""
	engine := NewEngine(config, log.DefaultLogger)
	defer engine.Close()

	engine.RegisterHook("multi_test", "Test multiple scripts")

	// Script 1: Set initial value
	script1 := &Script{
		Name: "script1",
		Hook: "multi_test",
		Source: `
local log = require "kratos_logger"

function execute()
    local ctx = __get_ctx()
    ctx.set("value", 10)
    log.info("Script 1: value = 10")
    return true
end
`,
		Enabled:  true,
		Priority: 1,
	}

	// Script 2: Add to value
	script2 := &Script{
		Name: "script2",
		Hook: "multi_test",
		Source: `
local log = require "kratos_logger"

function execute()
    local ctx = __get_ctx()
    local val = ctx.get("value")
    ctx.set("value", val + 5)
    log.info("Script 2: value = " .. (val + 5))
    return true
end
`,
		Enabled:  true,
		Priority: 2,
	}

	// Script 3: Multiply value
	script3 := &Script{
		Name: "script3",
		Hook: "multi_test",
		Source: `
local log = require "kratos_logger"

function execute()
    local ctx = __get_ctx()
    local val = ctx.get("value")
    ctx.set("value", val * 2)
    log.info("Script 3: value = " .. (val * 2))
    return true
end
`,
		Enabled:  true,
		Priority: 3,
	}

	engine.AddScript("multi_test", script1)
	engine.AddScript("multi_test", script2)
	engine.AddScript("multi_test", script3)

	execCtx := NewContext("multi_test")

	err := engine.ExecuteHook(context.Background(), "multi_test", execCtx)
	if err != nil {
		t.Fatalf("Hook execution failed: %v", err)
	}

	// Result should be: (10 + 5) * 2 = 30
	result := execCtx.GetInt("value")
	if result != 30 {
		t.Errorf("Expected value=30, got %d", result)
	}

	t.Logf("✓ Multiple scripts test passed (value=%d)", result)
}
