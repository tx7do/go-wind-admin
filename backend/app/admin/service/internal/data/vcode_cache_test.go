// VCodeCache 的 miniredis 测试。
//
// 一次性验证码缓存（找回密码 / 换绑联系方式），测试盯四件事：
//   - 消费型校验：匹配即删，同一个码第二次验证必失败（防重放）；
//   - purpose 隔离：reset_password 与 bind_contact 的码互不可用——两场景共用
//     identifier（如同一邮箱）时不能串门；
//   - 错误码不消耗：输错后正确码仍可用（用户手滑不应废掉整封邮件）；
//   - TTL 到期自动失效。
package data

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	conf "github.com/tx7do/kratos-bootstrap/api/gen/go/conf/v1"
	bBootstrap "github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
)

// newVCodeCacheForTest 起 miniredis 并构造缓存，返回供 FastForward 操控时钟。
func newVCodeCacheForTest(t *testing.T) (*VCodeCache, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err, "miniredis 启动失败")
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	bctx := bBootstrap.NewContextWithParam(
		context.Background(),
		&conf.AppInfo{},
		&conf.Bootstrap{Data: &conf.Data{Redis: &conf.Data_Redis{Addr: mr.Addr()}}},
		bLogger.NopLogger(),
	)
	return NewVCodeCache(bctx, rdb), mr
}

// TestVCodeCache_VerifyConsumesOnMatch 验证消费型语义：
// 匹配即删——同一码第二次验证失败（一次性），不匹配不消耗。
func TestVCodeCache_VerifyConsumesOnMatch(t *testing.T) {
	c, _ := newVCodeCacheForTest(t)

	require.NoError(t, c.Save("reset_password", "user@example.com", "123456", 5*time.Minute))

	require.True(t, c.Verify("reset_password", "user@example.com", "123456"),
		"正确的码应验证通过")
	require.False(t, c.Verify("reset_password", "user@example.com", "123456"),
		"同一码第二次验证应失败（消费型，防重放）")
}

// TestVCodeCache_WrongCodeNotConsumed 验证错误码不消耗：
// 输错后正确码仍可用——验证码邮件不该被一次手滑废掉。
func TestVCodeCache_WrongCodeNotConsumed(t *testing.T) {
	c, _ := newVCodeCacheForTest(t)

	require.NoError(t, c.Save("reset_password", "user@example.com", "123456", 5*time.Minute))

	require.False(t, c.Verify("reset_password", "user@example.com", "999999"),
		"错误的码应验证失败")
	require.True(t, c.Verify("reset_password", "user@example.com", "123456"),
		"错误尝试不应消耗正确码")
}

// TestVCodeCache_PurposeIsolation 验证 purpose 隔离：
// 同一 identifier 在不同业务场景下的码互不可用。
func TestVCodeCache_PurposeIsolation(t *testing.T) {
	c, _ := newVCodeCacheForTest(t)

	require.NoError(t, c.Save("reset_password", "user@example.com", "123456", 5*time.Minute))

	require.False(t, c.Verify("bind_contact", "user@example.com", "123456"),
		"reset_password 的码不能用于 bind_contact 场景")
	require.True(t, c.Verify("reset_password", "user@example.com", "123456"),
		"错误场景的验证不应消耗原码")
}

// TestVCodeCache_IdentifierIsolation 验证 identifier 隔离：
// 同场景不同接收人的码互不串门。
func TestVCodeCache_IdentifierIsolation(t *testing.T) {
	c, _ := newVCodeCacheForTest(t)

	require.NoError(t, c.Save("reset_password", "a@example.com", "111111", 5*time.Minute))
	require.NoError(t, c.Save("reset_password", "b@example.com", "222222", 5*time.Minute))

	require.True(t, c.Verify("reset_password", "b@example.com", "222222"))
	require.False(t, c.Verify("reset_password", "a@example.com", "222222"),
		"b 的码不能验证 a 的验证码")
}

// TestVCodeCache_TTLExpiry 验证过期语义：TTL 到期后验证失败。
// Save 的 ttl 由调用方按业务场景传（如找回密码 10 分钟），缓存层不过问具体值。
func TestVCodeCache_TTLExpiry(t *testing.T) {
	c, mr := newVCodeCacheForTest(t)

	require.NoError(t, c.Save("reset_password", "user@example.com", "123456", 2*time.Minute))

	mr.FastForward(2*time.Minute + time.Second)
	require.False(t, c.Verify("reset_password", "user@example.com", "123456"),
		"过期验证码应失效")
}

// TestVCodeCache_OverwriteSameKey 验证重发覆盖：
// 同 (purpose, identifier) 重发新码后，旧码作废、新码生效
// （"再发一次验证码"是找回密码页的常规操作）。
func TestVCodeCache_OverwriteSameKey(t *testing.T) {
	c, _ := newVCodeCacheForTest(t)

	require.NoError(t, c.Save("reset_password", "user@example.com", "111111", 5*time.Minute))
	require.NoError(t, c.Save("reset_password", "user@example.com", "222222", 5*time.Minute))

	require.False(t, c.Verify("reset_password", "user@example.com", "111111"),
		"重发后旧码应作废")
	require.True(t, c.Verify("reset_password", "user@example.com", "222222"),
		"重发后的新码应生效")
}
