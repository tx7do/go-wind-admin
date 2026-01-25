package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
)

const (
	// AccessTokenKeyFormat 访问令牌键格式 at:{ct}:{uid}
	AccessTokenKeyFormat = "at:%d:%d"
	// RefreshTokenKeyFormat 刷新令牌键格式 rt:{ct}:{uid}

	RefreshTokenKeyFormat = "rt:%d:%d"

	// BlacklistKeyFormat 访问令牌黑名单键格式 bl:{jti}
	BlacklistKeyFormat = "bl:%s"
)

// UserTokenCache 用户令牌缓存
type UserTokenCache struct {
	log *log.Helper
	rdb *redis.Client
}

func NewUserTokenCache(ctx *bootstrap.Context, rdb *redis.Client) *UserTokenCache {
	utc := &UserTokenCache{
		rdb: rdb,
		log: ctx.NewLoggerHelper("user-token/cache"),
	}
	return utc
}

// AddTokenPair 添加令牌对
func (r *UserTokenCache) AddTokenPair(
	ctx context.Context,
	clientType authenticationV1.ClientType,
	userId uint32,
	jti string,
	accessToken string,
	refreshToken string,
	accessTokenExpires time.Duration,
	refreshTokenExpires time.Duration,
) error {
	var err error
	pipe := r.rdb.TxPipeline()

	atKey := r.makeAccessTokenKey(clientType, userId)
	pipe.HSet(ctx, atKey, jti, accessToken)
	if accessTokenExpires > 0 {
		// WARN: HExpire有版本要求，请确保使用的redis版本支持该命令。
		pipe.HExpire(ctx, atKey, accessTokenExpires, jti)
	}

	rtKey := r.makeRefreshTokenKey(clientType, userId)
	pipe.HSet(ctx, rtKey, jti, refreshToken)
	if refreshTokenExpires > 0 {
		// WARN: HExpire有版本要求，请确保使用的redis版本支持该命令。
		pipe.HExpire(ctx, rtKey, refreshTokenExpires, jti)
	}

	_, err = pipe.Exec(ctx)

	return err
}

// AddAccessToken 添加访问令牌
func (r *UserTokenCache) AddAccessToken(
	ctx context.Context,
	clientType authenticationV1.ClientType,
	userId uint32,
	jti string,
	accessToken string,
	expires time.Duration,
) error {
	key := r.makeAccessTokenKey(clientType, userId)
	return r.hset(ctx, key, jti, accessToken, expires)
}

// AddRefreshToken 添加刷新令牌
func (r *UserTokenCache) AddRefreshToken(
	ctx context.Context,
	clientType authenticationV1.ClientType,
	userId uint32,
	jti string,
	refreshToken string,
	expires time.Duration,
) error {
	key := r.makeRefreshTokenKey(clientType, userId)
	return r.hset(ctx, key, jti, refreshToken, expires)
}

// AddBlockedAccessToken 添加被阻止的访问令牌
func (r *UserTokenCache) AddBlockedAccessToken(ctx context.Context, jti string, reason string, expires time.Duration) error {
	key := r.makeBlacklistKey(jti)
	return r.set(ctx, key, reason, expires)
}

// GetAccessTokens 获取访问令牌
func (r *UserTokenCache) GetAccessTokens(ctx context.Context, clientType authenticationV1.ClientType, userId uint32) []string {
	key := r.makeAccessTokenKey(clientType, userId)
	return r.hgetValues(ctx, key)
}

// GetRefreshTokens 获取刷新令牌
func (r *UserTokenCache) GetRefreshTokens(ctx context.Context, clientType authenticationV1.ClientType, userId uint32) []string {
	key := r.makeRefreshTokenKey(clientType, userId)
	return r.hgetValues(ctx, key)
}

// RevokeToken 移除所有令牌
func (r *UserTokenCache) RevokeToken(ctx context.Context, clientType authenticationV1.ClientType, userId uint32) error {
	var err error
	if err = r.RevokeUserAllAccessToken(ctx, clientType, userId); err != nil {
		r.log.Errorf("remove user access token failed: [%v]", err)
	}

	if err = r.RevokeUserAllRefreshToken(ctx, clientType, userId); err != nil {
		r.log.Errorf("remove user refresh token failed: [%v]", err)
	}

	return err
}

