// MfaChallengeCache 的 miniredis 测试。
//
// MFA 登录挑战是"密码对了一半、等第二因子"的悬空凭证，缓存层错一步就是
// 绕过第二因子或反复重放，测试盯四件事：
//   - verify-and-delete 语义：Peek 只读可重试，Atomic Take 只有一个赢家——
//     并发双花（同 opId 签两组 token）靠 GET+DEL 原子脚本裁决，第二个必 false；
//   - 失败上限：3 次验证失败作废挑战（封死逐码暴破），且作废后 Peek 拿不到；
//   - TTL：挑战 5 分钟过期，过期即 ErrMfaChallengeNotFound；
//   - 注册频控：同用户 30s 冷却，冷却期外可再次发起。
//
// Redis 异常路径按生产语义钉住：失败计数 fail-closed（宁拒勿放）、
// 频控 fail-open（不阻断正常注册）。
package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
)

// newMfaChallengeCacheForTest 起 miniredis 并构造缓存，返回供 FastForward 操控时钟。
func newMfaChallengeCacheForTest(t *testing.T) (*MfaChallengeCache, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err, "miniredis 启动失败")
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	return NewMfaChallengeCacheForTest(rdb), mr
}

func newLoginPayload(userID uint64) *authenticationV1.UserTokenPayload {
	return &authenticationV1.UserTokenPayload{
		UserId:   userID,
		TenantId: trans.Ptr(uint64(7)),
		Username: trans.Ptr("alice"),
	}
}

// TestMfaChallengeCache_LoginChallengeRoundTrip 验证登录挑战的读写回路：
// 写入返回 operation_id，Peek 还原 payload 与 clientType，且 Peek 不消耗
// （密码验证失败后的重试路径依赖只读语义）。
func TestMfaChallengeCache_LoginChallengeRoundTrip(t *testing.T) {
	c, mr := newMfaChallengeCacheForTest(t)
	ctx := context.Background()

	opId, err := c.SetLoginChallenge(ctx, newLoginPayload(42), authenticationV1.ClientType_admin)
	require.NoError(t, err)
	require.NotEmpty(t, opId, "operation_id 应由缓存生成且不可预测")
	require.Len(t, mr.Keys(), 1, "一次写入应恰好落一个键")

	got, err := c.PeekLoginChallenge(ctx, opId)
	require.NoError(t, err)
	require.Equal(t, uint64(42), got.Payload.GetUserId())
	require.Equal(t, uint64(7), got.Payload.GetTenantId())
	require.Equal(t, "alice", got.Payload.GetUsername())
	require.Equal(t, authenticationV1.ClientType_admin, got.ClientType,
		"clientType 应随信封原样还原")

	// Peek 不删除：重试路径还能再读
	_, err = c.PeekLoginChallenge(ctx, opId)
	require.NoError(t, err, "Peek 应只读不消耗")

	// 未知 opId：区分「没有这行」与「查库失败」
	_, err = c.PeekLoginChallenge(ctx, "no-such-op")
	require.ErrorIs(t, err, ErrMfaChallengeNotFound, "不存在的挑战应返回哨兵错误")
}

// TestMfaChallengeCache_LoginChallengeTTL 验证挑战 5 分钟过期：
// 过期后 Peek 报 ErrMfaChallengeNotFound（第二因子悬空凭证不能比登录页停留时间活得更久）。
func TestMfaChallengeCache_LoginChallengeTTL(t *testing.T) {
	c, mr := newMfaChallengeCacheForTest(t)
	ctx := context.Background()

	opId, err := c.SetLoginChallenge(ctx, newLoginPayload(1), authenticationV1.ClientType_admin)
	require.NoError(t, err)

	ttl := mr.TTL("mfa:login:" + opId)
	require.Equal(t, MfaChallengeTTL, ttl, "挑战键应按 MfaChallengeTTL 过期")

	mr.FastForward(MfaChallengeTTL + time.Second)
	_, err = c.PeekLoginChallenge(ctx, opId)
	require.ErrorIs(t, err, ErrMfaChallengeNotFound, "过期挑战应视同不存在")
}

