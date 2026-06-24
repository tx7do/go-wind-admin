package scripting

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	lua "github.com/yuin/gopher-lua"

	"go-wind-admin/pkg/scripting/api"
)

func TestHookAPI_RegisterHook(t *testing.T) {
	logger := log.NewHelper(log.DefaultLogger)
	engine := NewEngine(DefaultConfig(), log.DefaultLogger)
	defer engine.Close()

	L := lua.NewState()
	defer L.Close()

	api.RegisterLogger(L, logger)
	api.RegisterHookAPI(L, &luaHookAdapter{orchestrator: engine}, logger)

	script := `
		local hook = require "kratos_hook"

		-- Register a new hook
		local ok = hook.register("custom_hook", "A custom hook for testing")
		if not ok then
			error("Failed to register hook")
		end

		-- Verify it was registered
		local hooks = hook.list()
		local found = false
		for i, name in ipairs(hooks) do
			if name == "custom_hook" then
				found = true
				break
			end
		end

		if not found then
			error("Hook not found in list")
		end

		return true
	`

	err := L.DoString(script)
	if err != nil {
		t.Fatalf("Lua script error: %v", err)
	}

	t.Log("✓ Register hook test passed")
}

func TestHookAPI_AddScript(t *testing.T) {
	logger := log.NewHelper(log.DefaultLogger)
	engine := NewEngine(DefaultConfig(), log.DefaultLogger)
	defer engine.Close()

	L := lua.NewState()
	defer L.Close()

	api.RegisterLogger(L, logger)
	api.RegisterHookAPI(L, &luaHookAdapter{orchestrator: engine}, logger)

	// First script: registers a hook and adds itself
	initScript := `
		local log = require "kratos_logger"
		local hook = require "kratos_hook"

		-- Register a hook
		hook.register("self_test", "Hook for self-registration test")

		-- Add a script to the hook
		hook.add_script("self_test", {
			name = "dynamic_script",
			source = [[
				local log = require "kratos_logger"
				function execute()
					log.info("Dynamic script executed!")
					local ctx = __get_ctx()
					ctx.set("executed", true)
					return true
				end
			]],
			enabled = true,
			priority = 10,
			description = "Dynamically added script"
		})

		return true
	`

	err := L.DoString(initScript)
	if err != nil {
		t.Fatalf("Init script error: %v", err)
	}

	// Now execute the hook to verify the script was added
	ctx := &Context{
		ID:       "test-ctx",
		HookName: "self_test",
		Data:     make(map[string]interface{}),
	}

	err = engine.ExecuteHook(context.Background(), "self_test", ctx)
	if err != nil {
		t.Fatalf("ExecuteHook error: %v", err)
	}

	// Check if the dynamically added script executed
	if executed, ok := ctx.Data["executed"].(bool); !ok || !executed {
		t.Error("Dynamically added script did not execute")
	}

	t.Log("✓ Add script test passed")
}

func TestHookAPI_SelfRegistration(t *testing.T) {
	logger := log.NewHelper(log.DefaultLogger)
	engine := NewEngine(DefaultConfig(), log.DefaultLogger)
	defer engine.Close()

	L := lua.NewState()
	defer L.Close()

	api.RegisterLogger(L, logger)
	api.RegisterHookAPI(L, &luaHookAdapter{orchestrator: engine}, logger)

	// Script that registers itself
	selfRegisterScript := `
		local log = require "kratos_logger"
		local hook = require "kratos_hook"

		-- Self-registration pattern
		local HOOK_NAME = "user.created"
		local SCRIPT_NAME = "welcome_email"

		-- Register the hook if it doesn't exist
		hook.register(HOOK_NAME, "Triggered when a new user is created")

		-- Register this script to the hook
		hook.add_script(HOOK_NAME, {
			name = SCRIPT_NAME,
				source = [[
					local log = require "kratos_logger"
					function execute()
						local ctx = __get_ctx()
						local user_id = ctx.get("user_id")
						local email = ctx.get("email")
						log.info("Sending welcome email to: " .. email)
						ctx.set("email_sent", true)
						return true
					end
				]],
			enabled = true,
			priority = 5,
			description = "Sends welcome email to new users"
		})

		log.info("Script self-registered successfully")
		return true
	`

	err := L.DoString(selfRegisterScript)
	if err != nil {
		t.Fatalf("Self-registration script error: %v", err)
	}

	// Test the registered hook
	ctx := &Context{
		ID:       "test-user-creation",
		HookName: "user.created",
		Data: map[string]interface{}{
			"user_id": 123,
			"email":   "test@example.com",
		},
	}

	err = engine.ExecuteHook(context.Background(), "user.created", ctx)
	if err != nil {
		t.Fatalf("ExecuteHook error: %v", err)
	}

	if emailSent, ok := ctx.Data["email_sent"].(bool); !ok || !emailSent {
		t.Error("Welcome email script did not execute")
	}

	t.Log("✓ Self-registration test passed")
}

