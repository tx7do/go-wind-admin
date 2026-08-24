package data

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/redis/go-redis/v9"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
)

const (
	// AccessTokenKeyFormat 访问令牌键前缀格式 at:{ct}:{uid}
	AccessTokenKeyFormat = "at:%d:%d"
	// AccessTokenFieldKeyFormat 访问令牌键格式（含 jti）at:{ct}:{uid}:{jti}
	AccessTokenFieldKeyFormat = "at:%d:%d:%s"

	// RefreshTokenKeyFormat 刷新令牌键前缀格式 rt:{ct}:{uid}
	RefreshTokenKeyFormat = "rt:%d:%d"
	// RefreshTokenFieldKeyFormat 刷新令牌键格式（含 jti）rt:{ct}:{uid}:{jti}
	RefreshTokenFieldKeyFormat = "rt:%d:%d:%s"

	// BlacklistKeyFormat 访问令牌黑名单键格式 bl:{jti}
	BlacklistKeyFormat = "bl:%s"

	// scanCount 每次 SCAN 返回的键数量提示（仅是建议值，非强制）
	scanCount = 100
)

// verifyAndRevokeRefreshTokenScript 原子验证并吊销刷新令牌的 Lua 脚本。
// 在单次 Redis 调用中完成「验证 RT → 删除 RT → 删除 AT」，避免 TOCTOU 竞态。
// 注意：迁移到 String key 后，rtKey/atKey 均为含 jti 的完整 key，使用 GET/DEL 操作。
// 返回值: 1=验证成功, 0=令牌不存在或值不匹配
var verifyAndRevokeRefreshTokenScript = redis.NewScript(`
	local rtKey = KEYS[1]
	local atKey = KEYS[2]
	local refreshToken = ARGV[1]

	local stored = redis.call('GET', rtKey)
	if not stored or stored ~= refreshToken then
		return 0
	end

	redis.call('DEL', rtKey)
	redis.call('DEL', atKey)
	return 1
`)

// UserTokenCache 用户令牌缓存
type UserTokenCache struct {
	log *bLogger.Helper
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

	atKey := r.makeAccessTokenFieldKey(clientType, userId, jti)
	pipe.Set(ctx, atKey, accessToken, accessTokenExpires)

	rtKey := r.makeRefreshTokenFieldKey(clientType, userId, jti)
	pipe.Set(ctx, rtKey, refreshToken, refreshTokenExpires)

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
	key := r.makeAccessTokenFieldKey(clientType, userId, jti)
	return r.set(ctx, key, accessToken, expires)
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
	key := r.makeRefreshTokenFieldKey(clientType, userId, jti)
	return r.set(ctx, key, refreshToken, expires)
}

// AddBlockedAccessToken 添加被阻止的访问令牌
func (r *UserTokenCache) AddBlockedAccessToken(ctx context.Context, jti string, reason string, expires time.Duration) error {
	key := r.makeBlacklistKey(jti)
	return r.set(ctx, key, reason, expires)
}

// GetAccessTokens 获取访问令牌
func (r *UserTokenCache) GetAccessTokens(ctx context.Context, clientType authenticationV1.ClientType, userId uint32) []string {
	pattern := r.makeAccessTokenKey(clientType, userId) + ":*"
	return r.scanValues(ctx, pattern)
}

// GetRefreshTokens 获取刷新令牌
func (r *UserTokenCache) GetRefreshTokens(ctx context.Context, clientType authenticationV1.ClientType, userId uint32) []string {
	pattern := r.makeRefreshTokenKey(clientType, userId) + ":*"
	return r.scanValues(ctx, pattern)
}

// RevokeToken 移除所有令牌
func (r *UserTokenCache) RevokeToken(ctx context.Context, clientType authenticationV1.ClientType, userId uint32) error {
	var err error
	if err = r.RevokeUserAllAccessToken(ctx, clientType, userId); err != nil {
		r.log.Errorf(ctx, "remove user access token failed: [%v]", err)
	}

	if err = r.RevokeUserAllRefreshToken(ctx, clientType, userId); err != nil {
		r.log.Errorf(ctx, "remove user refresh token failed: [%v]", err)
	}

	return err
}

