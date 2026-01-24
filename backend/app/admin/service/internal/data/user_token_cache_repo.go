package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	authnEngine "github.com/tx7do/kratos-authn/engine"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"

	"go-wind-admin/pkg/jwt"
)

const (
	// DefaultAccessTokenExpires  默认访问令牌过期时间
	DefaultAccessTokenExpires = time.Minute * 15

	// DefaultRefreshTokenExpires 默认刷新令牌过期时间
	DefaultRefreshTokenExpires = time.Hour * 24 * 7
)

type UserTokenCacheRepo struct {
	log *log.Helper

	rdb *redis.Client // redis客户端

	authenticator authnEngine.Authenticator // 身份验证器

	accessTokenKeyPrefix  string // 访问令牌键前缀
	refreshTokenKeyPrefix string // 刷新令牌键前缀

	accessTokenExpires  time.Duration // 访问令牌过期时间
	refreshTokenExpires time.Duration // 刷新令牌过期时间
}

func NewUserTokenCacheRepo(
	ctx *bootstrap.Context,
	rdb *redis.Client,
	authenticator authnEngine.Authenticator,
	accessTokenKeyPrefix string,
	refreshTokenKeyPrefix string,
	accessTokenExpires time.Duration,
	refreshTokenExpires time.Duration,
) *UserTokenCacheRepo {
	return &UserTokenCacheRepo{
		rdb:                   rdb,
		log:                   ctx.NewLoggerHelper("user-token/cache/admin-service"),
		authenticator:         authenticator,
		accessTokenKeyPrefix:  accessTokenKeyPrefix,
		refreshTokenKeyPrefix: refreshTokenKeyPrefix,
		accessTokenExpires:    accessTokenExpires,
		refreshTokenExpires:   refreshTokenExpires,
	}
}

// GetAccessTokenExpires 获取访问令牌过期时间
func (r *UserTokenCacheRepo) GetAccessTokenExpires() time.Duration {
	return r.accessTokenExpires
}

// GetRefreshTokenExpires 获取刷新令牌过期时间
func (r *UserTokenCacheRepo) GetRefreshTokenExpires() time.Duration {
	return r.refreshTokenExpires
}

// GenerateToken 创建令牌
func (r *UserTokenCacheRepo) GenerateToken(
	ctx context.Context,
	tokenPayload *authenticationV1.UserTokenPayload,
) (accessToken, refreshToken string, err error) {
	// 创建访问令牌
	if accessToken, err = r.GenerateAccessToken(
		ctx,
		tokenPayload,
	); accessToken == "" {
		err = errors.New("create access token failed")
		return
	}

	// 创建刷新令牌
	if refreshToken, err = r.GenerateRefreshToken(ctx, tokenPayload.UserId); refreshToken == "" {
		err = errors.New("create refresh token failed")
		return
	}

	return
}

// GenerateAccessToken 创建访问令牌
func (r *UserTokenCacheRepo) GenerateAccessToken(
	ctx context.Context,
	tokenPayload *authenticationV1.UserTokenPayload,
) (accessToken string, err error) {
	if accessToken = r.newAccessJwtToken(tokenPayload); accessToken == "" {
		err = errors.New("create access token failed")
		return
	}

	if err = r.setAccessTokenToRedis(ctx, tokenPayload.GetUserId(), accessToken, r.accessTokenExpires); err != nil {
		return
	}

	return
}

// GenerateRefreshToken 创建刷新令牌
func (r *UserTokenCacheRepo) GenerateRefreshToken(ctx context.Context, userID uint32) (refreshToken string, err error) {
	if refreshToken = r.newRefreshToken(); refreshToken == "" {
		err = errors.New("create refresh token failed")
		return
	}

	if err = r.setRefreshTokenToRedis(ctx, userID, refreshToken, r.refreshTokenExpires); err != nil {
		return
	}

	return
}

// AddBlockedAccessToken 添加被阻止的访问令牌
func (r *UserTokenCacheRepo) AddBlockedAccessToken(ctx context.Context, userId uint32, accessToken string, expires time.Duration) error {
	key := r.makeAccessTokenBlockerKey(userId)
	return r.set(ctx, key, accessToken, expires)
}

// GetAccessTokens 获取访问令牌列表
func (r *UserTokenCacheRepo) GetAccessTokens(ctx context.Context, userId uint32) []string {
	key := r.makeAccessTokenKey(userId)
	return r.get(ctx, key)
}

// GetRefreshTokens 获取刷新令牌列表
func (r *UserTokenCacheRepo) GetRefreshTokens(ctx context.Context, userId uint32) []string {
	key := r.makeRefreshTokenKey(userId)
	return r.get(ctx, key)
}

// RemoveToken 移除所有令牌
func (r *UserTokenCacheRepo) RemoveToken(ctx context.Context, userId uint32) error {
	var err error
	if err = r.deleteAccessTokenFromRedis(ctx, userId); err != nil {
		r.log.Errorf("remove user access token failed: [%v]", err)
	}

	if err = r.deleteRefreshTokenFromRedis(ctx, userId); err != nil {
		r.log.Errorf("remove user refresh token failed: [%v]", err)
	}

	return err
}

// RemoveAccessToken 访问令牌
func (r *UserTokenCacheRepo) RemoveAccessToken(ctx context.Context, userId uint32, accessToken string) error {
	key := r.makeAccessTokenKey(userId)
	return r.delField(ctx, key, accessToken)
}