// TestMfaChallengeCache_RecordLoginFailure 验证失败上限：
// 前 2 次失败 exceeded=false（用户输错可重试），第 3 次达上限并作废挑战
// （再 Peek 已不存在——TOTP 有效码空间 10^6，3 次封死逐码暴破）。
func TestMfaChallengeCache_RecordLoginFailure(t *testing.T) {
	c, _ := newMfaChallengeCacheForTest(t)
	ctx := context.Background()

	opId, err := c.SetLoginChallenge(ctx, newLoginPayload(1), authenticationV1.ClientType_admin)
	require.NoError(t, err)

	require.False(t, c.RecordLoginFailure(ctx, opId), "第 1 次失败不应达上限")
	require.False(t, c.RecordLoginFailure(ctx, opId), "第 2 次失败不应达上限")

	// 上限内挑战仍可读（重试路径）
	_, err = c.PeekLoginChallenge(ctx, opId)
	require.NoError(t, err)

	require.True(t, c.RecordLoginFailure(ctx, opId), "第 3 次失败应达上限")
	_, err = c.PeekLoginChallenge(ctx, opId)
	require.ErrorIs(t, err, ErrMfaChallengeNotFound, "达上限后挑战应被作废")
}

// TestMfaChallengeCache_TakeAtomicDoubleSpend 验证原子消耗的裁决语义：
// 同一 opId 只有第一次 Take 返回 true，第二次必 false——发 token 路径靠它防
// 并发双花（Peek+DEL 组合下两个并发请求都能通过验证并各自签发一组令牌）。
func TestMfaChallengeCache_TakeAtomicDoubleSpend(t *testing.T) {
	c, _ := newMfaChallengeCacheForTest(t)
	ctx := context.Background()

	opId, err := c.SetLoginChallenge(ctx, newLoginPayload(1), authenticationV1.ClientType_admin)
	require.NoError(t, err)
	c.RecordLoginFailure(ctx, opId) // 留一个失败计数键，验证 Take 顺带清理

	require.True(t, c.TakeLoginChallengeAtomic(ctx, opId), "第一次原子消耗应抢到裁决权")
	require.False(t, c.TakeLoginChallengeAtomic(ctx, opId), "第二次消耗必失败（值已删）")

	_, err = c.PeekLoginChallenge(ctx, opId)
	require.ErrorIs(t, err, ErrMfaChallengeNotFound, "消耗后挑战应不存在")
}

// TestMfaChallengeCache_ConsumeLoginChallenge 验证普通消耗：
// 挑战键与失败计数键一起删（上限/因子缺失等无需裁决的场景走这条）。
func TestMfaChallengeCache_ConsumeLoginChallenge(t *testing.T) {
	c, mr := newMfaChallengeCacheForTest(t)
	ctx := context.Background()

	opId, err := c.SetLoginChallenge(ctx, newLoginPayload(1), authenticationV1.ClientType_admin)
	require.NoError(t, err)
	c.RecordLoginFailure(ctx, opId)

	c.ConsumeLoginChallenge(ctx, opId)
	require.NotContains(t, mr.Keys(), "mfa:login:"+opId, "挑战键应被删除")
	require.NotContains(t, mr.Keys(), "mfa:loginfail:"+opId, "失败计数键应一并删除")
}

