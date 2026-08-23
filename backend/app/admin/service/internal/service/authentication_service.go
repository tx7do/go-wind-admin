package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/go-crud/viewer"
	"github.com/tx7do/go-utils/captcha"
	"github.com/tx7do/go-utils/crypto"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent/privacy"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	ktransport "github.com/go-kratos/kratos/v2/transport"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"

	"go-wind-admin/pkg/constants"
	"go-wind-admin/pkg/middleware/auth"
	"go-wind-admin/pkg/netutil"
)

// 验证码相关请求头（H5：登录强制验证码，通过 header 传递以避免改动 proto 与三套前端生成代码）。
const (
	headerCaptchaID    = "X-Captcha-Id"
	headerCaptchaValue = "X-Captcha-Value"
)

// CaptchaEnabled 控制登录是否强制校验验证码。
// 开发/无 Redis 等环境可改为 false 跳过验证码校验，避免登录被 400 invalid or missing captcha 阻断。
const CaptchaEnabled = true

// refresh token cookie 相关常量。
// refresh token 以 HttpOnly Cookie 传输，Path 收窄到刷新端点，SameSite=Lax 阻断跨站 POST。
// refresh_exp 为非 HttpOnly 的过期时间戳 cookie，供前端定时器读取以调度主动刷新。
const (
	refreshTokenCookieName = "refresh_token"
	refreshExpCookieName   = "refresh_exp"
	refreshCookiePath      = "/admin/v1/refresh-token"
)

// setRefreshCookies 将 refresh token 及其过期时间戳写入 Set-Cookie 响应头。
// refresh_token cookie 为 HttpOnly（JS 不可读），refresh_exp cookie 非 HttpOnly（前端定时器可读）。
// 两者均 Secure + SameSite=Lax + Path 收窄到刷新端点。
func setRefreshCookies(ctx context.Context, refreshToken string, refreshExpiresInSeconds int64) {
	tr, ok := ktransport.FromServerContext(ctx)
	if !ok {
		return
	}
	htr, hok := tr.(*khttp.Transport)
	if !hok {
		return
	}
	header := htr.ReplyHeader()

	// HttpOnly refresh token cookie
	rtCookie := &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    refreshToken,
		Path:     refreshCookiePath,
		MaxAge:   int(refreshExpiresInSeconds),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	header.Add("Set-Cookie", rtCookie.String())

	// 非 HttpOnly 过期时间戳 cookie（只含 Unix 秒，无敏感信息）
	expCookie := &http.Cookie{
		Name:     refreshExpCookieName,
		Value:    fmt.Sprintf("%d", time.Now().Unix()+refreshExpiresInSeconds),
		Path:     refreshCookiePath,
		MaxAge:   int(refreshExpiresInSeconds),
		Secure:   true,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	}
	header.Add("Set-Cookie", expCookie.String())
}

// clearRefreshCookies 向响应头写入 Max-Age=0 的清除 cookie，使浏览器立即删除 refresh token 相关 cookie。
func clearRefreshCookies(ctx context.Context) {
	tr, ok := ktransport.FromServerContext(ctx)
	if !ok {
		return
	}
	htr, hok := tr.(*khttp.Transport)
	if !hok {
		return
	}
	header := htr.ReplyHeader()

	for _, name := range []string{refreshTokenCookieName, refreshExpCookieName} {
		cookie := &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     refreshCookiePath,
			MaxAge:   -1,
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		}
		header.Add("Set-Cookie", cookie.String())
	}
}

// normalizeLoginVerifyError 将登录凭证校验的多种细分错误统一对外成 INVALID_PASSWORD，
// 防止攻击者通过区分"用户不存在(404)/账号冻结(401)/密码错误(400)"来枚举有效用户名。
// 真实原因仍保留在服务端日志与审计中间件的 FailureReason 中，不影响可观测性。
func normalizeLoginVerifyError(err error) error {
	switch {
	case authenticationV1.IsUserNotFound(err),
		authenticationV1.IsUserFreeze(err),
		authenticationV1.IsInvalidPassword(err):
		return authenticationV1.ErrorInvalidPassword("invalid username or password")
	default:
		return err
	}
}

type AuthenticationService struct {
	adminV1.AuthenticationServiceHTTPServer

	log *log.Helper

	userRepo           data.UserRepo
	userCredentialRepo *data.UserCredentialRepo

	roleRepo       *data.RoleRepo
	tenantRepo     *data.TenantRepo
	membershipRepo *data.MembershipRepo
	orgUnitRepo    *data.OrgUnitRepo
	permissionRepo *data.PermissionRepo

	authenticator *data.Authenticator
	clientType    authenticationV1.ClientType

	captchaClient *captcha.Captcha
	rateLimiter   *data.LoginRateLimiter

	loginPolicyRepo *data.LoginPolicyRepo

	mfaFactorRepo     *data.UserMfaFactorRepo
	mfaChallengeCache *data.MfaChallengeCache
}

