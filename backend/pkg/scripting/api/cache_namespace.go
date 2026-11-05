package api

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// 脚本 cache 模块的命名空间隔离与限额。
//
// Redis 中同时存放着平台最敏感的运行时资产：访问/刷新令牌（at:/rt:/rtg:，明文值、
// 令牌校验即与 Redis 值精确比对）、MFA TOTP 密钥（mfa:）、一次性验证码、会话/黑名单
// 台账（us:/bl:/lin:）等。脚本此前可用任意 key 读写这些资产——cache.keys("at:*")+get
// 即可窃取任意用户（含平台管理员）的会话，cache.set 可向令牌槽写入自己的令牌完成
// 会话伪造，cache.keys("*")（底层 KEYS 命令）还会阻塞整个 Redis 实例。
//
// 处置：所有 cache 操作的 key 一律自动归一到 script: 命名空间内——
//   - 对既有脚本零破坏（原 key 原样加前缀继续可用）；
//   - 无绕过可能（任何输入都映射进命名空间，"at:..." 变成 "script:at:..."，与真实
//     令牌键 "at:..." 不同名）；
//   - keys 模式同样加前缀 + 改 SCAN 限量，扫不出命名空间外的键，也不会阻塞实例。
const (
	scriptCacheKeyPrefix = "script:"
	// scriptCacheMaxKeys keys 扫描返回的键数上限。
	scriptCacheMaxKeys = 200
	// scriptCacheMaxValueSize set 写入值（序列化后）的大小上限。
	scriptCacheMaxValueSize = 64 << 10 // 64KB
	// scriptCacheMaxKeyLength 单个 key 长度上限（含前缀）。
	scriptCacheMaxKeyLength = 512
)

// scriptCacheKey 把脚本侧的 key 归一到 script: 命名空间。
// 仅做长度校验与前缀拼接，脚本无法通过任何输入逃出命名空间。
func scriptCacheKey(key string) (string, error) {
	if key == "" {
		return "", errCacheNamespace("cache key is empty")
	}
	full := scriptCacheKeyPrefix + key
	if len(full) > scriptCacheMaxKeyLength {
		return "", errCacheNamespace("cache key too long (max %d bytes)", scriptCacheMaxKeyLength)
	}
	return full, nil
}

// scriptCachePattern 把 keys 的匹配模式归一到 script: 命名空间。
// "*" 之类的全量模式也只会落在 "script:*" 内。
func scriptCachePattern(pattern string) (string, error) {
	if pattern == "" {
		return "", errCacheNamespace("cache key pattern is empty")
	}
	return scriptCacheKeyPrefix + pattern, nil
}

// scriptCacheScanKeys 用 SCAN 按 pattern 扫描命名空间内的键，最多返回 scriptCacheMaxKeys 个。
// 取代 KEYS 命令（KEYS 在大库上会阻塞整个 Redis 实例）。
func scriptCacheScanKeys(ctx context.Context, rdb *redis.Client, pattern string) ([]string, error) {
	full, err := scriptCachePattern(pattern)
	if err != nil {
		return nil, err
	}

	var keys []string
	var cursor uint64
	for {
		batch, next, err := rdb.Scan(ctx, cursor, full, 100).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, batch...)
		if next == 0 || len(keys) >= scriptCacheMaxKeys {
			break
		}
		cursor = next
	}
	if len(keys) > scriptCacheMaxKeys {
		keys = keys[:scriptCacheMaxKeys]
	}
	return keys, nil
}

// errCacheNamespace 构造命名空间守卫错误。
func errCacheNamespace(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
