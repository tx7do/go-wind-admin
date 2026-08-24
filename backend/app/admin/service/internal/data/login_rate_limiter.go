package data

import (
	"context"
	"fmt"
	"time"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/redis/go-redis/v9"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
)

// 登录失败计数与锁定相关常量（H5：登录限流）。
const (
	// loginFailThreshold 触发锁定的失败次数阈值。
	loginFailThreshold = 5
	// loginLockoutDuration 触发锁定后的窗口时长。
	loginLockoutDuration = 15 * time.Minute
	// loginFailKeyTTL 失败计数器的存活时长（滑动窗口内累计）。
	// 与锁定窗口一致：超过窗口未再失败则计数自然过期。
	loginFailKeyTTL = loginLockoutDuration
)

// 登录失败计数键格式。
const (
	// loginFailKeyByIP 按 IP 维度计数的 Redis 键。
	// 采用项目通用前缀 + IP，便于运维统一排查/清理。
	loginFailKeyByIPFmt = "gowind:login:fail:ip:%s"
	// loginFailKeyByUser 按用户名维度计数的 Redis 键。
	loginFailKeyByUserFmt = "gowind:login:fail:user:%s"
)

// incrIfNotLockedScript 原子「判定是否已锁定 → 自增失败计数 → 设置 TTL」Lua 脚本。
// 语义：若当前计数已 >= 阈值，则判定为锁定状态、不再自增；否则自增并在首次失败时设置 TTL。
// 返回值：{已锁定 flag(0/1), 当前失败次数}
var incrIfNotLockedScript = redis.NewScript(`
	local key = KEYS[1]
	local threshold = tonumber(ARGV[1])
	local ttl = tonumber(ARGV[2])

	local current = tonumber(redis.call('GET', key) or "0")
	if current >= threshold then
		return {1, current}
	end

	current = redis.call('INCR', key)
	if current == 1 then
		redis.call('EXPIRE', key, ttl)
	end
	return {0, current}
`)

// LoginRateLimiter 基于 Redis 的登录失败计数器，按 IP 与用户名双维度限流。
// 用于 H5：防止登录暴力破解——失败超阈值后锁定相应窗口。
type LoginRateLimiter struct {
	log *bLogger.Helper
	rdb *redis.Client
}

// NewLoginRateLimiter 构造登录限流器。
func NewLoginRateLimiter(ctx *bootstrap.Context, rdb *redis.Client) *LoginRateLimiter {
	return &LoginRateLimiter{
		rdb: rdb,
		log: ctx.NewLoggerHelper("login/rate-limiter"),
	}
}

// CheckAndIncr 在登录失败时自增失败计数，并返回是否已触发锁定。
// 同时按 IP 与用户名两个维度计数；任一维度超阈值即视为锁定。
// locked=true 表示该 IP 或用户名已达失败阈值，调用方应拒绝登录并提示稍后重试。
// retryAfter 为建议的重试等待时长（当前固定为锁定窗口）。
func (l *LoginRateLimiter) CheckAndIncr(ctx context.Context, ip, username string) (locked bool, attempts int, retryAfter time.Duration, err error) {
	if l == nil || l.rdb == nil {
		// Redis 不可用时降级：不阻断登录（fail-open），仅记录告警。
		// 限流属于防御性增强，不应让 Redis 故障导致登录全部不可用。
		return false, 0, 0, nil
	}

	keys := l.failKeys(ip, username)
	threshold := int64(loginFailThreshold)
	ttl := int64(loginFailKeyTTL.Seconds())

	maxAttempts := 0
	for _, key := range keys {
		res, rerr := incrIfNotLockedScript.Run(ctx, l.rdb, []string{key}, threshold, ttl).Result()
		if rerr != nil {
			l.log.Errorf(ctx, "incr login fail counter failed (key=%s): %s", key, rerr.Error())
			continue
		}
		// res 形如 [1 "5"] 或 [0 "3"]
		vals, ok := res.([]interface{})
		if !ok || len(vals) < 2 {
			continue
		}
		isLocked, _ := vals[0].(int64)
		count, _ := vals[1].(int64)
		if int(count) > maxAttempts {
			maxAttempts = int(count)
		}
		if isLocked == 1 {
			l.log.Warnf(ctx, "login locked: key=%s, attempts=%d", key, count)
			return true, int(count), loginLockoutDuration, nil
		}
	}
	return false, maxAttempts, 0, nil
}

// IsLocked 查询当前 IP/用户名是否处于锁定状态（不自增计数）。
// 用于在真正执行登录前做前置拦截。
func (l *LoginRateLimiter) IsLocked(ctx context.Context, ip, username string) (bool, error) {
	if l == nil || l.rdb == nil {
		return false, nil
	}
	threshold := int64(loginFailThreshold)
	for _, key := range l.failKeys(ip, username) {
		cnt, rerr := l.rdb.Get(ctx, key).Int64()
		if rerr != nil && rerr != redis.Nil {
			l.log.Errorf(ctx, "get login fail counter failed (key=%s): %s", key, rerr.Error())
			continue
		}
		if cnt >= threshold {
			return true, nil
		}
	}
	return false, nil
}

// Reset 登录成功后清零失败计数。
func (l *LoginRateLimiter) Reset(ctx context.Context, ip, username string) {
	if l == nil || l.rdb == nil {
		return
	}
	for _, key := range l.failKeys(ip, username) {
		if err := l.rdb.Del(ctx, key).Err(); err != nil {
			l.log.Errorf(ctx, "reset login fail counter failed (key=%s): %s", key, err.Error())
		}
	}
}

// failKeys 根据 IP 与用户名生成两个维度的失败计数键。
// 空值维度跳过（例如内网无 X-Forwarded-For 时仅有用户名维度）。
func (l *LoginRateLimiter) failKeys(ip, username string) []string {
	keys := make([]string, 0, 2)
	if ip != "" {
		keys = append(keys, fmt.Sprintf(loginFailKeyByIPFmt, ip))
	}
	if username != "" {
		keys = append(keys, fmt.Sprintf(loginFailKeyByUserFmt, username))
	}
	return keys
}