func NewAuthenticationService(
	ctx *bootstrap.Context,
	userRepo data.UserRepo,
	userCredentialRepo *data.UserCredentialRepo,
	roleRepo *data.RoleRepo,
	tenantRepo *data.TenantRepo,
	membershipRepo *data.MembershipRepo,
	orgUnitRepo *data.OrgUnitRepo,
	permissionRepo *data.PermissionRepo,
	authenticator *data.Authenticator,
	clientType authenticationV1.ClientType,
	captchaClient *captcha.Captcha,
	rateLimiter *data.LoginRateLimiter,
	loginPolicyRepo *data.LoginPolicyRepo,
	mfaFactorRepo *data.UserMfaFactorRepo,
	mfaChallengeCache *data.MfaChallengeCache,
) *AuthenticationService {
	return &AuthenticationService{
		log:                ctx.NewLoggerHelper("authn/service/admin-service"),
		userRepo:           userRepo,
		userCredentialRepo: userCredentialRepo,
		tenantRepo:         tenantRepo,
		roleRepo:           roleRepo,
		membershipRepo:     membershipRepo,
		orgUnitRepo:        orgUnitRepo,
		permissionRepo:     permissionRepo,
		authenticator:      authenticator,
		clientType:         clientType,
		captchaClient:      captchaClient,
		rateLimiter:        rateLimiter,
		loginPolicyRepo:    loginPolicyRepo,
		mfaFactorRepo:      mfaFactorRepo,
		mfaChallengeCache:  mfaChallengeCache,
	}
}

// checkLoginPolicies 拉取租户登录策略并按当前上下文匹配。
// userId 传 0 时只匹配全局条目（target_id 为空）；密码校验前与取到 user 后各调用一次。
// 匹配逻辑见 data.MatchLoginPolicy（纯函数，含单测）。
// 策略查询失败时 fail-open（仅告警）——登录可用性优先于策略拦截，与验证码开关的容错取向一致。
func (s *AuthenticationService) checkLoginPolicies(ctx context.Context, tenantID, userId uint32, clientIP, deviceId string) (bool, string) {
	policies, err := s.loginPolicyRepo.ListForLogin(ctx, tenantID)
	if err != nil {
		s.log.Errorf("list login policies failed for tenant [%d]: %s", tenantID, err.Error())
		return false, ""
	}
	return data.MatchLoginPolicy(policies, userId, clientIP, deviceId, time.Now())
}

func (s *AuthenticationService) resetContextForLogin(ctx context.Context) context.Context {
	// 没有 viewer 信息，使用空的 NoopContext
	ctx = viewer.WithContext(ctx, viewer.NewNoopContext())
	// 绕过隐私保护中间件
	ctx = privacy.DecisionContext(ctx, privacy.Allow)

	return ctx
}

// Login 登录
func (s *AuthenticationService) Login(ctx context.Context, req *authenticationV1.LoginRequest) (*authenticationV1.LoginResponse, error) {
	switch req.GetGrantType() {
	case authenticationV1.GrantType_password:
		return s.doGrantTypePassword(ctx, req)

	case authenticationV1.GrantType_refresh_token:
		// refresh token 刷新已迁移到 /admin/v1/refresh-token 端点（HttpOnly Cookie 传输），
		// login 端点不再处理 refresh_token grant type。
		return nil, authenticationV1.ErrorInvalidGrantType("use /admin/v1/refresh-token for token refresh")

	case authenticationV1.GrantType_client_credentials:
		return s.doGrantTypeClientCredentials(ctx, req)

	default:
		return nil, authenticationV1.ErrorInvalidGrantType("invalid grant type")
	}
}
// containsPermission 检查权限代码列表中是否包含指定权限代码
func containsPermission(perms []string, target string) bool {
	for _, p := range perms {
		if p == target {
			return true
		}
	}
	return false
}

