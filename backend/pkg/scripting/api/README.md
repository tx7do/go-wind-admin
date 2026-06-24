# Lua API Modules

This directory contains the Lua API modules that expose Go functionality to Lua scripts.

## Available Modules

### Core Modules

| Module | Require Name | Description | Setup Required |
|--------|--------------|-------------|----------------|
| **Logger** | `kratos_logger` | Logging functionality | No (always available) |
| **Hook** | `kratos_hook` | Hook registration and management | No (always available) |
| **Crypto** | `kratos_crypto` | Encryption/decryption (AES-256-GCM) | No (always available) |
| **Util** | `kratos_util` | Utility functions (sleep, time, date) | No (always available) |
| **Cache** | `kratos_cache` | Redis cache operations | Yes - `SetRedis()` |
| **EventBus** | `kratos_eventbus` | Event publishing/subscribing | Yes - `SetEventBus()` |
| **OSS** | `kratos_oss` | Object storage (MinIO) operations | Yes - `SetOSS()` |

## Usage

### In Go

```go
import (
    "go-wind-admin/pkg/scripting"
    "go-wind-admin/pkg/oss"
)

// Create Lua engine
engine := lua.NewEngine(lua.DefaultConfig(), logger)

// Configure optional modules
engine.SetRedis(redisClient)      // Enable cache API
engine.SetEventBus(eventBusManager) // Enable eventbus API
engine.SetOSS(ossClient)          // Enable OSS API
```

### In Lua Scripts

```lua
-- Logger (always available)
local log = require "kratos_logger"
log.info("Hello, World!")

-- Hook API (always available)
local hook = require "kratos_hook"
hook.register("my_hook", "Description", function(ctx)
    return true
end)

-- Crypto API (always available)
local crypto = require "kratos_crypto"
local encrypted = crypto.encrypt("sensitive data")
local decrypted = crypto.decrypt(encrypted)

-- Util API (always available)
local util = require "kratos_util"
util.sleep(1)  -- Sleep for 1 second
local timestamp = util.timestamp()

-- Cache API (if configured)
local cache = require "kratos_cache"
cache.set("key", "value", 3600)

-- EventBus API (if configured)
local eventbus = require "kratos_eventbus"
eventbus.publish("event.name", {data = "value"})

-- OSS API (if configured)
local oss = require "kratos_oss"
local result = oss.upload_url({
    content_type = "image/jpeg"
})
```

## Module Documentation

- **[logger.go](logger.go)** - Logging API
- **[hook.go](hook.go)** - Hook management API
- **[crypto.go](crypto.go)** - Encryption/decryption API
- **[util.go](util.go)** - Utility API (sleep, time, date)
- **[cache.go](cache.go)** - Redis cache API
- **[eventbus.go](eventbus.go)** - Event bus API
- **[oss.go](oss.go)** - Object storage API

## Detailed Guides

- **[UTIL_API.md](UTIL_API.md)** - Complete Util API reference
- **[../CRYPTO_API.md](../CRYPTO_API.md)** - Complete Crypto API reference
- **[../OSS_API.md](../OSS_API.md)** - Complete OSS API reference
- **[../OSS_INTEGRATION.md](../OSS_INTEGRATION.md)** - OSS integration guide
- **[../PASSING_DATA_TO_HOOKS.md](../PASSING_DATA_TO_HOOKS.md)** - Data passing guide

## Creating New API Modules

To create a new Lua API module:

1. **Create the API file** (e.g., `myapi.go`):

```go
package api

import (
    "github.com/go-kratos/kratos/v2/log"
    lua "github.com/yuin/gopher-lua"
)

// RegisterMyAPI registers the API for Lua
func RegisterMyAPI(L *lua.LState, service *MyService, logger *log.Helper) {
    loader := func(L *lua.LState) int {
        // Create module table
        module := L.NewTable()

        // Add API functions
        module.RawSetString("do_something", L.NewFunction(func(L *lua.LState) int {
            param := L.CheckString(1)

            result := service.DoSomething(param)

            L.Push(lua.LString(result))
            return 1
        }))

        L.Push(module)
        return 1
    }

    // Register in package.preload
    L.PreloadModule("kratos_myapi", loader)
}
```

2. **Update engine.go**:

```go
// In Engine struct
type Engine struct {
    // ... existing fields ...
    myService *MyService
}

// In registerAPIs()
func (e *Engine) registerAPIs(L *lua.LState) {
    // ... existing registrations ...

    if e.myService != nil {
        api.RegisterMyAPI(L, e.myService, e.logger)
    }
}

// Add setter
func (e *Engine) SetMyService(service *MyService) {
    e.mu.Lock()
    defer e.mu.Unlock()
    e.myService = service
    e.logger.Info("MyService configured for Lua API")
}
```

3. **Create tests** (`myapi_test.go`):

```go
func TestMyAPI(t *testing.T) {
    L := lua.NewState()
    defer L.Close()

    service := NewMyService()
    logger := log.NewHelper(log.DefaultLogger)

    RegisterMyAPI(L, service, logger)

    script := `
        local myapi = require "kratos_myapi"
        local result = myapi.do_something("test")
        assert(result ~= nil)
        return result
    `

    err := L.DoString(script)
    assert.NoError(t, err)
}
```

4. **Document the API** in a markdown file

## Best Practices

1. **Error Handling**: Use consistent error return patterns
   ```lua
   -- Pattern 1: boolean + error message
   local ok, err = api.function()
   if not ok then
       handle_error(err)
   end

   -- Pattern 2: result or raise error
   local result = api.function()  -- throws on error
   ```

2. **Naming**: Use snake_case for Lua functions
   ```lua
   oss.upload_url()      -- Good
   oss.uploadUrl()       -- Avoid
   ```

3. **Tables for Complex Parameters**: Use tables for functions with multiple parameters
   ```lua
   oss.upload_url({
       content_type = "image/jpeg",
       file_name = "photo.jpg"
   })
   ```

4. **Documentation**: Document all functions with:
   - Purpose
   - Parameters (type, required/optional)
   - Return values
   - Examples

5. **Logging**: Use the logger API for debugging
   ```lua
   local log = require "kratos_logger"
   log.debug("API call: " .. param)
   ```

## Testing

```bash
# Test all API modules
cd pkg/scripting/api
go test -v

# Test specific module
go test -v -run TestOSS

# Test with coverage
go test -cover
```

## See Also

- [../engine.go](../engine.go) - Lua engine implementation
- [../SCRIPT_LOADING_GUIDE.md](../SCRIPT_LOADING_GUIDE.md) - Script loading
- [../examples/](../examples/) - Example scripts
