package api

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	lua "github.com/yuin/gopher-lua"

	"go-wind-admin/pkg/scripting/internal/convert"
)

// RegisterCache registers the Redis cache API for Lua as a requireable module
func RegisterCache(L *lua.LState, rdb *redis.Client, logger *log.Helper) {
	// Register in package.preload so it can be required
	L.PreloadModule("kratos_cache", LoaderCache(rdb, logger))
}

// LoaderCache 返回 cache 模块（kratos_cache）的 loader，供 go-scripts 引擎 RegisterModule 使用。
// rdb 为 nil 时返回空模块。
func LoaderCache(rdb *redis.Client, logger *log.Helper) lua.LGFunction {
	return func(L *lua.LState) int {
		if rdb == nil {
			L.Push(L.NewTable())
			return 1
		}
		// Create cache module
		cacheModule := L.NewTable()

		// cache.get(key)
		cacheModule.RawSetString("get", L.NewFunction(func(L *lua.LState) int {
			key := L.CheckString(1)

			val, err := rdb.Get(context.Background(), key).Result()
			if err != nil {
				if err == redis.Nil {
					// Key doesn't exist
					L.Push(lua.LNil)
					return 1
				}
				logger.Errorf("cache.get error: %v", err)
				L.Push(lua.LNil)
				L.Push(lua.LString(err.Error()))
				return 2
			}

			// Try to decode as JSON first
			var jsonVal interface{}
			if err := json.Unmarshal([]byte(val), &jsonVal); err == nil {
				// Successfully decoded JSON
				L.Push(convert.ToLuaValue(L, jsonVal))
			} else {
				// Return as string
				L.Push(lua.LString(val))
			}
			return 1
		}))

		// cache.set(key, value, ttl_seconds)
		cacheModule.RawSetString("set", L.NewFunction(func(L *lua.LState) int {
			key := L.CheckString(1)
			value := L.Get(2)
			ttl := L.OptInt(3, 0) // Default: no expiration

			// ConvertCode Lua value to Go value
			goVal := convert.ToGoValue(value)

			// Serialize to JSON if it's a table/object
			var strVal string
			switch v := goVal.(type) {
			case string:
				strVal = v
			case nil:
				strVal = ""
			default:
				// Serialize complex types as JSON
				jsonBytes, err := json.Marshal(goVal)
				if err != nil {
					logger.Errorf("cache.set JSON marshal error: %v", err)
					L.Push(lua.LBool(false))
					L.Push(lua.LString(err.Error()))
					return 2
				}
				strVal = string(jsonBytes)
			}

			// Set with expiration
			var err error
			if ttl > 0 {
				err = rdb.Set(context.Background(), key, strVal, time.Duration(ttl)*time.Second).Err()
			} else {
				err = rdb.Set(context.Background(), key, strVal, 0).Err()
			}

			if err != nil {
				logger.Errorf("cache.set error: %v", err)
				L.Push(lua.LBool(false))
				L.Push(lua.LString(err.Error()))
				return 2
			}

			L.Push(lua.LBool(true))
			return 1
		}))

		// cache.delete(key)
		cacheModule.RawSetString("delete", L.NewFunction(func(L *lua.LState) int {
			key := L.CheckString(1)

			err := rdb.Del(context.Background(), key).Err()
			if err != nil {
				logger.Errorf("cache.delete error: %v", err)
				L.Push(lua.LBool(false))
				L.Push(lua.LString(err.Error()))
				return 2
			}

			L.Push(lua.LBool(true))
			return 1
		}))

		// cache.exists(key)
		cacheModule.RawSetString("exists", L.NewFunction(func(L *lua.LState) int {
			key := L.CheckString(1)

			count, err := rdb.Exists(context.Background(), key).Result()
			if err != nil {
				logger.Errorf("cache.exists error: %v", err)
				L.Push(lua.LBool(false))
				return 1
			}

			L.Push(lua.LBool(count > 0))
			return 1
		}))

		// cache.expire(key, ttl_seconds)
		cacheModule.RawSetString("expire", L.NewFunction(func(L *lua.LState) int {
			key := L.CheckString(1)
			ttl := L.CheckInt(2)

			ok, err := rdb.Expire(context.Background(), key, time.Duration(ttl)*time.Second).Result()
			if err != nil {
				logger.Errorf("cache.expire error: %v", err)
				L.Push(lua.LBool(false))
				L.Push(lua.LString(err.Error()))
				return 2
			}

			L.Push(lua.LBool(ok))
			return 1
		}))

		// cache.incr(key)
		cacheModule.RawSetString("incr", L.NewFunction(func(L *lua.LState) int {
			key := L.CheckString(1)

			val, err := rdb.Incr(context.Background(), key).Result()
			if err != nil {
				logger.Errorf("cache.incr error: %v", err)
				L.Push(lua.LNil)
				L.Push(lua.LString(err.Error()))
				return 2
			}

			L.Push(lua.LNumber(val))
			return 1
		}))

		// cache.decr(key)
		cacheModule.RawSetString("decr", L.NewFunction(func(L *lua.LState) int {
			key := L.CheckString(1)

			val, err := rdb.Decr(context.Background(), key).Result()
			if err != nil {
				logger.Errorf("cache.decr error: %v", err)
				L.Push(lua.LNil)
				L.Push(lua.LString(err.Error()))
				return 2
			}

			L.Push(lua.LNumber(val))
			return 1
		}))

		// cache.incrby(key, increment)
		cacheModule.RawSetString("incrby", L.NewFunction(func(L *lua.LState) int {
			key := L.CheckString(1)
			increment := L.CheckInt(2)

			val, err := rdb.IncrBy(context.Background(), key, int64(increment)).Result()
			if err != nil {
				logger.Errorf("cache.incrby error: %v", err)
				L.Push(lua.LNil)
				L.Push(lua.LString(err.Error()))
				return 2
			}

			L.Push(lua.LNumber(val))
			return 1
		}))

		// cache.ttl(key) - Get remaining TTL in seconds
		cacheModule.RawSetString("ttl", L.NewFunction(func(L *lua.LState) int {
			key := L.CheckString(1)

			duration, err := rdb.TTL(context.Background(), key).Result()
			if err != nil {
				logger.Errorf("cache.ttl error: %v", err)
				L.Push(lua.LNumber(-2)) // Key doesn't exist
				return 1
			}

			if duration == -1 {
				L.Push(lua.LNumber(-1)) // No expiration
			} else if duration == -2 {
				L.Push(lua.LNumber(-2)) // Key doesn't exist
			} else {
				L.Push(lua.LNumber(int64(duration.Seconds())))
			}
			return 1
		}))

		// cache.keys(pattern) - Find keys by pattern
		cacheModule.RawSetString("keys", L.NewFunction(func(L *lua.LState) int {
			pattern := L.CheckString(1)

			keys, err := rdb.Keys(context.Background(), pattern).Result()
			if err != nil {
				logger.Errorf("cache.keys error: %v", err)
				L.Push(lua.LNil)
				L.Push(lua.LString(err.Error()))
				return 2
			}

			// ConvertCode to Lua table (array)
			table := L.NewTable()
			for i, key := range keys {
				table.RawSetInt(i+1, lua.LString(key))
			}

			L.Push(table)
			return 1
		}))

		// cache.hget(key, field) - Get hash field
		cacheModule.RawSetString("hget", L.NewFunction(func(L *lua.LState) int {
			key := L.CheckString(1)
			field := L.CheckString(2)

			val, err := rdb.HGet(context.Background(), key, field).Result()
			if err != nil {
				if err == redis.Nil {
					L.Push(lua.LNil)
					return 1
				}
				logger.Errorf("cache.hget error: %v", err)
				L.Push(lua.LNil)
				L.Push(lua.LString(err.Error()))
				return 2
			}

			L.Push(lua.LString(val))
			return 1
		}))

		// cache.hset(key, field, value) - Set hash field
		cacheModule.RawSetString("hset", L.NewFunction(func(L *lua.LState) int {
			key := L.CheckString(1)
			field := L.CheckString(2)
			value := L.CheckString(3)

			err := rdb.HSet(context.Background(), key, field, value).Err()
			if err != nil {
				logger.Errorf("cache.hset error: %v", err)
				L.Push(lua.LBool(false))
				L.Push(lua.LString(err.Error()))
				return 2
			}

			L.Push(lua.LBool(true))
			return 1
		}))

		// cache.hgetall(key) - Get all hash fields
		cacheModule.RawSetString("hgetall", L.NewFunction(func(L *lua.LState) int {
			key := L.CheckString(1)

			vals, err := rdb.HGetAll(context.Background(), key).Result()
			if err != nil {
				logger.Errorf("cache.hgetall error: %v", err)
				L.Push(lua.LNil)
				L.Push(lua.LString(err.Error()))
				return 2
			}

			// ConvertCode to Lua table (map)
			table := L.NewTable()
			for field, value := range vals {
				table.RawSetString(field, lua.LString(value))
			}

			L.Push(table)
			return 1
		}))

		L.Push(cacheModule)
		return 1
	}
}