// authorizeAndEnrichUserTokenPayloadUserTenantRelationOneToOne 一对一用户-租户关系的授权与丰富
func (s *AuthenticationService) authorizeAndEnrichUserTokenPayloadUserTenantRelationOneToOne(ctx context.Context, userID, tenantID uint32, tokenPayload *authenticationV1.UserTokenPayload) error {
	hasBackendAccess := false

	if tenantID > 0 {
		// 检查租户状态
		tenant, _ := s.tenantRepo.Get(ctx, &identityV1.GetTenantRequest{
			QueryBy: &identityV1.GetTenantRequest_Id{Id: tenantID},
		})
		if tenant == nil || tenant.GetStatus() != identityV1.Tenant_ON {
			return authenticationV1.ErrorForbidden("insufficient authority")
		}
	}

	// 获取角色 ID 列表
	roleIDs, err := s.userRepo.ListRoleIDsByUserID(ctx, userID)
	if err != nil || len(roleIDs) == 0 {
		s.log.Errorf("get roles by user [%d] failed [%v]", userID, err)
		return authenticationV1.ErrorForbidden("insufficient authority")
	}

	// 获取权限 ID 列表
	permissionIDs, err := s.roleRepo.ListPermissionIDsByRoleIDs(ctx, roleIDs)
	if err != nil || len(permissionIDs) == 0 {
		s.log.Errorf("get permissions by role ids failed [%v]", err)
		return authenticationV1.ErrorForbidden("insufficient authority")
	}

	// 获取权限代码列表
	permissionCodes, err := s.permissionRepo.GetPermissionCodesByIDs(ctx, permissionIDs)
	if err != nil || len(permissionCodes) == 0 {
		s.log.Errorf("get permission codes by ids failed [%v]", err)
		return authenticationV1.ErrorForbidden("insufficient authority")
	}

	// 检查是否包含系统访问后台权限
	if containsPermission(permissionCodes, constants.SystemAccessBackendPermissionCode) {
		hasBackendAccess = true
	}

	// 授权决策
	if !hasBackendAccess {
		s.log.Errorf("user [%d] has no backend access permission", userID)
		return authenticationV1.ErrorForbidden("insufficient authority")
	}

	// 获取角色代码列表
	roleCodes, err := s.roleRepo.ListRoleCodesByRoleIds(ctx, roleIDs)
	if err != nil || len(roleCodes) == 0 {
		s.log.Errorf("list role codes by role ids failed [%v]", err)
		return authenticationV1.ErrorForbidden("insufficient authority")
	}
	tokenPayload.Roles = roleCodes
	fillAdminFlags(tokenPayload, roleCodes)

	return nil
}

// authorizeAndEnrichUserTokenPayloadUserTenantRelationOneToMany 一对多用户-租户关系的授权与丰富
func (s *AuthenticationService) authorizeAndEnrichUserTokenPayloadUserTenantRelationOneToMany(ctx context.Context, userID, tenantID uint32, tokenPayload *authenticationV1.UserTokenPayload) error {
	var memberships []*identityV1.Membership
	if tenantID > 0 {
		// 指定租户
		membership, err := s.membershipRepo.GetMembershipByUserTenant(ctx, userID, tenantID)
		if err != nil {
			s.log.Errorf("get user [%d] membership for tenant [%d] failed [%s]", userID, tenantID, err.Error())
			return authenticationV1.ErrorForbidden("insufficient authority")
		}
		memberships = []*identityV1.Membership{membership}
	} else {
		var err error
		// 获取所有活跃成员身份
		memberships, err = s.membershipRepo.GetUserActiveMemberships(ctx, userID)
		if err != nil || len(memberships) == 0 {
			s.log.Errorf("list user [%d] active memberships failed [%v]", userID, err)
			return authenticationV1.ErrorForbidden("insufficient authority")
		}
	}

	hasBackendAccess := false
	var validRoleIDs []uint32
	for _, m := range memberships {
		if m.GetTenantId() > 0 {
			// 检查租户状态
			tenant, _ := s.tenantRepo.Get(ctx, &identityV1.GetTenantRequest{
				QueryBy: &identityV1.GetTenantRequest_Id{Id: m.GetTenantId()},
			})
			if tenant == nil || tenant.GetStatus() != identityV1.Tenant_ON {
				continue
			}
		}

		// 获取角色 ID 列表
		roleIDs, err := s.membershipRepo.GetRoleIDsByMembership(ctx, m.GetId())
		if err != nil || len(roleIDs) == 0 {
			s.log.Errorf("get roles by membership [%d] failed [%v]", m.GetId(), err)
			continue
		}

		// 获取权限 ID 列表
		permissionIDs, err := s.roleRepo.ListPermissionIDsByRoleIDs(ctx, roleIDs)
		if err != nil || len(permissionIDs) == 0 {
			s.log.Errorf("get permissions by role ids failed [%v]", err)
			continue
		}

		// 获取权限代码列表
		permissionCodes, _ := s.permissionRepo.GetPermissionCodesByIDs(ctx, permissionIDs)

		s.log.Infof("user [%d] membership [%d] permission codes: %v", userID, m.GetId(), permissionCodes)

		// 检查是否包含系统访问后台权限
		if containsPermission(permissionCodes, constants.SystemAccessBackendPermissionCode) {
			hasBackendAccess = true
			validRoleIDs = append(validRoleIDs, roleIDs...)
		}
	}

	// 授权决策
	if !hasBackendAccess {
		s.log.Errorf("user [%d] has no backend access permission", userID)
		return authenticationV1.ErrorForbidden("insufficient authority")
	}

	// 获取角色代码列表
	roleCodes, err := s.roleRepo.ListRoleCodesByRoleIds(ctx, validRoleIDs)
	if err != nil || len(roleCodes) == 0 {
		s.log.Errorf("list role codes by role ids failed [%v]", err)
		return authenticationV1.ErrorForbidden("insufficient authority")
	}
	tokenPayload.Roles = roleCodes
fillAdminFlags(tokenPayload, roleCodes)

	return nil
}

