package service

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/go-utils/trans"
	authnEngine "github.com/tx7do/kratos-authn/engine"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	userV1 "go-wind-admin/api/gen/go/user/service/v1"

	"go-wind-admin/pkg/jwt"
	"go-wind-admin/pkg/middleware/auth"
)

type AuthenticationService struct {
	adminV1.AuthenticationServiceHTTPServer

	userRepo           data.UserRepo
	userCredentialRepo *data.UserCredentialRepo
	roleRepo           *data.RoleRepo
	tenantRepo         *data.TenantRepo
	membershipRepo     *data.MembershipRepo
	orgUnitRepo        *data.OrgUnitRepo

	userToken *data.UserTokenCacheRepo

	authenticator authnEngine.Authenticator

	log *log.Helper
}

func NewAuthenticationService(
	ctx *bootstrap.Context,
	userRepo data.UserRepo,
	userCredentialRepo *data.UserCredentialRepo,
	roleRepo *data.RoleRepo,
	tenantRepo *data.TenantRepo,
	membershipRepo *data.MembershipRepo,
	orgUnitRepo *data.OrgUnitRepo,
	userToken *data.UserTokenCacheRepo,
	authenticator authnEngine.Authenticator,
) *AuthenticationService {
	return &AuthenticationService{
		log:                ctx.NewLoggerHelper("authn/service/admin-service"),
		userRepo:           userRepo,
		userCredentialRepo: userCredentialRepo,
		tenantRepo:         tenantRepo,
		roleRepo:           roleRepo,
		membershipRepo:     membershipRepo,
		orgUnitRepo:        orgUnitRepo,
		userToken:          userToken,
		authenticator:      authenticator,
	}
}

// Login 登录
func (s *AuthenticationService) Login(ctx context.Context, req *authenticationV1.LoginRequest) (*authenticationV1.LoginResponse, error) {
	switch req.GetGrantType() {
	case authenticationV1.GrantType_password:
		return s.doGrantTypePassword(ctx, req)

	case authenticationV1.GrantType_refresh_token:
		return s.doGrantTypeRefreshToken(ctx, req)

	case authenticationV1.GrantType_client_credentials:
		return s.doGrantTypeClientCredentials(ctx, req)

	default:
		return nil, authenticationV1.ErrorInvalidGrantType("invalid grant type")
	}
}

var allowedPlatformCodes = map[string]struct{}{
	"super":       {},
	"super_admin": {},
	"superadmin":  {}, // 兼容不同命名
}

// HasPlatformAdminRole 判断用户是否拥有平台角色
func (s *AuthenticationService) HasPlatformAdminRole(roles []*userV1.Role) bool {
	if len(roles) == 0 {
		return false
	}

	for _, role := range roles {
		if role == nil {
			continue
		}

		// 拥有全局数据权限
		if role.GetDataScope() == userV1.Role_ALL &&
			role.GetType() == userV1.Role_SYSTEM {
			s.log.Debugf("role id [%d] has all data scope and system type, is platform admin", role.GetId())
			return true
		}

		// 根据 code 判定（大小写不敏感）
		if roleCode := strings.ToLower(role.GetCode()); roleCode != "" {
			if _, ok := allowedPlatformCodes[roleCode]; ok {
				s.log.Debugf("role code [%s] is allowed platform admin code", roleCode)
				return true
			}
		}
	}

	return false
}

var tenantAdminCodes = map[string]struct{}{
	"tenant_admin": {},
	"tenantadmin":  {},
	"tenant-admin": {},
}

// HasTenantAdminRole 判断用户是否拥有租户管理员角色
func (s *AuthenticationService) HasTenantAdminRole(roles []*userV1.Role) bool {
	if len(roles) == 0 {
		return false
	}

	for _, role := range roles {
		if role == nil {
			continue
		}

		ds := role.GetDataScope()
		if ds == userV1.Role_ALL || ds == userV1.Role_UNIT_AND_CHILD {
			return true
		}

		if rc := strings.ToLower(role.GetCode()); rc != "" {
			if _, ok := tenantAdminCodes[rc]; ok {
				return true
			}
		}
	}

	return false
}

var priorityDataScope = map[userV1.Role_DataScope]int{
	userV1.Role_SELF:           1,
	userV1.Role_UNIT_ONLY:      2,
	userV1.Role_UNIT_AND_CHILD: 3,
	userV1.Role_SELECTED_UNITS: 4,
	userV1.Role_ALL:            5,
}

// mergeRolesDataScope 合并角色数据权限
func (s *AuthenticationService) mergeRolesDataScope(roles []*userV1.Role) userV1.Role_DataScope {
	if len(roles) == 0 {
		return userV1.Role_SELF
	}

	final := userV1.Role_SELF
	bestPrio := 0

	for _, r := range roles {
		if r == nil {
			continue
		}
		ds := r.GetDataScope()

		// 最优先短路
		if ds == userV1.Role_ALL {
			return userV1.Role_ALL
		}

		if p, ok := priorityDataScope[ds]; ok {
			if p > bestPrio {
				bestPrio = p
				final = ds
			}
		}
	}

	return final
}

