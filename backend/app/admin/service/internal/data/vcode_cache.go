package data

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	bLogger "github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/kratos-bootstrap/logger"

	"go-wind-admin/pkg/serviceid"
)

// vcodeKeyPrefix 验证码键前缀（与 captcha 同前缀域便于统一清理）
const vcodeKeyPrefix = serviceid.ProjectName + ":vcode:"

// vcodeMaxVerifyAttempts 单枚验证码允许的最大校验失败次数：达到即销毁验证码、
// 强制重新发码。免鉴权的验证码消费端点（找回密码/换绑联系方式）此前无失败计数，
// 6 位数字码（10^6 空间）可在 10 分钟 TTL 内被在线爆破跑完。
const vcodeMaxVerifyAttempts = 5

// vcodeCounterTTL 失败计数器的存活时长，与两处发码调用方的码 TTL（10 分钟）一致。
const vcodeCounterTTL = 10 * time.Minute

// verifyVCodeScript 原子校验 Lua：失败计数 → 比对 → 消费/销毁单次往返完成。
// 此前 GET→比较→成功才 DEL 三步非原子，并发请求可同时通过"单次有效"。
// KEYS[1]=验证码键 KEYS[2]=失败计数键；ARGV[1]=提交的码 ARGV[2]=最大失败次数
// ARGV[3]=计数器 TTL 秒。
// 返回：1=命中并消费；0=验证码不存在/已过期；-1=失败次数超限（已销毁，须重新发码）；
// -2=不匹配。
var verifyVCodeScript = redis.NewScript(`
local code = redis.call('GET', KEYS[1])
if not code then
	return 0
end
local attempts = tonumber(redis.call('GET', KEYS[2]) or '0')
if attempts >= tonumber(ARGV[2]) then
	redis.call('DEL', KEYS[1], KEYS[2])
	return -1
end
if code == ARGV[1] then
	redis.call('DEL', KEYS[1], KEYS[2])
	return 1
end
attempts = attempts + 1
redis.call('SET', KEYS[2], attempts, 'EX', ARGV[3])
return -2
`)

// VCodeCache 一次性验证码缓存（Redis）。
// purpose 区分业务场景（reset_password / bind_contact），
// 校验成功即删除（单次有效），TTL 到期自动清理；
// 校验失败计数超限即销毁（防在线爆破），发码侧有频控（防轰炸）。
type VCodeCache struct {
	rdb *redis.Client
	log *logger.Helper
}

func NewVCodeCache(ctx *bLogger.Context, rdb *redis.Client) *VCodeCache {
	return &VCodeCache{
		rdb: rdb,
		log: ctx.NewLoggerHelper("vcode/cache"),
	}
}

// vcodeKeys 返回验证码键与失败计数键。
func vcodeKeys(purpose, identifier string) (string, string) {
	base := vcodeKeyPrefix + purpose + ":" + identifier
	return base, base + ":attempts"
}

// Save 保存验证码，ttl 到期自动删除；同时重置该标识的失败计数
//（新码 = 新的尝试预算，正常用户重试不被旧失败拖累）。
func (c *VCodeCache) Save(purpose, identifier, code string, ttl time.Duration) error {
	key, attemptsKey := vcodeKeys(purpose, identifier)
	pipe := c.rdb.Pipeline()
	pipe.Set(context.Background(), key, code, ttl)
	pipe.Del(context.Background(), attemptsKey)
	if _, err := pipe.Exec(context.Background()); err != nil {
		c.log.Errorf(context.Background(), "save vcode failed: %v", err)
		return err
	}
	return nil
}

// Verify 校验验证码（消费型：匹配即删除，单次有效；失败计数超限即销毁）。
// 仅在原子比对命中时返回 true；超限锁定/不匹配/不存在均返回 false，
// 不向调用方区分锁定与错误（避免给爆破者反馈防御状态）。
func (c *VCodeCache) Verify(purpose, identifier, code string) bool {
	key, attemptsKey := vcodeKeys(purpose, identifier)
	status, err := verifyVCodeScript.Run(context.Background(), c.rdb,
		[]string{key, attemptsKey}, code, vcodeMaxVerifyAttempts,
		int(vcodeCounterTTL.Seconds()),
	).Int()
	if err != nil {
		c.log.Errorf(context.Background(), "verify vcode failed: %v", err)
		return false
	}
	switch status {
	case 1:
		return true
	case -1:
		c.log.Warnf(context.Background(), "vcode [%s:%s] destroyed after %d failed attempts",
			purpose, identifier, vcodeMaxVerifyAttempts)
	}
	return false
}

// Throttle 发码频控：同一 purpose+identifier 在 window 内只允许发一次。
// 返回 false 表示处于冷却期（本次不应发码）；调用方应静默成功（保持防枚举口径一致）。
func (c *VCodeCache) Throttle(purpose, identifier string, window time.Duration) bool {
	key := vcodeKeyPrefix + purpose + ":throttle:" + identifier
	ok, err := c.rdb.SetNX(context.Background(), key, 1, window).Result()
	if err != nil {
		// 频控组件故障 fail-open：宁可多发不可拦死正常用户的找回流程
		c.log.Errorf(context.Background(), "vcode throttle check failed: %v", err)
		return true
	}
	return ok
}