// authorizeAndEnrichUserTokenPayload 授权并丰富用户令牌载荷
func (s *AuthenticationService) authorizeAndEnrichUserTokenPayload(ctx context.Context, userID, tenantID uint32, tokenPayload *authenticationV1.UserTokenPayload) error {
	switch constants.DefaultUserTenantRelationType {
	case constants.UserTenantRelationOneToOne:
		return s.authorizeAndEnrichUserTokenPayloadUserTenantRelationOneToOne(ctx, userID, tenantID, tokenPayload)

	case constants.UserTenantRelationOneToMany:
		return s.authorizeAndEnrichUserTokenPayloadUserTenantRelationOneToMany(ctx, userID, tenantID, tokenPayload)

	default:
		s.log.Errorf("unsupported user-tenant relation type: %d", constants.DefaultUserTenantRelationType)
		return authenticationV1.ErrorServiceUnavailable("unsupported user-tenant relation type")
	}
}

// resolveUserAuthority 解析用户权限信息
func (s *AuthenticationService) resolveUserAuthority(ctx context.Context, user *identityV1.User, tokenPayload *authenticationV1.UserTokenPayload) error {
	if user.GetStatus() != identityV1.User_NORMAL {
		s.log.Errorf("user [%d] is [%v]", user.GetId(), user.GetStatus())
		return authenticationV1.ErrorForbidden("user is disabled")
	}

	if err := s.authorizeAndEnrichUserTokenPayload(ctx, user.GetId(), user.GetTenantId(), tokenPayload); err != nil {
		return err
	}

	return nil
}

