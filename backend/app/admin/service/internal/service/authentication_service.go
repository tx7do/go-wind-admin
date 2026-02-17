package service

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/go-crud/viewer"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent/privacy"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"

	"go-wind-admin/pkg/constants"
	"go-wind-admin/pkg/middleware/auth"
)

type AuthenticationService struct {
	adminV1.AuthenticationServiceHTTPServer

	userRepo           data.UserRepo
	userCredentialRepo *data.UserCredentialRepo

	roleRepo       *data.RoleRepo
	tenantRepo     *data.TenantRepo
	membershipRepo *data.MembershipRepo
	orgUnitRepo    *data.OrgUnitRepo
	permissionRepo *data.PermissionRepo

	authenticator *data.Authenticator
	clientType    authenticationV1.ClientType

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
	permissionRepo *data.PermissionRepo,
	authenticator *data.Authenticator,
	clientType authenticationV1.ClientType,
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
	}
}

// Login 登录
func (s *AuthenticationService) Login(ctx context.Context, req *authenticationV1.LoginRequest) (*authenticationV1.LoginResponse, error) {
	// 没有 viewer 信息，使用空的 NoopContext
	ctx = viewer.WithContext(ctx, viewer.NewNoopContext())
	// 绕过隐私保护中间件
	ctx = privacy.DecisionContext(ctx, privacy.Allow)

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

var priorityDataScope = map[identityV1.DataScope]int{
	identityV1.DataScope_SELF:           1,
	identityV1.DataScope_UNIT_ONLY:      2,
	identityV1.DataScope_UNIT_AND_CHILD: 3,
	identityV1.DataScope_SELECTED_UNITS: 4,
	identityV1.DataScope_ALL:            5,
}

// mergeDataScopes 合并角色数据权限
func (s *AuthenticationService) mergeDataScopes(dataScopes []identityV1.DataScope) identityV1.DataScope {
	if len(dataScopes) == 0 {
		return identityV1.DataScope_SELF
	}

	final := identityV1.DataScope_SELF
	bestPrio := 0

	for _, ds := range dataScopes {
		// 最优先短路
		if ds == identityV1.DataScope_ALL {
			return identityV1.DataScope_ALL
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
func (s *AuthenticationService) pickMostSpecificOrgUnit(units []*identityV1.OrgUnit) *identityV1.OrgUnit {
	if len(units) == 0 {
		return nil
	}

	var best *identityV1.OrgUnit
	bestDepth := -1

	for _, u := range units {
		if u == nil {
			continue
		}
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

	return nil
}

// authorizeAndEnrichUserTokenPayloadUserTenantRelationOneToMany 一对多用户-租户关系的授权与丰富
func (s *AuthenticationService) authorizeAndEnrichUserTokenPayloadUserTenantRelationOneToMany(ctx context.Context, userID, tenantID uint32, tokenPayload *authenticationV1.UserTokenPayload) error {
	var memberships []*identityV1.Membership
	if tenantID > 0 {
		// 指定租户
		membership, err := s.membershipRepo.GetMembershipByUserTenant(ctx, userID)
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
	var validMemberships []*identityV1.Membership
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
			validMemberships = append(validMemberships, m)
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
	var user *identityV1.User
	user, err = s.userRepo.Get(ctx, &identityV1.GetUserRequest{QueryBy: &identityV1.GetUserRequest_Username{Username: req.GetUsername()}})
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
	err = s.resolveUserAuthority(ctx, user, tokenPayload)
	if err != nil {
		s.log.Errorf("resolve user [%d] authority failed [%s]", user.GetId(), err.Error())
		return nil, err
	}

	// 生成令牌
	accessToken, refreshToken, err := s.authenticator.CreateUserToken(ctx, req.GetClientType(), tokenPayload)
	if err != nil {
		return nil, err
	}

	return &authenticationV1.LoginResponse{
		TokenType:        authenticationV1.TokenType_bearer,
		AccessToken:      accessToken,
		RefreshToken:     trans.Ptr(refreshToken),
		ExpiresIn:        int64(s.authenticator.GetAccessTokenExpires(req.GetClientType()).Seconds()),
		RefreshExpiresIn: trans.Ptr(int64(s.authenticator.GetRefreshTokenExpires(req.GetClientType()).Seconds())),
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
	user, err := s.userRepo.Get(ctx, &identityV1.GetUserRequest{
		QueryBy: &identityV1.GetUserRequest_Id{
			Id: operator.UserId,
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

	// 验证刷新令牌
	if err = s.authenticator.VerifyRefreshToken(ctx, req.GetClientType(), req.GetUserId(), operator.GetJti(), req.GetRefreshToken()); err != nil {
		s.log.Errorf("verify refresh token failed for user [%d]: [%s]", req.GetUserId(), err)
		return nil, authenticationV1.ErrorIncorrectRefreshToken("invalid refresh token")
	}

	// 生成令牌
	accessToken, refreshToken, err := s.authenticator.CreateUserToken(ctx, req.GetClientType(), tokenPayload)
	if err != nil {
		return nil, err
	}

	return &authenticationV1.LoginResponse{
		TokenType:        authenticationV1.TokenType_bearer,
		AccessToken:      accessToken,
		RefreshToken:     trans.Ptr(refreshToken),
		ExpiresIn:        int64(s.authenticator.GetAccessTokenExpires(req.GetClientType()).Seconds()),
		RefreshExpiresIn: trans.Ptr(int64(s.authenticator.GetRefreshTokenExpires(req.GetClientType()).Seconds())),
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
func (s *AuthenticationService) ValidateToken(ctx context.Context, req *authenticationV1.ValidateTokenRequest) (*authenticationV1.ValidateTokenResponse, error) {
	return s.authenticator.Authenticate(ctx, req)
}

// RegisterUser 注册前台用户
func (s *AuthenticationService) RegisterUser(ctx context.Context, req *authenticationV1.RegisterUserRequest) (*authenticationV1.RegisterUserResponse, error) {
	var err error

	var tenantId *uint32
	if constants.IsTenantModeEnabled {
		var tenant *identityV1.Tenant
		tenant, err = s.tenantRepo.Get(ctx, &identityV1.GetTenantRequest{
			QueryBy: &identityV1.GetTenantRequest_Code{Code: req.GetTenantCode()},
		})
		if err != nil {
			s.log.Errorf("get tenant by code [%s] failed [%s]", req.GetTenantCode(), err.Error())
			return nil, err
		}

		if tenant != nil {
			tenantId = tenant.Id
		}
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
