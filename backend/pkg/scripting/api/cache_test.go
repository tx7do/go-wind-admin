package api

/*
import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	lua "github.com/yuin/gopher-lua"
)

func setupTestRedis(t *testing.T) (*redis.Client, func()) {
	// Create mini Redis server
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	// Create Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return rdb, func() {
		rdb.Close()
		mr.Close()
	}
}

// setupLuaState creates a Lua state with package system for module support
func setupLuaState() *lua.LState {
	L := lua.NewState()

	// Create package.preload table for module system
	packageTable := L.NewTable()
	preloadTable := L.NewTable()
	packageTable.RawSetString("preload", preloadTable)
	L.SetGlobal("package", packageTable)

	// Create minimal require() function
	L.SetGlobal("require", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)

		// Check if module is already loaded
		pkg := L.GetGlobal("package")
		pkgTable := pkg.(*lua.LTable)

		loaded := pkgTable.RawGetString("loaded")
		if loaded == lua.LNil {
			loaded = L.NewTable()
			pkgTable.RawSetString("loaded", loaded)
		}
		loadedTable := loaded.(*lua.LTable)

		// Return cached module if already loaded
		cached := loadedTable.RawGetString(name)
		if cached != lua.LNil {
			L.Push(cached)
			return 1
		}

		// Get loader from package.preload
		preload := pkgTable.RawGetString("preload")
		preloadTable := preload.(*lua.LTable)

		loader := preloadTable.RawGetString(name)
		if loader == lua.LNil {
			L.RaiseError("module '%s' not found", name)
			return 0
		}

		// Call the loader function
		L.Push(loader)
		L.Call(0, 1)

		// Cache the result
		result := L.Get(-1)
		loadedTable.RawSetString(name, result)

		L.Push(result)
		return 1
	}))

	return L
}

func TestCacheAPI_SetGet(t *testing.T) {
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	L := setupLuaState()
	defer L.Close()

	logger := log.NewHelper(log.DefaultLogger)
	RegisterCache(L, rdb, logger)

	// Test script
	script := `
		local cache = require "kratos_cache"

		-- Set a string value
		local ok = cache.set("test_key", "test_value", 60)
		if not ok then
			error("Failed to set cache")
		end

		-- Get the value back
		local val = cache.get("test_key")
		if val ~= "test_value" then
			error("Expected 'test_value', got: " .. tostring(val))
		end

		return val
	`

	err := L.DoString(script)
	if err != nil {
		t.Fatalf("Lua script error: %v", err)
	}

	result := L.Get(-1)
	if result.String() != "test_value" {
		t.Errorf("Expected 'test_value', got %s", result.String())
	}

	t.Log("✓ Set/Get test passed")
}

func TestCacheAPI_IncrDecr(t *testing.T) {
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	L := lua.NewState()
	defer L.Close()

	logger := log.NewHelper(log.DefaultLogger)
	RegisterCache(L, rdb, logger)

	script := `
		-- Increment counter
		local val1 = cache.incr("counter")
		if val1 ~= 1 then
			error("Expected 1, got: " .. tostring(val1))
		end

		local val2 = cache.incr("counter")
		if val2 ~= 2 then
			error("Expected 2, got: " .. tostring(val2))
		end

		-- Decrement
		local val3 = cache.decr("counter")
		if val3 ~= 1 then
			error("Expected 1, got: " .. tostring(val3))
		end

		-- IncrBy
		local val4 = cache.incrby("counter", 10)
		if val4 ~= 11 then
			error("Expected 11, got: " .. tostring(val4))
		end

		return val4
	`

	err := L.DoString(script)
	if err != nil {
		t.Fatalf("Lua script error: %v", err)
	}

	t.Log("✓ Incr/Decr test passed")
}

func TestCacheAPI_Exists(t *testing.T) {
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	L := lua.NewState()
	defer L.Close()

	logger := log.NewHelper(log.DefaultLogger)
	RegisterCache(L, rdb, logger)

	script := `
		-- Check non-existent key
		local exists1 = cache.exists("nonexistent")
		if exists1 then
			error("Key should not exist")
		end

		-- Set and check
		cache.set("existing", "value")
		local exists2 = cache.exists("existing")
		if not exists2 then
			error("Key should exist")
		end

		-- Delete and check
		cache.delete("existing")
		local exists3 = cache.exists("existing")
		if exists3 then
			error("Key should not exist after delete")
		end

		return true
	`

	err := L.DoString(script)
	if err != nil {
		t.Fatalf("Lua script error: %v", err)
	}

	t.Log("✓ Exists test passed")
}

func TestCacheAPI_TTL(t *testing.T) {
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	L := lua.NewState()
	defer L.Close()

	logger := log.NewHelper(log.DefaultLogger)
	RegisterCache(L, rdb, logger)

	script := `
		-- Set with TTL
		cache.set("temp_key", "value", 300)

		-- Check TTL
		local ttl = cache.ttl("temp_key")
		if ttl < 290 or ttl > 300 then
			error("TTL should be around 300, got: " .. tostring(ttl))
		end

		-- Update TTL
		cache.expire("temp_key", 600)
		local new_ttl = cache.ttl("temp_key")
		if new_ttl < 590 or new_ttl > 600 then
			error("New TTL should be around 600, got: " .. tostring(new_ttl))
		end

		return true
	`

	err := L.DoString(script)
	if err != nil {
		t.Fatalf("Lua script error: %v", err)
	}

	t.Log("✓ TTL test passed")
}

func TestCacheAPI_Hash(t *testing.T) {
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	L := lua.NewState()
	defer L.Close()

	logger := log.NewHelper(log.DefaultLogger)
	RegisterCache(L, rdb, logger)

	script := `
		-- Set hash fields
		cache.hset("user:1", "name", "John")
		cache.hset("user:1", "email", "john@example.com")
		cache.hset("user:1", "age", "30")

		-- Get single field
		local name = cache.hget("user:1", "name")
		if name ~= "John" then
			error("Expected 'John', got: " .. tostring(name))
		end

		-- Get all fields
		local user = cache.hgetall("user:1")
		if user.name ~= "John" then
			error("Expected name='John'")
		end
		if user.email ~= "john@example.com" then
			error("Expected email='john@example.com'")
		end

		return true
	`

	err := L.DoString(script)
	if err != nil {
		t.Fatalf("Lua script error: %v", err)
	}

	t.Log("✓ Hash operations test passed")
}

func TestCacheAPI_Keys(t *testing.T) {
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	L := lua.NewState()
	defer L.Close()

	logger := log.NewHelper(log.DefaultLogger)
	RegisterCache(L, rdb, logger)

	script := `
		-- Set multiple keys
		cache.set("user:1", "alice")
		cache.set("user:2", "bob")
		cache.set("user:3", "charlie")
		cache.set("post:1", "hello")

		-- Find user keys
		local keys = cache.keys("user:*")
		if #keys ~= 3 then
			error("Expected 3 user keys, got: " .. tostring(#keys))
		end

		return #keys
	`

	err := L.DoString(script)
	if err != nil {
		t.Fatalf("Lua script error: %v", err)
	}

	result := L.Get(-1)
	if result.String() != "3" {
		t.Errorf("Expected 3 keys, got %s", result.String())
	}

	t.Log("✓ Keys pattern matching test passed")
}

func TestCacheAPI_JSONSupport(t *testing.T) {
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	L := lua.NewState()
	defer L.Close()

	logger := log.NewHelper(log.DefaultLogger)
	// Register both cache and logger APIs
	RegisterLogger(L, logger)
	RegisterCache(L, rdb, logger)

	script := `
		-- Set complex object
		local user = {
			id = 123,
			name = "Alice",
			email = "alice@example.com",
			roles = {"admin", "user"}
		}

		cache.set("user:obj", user, 60)

		-- Get it back
		local retrieved = cache.get("user:obj")

		-- Debug: print type and content
		log.info("Retrieved type: " .. type(retrieved))
		if type(retrieved) == "table" then
			if retrieved.name then
				log.info("Name: " .. tostring(retrieved.name))
			end
		else
			log.info("Retrieved value: " .. tostring(retrieved))
		end

		if type(retrieved) ~= "table" then
			error("Expected table, got: " .. type(retrieved))
		end
		if retrieved.name ~= "Alice" then
			error("Expected name='Alice', got: " .. tostring(retrieved.name))
		end

		return true
	`

	err := L.DoString(script)
	if err != nil {
		t.Fatalf("Lua script error: %v", err)
	}

	t.Log("✓ JSON serialization test passed")
}
*/