// RemoveRefreshToken 刷新令牌
func (r *UserTokenCacheRepo) RemoveRefreshToken(ctx context.Context, userId uint32, refreshToken string) error {
	key := r.makeRefreshTokenKey(userId)
	return r.delField(ctx, key, refreshToken)
}

// RemoveBlockedAccessToken 移除被阻止的访问令牌
func (r *UserTokenCacheRepo) RemoveBlockedAccessToken(ctx context.Context, userId uint32, accessToken string) error {
	key := r.makeAccessTokenBlockerKey(userId)
	return r.delField(ctx, key, accessToken)
}

// IsExistAccessToken 访问令牌是否存在
func (r *UserTokenCacheRepo) IsExistAccessToken(ctx context.Context, userId uint32, accessToken string) bool {
	key := r.makeAccessTokenKey(userId)
	return r.exists(ctx, key, accessToken)
}

// IsExistRefreshToken 刷新令牌是否存在
func (r *UserTokenCacheRepo) IsExistRefreshToken(ctx context.Context, userId uint32, refreshToken string) bool {
	key := r.makeRefreshTokenKey(userId)
	return r.exists(ctx, key, refreshToken)
}

// IsBlockedAccessToken 访问令牌是否被阻止
func (r *UserTokenCacheRepo) IsBlockedAccessToken(ctx context.Context, userId uint32, accessToken string) bool {
	key := r.makeAccessTokenBlockerKey(userId)
	return r.exists(ctx, key, accessToken)
}

// setAccessTokenToRedis 设置访问令牌
func (r *UserTokenCacheRepo) setAccessTokenToRedis(ctx context.Context, userId uint32, token string, expires time.Duration) error {
	key := r.makeAccessTokenKey(userId)
	return r.set(ctx, key, token, expires)
}

// set 设置字段
func (r *UserTokenCacheRepo) set(ctx context.Context, key string, token string, expires time.Duration) error {
	var err error
	if err = r.rdb.HSet(ctx, key, token, "").Err(); err != nil {
		return err
	}

	if expires > 0 {
		if err = r.rdb.HExpire(ctx, key, expires, token).Err(); err != nil {
			return err
		}
	}

	return nil
}

// get 获取所有字段
func (r *UserTokenCacheRepo) get(ctx context.Context, key string) []string {
	n, err := r.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return []string{}
	}

	var tokens []string
	for k, _ := range n {
		tokens = append(tokens, k)
	}

	return tokens
}

// exists 判断字段是否存在
func (r *UserTokenCacheRepo) exists(ctx context.Context, key string, token string) bool {
	n, err := r.rdb.HExists(ctx, key, token).Result()
	if err != nil {
		return false
	}
	return n
}

// del 删除键
func (r *UserTokenCacheRepo) del(ctx context.Context, key string) error {
	return r.rdb.Del(ctx, key).Err()
}

// delField 删除字段
func (r *UserTokenCacheRepo) delField(ctx context.Context, key string, token string) error {
	return r.rdb.HDel(ctx, key, token).Err()
}

// deleteAccessTokenFromRedis 删除访问令牌
func (r *UserTokenCacheRepo) deleteAccessTokenFromRedis(ctx context.Context, userId uint32) error {
	key := r.makeAccessTokenKey(userId)
	return r.del(ctx, key)
}

// setRefreshTokenToRedis 设置刷新令牌
func (r *UserTokenCacheRepo) setRefreshTokenToRedis(ctx context.Context, userId uint32, token string, expires time.Duration) error {
	key := r.makeRefreshTokenKey(userId)
	return r.set(ctx, key, token, expires)
}

// deleteRefreshTokenFromRedis 删除刷新令牌
func (r *UserTokenCacheRepo) deleteRefreshTokenFromRedis(ctx context.Context, userId uint32) error {
	key := r.makeRefreshTokenKey(userId)
	return r.del(ctx, key)
}

// newAccessJwtToken 生成JWT访问令牌
func (r *UserTokenCacheRepo) newAccessJwtToken(
	tokenPayload *authenticationV1.UserTokenPayload,
) string {
	expTime := time.Now().Add(r.accessTokenExpires)
	authClaims := jwt.NewUserTokenAuthClaims(tokenPayload, &expTime)

	signedToken, err := r.authenticator.CreateIdentity(*authClaims)
	if err != nil {
		r.log.Error("create access token failed: [%v]", err)
	}

	return signedToken
}

// newRefreshToken 生成刷新令牌
func (r *UserTokenCacheRepo) newRefreshToken() string {
	strUUID := uuid.New()
	return strUUID.String()
}

// makeAccessTokenKey 生成访问令牌键
func (r *UserTokenCacheRepo) makeAccessTokenKey(userId uint32) string {
	return fmt.Sprintf("%s%d", r.accessTokenKeyPrefix, userId)
}

// makeRefreshTokenKey 生成刷新令牌键
func (r *UserTokenCacheRepo) makeRefreshTokenKey(userId uint32) string {
	return fmt.Sprintf("%s%d", r.refreshTokenKeyPrefix, userId)
}

// makeAccessTokenBlockerKey 生成访问令牌阻止键
func (r *UserTokenCacheRepo) makeAccessTokenBlockerKey(userId uint32) string {
	return fmt.Sprintf("%sblocker:%d", r.accessTokenKeyPrefix, userId)
}
