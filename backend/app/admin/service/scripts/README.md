# Lua Scripts Directory

This directory contains Lua scripts that are automatically loaded when the server starts.

## How It Works

1. **Server startup** - When the server starts, it calls `LoadScriptsFromDir()` to load all `.lua` files from this directory
2. **Script execution** - Each script is executed in a Lua VM, allowing it to register hooks and callbacks
3. **Hook triggering** - After all scripts are loaded, the server triggers the `on_server_start` hook

## Script Loading Order

Scripts are loaded in alphabetical order based on filename. If you need a specific loading order, prefix your scripts with numbers:

```
01_init.lua
02_database.lua
03_cache.lua
10_on_server_start.lua
```

## Available APIs

All scripts have access to the following Lua APIs:

### Log API
```lua
log.debug("Debug message")
log.info("Info message")
log.warn("Warning message")
log.error("Error message")
```

### Hook API
```lua
-- Register hook with callback
hook.register("hook_name", "Description", function(ctx)
    local value = ctx.get("key")
    ctx.set("result", value)
    return true
end)

-- Register hook without callback
hook.register("hook_name", "Description")

-- Add script to hook
hook.add_script("hook_name", {
    name = "script_name",
    source = [[
        function execute(ctx)
            -- script code
            return true
        end
    ]],
    priority = 10,
    enabled = true
})

-- List all hooks
local hooks = hook.list()
```

### Cache API (if Redis is configured)
```lua
cache.set("key", "value", 3600)  -- TTL in seconds
local value = cache.get("key")
cache.delete("key")
cache.incr("counter", 1)
cache.decr("counter", 1)
cache.expire("key", 3600)
local exists = cache.exists("key")
local keys = cache.keys("pattern:*")
```

### EventBus API (if EventBus is configured)
```lua
-- Publish event
eventbus.publish("event.name", {
    data = "value"
}, "channel")

-- Publish async
eventbus.publish_async("event.name", {
    data = "value"
}, "channel")

-- Subscribe to events
eventbus.subscribe("event.name", function(event)
    log.info("Received event: " .. event.name)
end)
```

### Context API
Within hook callbacks, you have access to the context:

```lua
function(ctx)
    -- Get data from context
    local value = ctx.get("key")

    -- Set data in context
    ctx.set("result", "value")

    -- Stop execution chain
    ctx.stop("Reason for stopping")

    -- Return false to abort (same as ctx.stop)
    return false
end
```

## Common Hooks

### `on_server_start`
Triggered when the server starts (before accepting requests).

Example:
```lua
hook.register("on_server_start", "Server startup", function(ctx)
    log.info("Server is starting...")
    -- Initialize services
    -- Check system health
    -- Load configuration
    return true
end)
```

### Custom Application Hooks
You can create your own hooks in your application code:

```go
// In your Go code
err := scriptingEngine.ExecuteHook(ctx, "user.created", &scripting.Context{
    ID: "user-123",
    HookName: "user.created",
    Data: map[string]interface{}{
        "user_id": 123,
        "email": "user@example.com",
        "username": "johndoe",
    },
})
```

Then handle it in Lua:
```lua
hook.register("user.created", "New user created", function(ctx)
    local user_id = ctx.get("user_id")
    local email = ctx.get("email")

    -- Send welcome email
    eventbus.publish("email.send", {
        to = email,
        template = "welcome",
        data = { user_id = user_id }
    }, "email")

    log.info("Welcome email queued for: " .. email)
    return true
end)
```

## Best Practices

1. **Use descriptive filenames** - `on_server_start.lua`, not `script1.lua`
2. **Include comments** - Explain what your script does
3. **Handle errors** - Return `false` or use `ctx.stop()` on errors
4. **Log important events** - Use `log.info()` for key operations
5. **Keep scripts focused** - One script should do one thing well
6. **Test independently** - Each hook should work standalone

## Testing Your Scripts

You can test scripts locally:

```lua
-- test_script.lua
log.info("Testing script loading")

hook.register("test.hook", "Test hook", function(ctx)
    log.info("Test hook executed!")
    return true
end)

log.info("Script loaded successfully")
```

Run your server and check the logs for:
- "Loading Lua scripts from directory: ..."
- "Loaded N Lua scripts from ..."
- Your script's log messages

## Troubleshooting

### Script not loading
- Check file has `.lua` extension
- Check file permissions (should be readable)
- Check for syntax errors in your Lua code
- Enable debug logging to see script execution

### Hook not executing
- Ensure script loaded successfully (check logs)
- Verify hook is registered: `hook.list()` in another script
- Check if hook is being triggered in Go code
- Ensure callback returns `true` (not `false` or `nil`)

### Syntax errors
Common Lua syntax issues:
```lua
-- Wrong: using = instead of ==
if value = 10 then end

-- Correct:
if value == 10 then end

-- Wrong: missing 'then'
if value == 10

-- Correct:
if value == 10 then end

-- Wrong: missing 'end'
function test()
    log.info("test")

-- Correct:
function test()
    log.info("test")
end
```

## Example Scripts

See the example scripts in this directory:
- `on_server_start.lua` - Server startup hook
- Check `pkg/scripting/` for more examples:
  - `example_callback_registration.lua`
  - `example_self_register.lua`