// doGrantTypePassword 处理授权类型 - 密码
func (s *AuthenticationService) doGrantTypePassword(ctx context.Context, req *authenticationV1.LoginRequest) (*authenticationV1.LoginResponse, error) {
	ctx = s.resetContextForLogin(ctx)

	// 取客户端 IP（供限流维度使用），失败不阻断登录
	clientIP := netutil.ClientIPFromContext(ctx)
	username := req.GetUsername()

	// ===== H5 闸门 1：登录限流预检（按 IP + 用户名双维度）=====
	if s.rateLimiter != nil {
		if locked, lerr := s.rateLimiter.IsLocked(ctx, clientIP, username); lerr != nil {
			s.log.Errorf("login rate limiter pre-check failed: %s", lerr.Error())
		} else if locked {
			s.log.Warnf("login blocked by rate limiter: ip=%s, username=%s", clientIP, username)
			return nil, authenticationV1.ErrorBadRequest("too many login failures, please try again later")
		}
	}

	// ===== H5 闸门 2：强制验证码（始终启用；通过 HTTP Header 传递，避免改动 proto/前端生成代码）=====
	if !s.verifyLoginCaptcha(ctx) {
		return nil, authenticationV1.ErrorBadRequest("invalid or missing captcha")
	}

	// ===== 租户解析：tenant_code 留空视为平台（tenant 0），非空则按编号定位租户 =====
	// 解析后的 tenantID 限定后续凭证查询范围，消除同名 identifier 跨租户歧义。
	var tenantID uint32 = 0
	if code := req.GetTenantCode(); strings.TrimSpace(code) != "" {
		tenant, _ := s.tenantRepo.Get(ctx, &identityV1.GetTenantRequest{
			QueryBy: &identityV1.GetTenantRequest_Code{Code: code},
		})
		// 查不到、或租户非启用状态，统一返回同一文案，防止通过返回差异枚举有效租户编号
		if tenant == nil || tenant.GetStatus() != identityV1.Tenant_ON {
			return nil, authenticationV1.ErrorBadRequest("invalid tenant")
		}
		tenantID = tenant.GetId()
	}

	// ===== 登录策略闸门（全局部分）：target_id 为空的策略不依赖用户身份， =====
	// ===== 在 identifier 反查与密码校验之前拦截，被封锁的 IP 连 user 表查询都省掉。
	// ===== 用户定向策略（target_id = userId）在取到 user 后二次检查。
	if s.loginPolicyRepo != nil {
		if blocked, reason := s.checkLoginPolicies(ctx, tenantID, 0, clientIP, req.GetDeviceId()); blocked {
			s.log.Warnf("login blocked by policy: ip=%s username=%s reason=%s", clientIP, username, reason)
			return nil, authenticationV1.ErrorForbidden("login blocked by security policy")
		}
	}

	// ===== identifier 智能解析：输入含 @ 视为 email、纯数字视为 mobile， =====
	// ===== 经 user 表反查得到真实 username 后仍走 USERNAME 维度凭证校验。
	// 未命中时原样返回，交由凭证校验走统一失败路径（防枚举）；mobile 多行歧义直接拒绝。
	if resolved, _, rerr := s.userRepo.FindUsernameByIdentifier(ctx, tenantID, username); rerr != nil {
		return nil, rerr
	} else if resolved != username {
		username = resolved
	}

	// ===== 凭证校验：在解析出的 tenant 范围内查单条凭证并校验密码 =====
	// 注意用解析后的 username 局部变量（可能是 email/mobile 反查所得），而非 req 原始输入
	var matchedUserID uint32
	var err error
	matchedUserID, err = s.userCredentialRepo.FindUserCredential(ctx, tenantID, authenticationV1.UserCredential_USERNAME, username, req.GetPassword(), true)
	if err != nil {
		// 服务端日志保留真实原因（USER_NOT_FOUND / USER_FREEZE / INVALID_PASSWORD），便于运维排查
		s.log.Errorf("verify user credential failed for username [%s]: %s", username, err.Error())

		// H5：登录失败时自增失败计数（按 IP + 用户名双维度）
		if s.rateLimiter != nil {
			if _, _, _, cerr := s.rateLimiter.CheckAndIncr(ctx, clientIP, username); cerr != nil {
				s.log.Errorf("login rate limiter incr failed: %s", cerr.Error())
			}
		}

		// 对客户端统一返回同一文案，避免通过 HTTP 状态码/reason 枚举有效用户名（H11）
		return nil, normalizeLoginVerifyError(err)
	}

	// 获取用户信息（按凭证归属的 user_id 精确查找，避免同 identifier 多租户歧义）
	var user *identityV1.User
	user, err = s.userRepo.Get(ctx, &identityV1.GetUserRequest{QueryBy: &identityV1.GetUserRequest_Id{Id: matchedUserID}})
	if err != nil {
		s.log.Errorf("get user by id [%d] failed [%s]", matchedUserID, err.Error())
		return nil, err
	}

	// 纵深防御：凭证行的 tenant 必须与用户行的 tenant 一致，否则拒绝登录
	if user.GetTenantId() != tenantID {
		s.log.Errorf("tenant mismatch for user [%d]: credential tenant [%d] vs user tenant [%d]",
			matchedUserID, tenantID, user.GetTenantId())
		return nil, authenticationV1.ErrorBadRequest("invalid tenant")
	}

	// ===== 登录策略闸门（用户定向部分）：密码已通过、userId 已知， =====
	// ===== 检查 target_id 约束到该用户的策略条目。
	if s.loginPolicyRepo != nil {
		if blocked, reason := s.checkLoginPolicies(ctx, tenantID, user.GetId(), clientIP, req.GetDeviceId()); blocked {
			s.log.Warnf("login blocked by user-targeted policy: uid=%d ip=%s reason=%s", user.GetId(), clientIP, reason)
			return nil, authenticationV1.ErrorForbidden("login blocked by security policy")
		}
	}

	tokenPayload := &authenticationV1.UserTokenPayload{
		UserId:   user.GetId(),
		TenantId: user.TenantId,
		Username: user.Username,
		ClientId: req.ClientId,
		DeviceId: req.DeviceId,
	}

	// 解析用户权限信息
	err = s.resolveUserAuthority(ctx, user, tokenPayload)
	if err != nil {
		s.log.Errorf("resolve user [%d] authority failed [%s]", user.GetId(), err.Error())
		return nil, err
	}

	// ===== MFA 闸门：若该用户绑定了 ENABLED 的 TOTP 因子，则不签发 token， =====
	// ===== 改为签发 operation_id，要求前端走二次验证（MfaService.VerifyMFAChallenge）。
	// 注意：此处不清零限流计数——认证尚未完成；真正清零在 VerifyMFAChallenge 通过后。
	if s.mfaFactorRepo != nil {
		needMfa, merr := s.mfaFactorRepo.HasEnabledTotp(ctx, user.GetTenantId(), user.GetId())
		if merr != nil {
			// fail-closed：MFA 状态查询失败时拒绝登录，避免已绑定用户在 DB 故障时
			// 被降级为单因子放行（与上方凭证校验出错即拒登的行为一致）。
			s.log.Errorf("check mfa factor failed for user [%d]: %s", user.GetId(), merr.Error())
			return nil, authenticationV1.ErrorInternalServerError("mfa check failed")
		}
		if needMfa && s.mfaChallengeCache != nil {
			opId, cerr := s.mfaChallengeCache.SetLoginChallenge(ctx, tokenPayload, req.GetClientType())
			if cerr != nil {
				s.log.Errorf("set mfa login challenge failed for user [%d]: %s", user.GetId(), cerr.Error())
				return nil, authenticationV1.ErrorInternalServerError("mfa challenge failed")
			}
			return &authenticationV1.LoginResponse{
				TokenType:      authenticationV1.TokenType_bearer,
				AccessToken:    "",
				MfaOperationId: trans.Ptr(opId),
			}, nil
		}
	}

	// 生成令牌
	accessToken, refreshToken, err := s.authenticator.CreateUserToken(ctx, req.GetClientType(), tokenPayload)
	if err != nil {
		return nil, err
	}

	// H5：登录成功后清零失败计数
	if s.rateLimiter != nil {
		s.rateLimiter.Reset(ctx, clientIP, username)
	}

	// refresh token 通过 HttpOnly Cookie 下发，不再放入响应体
	refreshExpiresIn := int64(s.authenticator.GetRefreshTokenExpires(req.GetClientType()).Seconds())
	setRefreshCookies(ctx, refreshToken, refreshExpiresIn)

	return &authenticationV1.LoginResponse{
		TokenType:   authenticationV1.TokenType_bearer,
		AccessToken: accessToken,
		ExpiresIn:   int64(s.authenticator.GetAccessTokenExpires(req.GetClientType()).Seconds()),
	}, nil
}