func (r *UserTokenCache) RevokeTokenByJti(ctx context.Context, clientType authenticationV1.ClientType, userId uint32, jti string) error {
	var err error
	if err = r.RevokeAccessToken(ctx, clientType, userId, jti); err != nil {
		r.log.Errorf("remove user access token failed: [%v]", err)
	}

	if err = r.RevokeRefreshToken(ctx, clientType, userId, jti); err != nil {
		r.log.Errorf("remove user refresh token failed: [%v]", err)
	}

	return err
}

// RevokeAccessToken 移除访问令牌
func (r *UserTokenCache) RevokeAccessToken(
	ctx context.Context,
	clientType authenticationV1.ClientType,
	userId uint32,
	jti string,
) error {
	key := r.makeAccessTokenKey(clientType, userId)
	return r.hdel(ctx, key, jti)
}

// RevokeRefreshToken 移除刷新令牌
func (r *UserTokenCache) RevokeRefreshToken(
	ctx context.Context,
	clientType authenticationV1.ClientType,
	userId uint32,
	jti string,
) error {
	key := r.makeRefreshTokenKey(clientType, userId)
	return r.hdel(ctx, key, jti)
}

// RevokeBlockedAccessToken 撤销被阻止的访问令牌
func (r *UserTokenCache) RevokeBlockedAccessToken(ctx context.Context, jti string) error {
	key := r.makeBlacklistKey(jti)
	return r.del(ctx, key)
}

// IsValidAccessToken 访问令牌是否有效
func (r *UserTokenCache) IsValidAccessToken(
	ctx context.Context,
	clientType authenticationV1.ClientType,
	userId uint32,
	jti string,
	uploadedToken string,
) (bool, error) {
	key := r.makeAccessTokenKey(clientType, userId)

	storedToken, err := r.rdb.HGet(ctx, key, jti).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil // 令牌不存在或已过期
	}

	if storedToken != uploadedToken {
		return false, nil
	}

	return true, nil
}

func (r *UserTokenCache) IsExistAccessTokenByJti(
	ctx context.Context,
	clientType authenticationV1.ClientType,
	userId uint32,
	jti string,
) (bool, error) {
	key := r.makeAccessTokenKey(clientType, userId)

	_, err := r.rdb.HGet(ctx, key, jti).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil // 令牌不存在或已过期
	}

	return true, nil
}

// IsExistAccessToken 访问令牌是否存在
func (r *UserTokenCache) IsExistAccessToken(
	ctx context.Context,
	clientType authenticationV1.ClientType,
	userId uint32,
	uploadedToken string,
) (exist bool, jti string, err error) {
	key := r.makeAccessTokenKey(clientType, userId)

	all, err := r.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, "", nil
		}
		r.log.Errorf("hgetall key[%s] failed: %v", key, err)
		return false, "", err
	}

	for k, v := range all {
		if v == uploadedToken {
			return true, k, nil
		}
	}

	return false, "", nil
}

// IsExistRefreshToken 刷新令牌是否存在
func (r *UserTokenCache) IsExistRefreshToken(
	ctx context.Context,
	clientType authenticationV1.ClientType,
	userId uint32,
	uploadedToken string,
) (exist bool, jti string, err error) {
	key := r.makeRefreshTokenKey(clientType, userId)

	all, err := r.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, "", nil
		}
		r.log.Errorf("hgetall key[%s] failed: %v", key, err)
		return false, "", err
	}

	for k, v := range all {
		if v == uploadedToken {
			return true, k, nil
		}
	}

	return false, "", nil
}

// IsValidRefreshToken 刷新令牌是否有效
func (r *UserTokenCache) IsValidRefreshToken(
	ctx context.Context,
	clientType authenticationV1.ClientType,
	userId uint32,
	jti string,
	uploadedToken string,
) (bool, error) {
	key := r.makeRefreshTokenKey(clientType, userId)

	storedToken, err := r.rdb.HGet(ctx, key, jti).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil // 令牌不存在或已过期
	}

	if storedToken != uploadedToken {
		return false, nil
	}

	return true, nil
}

// IsBlockedAccessToken 访问令牌是否被阻止
func (r *UserTokenCache) IsBlockedAccessToken(ctx context.Context, jti string) bool {
	key := r.makeBlacklistKey(jti)
	return r.exists(ctx, key)
}