func TestHookAPI_ListHooks(t *testing.T) {
	logger := log.NewHelper(log.DefaultLogger)
	engine := NewEngine(DefaultConfig(), log.DefaultLogger)
	defer engine.Close()

	// Pre-register some hooks
	engine.RegisterHook("hook1", "First hook")
	engine.RegisterHook("hook2", "Second hook")
	engine.RegisterHook("hook3", "Third hook")

	L := lua.NewState()
	defer L.Close()

	api.RegisterLogger(L, logger)
	api.RegisterHookAPI(L, &luaHookAdapter{orchestrator: engine}, logger)

	script := `
		local log = require "kratos_logger"
		local hook = require "kratos_hook"

		local hooks = hook.list()

		-- Verify we got an array
		if #hooks < 3 then
			error("Expected at least 3 hooks, got: " .. #hooks)
		end

		-- Log all hooks
		for i, name in ipairs(hooks) do
			log.info("Hook " .. i .. ": " .. name)
		end

		return #hooks
	`

	err := L.DoString(script)
	if err != nil {
		t.Fatalf("Lua script error: %v", err)
	}

	result := L.Get(-1)
	if num, ok := result.(lua.LNumber); !ok || int(num) < 3 {
		t.Errorf("Expected at least 3 hooks, got: %v", result)
	}

	t.Log("✓ List hooks test passed")
}

func TestHookAPI_ComplexWorkflow(t *testing.T) {
	logger := log.NewHelper(log.DefaultLogger)
	engine := NewEngine(DefaultConfig(), log.DefaultLogger)
	defer engine.Close()

	L := lua.NewState()
	defer L.Close()

	api.RegisterLogger(L, logger)
	api.RegisterHookAPI(L, &luaHookAdapter{orchestrator: engine}, logger)

	// Complex workflow: script registers hooks and other scripts
	orchestratorScript := `
		local log = require "kratos_logger"
		local hook = require "kratos_hook"

		-- Orchestrator pattern: one script sets up multiple hooks and scripts

		-- 1. Register hooks
		hook.register("before_save", "Before saving data")
		hook.register("after_save", "After saving data")
		hook.register("on_error", "Error handling")

		-- 2. Add validation script
		hook.add_script("before_save", {
			name = "validate_data",
				source = [[
					local log = require "kratos_logger"
					function execute()
						local ctx = __get_ctx()
						local data = ctx.get("data")
						if not data or data == "" then
							log.error("Validation failed: data is empty")
							ctx.stop("Validation failed")
							return false
						end
						log.info("Validation passed")
						return true
					end
				]],
			priority = 1,  -- Run first
			enabled = true
		})

		-- 3. Add notification script
		hook.add_script("after_save", {
			name = "send_notification",
				source = [[
					local log = require "kratos_logger"
					function execute()
						local ctx = __get_ctx()
						local data = ctx.get("data")
						log.info("Notification sent for: " .. data)
						ctx.set("notified", true)
						return true
					end
				]],
			priority = 10,
			enabled = true
		})

		-- 4. Add error handler
		hook.add_script("on_error", {
			name = "log_error",
				source = [[
					local log = require "kratos_logger"
					function execute()
						local ctx = __get_ctx()
						local error = ctx.get("error")
						log.error("Error logged: " .. error)
						ctx.set("error_logged", true)
						return true
					end
				]],
			enabled = true
		})

		log.info("Orchestrator setup complete")
		return true
	`

	err := L.DoString(orchestratorScript)
	if err != nil {
		t.Fatalf("Orchestrator script error: %v", err)
	}

	// Test the workflow - successful save
	ctx := &Context{
		ID:       "test-save",
		HookName: "before_save",
		Data: map[string]interface{}{
			"data": "valid data",
		},
	}

	// Execute before_save hook
	err = engine.ExecuteHook(context.Background(), "before_save", ctx)
	if err != nil {
		t.Fatalf("before_save hook error: %v", err)
	}

	// Execute after_save hook
	ctx.HookName = "after_save"
	err = engine.ExecuteHook(context.Background(), "after_save", ctx)
	if err != nil {
		t.Fatalf("after_save hook error: %v", err)
	}

	if notified, ok := ctx.Data["notified"].(bool); !ok || !notified {
		t.Error("Notification was not sent")
	}

	t.Log("✓ Complex workflow test passed")
}