// decryptTransportSecret 解密前端 AES-CBC（base64）传输的秘密（注册/改密与登录同规）。
func decryptTransportSecret(secret string) (string, error) {
	bytesPass, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		return "", err
	}
	decrypted, err := crypto.AesDecrypt(bytesPass, crypto.DefaultAESKey, nil)
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
}

// fillAdminFlags 按角色码填充 token 的平台/租户管理员标志。
// 该标志此前从未被赋值（全仓库仅消费无生产），导致下游
// GetIsPlatformAdmin 判定（MFA 重置、用户管理越权校验等）形同虚设。
func fillAdminFlags(tokenPayload *authenticationV1.UserTokenPayload, roleCodes []string) {
	for _, rc := range roleCodes {
		if rc == constants.PlatformAdminRoleCode {
			tokenPayload.IsPlatformAdmin = trans.Ptr(true)
		}
		if rc == constants.TenantAdminRoleCode {
			tokenPayload.IsTenantAdmin = trans.Ptr(true)
		}
	}
}

// verifyLoginCaptcha 校验登录请求携带的验证码。
// 验证码 id/value 通过 HTTP Header（X-Captcha-Id / X-Captcha-Value）传递。
// captchaClient.Verify 已是 verify-and-delete 单次有效语义。
// 注意：refresh_token / client_credentials 等非密码授权不走此校验（仅 doGrantTypePassword 调用）。
func (s *AuthenticationService) verifyLoginCaptcha(ctx context.Context) bool {
	if !CaptchaEnabled {
		// 验证码开关关闭，跳过校验
		return true
	}
	if s.captchaClient == nil {
		// captcha 未配置时 fail-open（仅记录告警），避免影响登录基本功能
		return true
	}
	header := netutil.HeaderFromContext(ctx)
	if header == nil {
		return false
	}
	captchaID := strings.TrimSpace(header.Get(headerCaptchaID))
	captchaValue := strings.TrimSpace(header.Get(headerCaptchaValue))
	if captchaID == "" || captchaValue == "" {
		return false
	}
	ok, err := s.captchaClient.Verify(ctx, captchaID, captchaValue)
	if err != nil {
		s.log.Errorf("verify captcha failed: %s", err.Error())
		return false
	}
	return ok
}

