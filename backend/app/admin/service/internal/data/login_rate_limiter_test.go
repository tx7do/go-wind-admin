// LoginRateLimiter 的 miniredis 测试。
//
// 登录限流是防暴破的最后一道闸，测试盯三件"错一次就放进攻破者"的事：
//   - 阈值语义：恰好第 5 次失败触发锁定（不是第 6 次），锁定后计数不再自增——
//     否则攻击者可以无限拉高计数把锁定窗口越滚越长，或反过来刷掉别人；
//   - 双维度判定：IP 或用户名任一达阈值即锁（换 IP 也逃不掉用户名维度），
//     空维度（内网无 X-Forwarded-For）只按剩余维度计数；
//   - 窗口过期自然解锁：TTL 过后计数清零重来，锁定不是终身制。
//
// 附带钉住 fail-open 降级：Redis 不可用时限流放行而不是把登录一起拖死。
package data

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// newLoginRateLimiterForTest 起一个 miniredis 并构造限流器，返回供 FastForward 操控时钟。
func newLoginRateLimiterForTest(t *testing.T) (*LoginRateLimiter, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err, "miniredis 启动失败")
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	return NewLoginRateLimiterForTest(rdb), mr
}

// TestLoginRateLimiter_CheckAndIncr_LocksAtThreshold 验证阈值语义：
// 允许 loginFailThreshold 次失败（每次自增返回未锁），下一次 CheckAndIncr 才报告锁定
// （Lua 先判 current>=threshold 再自增，与 IsLocked 前置拦截的 5>=5 判定一致）；
// 锁定后计数不再自增（攻击者刷不出更长的窗口）；retryAfter 固定为锁定窗口。
func TestLoginRateLimiter_CheckAndIncr_LocksAtThreshold(t *testing.T) {
	l, _ := newLoginRateLimiterForTest(t)
	ctx := context.Background()

	for i := 1; i <= loginFailThreshold; i++ {
		locked, attempts, retryAfter, err := l.CheckAndIncr(ctx, "10.0.0.1", "alice")
		require.NoError(t, err, "第 %d 次失败的自增不应报错", i)
		require.False(t, locked, "第 %d 次失败（阈值 %d 内）不应触发锁定", i, loginFailThreshold)
		require.Equal(t, i, attempts, "未锁定时 attempts 应等于失败次数")
		require.Zero(t, retryAfter, "未锁定时 retryAfter 应为 0")
	}

	// 第 6 次失败：计数已到阈值，报告锁定且不再自增
	locked, attempts, retryAfter, err := l.CheckAndIncr(ctx, "10.0.0.1", "alice")
	require.NoError(t, err)
	require.True(t, locked, "计数达阈值后的下一次失败应被判定锁定")
	require.Equal(t, loginFailThreshold, attempts, "锁定判定不应继续自增")
	require.Equal(t, loginLockoutDuration, retryAfter, "retryAfter 应为锁定窗口")

	// 锁定后继续失败：Lua 脚本在已锁分支直接返回当前计数，不再 INCR
	locked, attempts, _, err = l.CheckAndIncr(ctx, "10.0.0.1", "alice")
	require.NoError(t, err)
	require.True(t, locked)
	require.Equal(t, loginFailThreshold, attempts, "锁定后计数不应继续自增")
}

// TestLoginRateLimiter_LocksPerDimension 验证双维度判定：
// 同一用户名换 IP 仍被锁定（用户名维度在）；同一 IP 换用户名也仍被锁定（IP 维度在）。
// 任一维度达标即锁，攻击者换壳无效。
func TestLoginRateLimiter_LocksPerDimension(t *testing.T) {
	l, _ := newLoginRateLimiterForTest(t)
	ctx := context.Background()

	for i := 0; i < loginFailThreshold; i++ {
		_, _, _, err := l.CheckAndIncr(ctx, "10.0.0.1", "alice")
		require.NoError(t, err)
	}

	// 换 IP、同一用户名：用户名维度仍达阈值
	locked, _, _, err := l.CheckAndIncr(ctx, "10.0.0.2", "alice")
	require.NoError(t, err)
	require.True(t, locked, "换 IP 后同用户名应仍被锁定（用户名维度）")

	// 换用户名、同一 IP：IP 维度仍达阈值
	locked, _, _, err = l.CheckAndIncr(ctx, "10.0.0.1", "bob")
	require.NoError(t, err)
	require.True(t, locked, "换用户名后同 IP 应仍被锁定（IP 维度）")

	// 全新的 IP + 用户名：不受他人连累
	locked, attempts, _, err := l.CheckAndIncr(ctx, "10.0.0.9", "carol")
	require.NoError(t, err)
	require.False(t, locked, "全新维度不应被别人的失败连累")
	require.Equal(t, 1, attempts)
}

