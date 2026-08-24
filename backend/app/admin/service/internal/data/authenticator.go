package data

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"time"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"github.com/tx7do/go-utils/jwtutil"
	"github.com/tx7do/go-utils/trans"

	authnEngine "github.com/tx7do/kratos-authn/engine"
	authnJwt "github.com/tx7do/kratos-authn/engine/jwt"

	conf "github.com/tx7do/kratos-bootstrap/api/gen/go/conf/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"

	"go-wind-admin/pkg/jwt"
)

// devSamplePrivateKeyFingerprint 是 configs/auth.yaml 内置开发示例 RSA 私钥的
// PEM body 指纹（首行），用于检测示例密钥是否仍生效。替换 yaml 或用 env 注入
// 生产密钥后即不再命中。
const devSamplePrivateKeyFingerprint = "MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCL8EdeTLDTf8AY"

// applyJwtKeyOverrides 环境变量密钥覆盖 + 开发示例密钥安全检查（见调用处注释）。
func applyJwtKeyOverrides(logHelper *bLogger.Helper, jwtCfg *conf.Authentication_Jwt) *conf.Authentication_Jwt {
	if jwtCfg == nil {
		return jwtCfg
	}
	if v := os.Getenv("GWA_AUTH_JWT_PRIVATE_KEY"); v != "" {
		jwtCfg.PrivateKey = trans.Ptr(v)
	}
	if v := os.Getenv("GWA_AUTH_JWT_PUBLIC_KEY"); v != "" {
		jwtCfg.PublicKey = trans.Ptr(v)
	}
	if v := os.Getenv("GWA_AUTH_JWT_KEY"); v != "" {
		jwtCfg.Key = v
	}

	if strings.Contains(jwtCfg.GetPrivateKey(), devSamplePrivateKeyFingerprint) {
		// 仅打醒目告警，不阻断启动——测试/本地环境普遍沿用示例密钥，
		// 强行 panic 会断掉开发与联调链路。生产替换由部署流程保证。
		logHelper.Warnf(context.Background(), "⚠️ configs/auth.yaml 中的开发示例 JWT 私钥仍在生效，生产环境必须替换："+
			"通过环境变量 GWA_AUTH_JWT_PRIVATE_KEY / GWA_AUTH_JWT_PUBLIC_KEY 注入，"+
			"或修改 yaml（生成命令见 auth.yaml 内注释）。")
	}
	return jwtCfg
}

const (
	// DefaultAccessTokenExpires  默认访问令牌过期时间（配置未指定时使用）
	DefaultAccessTokenExpires = time.Minute * 15

	// DefaultRefreshTokenExpires 默认刷新令牌过期时间（配置未指定时使用）
	DefaultRefreshTokenExpires = time.Hour * 24 * 7
)

type Authenticator struct {
	log *bLogger.Helper

	AdminAuthenticator authnEngine.Authenticator

	// jwtCfg 保留 JWT 配置，用于读取令牌过期时间。
	jwtCfg *conf.Authentication_Jwt

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

	jwtCfg := cfg.Authn.GetJwt()

	logHelper := ctx.NewLoggerHelper("authenticator/data/authentication-service")

	// 密钥注入与安全检查：
	// 1) 环境变量优先于 yaml——生产环境应通过 GWA_AUTH_JWT_PRIVATE_KEY /
	//    GWA_AUTH_JWT_PUBLIC_KEY（非对称）或 GWA_AUTH_JWT_KEY（对称）注入密钥，
	//    避免 yaml 中的开发示例密钥被带入生产。
	// 2) 检测到开发示例私钥仍生效时打醒目告警（不阻断启动，避免断开发/测试链路）。
	jwtCfg = applyJwtKeyOverrides(logHelper, jwtCfg)

	a := Authenticator{
		log:            logHelper,
		jwtCfg:         jwtCfg,
		userTokenCache: userTokenCache,
	}

	adminAuth, err := newAdminAuthenticator(jwtCfg)
	if err != nil {
		// 启动期密钥/算法配置错误属于不可恢复故障，直接 panic 以暴露问题。
		panic(fmt.Sprintf("init admin authenticator failed: %v", err))
	}
	a.AdminAuthenticator = adminAuth

	return &a
}

