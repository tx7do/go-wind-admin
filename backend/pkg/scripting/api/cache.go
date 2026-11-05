package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	lua "github.com/yuin/gopher-lua"

	"github.com/tx7do/go-scripts/lua/convert"
)

// RegisterCache registers the Redis cache API for Lua as a requireable module
func RegisterCache(L *lua.LState, rdb *redis.Client, logger *bLogger.Helper) {
	// Register in package.preload so it can be required
	L.PreloadModule("kratos_cache", LoaderCache(rdb, logger))
}

// LoaderCache 返回 cache 模块（kratos_cache）的 loader，供 go-scripts 引擎 RegisterModule 使用。
// rdb 为 nil 时返回空模块。
//
// 所有 key 均经 scriptCacheKey 自动归一到 script: 命名空间（见 cache_namespace.go）：
// Redis 里另存着平台令牌/MFA 密钥/验证码等资产键，脚本不得触及。
func LoaderCache(rdb *redis.Client, logger *bLogger.Helper) lua.LGFunction {
	return func(L *lua.LState) int {
		if rdb == nil {
			L.Push(L.NewTable())
			return 1
		}
		// Create cache module
		cacheModule := L.NewTable()

		// cache.get(key)
		cacheModule.RawSetString("get", L.NewFunction(func(L *lua.LState) int {
			key, kerr := scriptCacheKey(L.CheckString(1))
			if kerr != nil {
				L.Push(lua.LNil)
				L.Push(lua.LString(kerr.Error()))
				return 2
			}

			val, err := rdb.Get(context.Background(), key).Result()
			if err != nil {
				if err == redis.Nil {
					// Key doesn't exist
					L.Push(lua.LNil)
					return 1
				}
				logger.Errorf(context.Background(), "cache.get error: %v", err)
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
			key, kerr := scriptCacheKey(L.CheckString(1))
			if kerr != nil {
				L.Push(lua.LBool(false))
				L.Push(lua.LString(kerr.Error()))
				return 2
			}
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
					logger.Errorf(context.Background(), "cache.set JSON marshal error: %v", err)
					L.Push(lua.LBool(false))
					L.Push(lua.LString(err.Error()))
					return 2
				}
				strVal = string(jsonBytes)
			}

			// 写入值限额：防止脚本向 Redis 灌大对象撑爆内存
			if len(strVal) > scriptCacheMaxValueSize {
				L.Push(lua.LBool(false))
				L.Push(lua.LString(fmt.Sprintf("cache value exceeds %d bytes", scriptCacheMaxValueSize)))
				return 2
			}

			// Set with expiration
			var err error
			if ttl > 0 {
				err = rdb.Set(context.Background(), key, strVal, time.Duration(ttl)*time.Second).Err()
			} else {
				err = rdb.Set(context.Background(), key, strVal, 0).Err()
			}

			if err != nil {
				logger.Errorf(context.Background(), "cache.set error: %v", err)
				L.Push(lua.LBool(false))
				L.Push(lua.LString(err.Error()))
				return 2
			}

			L.Push(lua.LBool(true))
			return 1
		}))

		// cache.delete(key)
		cacheModule.RawSetString("delete", L.NewFunction(func(L *lua.LState) int {
			key, kerr := scriptCacheKey(L.CheckString(1))
			if kerr != nil {
				L.Push(lua.LBool(false))
				L.Push(lua.LString(kerr.Error()))
				return 2
			}

			err := rdb.Del(context.Background(), key).Err()
			if err != nil {
				logger.Errorf(context.Background(), "cache.delete error: %v", err)
				L.Push(lua.LBool(false))
				L.Push(lua.LString(err.Error()))
				return 2
			}

			L.Push(lua.LBool(true))
			return 1
		}))

		// cache.exists(key)
		cacheModule.RawSetString("exists", L.NewFunction(func(L *lua.LState) int {
			key, kerr := scriptCacheKey(L.CheckString(1))
			if kerr != nil {
				L.Push(lua.LBool(false))
				return 1
			}

			count, err := rdb.Exists(context.Background(), key).Result()
			if err != nil {
				logger.Errorf(context.Background(), "cache.exists error: %v", err)
				L.Push(lua.LBool(false))
				return 1
			}

			L.Push(lua.LBool(count > 0))
			return 1
		}))

		// cache.expire(key, ttl_seconds)
		cacheModule.RawSetString("expire", L.NewFunction(func(L *lua.LState) int {
			key, kerr := scriptCacheKey(L.CheckString(1))
			if kerr != nil {
				L.Push(lua.LBool(false))
				L.Push(lua.LString(kerr.Error()))
				return 2
			}
			ttl := L.CheckInt(2)

			ok, err := rdb.Expire(context.Background(), key, time.Duration(ttl)*time.Second).Result()
			if err != nil {
				logger.Errorf(context.Background(), "cache.expire error: %v", err)
				L.Push(lua.LBool(false))
				L.Push(lua.LString(err.Error()))
				return 2
			}

			L.Push(lua.LBool(ok))
			return 1
		}))

		// cache.incr(key)
		cacheModule.RawSetString("incr", L.NewFunction(func(L *lua.LState) int {
			key, kerr := scriptCacheKey(L.CheckString(1))
			if kerr != nil {
				L.Push(lua.LNil)
				L.Push(lua.LString(kerr.Error()))
				return 2
			}

			val, err := rdb.Incr(context.Background(), key).Result()
			if err != nil {
				logger.Errorf(context.Background(), "cache.incr error: %v", err)
				L.Push(lua.LNil)
				L.Push(lua.LString(err.Error()))
				return 2
			}

			L.Push(lua.LNumber(val))
			return 1
		}))

		// cache.decr(key)
		cacheModule.RawSetString("decr", L.NewFunction(func(L *lua.LState) int {
			key, kerr := scriptCacheKey(L.CheckString(1))
			if kerr != nil {
				L.Push(lua.LNil)
				L.Push(lua.LString(kerr.Error()))
				return 2
			}

			val, err := rdb.Decr(context.Background(), key).Result()
			if err != nil {
				logger.Errorf(context.Background(), "cache.decr error: %v", err)
				L.Push(lua.LNil)
				L.Push(lua.LString(err.Error()))
				return 2
			}

			L.Push(lua.LNumber(val))
			return 1
		}))

		// cache.incrby(key, increment)
		cacheModule.RawSetString("incrby", L.NewFunction(func(L *lua.LState) int {
			key, kerr := scriptCacheKey(L.CheckString(1))
			if kerr != nil {
				L.Push(lua.LNil)
				L.Push(lua.LString(kerr.Error()))
				return 2
			}
			increment := L.CheckInt(2)

			val, err := rdb.IncrBy(context.Background(), key, int64(increment)).Result()
			if err != nil {
				logger.Errorf(context.Background(), "cache.incrby error: %v", err)
				L.Push(lua.LNil)
				L.Push(lua.LString(err.Error()))
				return 2
			}

			L.Push(lua.LNumber(val))
			return 1
		}))

		// cache.ttl(key) - Get remaining TTL in seconds
		cacheModule.RawSetString("ttl", L.NewFunction(func(L *lua.LState) int {
			key, kerr := scriptCacheKey(L.CheckString(1))
			if kerr != nil {
				L.Push(lua.LNumber(-2)) // Key doesn't exist
				return 1
			}

			duration, err := rdb.TTL(context.Background(), key).Result()
			if err != nil {
				logger.Errorf(context.Background(), "cache.ttl error: %v", err)
				L.Push(lua.LNumber(-2)) // Key doesn't exist
				return 1
			}

			switch duration {
			case -1:
				L.Push(lua.LNumber(-1)) // No expiration
			case -2:
				L.Push(lua.LNumber(-2)) // Key doesn't exist
			default:
				L.Push(lua.LNumber(int64(duration.Seconds())))
			}
			return 1
		}))

		// cache.keys(pattern) - Find keys by pattern
		// 模式同样归一到 script: 命名空间，且走 SCAN 限量（KEYS 会阻塞整个实例）。
		cacheModule.RawSetString("keys", L.NewFunction(func(L *lua.LState) int {
			pattern := L.CheckString(1)

			keys, err := scriptCacheScanKeys(context.Background(), rdb, pattern)
			if err != nil {
				logger.Errorf(context.Background(), "cache.keys error: %v", err)
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
			key, kerr := scriptCacheKey(L.CheckString(1))
			if kerr != nil {
				L.Push(lua.LNil)
				L.Push(lua.LString(kerr.Error()))
				return 2
			}
			field := L.CheckString(2)

			val, err := rdb.HGet(context.Background(), key, field).Result()
			if err != nil {
				if err == redis.Nil {
					L.Push(lua.LNil)
					return 1
				}
				logger.Errorf(context.Background(), "cache.hget error: %v", err)
				L.Push(lua.LNil)
				L.Push(lua.LString(err.Error()))
				return 2
			}

			L.Push(lua.LString(val))
			return 1
		}))

		// cache.hset(key, field, value) - Set hash field
		cacheModule.RawSetString("hset", L.NewFunction(func(L *lua.LState) int {
			key, kerr := scriptCacheKey(L.CheckString(1))
			if kerr != nil {
				L.Push(lua.LBool(false))
				L.Push(lua.LString(kerr.Error()))
				return 2
			}
			field := L.CheckString(2)
			value := L.CheckString(3)

			if len(value) > scriptCacheMaxValueSize {
				L.Push(lua.LBool(false))
				L.Push(lua.LString(fmt.Sprintf("cache value exceeds %d bytes", scriptCacheMaxValueSize)))
				return 2
			}

			err := rdb.HSet(context.Background(), key, field, value).Err()
			if err != nil {
				logger.Errorf(context.Background(), "cache.hset error: %v", err)
				L.Push(lua.LBool(false))
				L.Push(lua.LString(err.Error()))
				return 2
			}

			L.Push(lua.LBool(true))
			return 1
		}))

		// cache.hgetall(key) - Get all hash fields
		cacheModule.RawSetString("hgetall", L.NewFunction(func(L *lua.LState) int {
			key, kerr := scriptCacheKey(L.CheckString(1))
			if kerr != nil {
				L.Push(lua.LNil)
				L.Push(lua.LString(kerr.Error()))
				return 2
			}

			vals, err := rdb.HGetAll(context.Background(), key).Result()
			if err != nil {
				logger.Errorf(context.Background(), "cache.hgetall error: %v", err)
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
