package jwt

import (
	"errors"
	"time"

	authn "github.com/tx7do/kratos-authn/engine"
)

// Refresh token 的 claim 字段复用 access token 的 ClaimFieldUserID ("uid")
// 与 authn.ClaimFieldJwtID ("jti")，加上标准的 exp/iat。
// refresh token JWT 不携带 sub/tid/roc/ds/ouid/ipa/ita/cid/did 等业务字段，
// 仅含定位 Redis 记录所需的最小信息（uid + jti），使其能脱离 access token
// 独立完成刷新鉴权。

// NewRefreshTokenAuthClaims 创建刷新令牌的最小认证声明。
// 仅包含 uid、jti、exp、iat，不含任何业务权限字段。
func NewRefreshTokenAuthClaims(
	userId uint32,
	jti string,
	expirationTime *time.Time,
) (*authn.AuthClaims, error) {
	if userId == 0 {
		return nil, errors.New("refresh token claims: userId is empty")
	}
	if jti == "" {
		return nil, errors.New("refresh token claims: jti is empty")
	}

	authClaims := authn.AuthClaims{
		ClaimFieldUserID:         userId,
		authn.ClaimFieldJwtID:    jti,
		authn.ClaimFieldIssuedAt: time.Now().Unix(),
	}

	if expirationTime != nil {
		authClaims[authn.ClaimFieldExpirationTime] = expirationTime.Unix()
	}

	return &authClaims, nil
}

// ParseRefreshTokenClaims 从已验签的 refresh token claims 中提取 uid 和 jti。
// 调用方须先用 authenticator.AuthenticateToken 验签取得 *authn.AuthClaims，
// 再调用本函数提取定位信息。
func ParseRefreshTokenClaims(claims *authn.AuthClaims) (userId uint32, jti string, err error) {
	if claims == nil {
		return 0, "", errors.New("refresh token claims: nil claims")
	}

	userId, err = claims.GetUint32(ClaimFieldUserID)
	if err != nil {
		return 0, "", err
	}
	if userId == 0 {
		return 0, "", errors.New("refresh token claims: uid is empty")
	}

	jti, err = claims.GetJwtID()
	if err != nil {
		return 0, "", err
	}
	if jti == "" {
		return 0, "", errors.New("refresh token claims: jti is empty")
	}

	return userId, jti, nil
}
