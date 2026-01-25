package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/go-utils/jwtutil"
	"github.com/tx7do/go-utils/trans"

	authnEngine "github.com/tx7do/kratos-authn/engine"
	authnJwt "github.com/tx7do/kratos-authn/engine/jwt"

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

type Authenticator struct {
	log *log.Helper

	AdminAuthenticator authnEngine.Authenticator

	userTokenCache *UserTokenCache
}

func NewAuthenticator(
	ctx *bootstrap.Context,
	userTokenCache *UserTokenCache,
) *Authenticator {
	cfg := ctx.GetConfig()
	if cfg == nil || cfg.Authn == nil {
		return nil
	}

	a := Authenticator{
		log:            ctx.NewLoggerHelper("authenticator/data/authentication-service"),
		userTokenCache: userTokenCache,
	}

	a.AdminAuthenticator, _ = authnJwt.NewAuthenticator(
		authnJwt.WithKey([]byte(cfg.Authn.GetJwt().GetKey())),
		authnJwt.WithSigningMethod(cfg.Authn.GetJwt().GetMethod()),
	)

	return &a
}

// GetAccessTokenExpires 获取访问令牌过期时间
func (a *Authenticator) GetAccessTokenExpires(clientType authenticationV1.ClientType) time.Duration {
	switch clientType {
	case authenticationV1.ClientType_admin:
		return DefaultAccessTokenExpires
	default:
		return DefaultAccessTokenExpires
	}
}

// GetRefreshTokenExpires 获取刷新令牌过期时间
func (a *Authenticator) GetRefreshTokenExpires(clientType authenticationV1.ClientType) time.Duration {
	switch clientType {
	case authenticationV1.ClientType_admin:
		return DefaultRefreshTokenExpires
	default:
		return DefaultRefreshTokenExpires
	}
}

// Authenticate 根据不同的客户端类型验证 Token
func (a *Authenticator) Authenticate(ctx context.Context, req *authenticationV1.ValidateTokenRequest) (*authenticationV1.ValidateTokenResponse, error) {
	if req == nil {
		return nil, authenticationV1.ErrorBadRequest("validate token request is nil")
	}

	if req.GetToken() == "" {
		return nil, authenticationV1.ErrorBadRequest("token is empty")
	}

	authenticator, err := a.getAuthenticator(req.GetClientType())
	if err != nil {
		return nil, err
	}

	switch req.GetTokenCategory() {
	case authenticationV1.TokenCategory_ACCESS:
		// Authenticate Token
		var claims *authnEngine.AuthClaims
		claims, err = authenticator.AuthenticateToken(req.GetToken())
		if err != nil {
			return nil, authenticationV1.ErrorUnauthorized("authenticate token failed: [%v]", err)
		}

		// Check Token Expiration
		if jwt.IsTokenExpired(claims) {
			return &authenticationV1.ValidateTokenResponse{
				IsValid: false,
			}, authenticationV1.ErrorUnauthorized("access token is expired")
		}

		// Check Token Validity
		//if !jwt.IsTokenNotValidYet(claims) {
		//	return &authenticationV1.ValidateTokenResponse{
		//		IsValid: false,
		//	}, authenticationV1.ErrorUnauthorized("access token is not valid yet")
		//}

		// Parse Token Payload
		var payload *authenticationV1.UserTokenPayload
		payload, err = jwt.NewUserTokenPayloadWithClaims(claims)
		if err != nil {
			return &authenticationV1.ValidateTokenResponse{
				IsValid: false,
			}, err
		}

		// Check token validity in cache
		if !req.GetSkipRedis() {
			var valid bool
			if valid, err = a.userTokenCache.IsValidAccessToken(ctx, req.GetClientType(), payload.GetUserId(), payload.GetJti(), req.GetToken()); err != nil {
				return &authenticationV1.ValidateTokenResponse{
					IsValid: false,
				}, authenticationV1.ErrorUnauthorized("invalid access token: [%v]", err)
			}
			if !valid {
				return &authenticationV1.ValidateTokenResponse{
					IsValid: false,
				}, authenticationV1.ErrorUnauthorized("access token is revoked or expired")
			}
		}

		// Check if token is blocked
		if !req.GetSkipBlacklist() {
			if a.userTokenCache.IsBlockedAccessToken(ctx, payload.GetJti()) {
				return &authenticationV1.ValidateTokenResponse{
					IsValid: false,
				}, authenticationV1.ErrorUnauthorized("access token is blocked")
			}
		}

		return &authenticationV1.ValidateTokenResponse{
			IsValid: true,
			Payload: payload,
		}, nil

	case authenticationV1.TokenCategory_REFRESH:
		var exist bool
		if exist, _, err = a.userTokenCache.IsExistRefreshToken(ctx, req.GetClientType(), req.GetUserId(), req.GetToken()); !exist {
			return &authenticationV1.ValidateTokenResponse{
				IsValid: false,
			}, authenticationV1.ErrorUnauthorized("refresh token not found for user")
		}

		return &authenticationV1.ValidateTokenResponse{
			IsValid: true,
		}, nil

	default:
		return nil, authenticationV1.ErrorBadRequest("invalid token category")
	}
}