// newAdminAuthenticator 根据配置构造 JWT 认证器，同时支持对称与非对称签名算法。
//   - 对称算法（HS256/HS384/HS512）：key 为共享秘钥。
//   - 非对称算法（RS256/RS384/RS512、PS256/PS384/PS512、ES256/...、EdDSA）：
//     private_key 用于签发，public_key 用于校验。若仅配置了 private_key（如本服务
//     既签发又校验的单体场景），则回退为从私钥派生公钥。
func newAdminAuthenticator(jwtCfg *conf.Authentication_Jwt) (authnEngine.Authenticator, error) {
	if jwtCfg == nil {
		return nil, fmt.Errorf("jwt config is nil")
	}

	method := jwtCfg.GetMethod()
	if method == "" {
		// 与底层库默认行为一致。
		method = "HS256"
	}

	opts := []authnJwt.Option{
		authnJwt.WithSigningMethod(method),
	}

	if isAsymmetricMethod(method) {
		// 非对称算法：优先使用独立配置的 private_key / public_key。
		switch {
		case jwtCfg.GetPrivateKey() != "" && jwtCfg.GetPublicKey() != "":
			opts = append(opts,
				authnJwt.WithPrivateKeyFromPEM([]byte(jwtCfg.GetPrivateKey())),
				authnJwt.WithPublicKeyFromPEM([]byte(jwtCfg.GetPublicKey())),
			)
		case jwtCfg.GetPrivateKey() != "":
			// 仅配置私钥：从私钥派生公钥（本服务既签发又校验的场景）。
			privPEM := []byte(jwtCfg.GetPrivateKey())
			publicKey, err := publicKeyFromPrivateKeyPEM(privPEM)
			if err != nil {
				return nil, fmt.Errorf("derive public key from private key: %w", err)
			}
			opts = append(opts,
				authnJwt.WithPrivateKeyFromPEM(privPEM),
				authnJwt.WithVerificationKey(publicKey),
			)
		case jwtCfg.GetPublicKey() != "":
			// 仅配置公钥：只能用于校验，无法签发。
			opts = append(opts, authnJwt.WithPublicKeyFromPEM([]byte(jwtCfg.GetPublicKey())))
		default:
			return nil, fmt.Errorf("asymmetric method %q requires private_key and/or public_key in config", method)
		}
	} else {
		// 对称算法：key 即 HMAC 共享秘钥，签发与校验共用。
		opts = append(opts, authnJwt.WithKey([]byte(jwtCfg.GetKey())))
	}

	return authnJwt.NewAuthenticator(opts...)
}

// isAsymmetricMethod 判断给定的签名算法是否为非对称（基于公钥/私钥对）算法。
func isAsymmetricMethod(method string) bool {
	switch strings.ToUpper(method) {
	case "RS256", "RS384", "RS512", // RSA
		"PS256", "PS384", "PS512", // RSA-PSS
		"ES256", "ES384", "ES512", // ECDSA
		"EDDSA": // Ed25519
		return true
	default:
		return false
	}
}

// publicKeyFromPrivateKeyPEM 从 PEM 编码的 RSA 私钥中派生出公钥。
// 适用于仅配置了私钥、本服务既签发又校验令牌的场景。
func publicKeyFromPrivateKeyPEM(pemBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from private key")
	}

	var privKey *rsa.PrivateKey
	switch block.Type {
	case "RSA PRIVATE KEY": // PKCS#1
		parsed, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse PKCS#1 private key: %w", err)
		}
		privKey = parsed
	case "PRIVATE KEY": // PKCS#8
		parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse PKCS#8 private key: %w", err)
		}
		rsaKey, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("PKCS#8 key is not an RSA private key")
		}
		privKey = rsaKey
	default:
		return nil, fmt.Errorf("unsupported PEM block type %q (expected RSA PRIVATE KEY or PRIVATE KEY)", block.Type)
	}

	return &privKey.PublicKey, nil
}

// GetAccessTokenExpires 获取访问令牌过期时间。
// 优先使用配置中的 access_token_expires，未配置时回退到默认值。
func (a *Authenticator) GetAccessTokenExpires(clientType authenticationV1.ClientType) time.Duration {
	if a.jwtCfg != nil && a.jwtCfg.GetAccessTokenExpires() != nil {
		return a.jwtCfg.GetAccessTokenExpires().AsDuration()
	}
	switch clientType {
	case authenticationV1.ClientType_admin:
		return DefaultAccessTokenExpires
	default:
		return DefaultAccessTokenExpires
	}
}