// TestMfaChallengeCache_EnrollChallenge 验证注册上下文回路：
// 写入→只读还原（首码输错可重试）→落库成功后显式删除；TTL 同为 5 分钟。
func TestMfaChallengeCache_EnrollChallenge(t *testing.T) {
	c, mr := newMfaChallengeCacheForTest(t)
	ctx := context.Background()

	opId, err := c.SetEnrollChallenge(ctx, "JBSWY3DPEHPK3PXP", 7, 42)
	require.NoError(t, err)
	require.NotEmpty(t, opId)

	got, err := c.PeekEnrollChallenge(ctx, opId)
	require.NoError(t, err)
	require.Equal(t, "JBSWY3DPEHPK3PXP", got.Secret)
	require.Equal(t, uint64(7), got.TenantID)
	require.Equal(t, uint64(42), got.UserID)

	_, err = c.PeekEnrollChallenge(ctx, opId)
	require.NoError(t, err, "注册上下文 Peek 只读（Confirm 失败可重试）")

	_, err = c.PeekEnrollChallenge(ctx, "no-such-op")
	require.ErrorIs(t, err, ErrMfaChallengeNotFound)

	c.DeleteEnrollChallenge(ctx, opId)
	require.NotContains(t, mr.Keys(), "mfa:enroll:"+opId, "删除后上下文应消失")
}

// TestMfaChallengeCache_EnrollCooldown 验证注册频控：
// 同 (tenant,user) 冷却期内第二次 SetNX 拿不到；冷却期外可再次发起；
// 键带 TTL 不会永久占坑（防循环 Start 塞满 Redis 的低成本 DoS）。
func TestMfaChallengeCache_EnrollCooldown(t *testing.T) {
	c, mr := newMfaChallengeCacheForTest(t)
	ctx := context.Background()

	require.True(t, c.TryAcquireEnrollCooldown(ctx, 7, 42), "首次发起应拿到频控")
	require.False(t, c.TryAcquireEnrollCooldown(ctx, 7, 42), "冷却期内第二次应被拒")
	require.True(t, c.TryAcquireEnrollCooldown(ctx, 7, 43), "同租户不同用户互不影响（各持各的冷却键）")

	ttl := mr.TTL("mfa:enrollcd:7:42")
	require.Equal(t, mfaEnrollCooldown, ttl,
		"频控键应按冷却时长过期")

	mr.FastForward(mfaEnrollCooldown + time.Second)
	require.True(t, c.TryAcquireEnrollCooldown(ctx, 7, 42), "冷却期外应可再次发起")
}

// TestMfaChallengeCache_FailClosedOnFailureCounter 钉住两条异常路径的生产语义：
//   - 失败计数遇 Redis 异常按「已达上限」处理（fail-closed：计数失效时若放行，
//     逐码暴破的封锁就随 Redis 抖动一起失效）；
//   - 注册频控遇 Redis 异常放行（fail-open：频控是防御性优化，不阻断正常注册）。
func TestMfaChallengeCache_FailClosedOnFailureCounter(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	addr := mr.Addr()
	mr.Close() // 直接关掉，制造连接失败

	rdb := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { _ = rdb.Close() })
	c := NewMfaChallengeCacheForTest(rdb)
	ctx := context.Background()

	require.True(t, c.RecordLoginFailure(ctx, "some-op"),
		"计数失败应按已达上限处理（fail-closed）")
	require.False(t, c.TakeLoginChallengeAtomic(ctx, "some-op"),
		"原子消耗遇 Redis 异常应返回未抢到（不放行发 token）")
	require.True(t, c.TryAcquireEnrollCooldown(ctx, 7, 42),
		"频控遇 Redis 异常应放行（fail-open）")
}

// TestMfaChallengeCache_UnknownOperationErrors 编译期顺带钉住哨兵错误可 errors.Is 判别。
func TestMfaChallengeCache_UnknownOperationErrors(t *testing.T) {
	c, _ := newMfaChallengeCacheForTest(t)
	ctx := context.Background()

	_, err := c.PeekLoginChallenge(ctx, "")
	require.ErrorIs(t, err, ErrMfaChallengeNotFound)
	require.False(t, errors.Is(err, context.DeadlineExceeded), "哨兵错误不应与其他错误族混淆")
}
