package scripting

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
)

// TestSandbox_BlocksDangerousLibs 验证沙箱配置生效：
// os/io/debug 等危险库被禁用，string/table/math 等安全库可用。
//
// 依赖 go-scripts/lua v0.0.8+ 的 SetOpenLibs 能力。
func TestSandbox_BlocksDangerousLibs(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ScriptDir = ""
	e := NewEngine(cfg, log.DefaultLogger)
	defer e.Close()

	// os 库应被禁用（os.execute 命令注入防护）
	err := e.LoadScriptString(context.Background(), "os_test", `
		local ok = pcall(function() return os.execute("echo hacked") end)
		if ok then error("os.execute should be blocked by sandbox") end
	`)
	if err != nil {
		t.Fatalf("os test failed: %v", err)
	}

	// io 库应被禁用（文件系统访问防护）
	err = e.LoadScriptString(context.Background(), "io_test", `
		local ok = pcall(function() return io.open("/etc/passwd") end)
		if ok then error("io.open should be blocked by sandbox") end
	`)
	if err != nil {
		t.Fatalf("io test failed: %v", err)
	}

	// debug 库应被禁用（反射绕过防护）
	err = e.LoadScriptString(context.Background(), "debug_test", `
		local ok = pcall(function() return debug.getinfo(print) end)
		if ok then error("debug should be blocked by sandbox") end
	`)
	if err != nil {
		t.Fatalf("debug test failed: %v", err)
	}

	// 安全库应可用：string
	err = e.LoadScriptString(context.Background(), "str_test", `
		local s = string.upper("hello")
		if s ~= "HELLO" then error("string lib broken") end
	`)
	if err != nil {
		t.Fatalf("string test failed: %v", err)
	}

	// 安全库应可用：math
	err = e.LoadScriptString(context.Background(), "math_test", `
		local v = math.floor(3.7)
		if v ~= 3 then error("math lib broken") end
	`)
	if err != nil {
		t.Fatalf("math test failed: %v", err)
	}

	// 安全库应可用：table
	err = e.LoadScriptString(context.Background(), "table_test", `
		local t = {1, 2, 3}
		if #t ~= 3 then error("table lib broken") end
	`)
	if err != nil {
		t.Fatalf("table test failed: %v", err)
	}

	t.Log("✓ Sandbox: os/io/debug blocked, string/math/table/base available")
}