// GetRefreshTokenExpires 获取刷新令牌过期时间。
// 优先使用配置中的 refresh_token_expires，未配置时回退到默认值。
func (a *Authenticator) GetRefreshTokenExpires(clientType authenticationV1.ClientType) time.Duration {
	if a.jwtCfg != nil && a.jwtCfg.GetRefreshTokenExpires() != nil {
		return a.jwtCfg.GetRefreshTokenExpires().AsDuration()
	}
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
		// refresh token 现为自描述 JWT，身份信息从 token 自身解析
		rtAuthenticator, rtErr := a.getAuthenticator(req.GetClientType())
		if rtErr != nil {
			return &authenticationV1.ValidateTokenResponse{IsValid: false}, rtErr
		}
		rtClaims, rtAuthErr := rtAuthenticator.AuthenticateToken(req.GetToken())
		if rtAuthErr != nil {
			return &authenticationV1.ValidateTokenResponse{IsValid: false},
				authenticationV1.ErrorUnauthorized("refresh token authentication failed")
		}
		if jwt.IsTokenExpired(rtClaims) {
			return &authenticationV1.ValidateTokenResponse{IsValid: false},
				authenticationV1.ErrorUnauthorized("refresh token is expired")
		}
		rtUserId, _, rtParseErr := jwt.ParseRefreshTokenClaims(rtClaims)
		if rtParseErr != nil {
			return &authenticationV1.ValidateTokenResponse{IsValid: false},
				authenticationV1.ErrorUnauthorized("invalid refresh token")
		}
		exist, _, err := a.userTokenCache.IsExistRefreshToken(ctx, req.GetClientType(), rtUserId, req.GetToken())
		if err != nil {
			a.log.Errorf(ctx, "check refresh token exist failed: %s", err.Error())
		}
		if err != nil || !exist {
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
	if refreshToken, err = a.newRefreshToken(clientType, tokenPayload); refreshToken == "" || err != nil {
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
		a.log.Error(ctx, "userTokenCache is nil")
		return authenticationV1.ErrorServiceUnavailable("token cache unavailable")
	}

	if userId == 0 {
		return authenticationV1.ErrorBadRequest("invalid user id")
	}

	if _, err := a.getAuthenticator(clientType); err != nil {
		return err
	}

	if err := a.userTokenCache.RevokeToken(ctx, clientType, userId); err != nil {
		a.log.Errorf(ctx, "revoke user token failed: %v", err)
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

// VerifyRefreshToken 验证刷新令牌，并原子地吊销旧令牌对。
// refresh token 现为自描述 JWT，本函数从 token 自身验签解析出 uid/jti，
// 不再依赖 access token 提供的身份信息。随后用 Lua 脚本保证「验证 RT → 删除 RT →
// 删除 AT」的原子性，避免 TOCTOU 竞态。
func (a *Authenticator) VerifyRefreshToken(
	ctx context.Context,
	clientType authenticationV1.ClientType,
	refreshToken string,
) (userId uint32, jti string, err error) {
	if a.userTokenCache == nil {
		a.log.Error(ctx, "userTokenCache is nil")
		return 0, "", authenticationV1.ErrorServiceUnavailable("token cache unavailable")
	}
	if refreshToken == "" {
		return 0, "", authenticationV1.ErrorBadRequest("refresh token is empty")
	}

	authenticator, err := a.getAuthenticator(clientType)
	if err != nil {
		return 0, "", err
	}

	// 验签 refresh token JWT，提取 uid/jti
	claims, authErr := authenticator.AuthenticateToken(refreshToken)
	if authErr != nil {
		a.log.Errorf(ctx, "authenticate refresh token failed: [%s]", authErr)
		return 0, "", authenticationV1.ErrorIncorrectRefreshToken("invalid refresh token")
	}
	if jwt.IsTokenExpired(claims) {
		return 0, "", authenticationV1.ErrorIncorrectRefreshToken("refresh token is expired")
	}
	userId, jti, err = jwt.ParseRefreshTokenClaims(claims)
	if err != nil {
		a.log.Errorf(ctx, "parse refresh token claims failed: [%s]", err)
		return 0, "", authenticationV1.ErrorIncorrectRefreshToken("invalid refresh token")
	}

	// 原子验证并吊销旧令牌对（Lua 脚本保证原子性）
	var valid bool
	if valid, err = a.userTokenCache.VerifyAndRevokeTokenPair(ctx, clientType, userId, jti, refreshToken); err != nil {
		a.log.Errorf(ctx, "verify refresh token failed for user [%d]: [%s]", userId, err)
		return 0, "", authenticationV1.ErrorServiceUnavailable("verify refresh token failed")
	}
	if !valid {
		a.log.Errorf(ctx, "invalid refresh token for user [%d]", userId)
		return 0, "", authenticationV1.ErrorIncorrectRefreshToken("invalid refresh token")
	}

	return userId, jti, nil
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
			a.log.Errorf(ctx, "check access token existence failed: [%v]", err)
			return authenticationV1.ErrorServiceUnavailable("check access token existence failed")
		}
		if !exist {
			a.log.Warnf(ctx, "access token not found for user [%d]", req.GetUserId())
			return authenticationV1.ErrorAccessTokenNotFound("access token not found")
		}

	case *authenticationV1.BlockTokenRequest_Jti:
		var exist bool
		exist, err = a.userTokenCache.IsExistAccessTokenByJti(ctx, req.GetClientType(), req.GetUserId(), req.GetJti())
		if err != nil {
			a.log.Errorf(ctx, "check access token existence by jti failed: [%v]", err)
			return authenticationV1.ErrorServiceUnavailable("check access token existence failed")
		}
		if !exist {
			a.log.Warnf(ctx, "access token not found for user [%d] by jti", req.GetUserId())
			return authenticationV1.ErrorAccessTokenNotFound("access token not found")
		}
		jti = req.GetJti()

	default:
		a.log.Error(ctx, "invalid block token request target")
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
			a.log.Errorf(ctx, "check access token existence failed: [%v]", err)
			return authenticationV1.ErrorServiceUnavailable("check access token existence failed")
		}
		if !exist {
			a.log.Warnf(ctx, "access token not found for user [%d]", req.GetUserId())
			return authenticationV1.ErrorAccessTokenNotFound("access token not found")
		}

	case *authenticationV1.UnblockTokenRequest_Jti:
		var exist bool
		exist, err = a.userTokenCache.IsExistAccessTokenByJti(ctx, req.GetClientType(), req.GetUserId(), req.GetJti())
		if err != nil {
			a.log.Errorf(ctx, "check access token existence by jti failed: [%v]", err)
			return authenticationV1.ErrorServiceUnavailable("check access token existence failed")
		}
		if !exist {
			a.log.Warnf(ctx, "access token not found for user [%d] by jti", req.GetUserId())
			return authenticationV1.ErrorAccessTokenNotFound("access token not found")
		}
		jti = req.GetJti()

	default:
		a.log.Error(ctx, "invalid block token request target")
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
		a.log.Error(context.Background(), "invalid client type: [%v]", clientType)
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
		a.log.Error(context.Background(), "token payload is nil")
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
		a.log.Error(context.Background(), "create access token failed: [%v]", err)
		return "", authenticationV1.ErrorServiceUnavailable("create access token failed")
	}

	return accessToken, nil
}

