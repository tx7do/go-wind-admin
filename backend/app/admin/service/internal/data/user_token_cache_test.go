package data

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

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
	bctx := bootstrap.NewContextWithParam(ctx, &conf.AppInfo{}, cfg, log.DefaultLogger)

	// 使用当前构造函数
	repo := NewUserTokenCache(bctx, rdb)
	assert.NotNil(t, repo)

	clientType := authenticationV1.ClientType_app
	var userId uint32 = 123

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
