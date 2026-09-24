package api

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
)

// ModuleCache 构建语言无关的 cache 模块（JS 等基于 map[string]any 桥接的语言使用）。
// 约定：Go 函数返回 (T, error) 时，goja 会把非 nil error 转成 JS 异常，脚本用 try/catch 处理。
// rdb 为 nil 时返回不含函数的空模块。
func ModuleCache(rdb *redis.Client, logger *bLogger.Helper) ModuleDef {
	if rdb == nil {
		return ModuleDef{Name: "cache", Funcs: map[string]any{}}
	}

	bg := context.Background()
	logError := func(op string, err error) {
		if logger != nil {
			logger.Errorf(bg, "cache.%s error: %v", op, err)
		}
	}

	return ModuleDef{
		Name: "cache",
		Funcs: map[string]any{
			// get(key) → 值（JSON 自动解码）| nil
			"get": func(key string) (any, error) {
				val, err := rdb.Get(bg, key).Result()
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
				var ttlSeconds time.Duration
				if len(ttl) > 0 && ttl[0] > 0 {
					ttlSeconds = time.Duration(ttl[0]) * time.Second
				}
				if err := rdb.Set(bg, key, strVal, ttlSeconds).Err(); err != nil {
					logError("set", err)
					return false, err
				}
				return true, nil
			},
			"delete": func(key string) (bool, error) {
				if err := rdb.Del(bg, key).Err(); err != nil {
					logError("delete", err)
					return false, err
				}
				return true, nil
			},
			"exists": func(key string) (bool, error) {
				count, err := rdb.Exists(bg, key).Result()
				if err != nil {
					logError("exists", err)
					return false, err
				}
				return count > 0, nil
			},
			"expire": func(key string, ttl int) (bool, error) {
				ok, err := rdb.Expire(bg, key, time.Duration(ttl)*time.Second).Result()
				if err != nil {
					logError("expire", err)
					return false, err
				}
				return ok, nil
			},
			"incr": func(key string) (int64, error) {
				return rdb.Incr(bg, key).Result()
			},
			"decr": func(key string) (int64, error) {
				return rdb.Decr(bg, key).Result()
			},
			"incrby": func(key string, increment int) (int64, error) {
				return rdb.IncrBy(bg, key, int64(increment)).Result()
			},
			"ttl": func(key string) (int64, error) {
				duration, err := rdb.TTL(bg, key).Result()
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
			"keys": func(pattern string) ([]string, error) {
				return rdb.Keys(bg, pattern).Result()
			},
			"hget": func(key, field string) (any, error) {
				val, err := rdb.HGet(bg, key, field).Result()
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
				if err := rdb.HSet(bg, key, field, value).Err(); err != nil {
					logError("hset", err)
					return false, err
				}
				return true, nil
			},
			"hgetall": func(key string) (map[string]string, error) {
				return rdb.HGetAll(bg, key).Result()
			},
		},
	}
}