// newRefreshToken 创建刷新令牌（自描述 JWT，仅含 uid/jti/exp/iat）。
// refresh token 以 HttpOnly Cookie 传输，需脱离 access token 独立鉴权，
// 因此自身携带定位 Redis 记录所需的最小 claims（uid + jti），不含业务权限字段。
func (a *Authenticator) newRefreshToken(
	clientType authenticationV1.ClientType,
	tokenPayload *authenticationV1.UserTokenPayload,
) (refreshToken string, err error) {
	if tokenPayload == nil {
		a.log.Error(context.Background(), "refresh token payload is nil")
		return "", authenticationV1.ErrorBadRequest("refresh token payload is nil")
	}

	expTime := time.Now().Add(a.GetRefreshTokenExpires(clientType))
	authClaims, err := jwt.NewRefreshTokenAuthClaims(tokenPayload.GetUserId(), tokenPayload.GetJti(), &expTime)
	if err != nil {
		a.log.Error(context.Background(), "create refresh token claims failed: [%v]", err)
		return "", authenticationV1.ErrorServiceUnavailable("create refresh token failed")
	}

	authenticator, err := a.getAuthenticator(clientType)
	if err != nil {
		return "", err
	}

	refreshToken, err = authenticator.CreateIdentity(*authClaims)
	if err != nil {
		a.log.Error(context.Background(), "create refresh token failed: [%v]", err)
		return "", authenticationV1.ErrorServiceUnavailable("create refresh token failed")
	}
	return refreshToken, nil
}

// newJwtId 创建 JWT ID
func (a *Authenticator) newJwtId() string {
	return jwtutil.NewJWTId()
}