// pickMostSpecificOrgUnit 从多个组织单元中选择最具体的一个
func (s *AuthenticationService) pickMostSpecificOrgUnit(units []*userV1.OrgUnit) *userV1.OrgUnit {
	if len(units) == 0 {
		return nil
	}

	var best *userV1.OrgUnit
	bestDepth := -1

	for _, u := range units {
		if u == nil {
			continue
		}
		// 假设 OrgUnit 有 GetPath() 返回例如 "1/2/3" 或 "/1/2/3/"
		p := strings.Trim(u.GetPath(), "/")
		depth := 0
		if p != "" {
			depth = len(strings.Split(p, "/"))
		}

		if depth > bestDepth {
			bestDepth = depth
			best = u
		}
	}

	return best
}

// enrichUserTokenPayload 填充用户令牌载荷的权限相关字段
func (s *AuthenticationService) enrichUserTokenPayload(ctx context.Context, userID, tenantID uint32, tokenPayload *authenticationV1.UserTokenPayload) error {
	// 初始化默认值
	tokenPayload.DataScope = trans.Ptr(userV1.Role_SELF)

	// 获取角色/岗位/组织单元 id 列表
	roleIDs, _, orgUnitIDs, err := s.membershipRepo.ListMembershipAllIDs(ctx, userID, tenantID)
	if err != nil {
		s.log.Errorf("list user [%d] membership ids failed [%s]", userID, err.Error())
		return authenticationV1.ErrorServiceUnavailable("获取用户角色失败")
	}

	// 获取角色信息并填充 role codes / data scope / admin flags
	roles, err := s.roleRepo.ListRolesByRoleIds(ctx, roleIDs)
	if err != nil {
		s.log.Errorf("list roles by ids failed [%s]", err.Error())
		// 继续：即使查角色失败也不直接返回，这与原实现一致（caller 侧应决定如何处理）
	}
	for _, role := range roles {
		if role == nil {
			continue
		}
		tokenPayload.Roles = append(tokenPayload.Roles, role.GetCode())
	}
	tokenPayload.DataScope = trans.Ptr(s.mergeRolesDataScope(roles))

	if tenantID == 0 && s.HasPlatformAdminRole(roles) {
		tokenPayload.IsPlatformAdmin = trans.Ptr(true)
		// 平台管理员赋予全局数据权限
		tokenPayload.DataScope = trans.Ptr(userV1.Role_ALL)
	} else if tenantID > 0 && s.HasTenantAdminRole(roles) {
		tokenPayload.IsTenantAdmin = trans.Ptr(true)
	}

	// 选取最具体的 org unit id
	orgUnits, err := s.orgUnitRepo.ListOrgUnitsByIds(ctx, orgUnitIDs)
	if err != nil {
		s.log.Errorf("list org units failed [%s]", err.Error())
	} else {
		if most := s.pickMostSpecificOrgUnit(orgUnits); most != nil {
			tokenPayload.OrgUnitId = most.Id
		}
	}

	return nil
}

// validateUserAuthority 只负责基于已经填充好的 tokenPayload 做权限校验
func (s *AuthenticationService) validateUserAuthority(userID uint32, tokenPayload *authenticationV1.UserTokenPayload) error {
	if !tokenPayload.GetIsPlatformAdmin() && !tokenPayload.GetIsTenantAdmin() {
		s.log.Errorf("user [%d] has no admin authority", userID)
		return authenticationV1.ErrorForbidden("insufficient authority")
	}
	if tokenPayload.GetDataScope() == userV1.Role_SELF {
		s.log.Errorf("user [%d] has insufficient data scope", userID)
		return authenticationV1.ErrorForbidden("insufficient data scope")
	}
	return nil
}

// resolveUserAuthority 解析用户权限信息
func (s *AuthenticationService) resolveUserAuthority(ctx context.Context, userID, tenantID uint32, tokenPayload *authenticationV1.UserTokenPayload) error {
	tokenPayload.DataScope = trans.Ptr(userV1.Role_SELF)

	if err := s.enrichUserTokenPayload(ctx, userID, tenantID, tokenPayload); err != nil {
		return err
	}
	return s.validateUserAuthority(userID, tokenPayload)
}

