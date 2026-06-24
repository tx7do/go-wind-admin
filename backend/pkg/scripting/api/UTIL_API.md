# Util API

The Util API provides common utility functions for Lua scripts including timing, sleeping, and date/time operations.

## Loading the Module

```lua
local util = require('util')
```

## Functions

### util.sleep(seconds)

Pause execution for the specified number of seconds.

**Parameters:**
- `seconds` (number): Duration to sleep in seconds (can be fractional)

**Returns:** None

**Example:**
```lua
local util = require('util')

-- Sleep for 1 second
util.sleep(1)

-- Sleep for 500 milliseconds
util.sleep(0.5)

-- Sleep for 2.5 seconds
util.sleep(2.5)
```

### util.time()

Get the current Unix timestamp in seconds.

**Parameters:** None

**Returns:**
- (number): Current Unix timestamp in seconds

**Example:**
```lua
local util = require('util')

local timestamp = util.time()
print("Current time: " .. timestamp)
-- Output: Current time: 1729518800
```

### util.timestamp()

Get the current Unix timestamp in milliseconds.

**Parameters:** None

**Returns:**
- (number): Current Unix timestamp in milliseconds

**Example:**
```lua
local util = require('util')

local start = util.timestamp()
-- Do some work...
util.sleep(1)
local elapsed = util.timestamp() - start
print("Elapsed: " .. elapsed .. "ms")
-- Output: Elapsed: 1005ms
```

### util.date(format)

Get a formatted date string.

**Parameters:**
- `format` (string, optional): Go time format string. Defaults to RFC3339

**Returns:**
- (string): Formatted date string

**Common Format Patterns:**
- Default (RFC3339): `"2006-01-02T15:04:05Z07:00"`
- Date only: `"2006-01-02"`
- Time only: `"15:04:05"`
- Custom: `"Mon Jan 2 15:04:05 MST 2006"`

**Example:**
```lua
local util = require('util')

-- Get current date in RFC3339 format (default)
local date = util.date()
print(date)
-- Output: 2025-10-20T11:15:30+03:00

-- Get date only
local date_only = util.date("2006-01-02")
print(date_only)
-- Output: 2025-10-20

-- Get time only
local time_only = util.date("15:04:05")
print(time_only)
-- Output: 11:15:30

-- Custom format
local custom = util.date("Mon Jan 2 15:04:05 2006")
print(custom)
-- Output: Mon Oct 20 11:15:30 2025
```

## Usage Examples

### Retry with Exponential Backoff

```lua
local util = require('util')
local log = require('logger')

function retry_with_backoff(fn, max_attempts)
    local attempts = 0
    local delay = 0.1  -- Start with 100ms

    while attempts < max_attempts do
        attempts = attempts + 1

        local success, result = pcall(fn)
        if success then
            return result
        end

        if attempts < max_attempts then
            log.warnf("Attempt %d failed, retrying in %.2fs...", attempts, delay)
            util.sleep(delay)
            delay = delay * 2  -- Exponential backoff
        end
    end

    error("Max retry attempts reached")
end
```

### Rate Limiting

```lua
local util = require('util')

function rate_limited_operation(items, max_per_second)
    local delay = 1.0 / max_per_second

    for i, item in ipairs(items) do
        -- Process item
        process_item(item)

        -- Rate limit
        if i < #items then
            util.sleep(delay)
        end
    end
end

-- Process 10 items per second
rate_limited_operation(my_items, 10)
```

### Performance Timing

```lua
local util = require('util')
local log = require('logger')

function timed_operation(name, fn)
    local start = util.timestamp()

    local result = fn()

    local elapsed = util.timestamp() - start
    log.infof("%s completed in %dms", name, elapsed)

    return result
end

-- Usage
local result = timed_operation("Database query", function()
    -- Your operation here
    return query_database()
end)
```

### Scheduled Task Execution

```lua
local util = require('util')
local log = require('logger')

function run_at_interval(task_fn, interval_seconds)
    while true do
        local start = util.time()

        -- Run the task
        local success, err = pcall(task_fn)
        if not success then
            log.errorf("Task failed: %s", err)
        end

        -- Calculate how long to sleep
        local elapsed = util.time() - start
        local sleep_time = interval_seconds - elapsed

        if sleep_time > 0 then
            util.sleep(sleep_time)
        end
    end
end

-- Run every 60 seconds
run_at_interval(function()
    log.info("Running scheduled task...")
    -- Your task logic here
end, 60)
```

## Integration with Other APIs

The util API works seamlessly with other Kratos APIs:

```lua
local util = require('util')
local log = require('logger')
local cache = require('cache')

-- Rate-limited cache operations
function safe_cache_set(key, value)
    -- Retry with delay on failure
    for attempt = 1, 3 do
        local success = cache.set(key, value, 3600)
        if success then
            return true
        end

        if attempt < 3 then
            log.warnf("Cache set failed, retrying in %ds...", attempt)
            util.sleep(attempt)  -- Exponential backoff: 1s, 2s
        end
    end

    return false
end

-- Periodic cache cleanup
function cleanup_old_cache_entries()
    local pattern = "temp:*"
    local keys = cache.keys(pattern)

    for i, key in ipairs(keys) do
        cache.del(key)

        -- Rate limit deletions to avoid overwhelming Redis
        if i % 100 == 0 then
            util.sleep(0.1)
        end
    end

    log.infof("Cleaned up %d cache entries", #keys)
end
```

## Notes

- All time functions use the server's local timezone
- The `sleep` function blocks the current Lua VM, so use it carefully in task handlers
- For long-running tasks, consider using task handlers with proper async/await patterns
- Timestamps are in UTC unless specified otherwise
