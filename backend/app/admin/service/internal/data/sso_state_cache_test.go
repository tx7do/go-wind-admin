// SsoStateCache 的 miniredis 测试。
//
// OIDC state 是防 CSRF 与回调重放的一次性票据，测试盯三件事：
//   - take-and-delete：同一个 state 只能换一次令牌（重放回调必失败）；
//   - 未知/过期 state 返回 ErrSsoStateNotFound（哨兵错误，service 层据此
//     给用户说人话而不是"内部错误"）；
//   - 多个 state 互不干扰（多实例部署下不同节点各自发起的登录并存）。
package data

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// newSsoStateCacheForTest 起 miniredis 并构造缓存，返回供 FastForward 操控时钟。
func newSsoStateCacheForTest(t *testing.T) (*SsoStateCache, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err, "miniredis 启动失败")
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	return NewSsoStateCacheForTest(rdb), mr
}

// TestSsoStateCache_TakeOnce 验证 state 的单次有效：Take 成功即删除，
// 重放同一 state 报 ErrSsoStateNotFound（防回调重放的核心语义）。
func TestSsoStateCache_TakeOnce(t *testing.T) {
	c, _ := newSsoStateCacheForTest(t)
	ctx := context.Background()

	require.NoError(t, c.Put(ctx, "state-abc"), "Put 应成功")

	ok, err := c.Take(ctx, "state-abc")
	require.NoError(t, err)
	require.True(t, ok, "存在的 state 应验证通过")

	ok, err = c.Take(ctx, "state-abc")
	require.ErrorIs(t, err, ErrSsoStateNotFound, "重放已用 state 应失败")
	require.False(t, ok)
}

// TestSsoStateCache_UnknownState 验证未知 state：
// 不存在的 state 不报"查询失败"，返回哨兵错误让回调链路给出明确拒绝。
func TestSsoStateCache_UnknownState(t *testing.T) {
	c, _ := newSsoStateCacheForTest(t)
	ctx := context.Background()

	ok, err := c.Take(ctx, "never-put")
	require.ErrorIs(t, err, ErrSsoStateNotFound)
	require.False(t, ok)
}

// TestSsoStateCache_TTL 验证 state 过期语义：
// 键按 SsoStateTTL（10 分钟）过期，过期后 Take 报哨兵错误
// （用户在 IdP 停留太久回来，state 已失效，需重新发起登录）。
func TestSsoStateCache_TTL(t *testing.T) {
	c, mr := newSsoStateCacheForTest(t)
	ctx := context.Background()

	require.NoError(t, c.Put(ctx, "state-abc"))

	ttl := mr.TTL("sso:state:state-abc")
	require.Equal(t, SsoStateTTL, ttl, "state 键应按 SsoStateTTL 过期")

	mr.FastForward(SsoStateTTL + time.Second)
	ok, err := c.Take(ctx, "state-abc")
	require.ErrorIs(t, err, ErrSsoStateNotFound, "过期 state 应视同不存在")
	require.False(t, ok)
}

// TestSsoStateCache_IndependentStates 验证多 state 并存：
// 两个登录流程各自的 state 互不干扰，消耗其一不影响另一个。
func TestSsoStateCache_IndependentStates(t *testing.T) {
	c, _ := newSsoStateCacheForTest(t)
	ctx := context.Background()

	require.NoError(t, c.Put(ctx, "state-1"))
	require.NoError(t, c.Put(ctx, "state-2"))

	ok, err := c.Take(ctx, "state-1")
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = c.Take(ctx, "state-2")
	require.NoError(t, err, "消耗 state-1 不应影响 state-2")
	require.True(t, ok)
}
