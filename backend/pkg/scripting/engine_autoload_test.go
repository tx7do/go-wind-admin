package scripting

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
)

func TestEngine_AutoLoadScripts(t *testing.T) {
	// Create temporary scripts directory
	tmpDir := t.TempDir()

	// Create a test script
	scriptContent := `
		local log = require "kratos_logger"
		local hook = require "kratos_hook"

		log.info("Auto-loaded script executing")

		hook.register("autoload.test", "Auto-load test hook", function(ctx)
			ctx.set("auto_loaded", true)
			log.info("Auto-loaded callback executed")
			return true
		end)
	`

	err := os.WriteFile(filepath.Join(tmpDir, "autoload.lua"), []byte(scriptContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test script: %v", err)
	}

	// Create engine with ScriptDir configured
	config := DefaultConfig()
	config.ScriptDir = tmpDir

	engine := NewEngine(config, log.DefaultLogger)
	defer engine.Close()

	// Script should have been loaded automatically
	// Verify by executing the hook
	ctx := &Context{
		ID:       "test-autoload",
		HookName: "autoload.test",
		Data:     make(map[string]interface{}),
	}

	err = engine.ExecuteHook(context.Background(), "autoload.test", ctx)
	if err != nil {
		t.Fatalf("Failed to execute hook: %v", err)
	}

	// Verify callback executed
	if loaded, ok := ctx.Data["auto_loaded"].(bool); !ok || !loaded {
		t.Error("Auto-loaded script callback was not executed")
	}

	t.Log("✓ Auto-load scripts test passed")
}

func TestEngine_NoAutoLoadWithoutScriptDir(t *testing.T) {
	// Create engine without ScriptDir
	config := DefaultConfig()
	config.ScriptDir = "" // Empty

	engine := NewEngine(config, log.DefaultLogger)
	defer engine.Close()

	// Should initialize without errors even without scripts
	t.Log("✓ Engine initialized without ScriptDir")
}

func TestEngine_AutoLoadNonExistentDir(t *testing.T) {
	// Create engine with non-existent ScriptDir
	config := DefaultConfig()
	config.ScriptDir = "/non/existent/path"

	// Should not panic, just log warning
	engine := NewEngine(config, log.DefaultLogger)
	defer engine.Close()

	t.Log("✓ Engine initialized with non-existent ScriptDir (warning logged)")
}