// CreateUserToken 创建用户令牌对（访问令牌和刷新令牌）
func (a *Authenticator) CreateUserToken(
	ctx context.Context,
	clientType authenticationV1.ClientType,
	tokenPayload *authenticationV1.UserTokenPayload,
) (accessToken, refreshToken string, err error) {
	if tokenPayload == nil {
		return "", "", authenticationV1.ErrorBadRequest("token payload is nil")
	}

	var jti string
	if jti = a.newJwtId(); jti == "" {
		return "", "", authenticationV1.ErrorServiceUnavailable("create jwt id failed")
	}

	tokenPayload.Jti = trans.Ptr(jti)

	// Create Access Token
	if accessToken, err = a.newAccessToken(clientType, tokenPayload); accessToken == "" || err != nil {
		return "", "", authenticationV1.ErrorServiceUnavailable("create access token failed")
	}

	// Create Refresh Token
	if refreshToken, err = a.newRefreshToken(); refreshToken == "" || err != nil {
		return "", "", authenticationV1.ErrorServiceUnavailable("create refresh token failed")
	}

	// Store tokens in cache
	if err = a.userTokenCache.AddTokenPair(
		ctx,
		clientType,
		tokenPayload.GetUserId(),
		jti,
		accessToken,
		refreshToken,
		a.GetAccessTokenExpires(clientType),
		a.GetRefreshTokenExpires(clientType),
	); err != nil {
		return "", "", err
	}

	return
}

// RevokeUserToken 撤销用户令牌
func (a *Authenticator) RevokeUserToken(ctx context.Context, clientType authenticationV1.ClientType, userId uint32) error {
	if a.userTokenCache == nil {
		a.log.Error("userTokenCache is nil")
		return authenticationV1.ErrorServiceUnavailable("token cache unavailable")
	}

	if userId == 0 {
		return authenticationV1.ErrorBadRequest("invalid user id")
	}

	if _, err := a.getAuthenticator(clientType); err != nil {
		return err
	}

	if err := a.userTokenCache.RevokeToken(ctx, clientType, userId); err != nil {
		a.log.Errorf("revoke user token failed: %v", err)
		return err
	}

	return nil
}

func (a *Authenticator) RevokeTokenByJti(ctx context.Context, clientType *authenticationV1.ClientType, userId uint32, jti string) error {
	if clientType != nil {
		if _, err := a.getAuthenticator(*clientType); err != nil {
			return err
		}
		return a.userTokenCache.RevokeTokenByJti(ctx, *clientType, userId, jti)
	}

	if err := a.userTokenCache.RevokeTokenByJti(ctx, authenticationV1.ClientType_admin, userId, jti); err != nil {
		return err
	}
	return a.userTokenCache.RevokeTokenByJti(ctx, authenticationV1.ClientType_app, userId, jti)
}

// VerifyRefreshToken 验证刷新令牌
func (a *Authenticator) VerifyRefreshToken(
	ctx context.Context,
	clientType authenticationV1.ClientType,
	userId uint32,
	jti string,
	refreshToken string,
) (err error) {
	if a.userTokenCache == nil {
		a.log.Error("userTokenCache is nil")
		return authenticationV1.ErrorServiceUnavailable("token cache unavailable")
	}
	if userId == 0 {
		return authenticationV1.ErrorBadRequest("invalid user id")
	}
	if jti == "" || refreshToken == "" {
		return authenticationV1.ErrorBadRequest("jti or refresh token is empty")
	}
	if _, err = a.getAuthenticator(clientType); err != nil {
		return err
	}

	// 校验刷新令牌
	var valid bool
	if valid, err = a.userTokenCache.IsValidRefreshToken(ctx, clientType, userId, jti, refreshToken); !valid || err != nil {
		a.log.Errorf("invalid refresh token for user [%d]: [%s]", userId, err)
		return authenticationV1.ErrorIncorrectRefreshToken("invalid refresh token")
	}

	// 撤销已使用的刷新令牌
	if err = a.userTokenCache.RevokeRefreshToken(ctx, clientType, userId, jti); err != nil {
		a.log.Errorf("remove refresh token failed [%s]", err.Error())
		return authenticationV1.ErrorServiceUnavailable("remove refresh token failed")
	}

	if err = a.userTokenCache.RevokeAccessToken(ctx, clientType, userId, jti); err != nil {
		a.log.Errorf("remove access token failed for user [%d] jti[%s]: %v", userId, jti, err)
		return authenticationV1.ErrorServiceUnavailable("remove access token failed")
	}

	return nil
}

// GetAccessTokens 获取用户的所有访问令牌
func (a *Authenticator) GetAccessTokens(
	ctx context.Context,
	clientType authenticationV1.ClientType,
	userId uint32,
) []string {
	return a.userTokenCache.GetAccessTokens(ctx, clientType, userId)
}

