package data

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	conf "github.com/tx7do/kratos-bootstrap/api/gen/go/conf/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
)

func TestUserTokenCache_BasicOperations(t *testing.T) {
	// 启动内存 redis
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	// 构造 redis client
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	ctx := context.Background()

	// 最小 bootstrap cfg，用于构造 logger/context
	cfg := &conf.Bootstrap{
		Server: &conf.Server{
			Rest: &conf.Server_REST{
				Middleware: &conf.Middleware{
					Auth: &conf.Middleware_Auth{
						Method: "HS256",
						Key:    "some_api_key",
					},
				},
			},
		},
		Data: &conf.Data{
			Redis: &conf.Data_Redis{
				Addr: mr.Addr(),
			},
		},
	}
	bctx := bootstrap.NewContextWithParam(ctx, &conf.AppInfo{}, cfg, bLogger.NopLogger())

	// 使用当前构造函数
	repo := NewUserTokenCache(bctx, rdb)
	assert.NotNil(t, repo)

	clientType := authenticationV1.ClientType_app
	var userId int64 = 123

	// 用于接收验证结果，避免重复短声明导致重定义
	var valid bool

	// access token 流程：使用 jti 作为字段名
	jtiAccess := "jti-1"
	accessValue := "access_token_1"
	err = repo.AddAccessToken(ctx, clientType, userId, jtiAccess, accessValue, 0)
	assert.NoError(t, err)

	// 注意：IsValidAccessToken 目前检查的是字段是否存在，因此传入 jti
	valid, err = repo.IsValidAccessToken(ctx, clientType, userId, jtiAccess, accessValue)
	assert.NoError(t, err)
	assert.True(t, valid)

	tokens := repo.GetAccessTokens(ctx, clientType, userId)
	assert.Contains(t, tokens, accessValue)

	// 删除（同样以 jti 作为字段）
	err = repo.RevokeAccessToken(ctx, clientType, userId, jtiAccess)
	assert.NoError(t, err)

	valid, err = repo.IsValidAccessToken(ctx, clientType, userId, jtiAccess, accessValue)
	assert.NoError(t, err)
	assert.False(t, valid)

	// refresh token 流程：使用 jti 作为字段名
	jtiRefresh := "rt-1"
	refreshValue := "refresh_token_1"
	err = repo.AddRefreshToken(ctx, clientType, userId, jtiRefresh, refreshValue, 0)
	assert.NoError(t, err)
	// 正确处理 IsValidRefreshToken 的 (bool, error) 返回值
	valid, err = repo.IsValidRefreshToken(ctx, clientType, userId, jtiRefresh, refreshValue)
	assert.NoError(t, err)
	assert.True(t, valid)
	rfs := repo.GetRefreshTokens(ctx, clientType, userId)
	assert.Contains(t, rfs, refreshValue)

	err = repo.RevokeRefreshToken(ctx, clientType, userId, jtiRefresh)
	assert.NoError(t, err)
	valid, err = repo.IsValidRefreshToken(ctx, clientType, userId, jtiRefresh, refreshValue)
	assert.NoError(t, err)
	assert.False(t, valid)

	// 黑名单（blocked token）流程
	blockJti := "jti-block"
	err = repo.AddBlockedAccessToken(ctx, blockJti, "test-reason", 0)
	assert.NoError(t, err)
	assert.True(t, repo.IsBlockedAccessToken(ctx, blockJti))

	err = repo.RevokeBlockedAccessToken(ctx, blockJti)
	assert.NoError(t, err)
	assert.False(t, repo.IsBlockedAccessToken(ctx, blockJti))

	// RevokeToken（清空用户所有 token），使用 AddTokenPair 添加一对
	pairJti := "pair-1"
	err = repo.AddTokenPair(ctx, clientType, userId, pairJti, "access2", "refresh2", 0, 0)
	assert.NoError(t, err)

	// 确认已加入
	valid, err = repo.IsValidAccessToken(ctx, clientType, userId, pairJti, "access2")
	assert.NoError(t, err)
	assert.True(t, valid)

	valid, err = repo.IsValidRefreshToken(ctx, clientType, userId, pairJti, "refresh2")
	assert.NoError(t, err)
	assert.True(t, valid)

	// 清空所有 token
	err = repo.RevokeToken(ctx, clientType, userId)
	assert.NoError(t, err)
	// 确认为空
	assert.Empty(t, repo.GetAccessTokens(ctx, clientType, userId))
	assert.Empty(t, repo.GetRefreshTokens(ctx, clientType, userId))

	// 额外：确认过期设置不会导致 panic（safety check）
	_ = repo.AddBlockedAccessToken(ctx, "tmp-jti", "r", 50*time.Millisecond)
}