func (r *UserTokenCache) RevokeTokenByJti(ctx context.Context, clientType authenticationV1.ClientType, userId uint32, jti string) error {
	var err error
	if err = r.RevokeAccessToken(ctx, clientType, userId, jti); err != nil {
		r.log.Errorf(ctx, "remove user access token failed: [%v]", err)
	}

	if err = r.RevokeRefreshToken(ctx, clientType, userId, jti); err != nil {
		r.log.Errorf(ctx, "remove user refresh token failed: [%v]", err)
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
	key := r.makeAccessTokenFieldKey(clientType, userId, jti)
	return r.del(ctx, key)
}

// RevokeRefreshToken 移除刷新令牌
func (r *UserTokenCache) RevokeRefreshToken(
	ctx context.Context,
	clientType authenticationV1.ClientType,
	userId uint32,
	jti string,
) error {
	key := r.makeRefreshTokenFieldKey(clientType, userId, jti)
	return r.del(ctx, key)
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
	key := r.makeAccessTokenFieldKey(clientType, userId, jti)

	storedToken, err := r.rdb.Get(ctx, key).Result()
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
	key := r.makeAccessTokenFieldKey(clientType, userId, jti)

	_, err := r.rdb.Get(ctx, key).Result()
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
	pattern := r.makeAccessTokenKey(clientType, userId) + ":*"
	return r.scanFindValue(ctx, pattern, uploadedToken)
}

// IsExistRefreshToken 刷新令牌是否存在
func (r *UserTokenCache) IsExistRefreshToken(
	ctx context.Context,
	clientType authenticationV1.ClientType,
	userId uint32,
	uploadedToken string,
) (exist bool, jti string, err error) {
	pattern := r.makeRefreshTokenKey(clientType, userId) + ":*"
	return r.scanFindValue(ctx, pattern, uploadedToken)
}

// IsValidRefreshToken 刷新令牌是否有效
func (r *UserTokenCache) IsValidRefreshToken(
	ctx context.Context,
	clientType authenticationV1.ClientType,
	userId uint32,
	jti string,
	uploadedToken string,
) (bool, error) {
	key := r.makeRefreshTokenFieldKey(clientType, userId, jti)

	storedToken, err := r.rdb.Get(ctx, key).Result()
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
	pattern := r.makeAccessTokenKey(clientType, userId) + ":*"
	return r.delByPattern(ctx, pattern)
}

// RevokeUserAllRefreshToken 删除刷新令牌
func (r *UserTokenCache) RevokeUserAllRefreshToken(
	ctx context.Context,
	clientType authenticationV1.ClientType,
	userId uint32,
) error {
	pattern := r.makeRefreshTokenKey(clientType, userId) + ":*"
	return r.delByPattern(ctx, pattern)
}

// VerifyAndRevokeTokenPair 原子验证并吊销刷新令牌及其关联的访问令牌。
// 使用 Lua 脚本保证「验证→删除」操作的原子性，避免 TOCTOU 竞态条件。
// 返回 (true, nil) 表示刷新令牌有效且已成功吊销旧令牌对。
func (r *UserTokenCache) VerifyAndRevokeTokenPair(
	ctx context.Context,
	clientType authenticationV1.ClientType,
	userId uint32,
	jti string,
	refreshToken string,
) (bool, error) {
	rtKey := r.makeRefreshTokenFieldKey(clientType, userId, jti)
	atKey := r.makeAccessTokenFieldKey(clientType, userId, jti)

	result, err := verifyAndRevokeRefreshTokenScript.Run(ctx, r.rdb, []string{rtKey, atKey}, refreshToken).Int64()
	if err != nil {
		r.log.Errorf(ctx, "verifyAndRevokeTokenPair failed for user [%d] jti [%s]: %v", userId, jti, err)
		return false, err
	}

	return result == 1, nil
}

// ==============================
// key 生成
// ==============================

// makeAccessTokenKey 生成访问令牌键前缀 at:{ct}:{uid}（用于 SCAN 匹配）
func (r *UserTokenCache) makeAccessTokenKey(clientType authenticationV1.ClientType, userId uint32) string {
	return fmt.Sprintf(AccessTokenKeyFormat, clientType.Number(), userId)
}

// makeAccessTokenFieldKey 生成访问令牌键（含 jti）at:{ct}:{uid}:{jti}
func (r *UserTokenCache) makeAccessTokenFieldKey(clientType authenticationV1.ClientType, userId uint32, jti string) string {
	return fmt.Sprintf(AccessTokenFieldKeyFormat, clientType.Number(), userId, jti)
}

// makeRefreshTokenKey 生成刷新令牌键前缀 rt:{ct}:{uid}（用于 SCAN 匹配）
func (r *UserTokenCache) makeRefreshTokenKey(clientType authenticationV1.ClientType, userId uint32) string {
	return fmt.Sprintf(RefreshTokenKeyFormat, clientType.Number(), userId)
}

// makeRefreshTokenFieldKey 生成刷新令牌键（含 jti）rt:{ct}:{uid}:{jti}
func (r *UserTokenCache) makeRefreshTokenFieldKey(clientType authenticationV1.ClientType, userId uint32, jti string) string {
	return fmt.Sprintf(RefreshTokenFieldKeyFormat, clientType.Number(), userId, jti)
}

// makeBlacklistKey 生成黑名单键
func (r *UserTokenCache) makeBlacklistKey(jti string) string {
	return fmt.Sprintf(BlacklistKeyFormat, jti)
}

// ==============================
// 基础 String 操作（黑名单等使用）
// ==============================

func (r *UserTokenCache) set(ctx context.Context, key string, value string, expires time.Duration) error {
	if err := r.rdb.Set(ctx, key, value, expires).Err(); err != nil {
		r.log.Errorf(ctx, "set key[%s] value[%s] failed: %v", key, value, err)
		return err
	}
	return nil
}

// del 删除键
func (r *UserTokenCache) del(ctx context.Context, key string) error {
	if err := r.rdb.Del(ctx, key).Err(); err != nil {
		r.log.Errorf(ctx, "del key[%s] failed: %v", key, err)
		return err
	}
	return nil
}

func (r *UserTokenCache) exists(ctx context.Context, key string) bool {
	n, err := r.rdb.Exists(ctx, key).Result()
	if err != nil {
		r.log.Errorf(ctx, "exists key[%s] failed: %v", key, err)
		return false
	}
	return n > 0
}

// ==============================
// 基于 SCAN 的批量操作（替代原 Hash 操作）
// ==============================

// scanKeys 使用 SCAN 遍历匹配 pattern 的所有 key。
// 正确处理游标迭代，直到游标归零为止，避免遗漏。
func (r *UserTokenCache) scanKeys(ctx context.Context, pattern string) ([]string, error) {
	var keys []string
	var cursor uint64
	var err error

	for {
		var batch []string
		batch, cursor, err = r.rdb.Scan(ctx, cursor, pattern, scanCount).Result()
		if err != nil {
			r.log.Errorf(ctx, "scan pattern[%s] failed: %v", pattern, err)
			return nil, err
		}

		keys = append(keys, batch...)

		if cursor == 0 {
			break
		}
	}

	return keys, nil
}

// scanValues 获取匹配 pattern 的所有 key 的值。
func (r *UserTokenCache) scanValues(ctx context.Context, pattern string) []string {
	keys, err := r.scanKeys(ctx, pattern)
	if err != nil || len(keys) == 0 {
		return []string{}
	}

	values, err := r.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		r.log.Errorf(ctx, "mget pattern[%s] failed: %v", pattern, err)
		return []string{}
	}

	var result []string
	for _, v := range values {
		if v == nil {
			continue // key 已过期
		}
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}

	return result
}

