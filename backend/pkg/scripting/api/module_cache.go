package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tx7do/go-scripts/hostmodule"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
)

// ModuleCache 构建语言无关的 cache 模块（JS 等基于 map[string]any 桥接的语言使用）。
// 约定：Go 函数返回 (T, error) 时，goja 会把非 nil error 转成 JS 异常，脚本用 try/catch 处理。
// rdb 为 nil 时返回不含函数的空模块。
//
// 所有 key 均经 scriptCacheKey 自动归一到 script: 命名空间（见 cache_namespace.go）：
// Redis 里另存着平台令牌/MFA 密钥/验证码等资产键，脚本不得触及。
func ModuleCache(rdb *redis.Client, logger *bLogger.Helper) hostmodule.ModuleDef {
	if rdb == nil {
		return hostmodule.ModuleDef{Name: "cache", Funcs: map[string]any{}}
	}

	bg := context.Background()
	logError := func(op string, err error) {
		if logger != nil {
			logger.Errorf(bg, "cache.%s error: %v", op, err)
		}
	}

	return hostmodule.ModuleDef{
		Name: "cache",
		Funcs: map[string]any{
			// get(key) → 值（JSON 自动解码）| nil
			"get": func(key string) (any, error) {
				rkey, kerr := scriptCacheKey(key)
				if kerr != nil {
					return nil, kerr
				}
				val, err := rdb.Get(bg, rkey).Result()
				if err != nil {
					if err == redis.Nil {
						return nil, nil
					}
					logError("get", err)
					return nil, err
				}
				var jsonVal any
				if err := json.Unmarshal([]byte(val), &jsonVal); err == nil {
					return jsonVal, nil
				}
				return val, nil
			},
			// set(key, value, ttl?) → bool；复杂类型序列化为 JSON
			"set": func(key string, value any, ttl ...int) (bool, error) {
				rkey, kerr := scriptCacheKey(key)
				if kerr != nil {
					return false, kerr
				}
				var strVal string
				switch v := value.(type) {
				case string:
					strVal = v
				case nil:
					strVal = ""
				default:
					jsonBytes, err := json.Marshal(v)
					if err != nil {
						logError("set", err)
						return false, err
					}
					strVal = string(jsonBytes)
				}
				// 写入值限额：防止脚本向 Redis 灌大对象撑爆内存
				if len(strVal) > scriptCacheMaxValueSize {
					return false, fmt.Errorf("cache value exceeds %d bytes", scriptCacheMaxValueSize)
				}
				var ttlSeconds time.Duration
				if len(ttl) > 0 && ttl[0] > 0 {
					ttlSeconds = time.Duration(ttl[0]) * time.Second
				}
				if err := rdb.Set(bg, rkey, strVal, ttlSeconds).Err(); err != nil {
					logError("set", err)
					return false, err
				}
				return true, nil
			},
			"delete": func(key string) (bool, error) {
				rkey, kerr := scriptCacheKey(key)
				if kerr != nil {
					return false, kerr
				}
				if err := rdb.Del(bg, rkey).Err(); err != nil {
					logError("delete", err)
					return false, err
				}
				return true, nil
			},
			"exists": func(key string) (bool, error) {
				rkey, kerr := scriptCacheKey(key)
				if kerr != nil {
					return false, kerr
				}
				count, err := rdb.Exists(bg, rkey).Result()
				if err != nil {
					logError("exists", err)
					return false, err
				}
				return count > 0, nil
			},
			"expire": func(key string, ttl int) (bool, error) {
				rkey, kerr := scriptCacheKey(key)
				if kerr != nil {
					return false, kerr
				}
				ok, err := rdb.Expire(bg, rkey, time.Duration(ttl)*time.Second).Result()
				if err != nil {
					logError("expire", err)
					return false, err
				}
				return ok, nil
			},
			"incr": func(key string) (int64, error) {
				rkey, kerr := scriptCacheKey(key)
				if kerr != nil {
					return 0, kerr
				}
				return rdb.Incr(bg, rkey).Result()
			},
			"decr": func(key string) (int64, error) {
				rkey, kerr := scriptCacheKey(key)
				if kerr != nil {
					return 0, kerr
				}
				return rdb.Decr(bg, rkey).Result()
			},
			"incrby": func(key string, increment int) (int64, error) {
				rkey, kerr := scriptCacheKey(key)
				if kerr != nil {
					return 0, kerr
				}
				return rdb.IncrBy(bg, rkey, int64(increment)).Result()
			},
			"ttl": func(key string) (int64, error) {
				rkey, kerr := scriptCacheKey(key)
				if kerr != nil {
					return -2, kerr
				}
				duration, err := rdb.TTL(bg, rkey).Result()
				if err != nil {
					return -2, err
				}
				switch duration {
				case -1:
					return -1, nil // 无过期时间
				case -2:
					return -2, nil // key 不存在
				default:
					return int64(duration.Seconds()), nil
				}
			},
			// keys(pattern) → 命名空间内的键（SCAN 限量，不阻塞实例）
			"keys": func(pattern string) ([]string, error) {
				return scriptCacheScanKeys(bg, rdb, pattern)
			},
			"hget": func(key, field string) (any, error) {
				rkey, kerr := scriptCacheKey(key)
				if kerr != nil {
					return nil, kerr
				}
				val, err := rdb.HGet(bg, rkey, field).Result()
				if err != nil {
					if err == redis.Nil {
						return nil, nil
					}
					logError("hget", err)
					return nil, err
				}
				return val, nil
			},
			"hset": func(key, field, value string) (bool, error) {
				rkey, kerr := scriptCacheKey(key)
				if kerr != nil {
					return false, kerr
				}
				if len(value) > scriptCacheMaxValueSize {
					return false, fmt.Errorf("cache value exceeds %d bytes", scriptCacheMaxValueSize)
				}
				if err := rdb.HSet(bg, rkey, field, value).Err(); err != nil {
					logError("hset", err)
					return false, err
				}
				return true, nil
			},
			"hgetall": func(key string) (map[string]string, error) {
				rkey, kerr := scriptCacheKey(key)
				if kerr != nil {
					return nil, kerr
				}
				return rdb.HGetAll(bg, rkey).Result()
			},
		},
	}
}