func TestUserTokenCache_VerifyAndRevokeTokenPair(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()

	cfg := &conf.Bootstrap{
		Server: &conf.Server{
			Rest: &conf.Server_REST{
				Middleware: &conf.Middleware{
					Auth: &conf.Middleware_Auth{Method: "HS256", Key: "some_api_key"},
				},
			},
		},
		Data: &conf.Data{Redis: &conf.Data_Redis{Addr: mr.Addr()}},
	}
	bctx := bootstrap.NewContextWithParam(ctx, &conf.AppInfo{}, cfg, bLogger.NopLogger())
	repo := NewUserTokenCache(bctx, rdb)

	clientType := authenticationV1.ClientType_app
	var userId int64 = 456
	jti := "jti-atomic"
	atVal := "access_atomic"
	rtVal := "refresh_atomic"

	// 添加令牌对
	err = repo.AddTokenPair(ctx, clientType, userId, jti, atVal, rtVal, 0, 0)
	assert.NoError(t, err)

	// 第一次原子验证并轮换：应成功
	valid, err := repo.VerifyAndRevokeTokenPair(ctx, clientType, userId, jti, rtVal)
	assert.NoError(t, err)
	assert.True(t, valid, "第一次验证应成功")

	// 轮换后：旧 RT 已删除；旧 AT 保留至自然 TTL（多标签页共享 cookie 的
	// 轮换不再互杀在用令牌）；旧 RT 进入宽限集合
	valid, err = repo.IsValidAccessToken(ctx, clientType, userId, jti, atVal)
	assert.NoError(t, err)
	assert.True(t, valid, "旧 Access Token 应保留至自然过期")

	valid, err = repo.IsValidRefreshToken(ctx, clientType, userId, jti, rtVal)
	assert.NoError(t, err)
	assert.False(t, valid, "旧 Refresh Token 应已被删除")

	assert.True(t, repo.IsGraceRefreshToken(ctx, clientType, userId, rtVal), "被轮换的 RT 应处于宽限集合")
	assert.False(t, repo.IsGraceRefreshToken(ctx, clientType, userId, "never_seen"), "陌生 RT 不应在宽限集合")

	// 第二次原子验证：应失败（已被轮换），但宽限判定放行
	valid, err = repo.VerifyAndRevokeTokenPair(ctx, clientType, userId, jti, rtVal)
	assert.NoError(t, err)
	assert.False(t, valid, "已轮换的 token 不应再次通过验证")

	// 测试值不匹配的情况
	err = repo.AddTokenPair(ctx, clientType, userId, jti, atVal, rtVal, 0, 0)
	assert.NoError(t, err)

	valid, err = repo.VerifyAndRevokeTokenPair(ctx, clientType, userId, jti, "wrong_token")
	assert.NoError(t, err)
	assert.False(t, valid, "值不匹配时验证应失败")

	// 验证令牌未被删除（因为验证失败了）
	valid, err = repo.IsValidRefreshToken(ctx, clientType, userId, jti, rtVal)
	assert.NoError(t, err)
	assert.True(t, valid, "验证失败时不应删除令牌")
}

func TestUserTokenCache_RevokeTokenByJtiWalksLineage(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()

	cfg := &conf.Bootstrap{
		Server: &conf.Server{
			Rest: &conf.Server_REST{
				Middleware: &conf.Middleware{
					Auth: &conf.Middleware_Auth{Method: "HS256", Key: "some_api_key"},
				},
			},
		},
		Data: &conf.Data{Redis: &conf.Data_Redis{Addr: mr.Addr()}},
	}
	bctx := bootstrap.NewContextWithParam(ctx, &conf.AppInfo{}, cfg, bLogger.NopLogger())
	repo := NewUserTokenCache(bctx, rdb)

	clientType := authenticationV1.ClientType_app
	var userId int64 = 789

	// 三代世系：gen1 → gen2 → gen3（每代各自 AT；RT 已随轮换删除故不建）
	const (
		jtiGen1 = "gen-1"
		jtiGen2 = "gen-2"
		jtiGen3 = "gen-3"
	)
	assert.NoError(t, repo.AddTokenPair(ctx, clientType, userId, jtiGen1, "at-1", "rt-1", 0, 0))
	assert.NoError(t, repo.AddTokenPair(ctx, clientType, userId, jtiGen2, "at-2", "rt-2", 0, 0))
	assert.NoError(t, repo.AddTokenPair(ctx, clientType, userId, jtiGen3, "at-3", "rt-3", 0, 0))
	assert.NoError(t, repo.AddTokenLineage(ctx, clientType, userId, jtiGen2, jtiGen1, 0))
	assert.NoError(t, repo.AddTokenLineage(ctx, clientType, userId, jtiGen3, jtiGen2, 0))

	// 踢最新一代：整条世系的 AT 都应被删除
	assert.NoError(t, repo.RevokeTokenByJti(ctx, clientType, userId, jtiGen3))

	for jti, at := range map[string]string{jtiGen1: "at-1", jtiGen2: "at-2", jtiGen3: "at-3"} {
		valid, err := repo.IsValidAccessToken(ctx, clientType, userId, jti, at)
		assert.NoError(t, err)
		assert.False(t, valid, "踢下线应沿代际链删除 %s 的 AT", jti)
	}

	// 无关会话不受影响
	assert.NoError(t, repo.AddTokenPair(ctx, clientType, userId, "other", "at-other", "rt-other", 0, 0))
	valid, err := repo.IsValidAccessToken(ctx, clientType, userId, "other", "at-other")
	assert.NoError(t, err)
	assert.True(t, valid, "踢下线不应波及无关会话")
}