// doGrantTypeRefreshToken 处理授权类型 - 刷新令牌
func (s *AuthenticationService) doGrantTypeRefreshToken(ctx context.Context, req *authenticationV1.LoginRequest, refreshToken string) (*authenticationV1.LoginResponse, error) {
	// refresh token 为自描述 JWT，VerifyRefreshToken 验签并原子吊销旧令牌对，返回 uid/jti。
	// 不再依赖 access token 的 auth.FromContext 提供身份信息。
	userId, _, err := s.authenticator.VerifyRefreshToken(ctx, req.GetClientType(), refreshToken)
	if err != nil {
		s.log.Errorf("verify refresh token failed: [%s]", err)
		return nil, authenticationV1.ErrorIncorrectRefreshToken("invalid refresh token")
	}

	// 获取用户信息
	user, err := s.userRepo.Get(ctx, &identityV1.GetUserRequest{
		QueryBy: &identityV1.GetUserRequest_Id{
			Id: userId,
		},
	})
	if err != nil {
		return nil, err
	}

	tokenPayload := &authenticationV1.UserTokenPayload{
		UserId:   user.GetId(),
		TenantId: user.TenantId,
		Username: user.Username,
		ClientId: req.ClientId,
		DeviceId: req.DeviceId,
	}

	// 解析用户权限信息
	err = s.resolveUserAuthority(ctx, user, tokenPayload)
	if err != nil {
		s.log.Errorf("resolve user [%d] authority failed [%s]", user.GetId(), err.Error())
		return nil, err
	}

	// 生成令牌
	accessToken, newRefreshToken, err := s.authenticator.CreateUserToken(ctx, req.GetClientType(), tokenPayload)
	if err != nil {
		return nil, err
	}

	// refresh token 通过 HttpOnly Cookie 下发，不再放入响应体
	refreshExpiresIn := int64(s.authenticator.GetRefreshTokenExpires(req.GetClientType()).Seconds())
	setRefreshCookies(ctx, newRefreshToken, refreshExpiresIn)

	return &authenticationV1.LoginResponse{
		TokenType:   authenticationV1.TokenType_bearer,
		AccessToken: accessToken,
		ExpiresIn:   int64(s.authenticator.GetAccessTokenExpires(req.GetClientType()).Seconds()),
	}, nil
}

// doGrantTypeClientCredentials 处理授权类型 - 客户端凭据
func (s *AuthenticationService) doGrantTypeClientCredentials(_ context.Context, _ *authenticationV1.LoginRequest) (*authenticationV1.LoginResponse, error) {
	return nil, authenticationV1.ErrorInvalidGrantType("invalid grant type")
}

// Logout 登出
func (s *AuthenticationService) Logout(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err = s.authenticator.RevokeUserToken(ctx, s.clientType, operator.GetUserId()); err != nil {
		return nil, err
	}

	// 清除 refresh token 相关 cookie
	clearRefreshCookies(ctx)

	return &emptypb.Empty{}, nil
}

// RefreshToken 刷新令牌
// refresh token 现以 HttpOnly Cookie 传输，本接口已加入白名单（无需 access token）。
// refresh token 为自描述 JWT，VerifyRefreshToken 从中解析 uid/jti 完成独立鉴权。
func (s *AuthenticationService) RefreshToken(ctx context.Context, req *authenticationV1.LoginRequest) (*authenticationV1.LoginResponse, error) {
	// 校验授权类型
	if req.GetGrantType() != authenticationV1.GrantType_refresh_token {
		return nil, authenticationV1.ErrorInvalidGrantType("invalid grant type")
	}

	// refresh token 从 HttpOnly Cookie 读取，不再从请求体获取
	refreshToken := netutil.CookieFromContext(ctx, refreshTokenCookieName)
	if refreshToken == "" {
		return nil, authenticationV1.ErrorIncorrectRefreshToken("refresh token cookie is missing")
	}

	// admin 客户端类型固定
	req.ClientType = trans.Ptr(authenticationV1.ClientType_admin)

	return s.doGrantTypeRefreshToken(ctx, req, refreshToken)
}

// ValidateToken 验证令牌
func (s *AuthenticationService) ValidateToken(ctx context.Context, req *authenticationV1.ValidateTokenRequest) (*authenticationV1.ValidateTokenResponse, error) {
	return s.authenticator.Authenticate(ctx, req)
}