// scanFindValue 在匹配 pattern 的所有 key 中查找值等于 target 的 key，并返回其 jti。
// 用于「按 token 值反查 jti」的场景（如黑名单操作）。
func (r *UserTokenCache) scanFindValue(ctx context.Context, pattern string, target string) (bool, string, error) {
	keys, err := r.scanKeys(ctx, pattern)
	if err != nil {
		return false, "", err
	}
	if len(keys) == 0 {
		return false, "", nil
	}

	values, err := r.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		r.log.Errorf(ctx, "mget pattern[%s] failed: %v", pattern, err)
		return false, "", err
	}

	for i, v := range values {
		if s, ok := v.(string); ok && s == target {
			return true, r.extractJtiFromKey(keys[i]), nil
		}
	}

	return false, "", nil
}

// delByPattern 删除匹配 pattern 的所有 key。
// 使用 SCAN + pipeline DEL，避免阻塞 Redis（不使用 KEYS）。
func (r *UserTokenCache) delByPattern(ctx context.Context, pattern string) error {
	keys, err := r.scanKeys(ctx, pattern)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}

	if err := r.rdb.Del(ctx, keys...).Err(); err != nil {
		r.log.Errorf(ctx, "del by pattern[%s] failed: %v", pattern, err)
		return err
	}

	return nil
}

// extractJtiFromKey 从键名 at:{ct}:{uid}:{jti} 中提取末段的 jti。
func (r *UserTokenCache) extractJtiFromKey(key string) string {
	parts := strings.Split(key, ":")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}