func TestHookAPI_CallbackRegistration(t *testing.T) {
	logger := log.NewHelper(log.DefaultLogger)
	engine := NewEngine(DefaultConfig(), log.DefaultLogger)
	defer engine.Close()

	L := lua.NewState()
	defer L.Close()

	api.RegisterLogger(L, logger)
	api.RegisterHookAPI(L, &luaHookAdapter{orchestrator: engine}, logger)

	// Test callback-based registration (user's requested API)
	callbackScript := `
		local log = require "kratos_logger"
		local hook = require "kratos_hook"

		-- Register hook with callback function
		local ok = hook.register("user.created", "Triggered when a new user is created", function(ctx)
			local email = ctx.get("email")
			local username = ctx.get("username")

			log.info("Callback executed for user: " .. username)

			-- Set a flag to verify execution
			ctx.set("callback_executed", true)
			ctx.set("processed_email", email)

			return true
		end)

		if not ok then
			error("Failed to register hook with callback")
		end

		log.info("Hook registered with callback successfully")
		return true
	`

	err := L.DoString(callbackScript)
	if err != nil {
		t.Fatalf("Callback registration script error: %v", err)
	}

	// Test the callback by executing the hook
	ctx := &Context{
		ID:       "test-callback",
		HookName: "user.created",
		Data: map[string]interface{}{
			"email":    "test@example.com",
			"username": "testuser",
		},
	}

	err = engine.ExecuteHook(context.Background(), "user.created", ctx)
	if err != nil {
		t.Fatalf("ExecuteHook error: %v", err)
	}

	// Verify callback executed
	if executed, ok := ctx.Data["callback_executed"].(bool); !ok || !executed {
		t.Error("Callback was not executed")
	}

	if email, ok := ctx.Data["processed_email"].(string); !ok || email != "test@example.com" {
		t.Errorf("Callback did not process email correctly, got: %v", email)
	}

	t.Log("✓ Callback registration test passed")
}

func TestHookAPI_CallbackWithoutDescription(t *testing.T) {
	logger := log.NewHelper(log.DefaultLogger)
	engine := NewEngine(DefaultConfig(), log.DefaultLogger)
	defer engine.Close()

	L := lua.NewState()
	defer L.Close()

	api.RegisterLogger(L, logger)
	api.RegisterHookAPI(L, &luaHookAdapter{orchestrator: engine}, logger)

	// Test minimal callback registration (hook name + callback only)
	minimalScript := `
		local hook = require "kratos_hook"

		-- Register hook with just name and callback (no description)
		hook.register("minimal.test", nil, function(ctx)
			ctx.set("minimal_callback", true)
			return true
		end)

		return true
	`

	err := L.DoString(minimalScript)
	if err != nil {
		t.Fatalf("Minimal callback script error: %v", err)
	}

	// Execute and verify
	ctx := &Context{
		ID:       "test-minimal",
		HookName: "minimal.test",
		Data:     make(map[string]interface{}),
	}

	err = engine.ExecuteHook(context.Background(), "minimal.test", ctx)
	if err != nil {
		t.Fatalf("ExecuteHook error: %v", err)
	}

	if executed, ok := ctx.Data["minimal_callback"].(bool); !ok || !executed {
		t.Error("Minimal callback was not executed")
	}

	t.Log("✓ Callback without description test passed")
}

func TestHookAPI_MixedRegistrationMethods(t *testing.T) {
	logger := log.NewHelper(log.DefaultLogger)
	engine := NewEngine(DefaultConfig(), log.DefaultLogger)
	defer engine.Close()

	L := lua.NewState()
	defer L.Close()

	api.RegisterLogger(L, logger)
	api.RegisterHookAPI(L, &luaHookAdapter{orchestrator: engine}, logger)

	// Test using both callback and add_script methods together
	mixedScript := `
		local log = require "kratos_logger"
		local hook = require "kratos_hook"

		-- Register hook with callback
		hook.register("mixed.test", "Mixed registration test", function(ctx)
			local count = ctx.get("count") or 0
			ctx.set("count", count + 1)
			log.info("Callback 1 executed")
			return true
		end)

		-- Add another script to the same hook using add_script
		hook.add_script("mixed.test", {
			name = "second_script",
				source = [[
					local log = require "kratos_logger"
					function execute()
						local ctx = __get_ctx()
						local count = ctx.get("count") or 0
						ctx.set("count", count + 1)
						log.info("Callback 2 executed")
						return true
					end
				]],
			priority = 5,
			enabled = true
		})

		return true
	`

	err := L.DoString(mixedScript)
	if err != nil {
		t.Fatalf("Mixed registration script error: %v", err)
	}

	// Execute hook and verify both scripts ran
	ctx := &Context{
		ID:       "test-mixed",
		HookName: "mixed.test",
		Data:     make(map[string]interface{}),
	}

	err = engine.ExecuteHook(context.Background(), "mixed.test", ctx)
	if err != nil {
		t.Fatalf("ExecuteHook error: %v", err)
	}

	// Both scripts should have incremented count
	count, ok := ctx.Data["count"].(float64)
	if !ok {
		t.Error("Count was not set")
	} else if int(count) != 2 {
		t.Errorf("Expected count to be 2, got: %d", int(count))
	}

	t.Log("✓ Mixed registration methods test passed")
}
