// TokenChecker 的集成测试（miniredis + 真 Authenticator）。
//
// TokenChecker 是 auth 中间件与 Authenticator 之间的适配层，逻辑只有映射：
// 认证错误/无效 → 拒，有效 → 连 payload 一起放行。但它映射的两个入参有安全语义，
// 必须钉住：
//   - IsValidAccessToken 的 skipRedis **透传**：skipRedis=false 时必须走 Redis
//     存活校验（吊销/过期令牌在缓存校验一步被拒），不能把"跳过校验"变成默认；
//   - IsBlockedAccessToken 恒走 SkipRedis=true（它要的是 JWT 层面的拉黑判定，
//     不该被 Redis 抖动连坐），且异常路径 fail-closed（错误视为已拉黑）。
package data

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	conf "github.com/tx7do/kratos-bootstrap/api/gen/go/conf/v1"
	bBootstrap "github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/tx7do/go-utils/trans"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
)

// newTokenCheckerForTest 组装 miniredis + 真 Authenticator 的 TokenChecker，
// 返回可供吊销/拉黑操作的 Authenticator 与一个已签发的有效令牌。
func newTokenCheckerForTest(t *testing.T) (checker *TokenChecker, authn *Authenticator, validToken string, userID uint32) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err, "miniredis 启动失败")
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	jwtCfg := &conf.Authentication_Jwt{
		Method: "HS256",
		Key: "test-signing-key",
	}
	authn, err = NewAuthenticatorForTest(jwtCfg, NewUserTokenCacheForTest(rdb))
	require.NoError(t, err, "Authenticator 测试装配失败")

	bctx := bBootstrap.NewContextWithParam(
		context.Background(),
		&conf.AppInfo{},
		&conf.Bootstrap{Data: &conf.Data{Redis: &conf.Data_Redis{Addr: mr.Addr()}}},
		bLogger.NopLogger(),
	)
	checker = NewTokenChecker(bctx, authn, authenticationV1.ClientType_admin).(*TokenChecker)

	// 签发一组真实令牌并写入缓存
	userID = 42
	access, _, err := authn.CreateUserToken(context.Background(), authenticationV1.ClientType_admin, &authenticationV1.UserTokenPayload{
		UserId:   userID,
		TenantId: trans.Ptr(uint32(7)),
		Username: trans.Ptr("alice"),
	})
	require.NoError(t, err, "签发测试令牌失败")
	validToken = access
	return
}

// TestTokenChecker_IsValidAccessToken_HappyPath 验证有效令牌放行并带出 payload；
// skipRedis=false 时走 Redis 存活校验（令牌在签发时已入缓存，应通过）。
func TestTokenChecker_IsValidAccessToken_HappyPath(t *testing.T) {
	checker, _, token, userID := newTokenCheckerForTest(t)
	ctx := context.Background()

	valid, payload := checker.IsValidAccessToken(ctx, token, false)
	require.True(t, valid, "已入缓存的有效令牌应通过校验")
	require.NotNil(t, payload)
	require.Equal(t, userID, payload.GetUserId(), "payload 应随校验带出")

	valid, payload = checker.IsValidAccessToken(ctx, token, true)
	require.True(t, valid, "跳过 Redis 的纯 JWT 校验也应通过")
	require.NotNil(t, payload)
}

// TestTokenChecker_IsValidAccessToken_RejectsInvalid 验证三类拒绝：
// 篡改签名、垃圾串、空串——一律 (false, nil)，不 panic 不放行。
func TestTokenChecker_IsValidAccessToken_RejectsInvalid(t *testing.T) {
	checker, _, _, _ := newTokenCheckerForTest(t)
	ctx := context.Background()

	cases := []struct {
		name  string
		token string
	}{
		{"篡改签名", "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.tampered-signature"},
		{"垃圾串", "not-a-jwt"},
		{"空串", ""},
	}
	for _, tc := range cases {
		valid, payload := checker.IsValidAccessToken(ctx, tc.token, true)
		require.False(t, valid, "形态[%s]应被拒绝", tc.name)
		require.Nil(t, payload, "拒绝时不应带出 payload")
	}
}

// TestTokenChecker_IsValidAccessToken_RevokedFromCache 验证 skipRedis=false
// 的存活校验：令牌从缓存删除（吊销）后，纯 JWT 校验仍通过、缓存校验被拒——
// 这就是 skipRedis 参数必须透传而不能写死 true 的原因。
func TestTokenChecker_IsValidAccessToken_RevokedFromCache(t *testing.T) {
	checker, authn, token, userID := newTokenCheckerForTest(t)
	ctx := context.Background()

	// 模拟吊销：把该用户的令牌从缓存清掉
	require.NoError(t, authn.RevokeUserToken(ctx, authenticationV1.ClientType_admin, userID),
	 "吊销令牌应成功")

	// skipRedis=true：JWT 层面仍有效
	valid, _ := checker.IsValidAccessToken(ctx, token, true)
	require.True(t, valid, "跳过 Redis 时 JWT 本身仍有效")

	// skipRedis=false：缓存里已不在，应拒绝
	valid, payload := checker.IsValidAccessToken(ctx, token, false)
	require.False(t, valid, "吊销后的令牌经缓存校验应被拒绝")
	require.Nil(t, payload)
}

// TestTokenChecker_IsBlockedAccessToken 验证拉黑判定的三态：
// 正常令牌未被拉黑 → false；BlockToken 后 → true（JWT 层面，无需 Redis 存活）；
// 垃圾串按已拉黑处理（fail-closed：宁可多拦也不放可疑令牌过黑名单检查）。
func TestTokenChecker_IsBlockedAccessToken(t *testing.T) {
	checker, authn, token, userID := newTokenCheckerForTest(t)
	ctx := context.Background()

	require.False(t, checker.IsBlockedAccessToken(ctx, token), "正常令牌不应被判拉黑")

	// 拉黑该令牌（BlockToken 按 token 定位 jti；UserId 必须是令牌属主，
// IsExistAccessToken 按 (clientType, userId, token) 三元组查缓存）
	require.NoError(t, authn.BlockToken(ctx, &authenticationV1.BlockTokenRequest{
		UserId:     userID,
		ClientType: authenticationV1.ClientType_admin,
		Target:     &authenticationV1.BlockTokenRequest_Token{Token: token},
	}), "拉黑令牌应成功")
	require.True(t, checker.IsBlockedAccessToken(ctx, token), "拉黑后的令牌应被判拉黑")

	require.True(t, checker.IsBlockedAccessToken(ctx, "garbage-token"),
		"无法认证的令牌应按已拉黑处理（fail-closed）")
}