// TestLoginRateLimiter_IsLockedAndReset 验证前置查询不自增计数，
// Reset（登录成功）后两个维度都清零。
func TestLoginRateLimiter_IsLockedAndReset(t *testing.T) {
	l, _ := newLoginRateLimiterForTest(t)
	ctx := context.Background()

	locked, err := l.IsLocked(ctx, "10.0.0.1", "alice")
	require.NoError(t, err)
	require.False(t, locked, "空表不应锁定")

	for i := 0; i < loginFailThreshold; i++ {
		_, _, _, err := l.CheckAndIncr(ctx, "10.0.0.1", "alice")
		require.NoError(t, err)
	}

	locked, err = l.IsLocked(ctx, "10.0.0.1", "alice")
	require.NoError(t, err)
	require.True(t, locked)

	l.Reset(ctx, "10.0.0.1", "alice")
	locked, err = l.IsLocked(ctx, "10.0.0.1", "alice")
	require.NoError(t, err)
	require.False(t, locked, "Reset 后应解锁")

	locked, attempts, _, err := l.CheckAndIncr(ctx, "10.0.0.1", "alice")
	require.NoError(t, err)
	require.False(t, locked)
	require.Equal(t, 1, attempts, "Reset 后计数应从 1 重新开始")
}

// TestLoginRateLimiter_WindowExpiry 验证失败计数的滑动窗口：
// 首次失败才设 TTL，窗口过后计数自然过期、锁定解除、重新从 1 计。
func TestLoginRateLimiter_WindowExpiry(t *testing.T) {
	l, mr := newLoginRateLimiterForTest(t)
	ctx := context.Background()

	_, _, _, err := l.CheckAndIncr(ctx, "10.0.0.1", "alice")
	require.NoError(t, err)

	ttl := mr.TTL("gowind:login:fail:ip:10.0.0.1")
	require.Equal(t, loginFailKeyTTL, ttl,
		"首次失败应按锁定窗口设置 TTL")

	// 时间推到窗口刚过 1 秒：计数应已过期，第 2 次失败从 1 计
	mr.FastForward(loginFailKeyTTL + time.Second)
	locked, attempts, _, err := l.CheckAndIncr(ctx, "10.0.0.1", "alice")
	require.NoError(t, err)
	require.False(t, locked, "窗口过期后不应仍处于锁定")
	require.Equal(t, 1, attempts, "过期后计数应重新从 1 开始")
}

// TestLoginRateLimiter_EmptyDimensions 验证空维度的边界：
// 只传用户名（内网无 X-Forwarded-For）只计用户名键；两个都空是无意义调用，
// 不报错也不误伤。
func TestLoginRateLimiter_EmptyDimensions(t *testing.T) {
	l, mr := newLoginRateLimiterForTest(t)
	ctx := context.Background()

	for i := 0; i < loginFailThreshold; i++ {
		locked, attempts, _, err := l.CheckAndIncr(ctx, "", "alice")
		require.NoError(t, err)
		require.False(t, locked)
		require.Equal(t, i+1, attempts)
	}
	require.NotContains(t, mr.Keys(), "gowind:login:fail:ip:", "空 IP 不该写出尾缀为空的 IP 键")

	// 双空维度：无键可计，静默返回未锁定
	locked, attempts, _, err := l.CheckAndIncr(ctx, "", "")
	require.NoError(t, err)
	require.False(t, locked)
	require.Equal(t, 0, attempts)
}

// TestLoginRateLimiter_FailOpenWhenRedisUnavailable 验证降级语义：
// Redis 不可用（rdb 为 nil / 连接失败）时限流放行（fail-open），登录主链路不被拖死。
// 限流是防御性增强，不该有"Redis 挂了全员无法登录"的连坐。
func TestLoginRateLimiter_FailOpenWhenRedisUnavailable(t *testing.T) {
	ctx := context.Background()

	// rdb 为 nil 的白盒实例
	nilRdb := &LoginRateLimiter{}
	locked, attempts, retryAfter, err := nilRdb.CheckAndIncr(ctx, "10.0.0.1", "alice")
	require.NoError(t, err, "Redis 不可用应降级放行而非报错")
	require.False(t, locked)
	require.Equal(t, 0, attempts)
	require.Zero(t, retryAfter)

	_, err = nilRdb.IsLocked(ctx, "10.0.0.1", "alice")
	require.NoError(t, err)
	require.False(t, locked)

	require.NotPanics(t, func() { nilRdb.Reset(ctx, "10.0.0.1", "alice") },
		"降级路径的 Reset 不应 panic")

	// 连接指向已关闭的 miniredis：同样 fail-open
	mr, err := miniredis.Run()
	require.NoError(t, err)
	addr := mr.Addr()
	mr.Close()

	dead := NewLoginRateLimiterForTest(redis.NewClient(&redis.Options{Addr: addr}))
	locked, _, _, err = dead.CheckAndIncr(ctx, "10.0.0.1", "alice")
	require.NoError(t, err, "连接失败也应降级放行（脚本错误被逐键吞掉、循环后返回未锁）")
	require.False(t, locked)
}