// BlockToken 封禁访问令牌
func (a *Authenticator) BlockToken(
	ctx context.Context,
	req *authenticationV1.BlockTokenRequest,
) (err error) {
	var jti string
	switch req.Target.(type) {
	case *authenticationV1.BlockTokenRequest_Token:
		var exist bool
		exist, jti, err = a.userTokenCache.IsExistAccessToken(ctx, req.GetClientType(), req.GetUserId(), req.GetToken())
		if err != nil {
			a.log.Errorf("check access token existence failed: [%v]", err)
			return authenticationV1.ErrorServiceUnavailable("check access token existence failed")
		}
		if !exist {
			a.log.Warnf("access token not found for user [%d]", req.GetUserId())
			return authenticationV1.ErrorAccessTokenNotFound("access token not found")
		}

	case *authenticationV1.BlockTokenRequest_Jti:
		var exist bool
		exist, err = a.userTokenCache.IsExistAccessTokenByJti(ctx, req.GetClientType(), req.GetUserId(), req.GetJti())
		if err != nil {
			a.log.Errorf("check access token existence by jti failed: [%v]", err)
			return authenticationV1.ErrorServiceUnavailable("check access token existence failed")
		}
		if !exist {
			a.log.Warnf("access token not found for user [%d] by jti", req.GetUserId())
			return authenticationV1.ErrorAccessTokenNotFound("access token not found")
		}
		jti = req.GetJti()

	default:
		a.log.Error("invalid block token request target")
		return authenticationV1.ErrorBadRequest("invalid block token request target")
	}

	return a.userTokenCache.AddBlockedAccessToken(ctx, jti, req.GetReason(), req.GetDuration().AsDuration())
}

func (a *Authenticator) UnblockToken(
	ctx context.Context,
	req *authenticationV1.UnblockTokenRequest,
) (err error) {
	var jti string
	switch req.Target.(type) {
	case *authenticationV1.UnblockTokenRequest_Token:
		var exist bool
		exist, jti, err = a.userTokenCache.IsExistAccessToken(ctx, req.GetClientType(), req.GetUserId(), req.GetToken())
		if err != nil {
			a.log.Errorf("check access token existence failed: [%v]", err)
			return authenticationV1.ErrorServiceUnavailable("check access token existence failed")
		}
		if !exist {
			a.log.Warnf("access token not found for user [%d]", req.GetUserId())
			return authenticationV1.ErrorAccessTokenNotFound("access token not found")
		}

	case *authenticationV1.UnblockTokenRequest_Jti:
		var exist bool
		exist, err = a.userTokenCache.IsExistAccessTokenByJti(ctx, req.GetClientType(), req.GetUserId(), req.GetJti())
		if err != nil {
			a.log.Errorf("check access token existence by jti failed: [%v]", err)
			return authenticationV1.ErrorServiceUnavailable("check access token existence failed")
		}
		if !exist {
			a.log.Warnf("access token not found for user [%d] by jti", req.GetUserId())
			return authenticationV1.ErrorAccessTokenNotFound("access token not found")
		}
		jti = req.GetJti()

	default:
		a.log.Error("invalid block token request target")
		return authenticationV1.ErrorBadRequest("invalid block token request target")
	}

	return a.userTokenCache.RevokeTokenByJti(ctx, req.GetClientType(), req.GetUserId(), jti)
}

// getAuthenticator 根据客户端类型获取认证器
func (a *Authenticator) getAuthenticator(clientType authenticationV1.ClientType) (authnEngine.Authenticator, error) {
	var authenticator authnEngine.Authenticator
	switch clientType {
	case authenticationV1.ClientType_admin:
		authenticator = a.AdminAuthenticator
	case authenticationV1.ClientType_app:
	default:
		a.log.Error("invalid client type: [%v]", clientType)
		return nil, authenticationV1.ErrorBadRequest("invalid client type")
	}
	return authenticator, nil
}

// newAccessToken 创建访问令牌
func (a *Authenticator) newAccessToken(
	clientType authenticationV1.ClientType,
	tokenPayload *authenticationV1.UserTokenPayload,
) (accessToken string, err error) {
	if tokenPayload == nil {
		a.log.Error("token payload is nil")
		return "", authenticationV1.ErrorBadRequest("token payload is nil")
	}

	expTime := time.Now().Add(a.GetAccessTokenExpires(clientType))
	authClaims := jwt.NewUserTokenAuthClaims(tokenPayload, &expTime)

	authenticator, err := a.getAuthenticator(clientType)
	if err != nil {
		return "", err
	}

	accessToken, err = authenticator.CreateIdentity(*authClaims)
	if err != nil {
		a.log.Error("create access token failed: [%v]", err)
		return "", authenticationV1.ErrorServiceUnavailable("create access token failed")
	}

	return accessToken, nil
}

// newRefreshToken 创建刷新令牌
func (a *Authenticator) newRefreshToken() (refreshToken string, err error) {
	refreshToken, err = jwtutil.NewRefreshToken()
	if err != nil {
		a.log.Error("create refresh token failed: [%v]", err)
		return "", authenticationV1.ErrorServiceUnavailable("create refresh token failed")
	}
	return refreshToken, nil
}

// newJwtId 创建 JWT ID
func (a *Authenticator) newJwtId() string {
	return jwtutil.NewJWTId()
}