// doGrantTypePassword 处理授权类型 - 密码
func (s *AuthenticationService) doGrantTypePassword(ctx context.Context, req *authenticationV1.LoginRequest) (*authenticationV1.LoginResponse, error) {
	var err error
	if _, err = s.userCredentialRepo.VerifyCredential(ctx, &authenticationV1.VerifyCredentialRequest{
		IdentityType: authenticationV1.UserCredential_USERNAME,
		Identifier:   req.GetUsername(),
		Credential:   req.GetPassword(),
		NeedDecrypt:  true,
	}); err != nil {
		s.log.Errorf("verify user credential failed for username [%s]: %s", req.GetUsername(), err.Error())
		return nil, err
	}

	// 获取用户信息
	var user *userV1.User
	user, err = s.userRepo.Get(ctx, &userV1.GetUserRequest{QueryBy: &userV1.GetUserRequest_Username{Username: req.GetUsername()}})
	if err != nil {
		s.log.Errorf("get user by username [%s] failed [%s]", req.GetUsername(), err.Error())
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
	err = s.resolveUserAuthority(ctx, user.GetId(), user.GetTenantId(), tokenPayload)
	if err != nil {
		s.log.Errorf("resolve user [%d] authority failed [%s]", user.GetId(), err.Error())
		return nil, err
	}

	// 生成令牌
	accessToken, refreshToken, err := s.userToken.GenerateToken(ctx, tokenPayload)
	if err != nil {
		return nil, err
	}

	return &authenticationV1.LoginResponse{
		TokenType:    authenticationV1.TokenType_bearer,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// doGrantTypeAuthorizationCode 处理授权类型 - 刷新令牌
func (s *AuthenticationService) doGrantTypeRefreshToken(ctx context.Context, req *authenticationV1.LoginRequest) (*authenticationV1.LoginResponse, error) {
	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	// 获取用户信息
	user, err := s.userRepo.Get(ctx, &userV1.GetUserRequest{
		QueryBy: &userV1.GetUserRequest_Id{
			Id: operator.UserId,
		},
	})
	if err != nil {
		return &authenticationV1.LoginResponse{}, err
	}

	tokenPayload := &authenticationV1.UserTokenPayload{
		UserId:   user.GetId(),
		TenantId: user.TenantId,
		Username: user.Username,
		ClientId: req.ClientId,
		DeviceId: req.DeviceId,
	}

	// 解析用户权限信息
	err = s.resolveUserAuthority(ctx, user.GetId(), user.GetTenantId(), tokenPayload)
	if err != nil {
		s.log.Errorf("resolve user [%d] authority failed [%s]", user.GetId(), err.Error())
		return nil, err
	}

	// 校验刷新令牌
	if !s.userToken.IsExistRefreshToken(ctx, operator.UserId, req.GetRefreshToken()) {
		return nil, authenticationV1.ErrorIncorrectRefreshToken("invalid refresh token")
	}

	if err = s.userToken.RemoveRefreshToken(ctx, operator.UserId, req.GetRefreshToken()); err != nil {
		s.log.Errorf("remove refresh token failed [%s]", err.Error())
	}

	// 生成令牌
	accessToken, refreshToken, err := s.userToken.GenerateToken(ctx, tokenPayload)
	if err != nil {
		return nil, authenticationV1.ErrorServiceUnavailable("generate token failed")
	}

	return &authenticationV1.LoginResponse{
		TokenType:    authenticationV1.TokenType_bearer,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
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

	if err = s.userToken.RemoveToken(ctx, operator.UserId); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// RefreshToken 刷新令牌
func (s *AuthenticationService) RefreshToken(ctx context.Context, req *authenticationV1.LoginRequest) (*authenticationV1.LoginResponse, error) {
	// 校验授权类型
	if req.GetGrantType() != authenticationV1.GrantType_refresh_token {
		return nil, authenticationV1.ErrorInvalidGrantType("invalid grant type")
	}

	return s.doGrantTypeRefreshToken(ctx, req)
}

// ValidateToken 验证令牌
func (s *AuthenticationService) ValidateToken(_ context.Context, req *authenticationV1.ValidateTokenRequest) (*authenticationV1.ValidateTokenResponse, error) {
	ret, err := s.authenticator.AuthenticateToken(req.GetToken())
	if err != nil {
		return &authenticationV1.ValidateTokenResponse{
			IsValid: false,
		}, err
	}

	claims, err := jwt.NewUserTokenPayloadWithClaims(ret)
	if err != nil {
		return &authenticationV1.ValidateTokenResponse{
			IsValid: false,
		}, err
	}

	return &authenticationV1.ValidateTokenResponse{
		IsValid: true,
		Claim:   claims,
	}, nil
}

// RegisterUser 注册前台用户
func (s *AuthenticationService) RegisterUser(ctx context.Context, req *authenticationV1.RegisterUserRequest) (*authenticationV1.RegisterUserResponse, error) {
	var err error

	var tenantId *uint32
	tenant, err := s.tenantRepo.Get(ctx, &userV1.GetTenantRequest{QueryBy: &userV1.GetTenantRequest_Code{Code: req.GetTenantCode()}})
	if tenant != nil {
		tenantId = tenant.Id
	}

	user, err := s.userRepo.Create(ctx, &userV1.CreateUserRequest{
		Data: &userV1.User{
			TenantId: tenantId,
			Username: trans.Ptr(req.Username),
			Email:    req.Email,
		},
	})
	if err != nil {
		s.log.Errorf("create user error: %v", err)
		return nil, err
	}

	if err = s.userCredentialRepo.Create(ctx, &authenticationV1.CreateUserCredentialRequest{
		Data: &authenticationV1.UserCredential{
			UserId:   user.Id,
			TenantId: user.TenantId,

			IdentityType: authenticationV1.UserCredential_USERNAME.Enum(),
			Identifier:   trans.Ptr(req.GetUsername()),

			CredentialType: authenticationV1.UserCredential_PASSWORD_HASH.Enum(),
			Credential:     trans.Ptr(req.GetPassword()),

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