// RegisterUser 注册前台用户
func (s *AuthenticationService) RegisterUser(ctx context.Context, req *authenticationV1.RegisterUserRequest) (*authenticationV1.RegisterUserResponse, error) {
	// 免鉴权接口（白名单），无 auth 中间件注入 viewer，ent 隐私层要求
	// ViewerContext——与 doGrantTypePassword 同款处理
	ctx = s.resetContextForLogin(ctx)

	var err error

	var tenantId *uint32
	if constants.IsTenantModeEnabled {
		// 租户模式：注册必须归属有效租户——无 tenant_code 时 user 落库会撞
		// tenant 外键报 500，此处提前以明确错误拒绝（平台账号由种子/管理端创建）
		if strings.TrimSpace(req.GetTenantCode()) == "" {
			return nil, authenticationV1.ErrorBadRequest("tenant code required")
		}
		var tenant *identityV1.Tenant
		tenant, err = s.tenantRepo.Get(ctx, &identityV1.GetTenantRequest{
			QueryBy: &identityV1.GetTenantRequest_Code{Code: req.GetTenantCode()},
		})
		if err != nil || tenant == nil {
			return nil, authenticationV1.ErrorBadRequest("invalid tenant")
		}

		tenantId = tenant.Id
	}

	user, err := s.userRepo.Create(ctx, &identityV1.CreateUserRequest{
		Data: &identityV1.User{
			TenantId: tenantId,
			Username: trans.Ptr(req.Username),
			Email:    req.Email,
		},
	})
	if err != nil {
		s.log.Errorf("create user error: %v", err)
		return nil, err
	}

	// 前端注册密码与登录一致为 AES 加密传输（base64），落库哈希必须基于明文：
	// 此前直接把 AES 密文传给 Create，存成 bcrypt(AES密文)，而登录校验是
	// bcrypt(解密后明文)——哈希对象不一致，注册用户永远无法登录。
	plainPassword, derr := decryptTransportSecret(req.GetPassword())
	if derr != nil {
		return nil, authenticationV1.ErrorBadRequest("invalid password format")
	}

	if err = s.userCredentialRepo.Create(ctx, &authenticationV1.CreateUserCredentialRequest{
		Data: &authenticationV1.UserCredential{
			UserId:   user.Id,
			TenantId: user.TenantId,

			IdentityType: authenticationV1.UserCredential_USERNAME.Enum(),
			Identifier:   trans.Ptr(req.GetUsername()),

			CredentialType: authenticationV1.UserCredential_PASSWORD_HASH.Enum(),
			Credential:     trans.Ptr(plainPassword),

			IsPrimary: trans.Ptr(true),
			Status:    authenticationV1.UserCredential_ENABLED.Enum(),
		},
	}); err != nil {
		s.log.Errorf("create user credentials error: %v", err)
		return nil, err
	}

	return &authenticationV1.RegisterUserResponse{
		UserId: user.GetId(),
	}, nil
}

func (s *AuthenticationService) WhoAmI(ctx context.Context, _ *emptypb.Empty) (*authenticationV1.WhoAmIResponse, error) {
	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	return &authenticationV1.WhoAmIResponse{
		UserId:   operator.GetUserId(),
		Username: operator.GetUsername(),
	}, nil
}

func (s *AuthenticationService) GenerateCaptcha(ctx context.Context, _ *emptypb.Empty) (*authenticationV1.GenerateCaptchaResponse, error) {
	captchaId, captchaImage, answer, err := s.captchaClient.Generate()
	if err != nil {
		s.log.Errorf("generate captcha failed: %s", err.Error())
		return nil, authenticationV1.ErrorInternalServerError("generate captcha failed")
	}

	// Generate() 只生成验证码但不落盘，必须手动 Save 到 Redis，否则 Verify 时查不到。
	if err = s.captchaClient.Save(ctx, captchaId, answer); err != nil {
		s.log.Errorf("save captcha failed: %s", err.Error())
		return nil, authenticationV1.ErrorInternalServerError("save captcha failed")
	}

	return &authenticationV1.GenerateCaptchaResponse{
		CaptchaId:   captchaId,
		ImageBase64: captchaImage,
	}, nil
}

func (s *AuthenticationService) VerifyCaptcha(ctx context.Context, req *authenticationV1.VerifyCaptchaRequest) (*authenticationV1.VerifyCaptchaResponse, error) {
	ok, err := s.captchaClient.Verify(ctx, req.GetCaptchaId(), req.GetUserInput())
	if err != nil {
		s.log.Errorf("verify captcha failed: %s", err.Error())
		return nil, authenticationV1.ErrorInternalServerError("verify captcha failed")
	}

	return &authenticationV1.VerifyCaptchaResponse{
		Valid: ok,
	}, nil
}
