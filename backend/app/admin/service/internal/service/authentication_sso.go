package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	ktransport "github.com/go-kratos/kratos/v2/transport"
	khttp "github.com/go-kratos/kratos/v2/transport/http"

	"github.com/tx7do/go-utils/id"
	"github.com/tx7do/go-utils/trans"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

// ssoStateCookieName state 浏览器绑定 cookie 名（login CSRF 防线）。
const (
	ssoStateCookieName = "sso_state"
	ssoStateCookiePath = "/admin/v1"
)

// setSsoStateCookie 把 state 以 HttpOnly cookie 绑定到发起浏览器。
// SameSite=Lax + 短时效（10 分钟足够走完 IdP 授权跳转）；REST 入口才种
// （非 HTTP 传输无 cookie 通道）。Secure 由 TLS 形态自适应。
func setSsoStateCookie(ctx context.Context, state string) {
	tr, ok := ktransport.FromServerContext(ctx)
	if !ok {
		return
	}
	htr, hok := tr.(*khttp.Transport)
	if !hok || htr.Response() == nil {
		return
	}
	secure := false
	if htr.Request() != nil && htr.Request().TLS != nil {
		secure = true
	}
	http.SetCookie(htr.Response(), &http.Cookie{
		Name:     ssoStateCookieName,
		Value:    state,
		Path:     ssoStateCookiePath,
		MaxAge:   600,
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// ==== OIDC SSO 配置（环境变量，opt-in）====

const (
	ssoEnvIssuer       = "SSO_OIDC_ISSUER" // 例：https://idp.example.com（不带路径，discovery 拼 /.well-known/...）
	ssoEnvClientID     = "SSO_OIDC_CLIENT_ID"
	ssoEnvClientSecret = "SSO_OIDC_CLIENT_SECRET"
	ssoEnvRedirectURL  = "SSO_OIDC_REDIRECT_URL" // IdP 回调地址（指向前端 /auth/sso/callback）
	ssoEnvScopes       = "SSO_OIDC_SCOPES"       // 默认 "openid email profile"
	ssoEnvAutoCreate   = "SSO_AUTO_CREATE"       // "true"/"1" 时按邮箱自动预置用户
	ssoEnvTenantCode   = "SSO_TENANT_CODE"       // SSO 用户归属租户（空 = 平台 0）

	ssoDiscoveryTTL   = time.Hour
	ssoDefaultScopes  = "openid email profile"
	ssoStateTTLClient = 10 * time.Minute
)

// ssoOIDCConfig 环境变量解析后的 SSO 配置；Enabled 齐备即开启。
type ssoOIDCConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       string
	AutoCreate   bool
	TenantCode   string
}

func (c *ssoOIDCConfig) Enabled() bool {
	return c.Issuer != "" && c.ClientID != "" && c.ClientSecret != "" && c.RedirectURL != ""
}

// ssoOIDCDiscovery OIDC discovery 文档中本流程用到的三个端点。
type ssoOIDCDiscovery struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
}

// ssoDiscoveryCache 进程内 discovery 缓存（issuer → 文档 + 过期时间）。
// 单例内存缓存即可：discovery 指纹变化极罕见，过期后下次登录重拉。
var (
	ssoDiscoveryMu     sync.Mutex
	ssoDiscoveryCache  = map[string]ssoOIDCDiscovery{}
	ssoDiscoveryExpiry = map[string]time.Time{}
)

// ssoHTTPClient OIDC 出站 HTTP 客户端；测试注入 httptest 假 IdP。
var ssoHTTPClient = &http.Client{Timeout: 15 * time.Second}

// ssoConfigFromEnv 读取当前 SSO 配置。
func ssoConfigFromEnv() ssoOIDCConfig {
	return ssoOIDCConfig{
		Issuer:       strings.TrimSpace(os.Getenv(ssoEnvIssuer)),
		ClientID:     strings.TrimSpace(os.Getenv(ssoEnvClientID)),
		ClientSecret: strings.TrimSpace(os.Getenv(ssoEnvClientSecret)),
		RedirectURL:  strings.TrimSpace(os.Getenv(ssoEnvRedirectURL)),
		Scopes:       strings.TrimSpace(firstNonEmpty(os.Getenv(ssoEnvScopes), ssoDefaultScopes)),
		AutoCreate:   isTruthyEnv(os.Getenv(ssoEnvAutoCreate)),
		TenantCode:   strings.TrimSpace(os.Getenv(ssoEnvTenantCode)),
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func isTruthyEnv(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// ssoFetchDiscovery 拉（带缓存）OIDC discovery 文档。
func (s *AuthenticationService) ssoFetchDiscovery(ctx context.Context, issuer string) (*ssoOIDCDiscovery, error) {
	ssoDiscoveryMu.Lock()
	if exp, ok := ssoDiscoveryExpiry[issuer]; ok && time.Now().Before(exp) {
		doc := ssoDiscoveryCache[issuer]
		ssoDiscoveryMu.Unlock()
		return &doc, nil
	}
	ssoDiscoveryMu.Unlock()

	wellKnown := strings.TrimSuffix(issuer, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wellKnown, nil)
	if err != nil {
		return nil, fmt.Errorf("build discovery request failed: %w", err)
	}
	resp, err := ssoHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch discovery failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("discovery returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var doc ssoOIDCDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode discovery failed: %w", err)
	}
	if doc.AuthorizationEndpoint == "" || doc.TokenEndpoint == "" || doc.UserinfoEndpoint == "" {
		return nil, fmt.Errorf("discovery document missing endpoints")
	}

	ssoDiscoveryMu.Lock()
	ssoDiscoveryCache[issuer] = doc
	ssoDiscoveryExpiry[issuer] = time.Now().Add(ssoDiscoveryTTL)
	ssoDiscoveryMu.Unlock()

	return &doc, nil
}

// GetSsoLoginInfo SSO 能力开关（登录页按钮显隐）。
func (s *AuthenticationService) GetSsoLoginInfo(ctx context.Context, _ *authenticationV1.GetSsoLoginInfoRequest) (*authenticationV1.GetSsoLoginInfoResponse, error) {
	cfg := ssoConfigFromEnv()
	return &authenticationV1.GetSsoLoginInfoResponse{Enabled: cfg.Enabled()}, nil
}

// GetSsoLoginUrl 生成 OIDC 授权跳转 URL，state 落 Redis（10 分钟单次有效）。
func (s *AuthenticationService) GetSsoLoginUrl(ctx context.Context, _ *authenticationV1.GetSsoLoginUrlRequest) (*authenticationV1.GetSsoLoginUrlResponse, error) {
	cfg := ssoConfigFromEnv()
	if !cfg.Enabled() {
		return nil, authenticationV1.ErrorBadRequest("sso is not configured")
	}

	discovery, err := s.ssoFetchDiscovery(ctx, cfg.Issuer)
	if err != nil {
		s.log.Errorf(ctx, "sso login url: discovery failed: %s", err.Error())
		return nil, authenticationV1.ErrorInternalServerError("fetch sso discovery failed")
	}

	state := id.NewGUIDv4(false)
	if s.ssoStateCache == nil {
		return nil, authenticationV1.ErrorInternalServerError("sso state cache is not wired")
	}
	if err := s.ssoStateCache.Put(ctx, state); err != nil {
		return nil, authenticationV1.ErrorInternalServerError("store sso state failed")
	}
	// login CSRF 防线（P2 修复）：state 以 HttpOnly cookie 绑定到发起浏览器，
	// 回调（SsoLogin）时双向比对——仅持有 state 值的攻击者无法替受害者完成登录。
	setSsoStateCookie(ctx, state)

	authURL := fmt.Sprintf("%s?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&state=%s",
		discovery.AuthorizationEndpoint,
		url.QueryEscape(cfg.ClientID),
		url.QueryEscape(cfg.RedirectURL),
		url.QueryEscape(cfg.Scopes),
		url.QueryEscape(state),
	)

	return &authenticationV1.GetSsoLoginUrlResponse{
		AuthorizationUrl: authURL,
		State:            state,
	}, nil
}

// SsoLogin OIDC 回调换本系统令牌。
//
// 流程：state 验证取删 → code 换 access_token → userinfo 取 email →
// 按邮箱定位用户（未找到且 SSO_AUTO_CREATE 时自动预置）→ resolveUserAuthority →
// 签发 JWT（refresh token 走 HttpOnly Cookie，与密码登录同形）。
// V1 边界：SSO 登录不触发本系统 TOTP MFA 闸门（认证已在 IdP 完成），
// 也不走密码登录的限流/登录策略（无口令攻击面）。
func (s *AuthenticationService) SsoLogin(ctx context.Context, req *authenticationV1.SsoLoginRequest) (*authenticationV1.LoginResponse, error) {
	cfg := ssoConfigFromEnv()
	if !cfg.Enabled() {
		return nil, authenticationV1.ErrorBadRequest("sso is not configured")
	}
	if req.GetCode() == "" || req.GetState() == "" {
		return nil, authenticationV1.ErrorBadRequest("code and state are required")
	}
	if s.ssoStateCache == nil {
		return nil, authenticationV1.ErrorInternalServerError("sso state cache is not wired")
	}

	// 与 doGrantTypePassword 同款：登录链路的用户查询需要绕过隐私层
	//（此时无 viewer，TenantPrivacy 会拒绝查询）。SSO 是登录入口，同权。
	ctx = s.resetContextForLogin(ctx)

	// 1. state 浏览器绑定校验（P2 修复，login CSRF 防线）：
	// REST 入口强制比对 cookie 与 query 中的 state——攻击者诱导受害者回调自己
	// 的 state+code 时，受害者浏览器没有对应 cookie，即 403。
	// 非 HTTP 传输（内部调用/单测）无 cookie 通道，跳过。
	if tr, tok := ktransport.FromServerContext(ctx); tok {
		if htr, hok := tr.(*khttp.Transport); hok {
			cookie, cerr := htr.Request().Cookie(ssoStateCookieName)
			if cerr != nil || cookie.Value != req.GetState() {
				s.log.Warnf(ctx, "sso login: state browser binding mismatch")
				return nil, authenticationV1.ErrorForbidden("sso state browser binding mismatch")
			}
			// 用后即清（一次浏览器绑定只服务一次回调）
			http.SetCookie(htr.Response(), &http.Cookie{
				Name:     ssoStateCookieName,
				Value:    "",
				Path:     ssoStateCookiePath,
				MaxAge:   -1,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
		}
	}

	// 2. state 单次验证取删（防 CSRF 与回调重放）
	ok, err := s.ssoStateCache.Take(ctx, req.GetState())
	if err != nil {
		return nil, authenticationV1.ErrorInternalServerError("validate sso state failed")
	}
	if !ok {
		return nil, authenticationV1.ErrorUnauthorized("sso state invalid or expired")
	}

	discovery, err := s.ssoFetchDiscovery(ctx, cfg.Issuer)
	if err != nil {
		s.log.Errorf(ctx, "sso login: discovery failed: %s", err.Error())
		return nil, authenticationV1.ErrorInternalServerError("fetch sso discovery failed")
	}

	// 2. code 换 access_token
	accessToken, err := s.ssoExchangeCode(ctx, discovery.TokenEndpoint, cfg, req.GetCode())
	if err != nil {
		s.log.Errorf(ctx, "sso login: exchange code failed: %s", err.Error())
		return nil, authenticationV1.ErrorUnauthorized("sso code exchange failed")
	}

	// 3. userinfo 取身份（email 为映射锚）
	claims, err := s.ssoFetchUserinfo(ctx, discovery.UserinfoEndpoint, accessToken)
	if err != nil {
		s.log.Errorf(ctx, "sso login: fetch userinfo failed: %s", err.Error())
		return nil, authenticationV1.ErrorUnauthorized("sso userinfo failed")
	}
	email := strings.TrimSpace(claims["email"])
	if email == "" {
		return nil, authenticationV1.ErrorUnauthorized("sso userinfo has no email claim")
	}

	// P2 修复：email_verified 校验——email 是本地账号映射的锚（含 AutoCreate
	// 自动建号）。IdP 侧允许用户自填且不强制验证邮箱时（不少企业 IdP 的默认
	// 策略、或自建 Keycloak 域配置疏漏），攻击者把 IdP 账号邮箱改成受害者邮箱
	// 即可登录成本地对应账号（含平台管理员，若其邮箱被填入 IdP）。
	// 口径：claim 存在且为 false → 拒绝；不存在 → 放行但告警（兼容不发该
	// claim 的 IdP，运维应改配 IdP 强制验证邮箱）。
	if ev, ok := claims["email_verified"]; ok {
		switch strings.ToLower(strings.TrimSpace(ev)) {
		case "true", "1", "yes":
			// 已验证：放行
		default:
			s.log.Warnf(ctx, "sso login: email [%s] not verified at IdP (email_verified=%q), rejected", email, ev)
			return nil, authenticationV1.ErrorUnauthorized("sso email not verified")
		}
	} else {
		s.log.Warnf(ctx, "sso login: IdP userinfo has no email_verified claim for [%s]; configure the IdP to verify emails", email)
	}

	// 4. 定位/预置用户
	tenantID := uint64(0)
	if cfg.TenantCode != "" {
		tenant, terr := s.tenantRepo.Get(ctx, &identityV1.GetTenantRequest{
			QueryBy: &identityV1.GetTenantRequest_Code{Code: cfg.TenantCode},
		})
		if terr != nil || tenant == nil || tenant.GetId() == 0 {
			s.log.Errorf(ctx, "sso login: tenant code [%s] not found", cfg.TenantCode)
			return nil, authenticationV1.ErrorInternalServerError("sso tenant misconfigured")
		}
		tenantID = tenant.GetId()
	}

	username, userID, ferr := s.userRepo.FindUsernameByIdentifier(ctx, tenantID, email)
	if ferr != nil {
		s.log.Errorf(ctx, "sso login: find user by email failed: %s", ferr.Error())
		return nil, authenticationV1.ErrorInternalServerError("find user failed")
	}
	if userID == 0 {
		if !cfg.AutoCreate {
			return nil, authenticationV1.ErrorForbidden("no local account matches the sso identity; ask an administrator to create one")
		}
		created, cerr := s.ssoAutoCreateUser(ctx, tenantID, email)
		if cerr != nil {
			s.log.Errorf(ctx, "sso login: auto create user failed: %s", cerr.Error())
			return nil, authenticationV1.ErrorInternalServerError("auto create user failed")
		}
		userID = created.GetId()
		username = created.GetUsername()
	}

	user, uerr := s.userRepo.Get(ctx, &identityV1.GetUserRequest{
		QueryBy: &identityV1.GetUserRequest_Id{Id: userID},
	})
	if uerr != nil || user == nil {
		return nil, authenticationV1.ErrorUnauthorized("sso user not found")
	}
	if sso := user.GetStatus(); sso != identityV1.User_NORMAL {
		return nil, authenticationV1.ErrorForbidden("account is frozen")
	}

	// 5. 权限解析 + 签发（与密码登录同一套）
	tokenPayload := &authenticationV1.UserTokenPayload{
		UserId:   user.GetId(),
		TenantId: user.TenantId,
		Username: user.Username,
		ClientId: trans.Ptr("sso"),
	}
	if err := s.resolveUserAuthority(ctx, user, tokenPayload); err != nil {
		s.log.Errorf(ctx, "sso login: resolve authority failed for user [%d]: %s", user.GetId(), err.Error())
		return nil, err
	}

	accessToken, refreshToken, terr := s.authenticator.CreateUserToken(ctx, authenticationV1.ClientType_admin, tokenPayload)
	if terr != nil {
		return nil, terr
	}

	recordSessionMeta(ctx, s.log, s.authenticator, authenticationV1.ClientType_admin, tokenPayload)
	recordUserLastLogin(ctx, s.log, s.userRepo, user.GetId(), "")

	refreshExpiresIn := int64(s.authenticator.GetRefreshTokenExpires(authenticationV1.ClientType_admin).Seconds())
	setRefreshCookies(ctx, refreshToken, refreshExpiresIn)

	s.log.Infof(ctx, "sso login: user [%s] (id=%d) logged in via oidc (%s)", username, user.GetId(), cfg.Issuer)

	return &authenticationV1.LoginResponse{
		TokenType:   authenticationV1.TokenType_bearer,
		AccessToken: accessToken,
		ExpiresIn:   int64(s.authenticator.GetAccessTokenExpires(authenticationV1.ClientType_admin).Seconds()),
	}, nil
}

// ssoExchangeCode 在 IdP token 端点用授权码换 access_token（client_secret_post）。
func (s *AuthenticationService) ssoExchangeCode(ctx context.Context, tokenEndpoint string, cfg ssoOIDCConfig, code string) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", cfg.RedirectURL)
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint,
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := ssoHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("decode token response failed: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("token response has no access_token")
	}
	return tokenResp.AccessToken, nil
}

// ssoFetchUserinfo 携 access_token 调 userinfo 端点，返回 claims。
func (s *AuthenticationService) ssoFetchUserinfo(ctx context.Context, userinfoEndpoint, accessToken string) (map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userinfoEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := ssoHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode userinfo failed: %w", err)
	}
	claims := make(map[string]string, len(raw))
	for k, v := range raw {
		switch tv := v.(type) {
		case string:
			claims[k] = tv
		case float64:
			claims[k] = fmt.Sprintf("%v", tv)
		case bool:
			claims[k] = fmt.Sprintf("%v", tv)
		default:
			// 嵌套结构（如 email_verified 之外的复杂 claim）不进平铺映射
		}
	}
	return claims, nil
}

// ssoAutoCreateUser 按邮箱预置用户：username 取邮箱本地部分，无密码凭证
// （管理员设置密码前该账号无法密码登录），状态 ON。
func (s *AuthenticationService) ssoAutoCreateUser(ctx context.Context, tenantID uint64, email string) (*identityV1.User, error) {
	local := email
	if at := strings.Index(local, "@"); at > 0 {
		local = local[:at]
	}
	username := fmt.Sprintf("sso_%s_%s", local, id.NewGUIDv4(false)[:8])

	return s.userRepo.Create(ctx, &identityV1.CreateUserRequest{
		Data: &identityV1.User{
			Username: trans.Ptr(username),
			Email:    trans.Ptr(email),
			TenantId: trans.Ptr(tenantID),
			Status:   identityV1.User_NORMAL.Enum(),
		},
	})
}