// RevokeUserAllAccessToken 删除访问令牌
func (r *UserTokenCache) RevokeUserAllAccessToken(
	ctx context.Context,
	clientType authenticationV1.ClientType,
	userId uint32,
) error {
	key := r.makeAccessTokenKey(clientType, userId)
	return r.del(ctx, key)
}

// RevokeUserAllRefreshToken 删除刷新令牌
func (r *UserTokenCache) RevokeUserAllRefreshToken(
	ctx context.Context,
	clientType authenticationV1.ClientType,
	userId uint32,
) error {
	key := r.makeRefreshTokenKey(clientType, userId)
	return r.del(ctx, key)
}

// makeAccessTokenKey 生成访问令牌键
func (r *UserTokenCache) makeAccessTokenKey(clientType authenticationV1.ClientType, userId uint32) string {
	return fmt.Sprintf(AccessTokenKeyFormat, clientType.Number(), userId)
}

// makeRefreshTokenKey 生成刷新令牌键
func (r *UserTokenCache) makeRefreshTokenKey(clientType authenticationV1.ClientType, userId uint32) string {
	return fmt.Sprintf(RefreshTokenKeyFormat, clientType.Number(), userId)
}

// makeBlacklistKey 生成黑名单键
func (r *UserTokenCache) makeBlacklistKey(jti string) string {
	return fmt.Sprintf(BlacklistKeyFormat, jti)
}

func (r *UserTokenCache) set(ctx context.Context, key string, value string, expires time.Duration) error {
	if err := r.rdb.Set(ctx, key, value, expires).Err(); err != nil {
		r.log.Errorf("set key[%s] value[%s] failed: %v", key, value, err)
		return err
	}
	return nil
}

func (r *UserTokenCache) get(ctx context.Context, key string) string {
	result, err := r.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ""
		}

		r.log.Errorf("get key[%s] failed: %v", key, err)
		return ""
	}
	return result
}

// del 删除键
func (r *UserTokenCache) del(ctx context.Context, key string) error {
	if err := r.rdb.Del(ctx, key).Err(); err != nil {
		r.log.Errorf("del key[%s] failed: %v", key, err)
		return err
	}
	return nil
}

func (r *UserTokenCache) exists(ctx context.Context, key string) bool {
	n, err := r.rdb.Exists(ctx, key).Result()
	if err != nil {
		r.log.Errorf("exists key[%s] failed: %v", key, err)
		return false
	}
	return n > 0
}

// hset 设置字段
func (r *UserTokenCache) hset(ctx context.Context, key string, field, value string, expires time.Duration) error {
	var err error
	if err = r.rdb.HSet(ctx, key, field, value).Err(); err != nil {
		r.log.Errorf("hset key[%s] field[%s] failed: %v", key, field, err)
		return err
	}

	if expires > 0 {
		// WARN: HExpire有版本要求，请确保使用的redis版本支持该命令。
		if err = r.rdb.HExpire(ctx, key, expires, field).Err(); err != nil {
			r.log.Errorf("hexpire key[%s] field[%s] failed: %v", key, field, err)
			return err
		}
	}

	return nil
}

// hgetValues 获取所有字段
func (r *UserTokenCache) hgetValues(ctx context.Context, key string) []string {
	n, err := r.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return []string{}
		}

		r.log.Errorf("hgetValues key[%s] failed: %v", key, err)
		return []string{}
	}

	var tokens []string
	for _, v := range n {
		tokens = append(tokens, v)
	}

	return tokens
}

// hexists 判断字段是否存在
func (r *UserTokenCache) hexists(ctx context.Context, key string, field string) bool {
	n, err := r.rdb.HExists(ctx, key, field).Result()
	if err != nil {
		r.log.Errorf("hexists key[%s] field[%s] failed: %v", key, field, err)
		return false
	}
	return n
}

// hdel 删除字段
func (r *UserTokenCache) hdel(ctx context.Context, key string, field string) error {
	if err := r.rdb.HDel(ctx, key, field).Err(); err != nil {
		r.log.Errorf("hdel key[%s] field[%s] failed: %v", key, field, err)
		return err
	}
	return nil
}
