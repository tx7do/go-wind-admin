package service

import (
	"context"
	"fmt"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/aggregator"
	"github.com/tx7do/go-utils/sliceutil"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"

	"go-wind-admin/pkg/constants"
	appViewer "go-wind-admin/pkg/entgo/viewer"
	"go-wind-admin/pkg/middleware/auth"
	"go-wind-admin/pkg/utils"
)

type UserService struct {
	adminV1.UserServiceHTTPServer

	log *bLogger.Helper

	userRepo           data.UserRepo
	userCredentialRepo *data.UserCredentialRepo

	roleRepo     *data.RoleRepo
	positionRepo *data.PositionRepo
	orgUnitRepo  *data.OrgUnitRepo
	tenantRepo   *data.TenantRepo

	membershipRepo *data.MembershipRepo
}

func NewUserService(
	ctx *bootstrap.Context,
	userRepo data.UserRepo,
	roleRepo *data.RoleRepo,
	userCredentialRepo *data.UserCredentialRepo,
	positionRepo *data.PositionRepo,
	orgUnitRepo *data.OrgUnitRepo,
	tenantRepo *data.TenantRepo,
	membershipRepo *data.MembershipRepo,
) *UserService {
	svc := &UserService{
		log:                ctx.NewLoggerHelper("user/service/admin-service"),
		userRepo:           userRepo,
		roleRepo:           roleRepo,
		userCredentialRepo: userCredentialRepo,
		positionRepo:       positionRepo,
		orgUnitRepo:        orgUnitRepo,
		tenantRepo:         tenantRepo,
		membershipRepo:     membershipRepo,
	}

	svc.init()

	return svc
}

func (s *UserService) init() {
	ctx := appViewer.NewSystemViewerContext(context.Background())
	if count, _ := s.userRepo.Count(ctx, nil); count == 0 {
		_ = s.createDefaultUser(ctx)
	}
}

func (s *UserService) extractRelationIDs(
	users []*identityV1.User,
	roleSet aggregator.ResourceMap[uint32, *permissionV1.Role],
	tenantSet aggregator.ResourceMap[uint32, *identityV1.Tenant],
	orgUnitSet aggregator.ResourceMap[uint32, *identityV1.OrgUnit],
	posSet aggregator.ResourceMap[uint32, *identityV1.Position],
) {
	for _, v := range users {
		if v == nil {
			continue
		}

		if id := v.GetTenantId(); id > 0 {
			tenantSet[id] = nil
		}

		for _, roleId := range v.RoleIds {
			if roleId > 0 {
				roleSet[roleId] = nil
			}
		}

		if v.GetOrgUnitId() > 0 {
			orgUnitSet[v.GetOrgUnitId()] = nil
		}
		if len(v.OrgUnitIds) > 0 {
			for _, orgID := range v.OrgUnitIds {
				if orgID > 0 {
					orgUnitSet[orgID] = nil
				}
			}
		}

		if v.GetPositionId() > 0 {
			posSet[v.GetPositionId()] = nil
		}
		if len(v.PositionIds) > 0 {
			for _, posID := range v.PositionIds {
				if posID > 0 {
					posSet[posID] = nil
				}
			}
		}

	}
}

func (s *UserService) fetchRelationInfo(
	ctx context.Context,
	roleSet aggregator.ResourceMap[uint32, *permissionV1.Role],
	tenantSet aggregator.ResourceMap[uint32, *identityV1.Tenant],
	orgUnitSet aggregator.ResourceMap[uint32, *identityV1.OrgUnit],
	posSet aggregator.ResourceMap[uint32, *identityV1.Position],
) error {
	if len(roleSet) > 0 {
		roleIds := make([]uint32, 0, len(roleSet))
		for id := range roleSet {
			roleIds = append(roleIds, id)
		}

		roles, err := s.roleRepo.ListRolesByRoleIds(ctx, roleIds)
		if err != nil {
			s.log.Errorf(context.Background(), "query roles err: %v", err)
			return err
		}

		for _, role := range roles {
			roleSet[role.GetId()] = role
		}
	}

	if len(tenantSet) > 0 {
		tenantIds := make([]uint32, 0, len(tenantSet))
		for id := range tenantSet {
			tenantIds = append(tenantIds, id)
		}

		tenants, err := s.tenantRepo.ListTenantsByIds(ctx, tenantIds)
		if err != nil {
			s.log.Errorf(context.Background(), "query tenants err: %v", err)
			return err
		}

		for _, tenant := range tenants {
			tenantSet[tenant.GetId()] = tenant
		}
	}

	if len(orgUnitSet) > 0 {
		orgUnitIds := make([]uint32, 0, len(orgUnitSet))
		for id := range orgUnitSet {
			orgUnitIds = append(orgUnitIds, id)
		}

		orgUnits, err := s.orgUnitRepo.ListOrgUnitsByIds(ctx, orgUnitIds)
		if err != nil {
			s.log.Errorf(context.Background(), "query orgUnits err: %v", err)
			return err
		}

		for _, orgUnit := range orgUnits {
			orgUnitSet[orgUnit.GetId()] = orgUnit
		}
	}

	if len(posSet) > 0 {
		posIds := make([]uint32, 0, len(posSet))
		for id := range posSet {
			posIds = append(posIds, id)
		}

		positions, err := s.positionRepo.ListPositionByIds(ctx, posIds)
		if err != nil {
			s.log.Errorf(context.Background(), "query positions err: %v", err)
			return err
		}

		for _, position := range positions {
			posSet[position.GetId()] = position
		}
	}

	return nil
}

func (s *UserService) bindRelations(
	users []*identityV1.User,
	roleSet aggregator.ResourceMap[uint32, *permissionV1.Role],
	tenantSet aggregator.ResourceMap[uint32, *identityV1.Tenant],
	orgUnitSet aggregator.ResourceMap[uint32, *identityV1.OrgUnit],
	posSet aggregator.ResourceMap[uint32, *identityV1.Position],
) {
	aggregator.PopulateMulti(
		users,
		roleSet,
		func(ou *identityV1.User) []uint32 { return ou.GetRoleIds() },
		func(ou *identityV1.User, r []*permissionV1.Role) {
			for _, role := range r {
				ou.RoleNames = append(ou.RoleNames, role.GetName())
				ou.Roles = append(ou.Roles, role.GetCode())
			}
		},
	)
	aggregator.Populate(
		users,
		roleSet,
		func(ou *identityV1.User) uint32 { return ou.GetRoleId() },
		func(ou *identityV1.User, r *permissionV1.Role) {
			ou.RoleNames = append(ou.RoleNames, r.GetName())
			ou.Roles = append(ou.Roles, r.GetCode())
		},
	)

	aggregator.Populate(
		users,
		tenantSet,
		func(ou *identityV1.User) uint32 { return ou.GetTenantId() },
		func(ou *identityV1.User, r *identityV1.Tenant) {
			ou.TenantName = r.Name
		},
	)

	aggregator.PopulateMulti(
		users,
		posSet,
		func(ou *identityV1.User) []uint32 { return ou.GetPositionIds() },
		func(ou *identityV1.User, r []*identityV1.Position) {
			for _, pos := range r {
				ou.PositionNames = append(ou.PositionNames, pos.GetName())
			}
		},
	)
	aggregator.Populate(
		users,
		posSet,
		func(ou *identityV1.User) uint32 { return ou.GetPositionId() },
		func(ou *identityV1.User, r *identityV1.Position) {
			ou.PositionName = r.Name
		},
	)

	aggregator.PopulateMulti(
		users,
		orgUnitSet,
		func(ou *identityV1.User) []uint32 { return ou.GetOrgUnitIds() },
		func(ou *identityV1.User, orgs []*identityV1.OrgUnit) {
			for _, org := range orgs {
				ou.OrgUnitNames = append(ou.OrgUnitNames, org.GetName())
			}
		},
	)
	aggregator.Populate(
		users,
		orgUnitSet,
		func(ou *identityV1.User) uint32 { return ou.GetOrgUnitId() },
		func(ou *identityV1.User, org *identityV1.OrgUnit) {
			ou.OrgUnitName = org.Name
		},
	)
}

func (s *UserService) enrichRelations(ctx context.Context, users []*identityV1.User) error {
	var roleSet = make(aggregator.ResourceMap[uint32, *permissionV1.Role])
	var tenantSet = make(aggregator.ResourceMap[uint32, *identityV1.Tenant])
	var orgUnitSet = make(aggregator.ResourceMap[uint32, *identityV1.OrgUnit])
	var posSet = make(aggregator.ResourceMap[uint32, *identityV1.Position])

	s.extractRelationIDs(users, roleSet, tenantSet, orgUnitSet, posSet)
	if err := s.fetchRelationInfo(ctx, roleSet, tenantSet, orgUnitSet, posSet); err != nil {
		return err
	}
	s.bindRelations(users, roleSet, tenantSet, orgUnitSet, posSet)
	return nil
}

func (s *UserService) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListUserResponse, error) {
	resp, err := s.userRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	_ = s.enrichRelations(ctx, resp.Items)

	return resp, nil
}

func (s *UserService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.CountUserResponse, error) {
	count, err := s.userRepo.Count(ctx, req)
	if err != nil {
		return nil, err
	}

	return &identityV1.CountUserResponse{
		Count: uint64(count),
	}, nil
}

func (s *UserService) Get(ctx context.Context, req *identityV1.GetUserRequest) (*identityV1.User, error) {
	resp, err := s.userRepo.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	fakeItems := []*identityV1.User{resp}
	_ = s.enrichRelations(ctx, fakeItems)

	return resp, nil
}

func (s *UserService) Create(ctx context.Context, req *identityV1.CreateUserRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	// 获取操作者的用户信息
	_, err = s.userRepo.Get(ctx, &identityV1.GetUserRequest{
		QueryBy: &identityV1.GetUserRequest_Id{
			Id: operator.UserId,
		},
	})
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)
	if operator.GetTenantId() > 0 {
		req.Data.TenantId = operator.TenantId
	}

	var roleIds []uint32
	if len(req.Data.GetRoleIds()) > 0 {
		roleIds = req.Data.GetRoleIds()
	}
	if req.Data.RoleId != nil && *req.Data.RoleId > 0 {
		roleIds = append(roleIds, *req.Data.RoleId)
	}
	roleIds = sliceutil.Unique(roleIds)
	if len(roleIds) == 0 {
		s.log.Errorf(ctx, "role_ids is required")
		return nil, adminV1.ErrorBadRequest("role_ids is required")
	}

	var queryString string
	if operator.GetTenantId() > 0 || req.Data.GetTenantId() > 0 {
		queryString = fmt.Sprintf(`{"id__in": "[%s]", "type": "TENANT", "tenant_id": %d}`,
			utils.NumberSliceToString(roleIds),
			req.Data.GetTenantId(),
		)
	} else {
		queryString = fmt.Sprintf(`{"id__in": "[%s]", "type": "SYSTEM"}`,
			utils.NumberSliceToString(roleIds),
		)
	}
	roles, err := s.roleRepo.List(ctx, &paginationV1.PagingRequest{
		NoPaging: trans.Ptr(true),
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: queryString,
		},
	})
	if err != nil {
		s.log.Errorf(ctx, "query roles err: %v", err)
		return nil, err
	}

	if len(roles.Items) != len(roleIds) {
		s.log.Errorf(ctx, "some roles not found, requested role ids: %v", roleIds)
		return nil, adminV1.ErrorBadRequest("some roles not found")
	}
	if len(roles.Items) == 0 {
		s.log.Errorf(ctx, "at least one role is required")
		return nil, adminV1.ErrorBadRequest("at least one role is required")
	}

	req.Data.RoleId = nil
	req.Data.RoleIds = roleIds

	// 创建用户
	var user *identityV1.User
	if user, err = s.userRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	if len(req.GetPassword()) == 0 {
		// 如果没有设置密码，则设置为默认密码。
		req.Password = trans.Ptr(constants.DefaultUserPassword)
	}

	if len(req.GetPassword()) > 0 {
		if err = s.userCredentialRepo.Create(ctx, &authenticationV1.CreateUserCredentialRequest{
			Data: &authenticationV1.UserCredential{
				UserId:   user.Id,
				TenantId: user.TenantId,

				IdentityType: authenticationV1.UserCredential_USERNAME.Enum(),
				Identifier:   req.Data.Username,

				CredentialType: authenticationV1.UserCredential_PASSWORD_HASH.Enum(),
				Credential:     req.Password,

				IsPrimary: trans.Ptr(true),
				Status:    authenticationV1.UserCredential_ENABLED.Enum(),
			},
		}); err != nil {
			return nil, err
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *UserService) Update(ctx context.Context, req *identityV1.UpdateUserRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	// 获取操作者的用户信息
	_, err = s.userRepo.Get(ctx, &identityV1.GetUserRequest{
		QueryBy: &identityV1.GetUserRequest_Id{
			Id: operator.UserId,
		},
	})
	if err != nil {
		return nil, err
	}

	req.Data.UpdatedBy = trans.Ptr(operator.UserId)
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	req.Data.Id = trans.Ptr(req.GetId())

	if operator.GetTenantId() > 0 {
		req.Data.TenantId = operator.TenantId
	}

	var roleIds []uint32
	if len(req.Data.GetRoleIds()) > 0 {
		roleIds = req.Data.GetRoleIds()
	}
	if req.Data.RoleId != nil && *req.Data.RoleId > 0 {
		roleIds = append(roleIds, *req.Data.RoleId)
	}
	roleIds = sliceutil.Unique(roleIds)
	if len(roleIds) == 0 {
		s.log.Errorf(ctx, "role_ids is required")
		return nil, adminV1.ErrorBadRequest("role_ids is required")
	}

	var queryString string
	if operator.GetTenantId() > 0 || req.Data.GetTenantId() > 0 {
		queryString = fmt.Sprintf(`{"id__in": "[%s]", "type": "TENANT", "tenant_id": %d}`,
			utils.NumberSliceToString(roleIds),
			req.Data.GetTenantId(),
		)
	} else {
		queryString = fmt.Sprintf(`{"id__in": "[%s]", "type": "SYSTEM"}`,
			utils.NumberSliceToString(roleIds),
		)
	}
	roles, err := s.roleRepo.List(ctx, &paginationV1.PagingRequest{
		NoPaging: trans.Ptr(true),
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: queryString,
		},
	})
	if err != nil {
		s.log.Errorf(ctx, "query roles err: %v", err)
		return nil, err
	}

	if len(roles.Items) != len(roleIds) {
		s.log.Errorf(ctx, "some roles not found, requested role ids: %v", roleIds)
		return nil, adminV1.ErrorBadRequest("some roles not found")
	}
	if len(roles.Items) == 0 {
		s.log.Errorf(ctx, "at least one role is required")
		return nil, adminV1.ErrorBadRequest("at least one role is required")
	}

	req.Data.RoleId = nil
	req.Data.RoleIds = roleIds

	// 更新用户
	if err = s.userRepo.Update(ctx, req); err != nil {
		s.log.Error(ctx, err.Error())
		return nil, err
	}

	if len(req.GetPassword()) > 0 {
		// 跨租户防护：非平台超级管理员重置密码前，校验目标用户真实租户
		// 注意：此处不能用 req.Data.GetTenantId()，因为上方已将其覆盖为操作者租户
		var targetUsername string
		if !operator.GetIsPlatformAdmin() {
			targetUser, gerr := s.userRepo.Get(ctx, &identityV1.GetUserRequest{
				QueryBy: &identityV1.GetUserRequest_Id{Id: req.GetId()},
			})
			if gerr != nil {
				return nil, gerr
			}
			if targetUser.GetTenantId() != operator.GetTenantId() {
				s.log.Errorf(ctx, "operator [%d] (tenant %d) has no permission to reset password of cross-tenant user [%d] (tenant %d)",
					operator.GetUserId(), operator.GetTenantId(), targetUser.GetId(), targetUser.GetTenantId())
				return nil, adminV1.ErrorForbidden("no permission to reset password of user in other tenant")
			}
			// 防 IDOR：必须用 DB 中已校验租户的目标用户名，而非请求体里可伪造的 username。
			// 否则攻击者可用 id=本租户用户 + username=他租户用户 绕过上面的租户校验。
			targetUsername = targetUser.GetUsername()
		} else {
			// 平台管理员：无跨租户限制，但仍按 id 从 DB 取 username——
			// 既校验目标用户存在，也保证重置目标与请求 id 严格一致
			// （避免 id=甲 + username=乙 的参数错位实际重置了乙）
			targetUser, gerr := s.userRepo.Get(ctx, &identityV1.GetUserRequest{
				QueryBy: &identityV1.GetUserRequest_Id{Id: req.GetId()},
			})
			if gerr != nil {
				return nil, gerr
			}
			targetUsername = targetUser.GetUsername()
		}

		if err = s.userCredentialRepo.ResetCredential(ctx, &authenticationV1.ResetCredentialRequest{
			IdentityType:  authenticationV1.UserCredential_USERNAME,
			Identifier:    targetUsername,
			NewCredential: req.GetPassword(),
			NeedDecrypt:   true, // 前端 AES 密文传输，解密后才走复杂度/哈希
		}); err != nil {
			return nil, err
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *UserService) Delete(ctx context.Context, req *identityV1.DeleteUserRequest) (*emptypb.Empty, error) {
	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	// 获取操作者的用户信息
	_, err = s.userRepo.Get(ctx, &identityV1.GetUserRequest{
		QueryBy: &identityV1.GetUserRequest_Id{
			Id: operator.UserId,
		},
	})
	if err != nil {
		return nil, err
	}

	// 获取将被删除的用户信息
	target, err := s.userRepo.Get(ctx, &identityV1.GetUserRequest{
		QueryBy: &identityV1.GetUserRequest_Id{
			Id: req.GetId(),
		},
	})
	if err != nil {
		return nil, err
	}

	// 禁止删除默认超级管理员：初始化时创建的平台级 admin（恒为 id=1，
	// 即便后续改名也由 id 兜底保护）。误删会导致系统失去超级管理员且无法自动重建。
	if target.GetId() == 1 ||
		(target.GetUsername() == constants.DefaultAdminUserName && target.GetTenantId() == 0) {
		s.log.Errorf(ctx, "operator [%d] attempted to delete default admin user [%d]",
			operator.GetUserId(), target.GetId())
		return nil, adminV1.ErrorBadRequest("default admin cannot be deleted")
	}

	// 禁止删除自己：误删自身账号将导致当前会话立即失去管理能力。
	if target.GetId() == operator.GetUserId() {
		s.log.Errorf(ctx, "operator [%d] attempted to delete self", operator.GetUserId())
		return nil, adminV1.ErrorBadRequest("cannot delete yourself")
	}

	// 删除用户
	err = s.userRepo.Delete(ctx, req)

	return &emptypb.Empty{}, err
}

func (s *UserService) UserExists(ctx context.Context, req *identityV1.UserExistsRequest) (*identityV1.UserExistsResponse, error) {
	return s.userRepo.UserExists(ctx, req)
}

// EditUserPassword 修改用户密码
func (s *UserService) EditUserPassword(ctx context.Context, req *identityV1.EditUserPasswordRequest) (*emptypb.Empty, error) {
	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	// 获取目标用户信息
	u, err := s.userRepo.Get(ctx, &identityV1.GetUserRequest{
		QueryBy: &identityV1.GetUserRequest_Id{
			Id: req.GetUserId(),
		},
	})
	if err != nil {
		return nil, err
	}

	// 跨租户防护：非平台超级管理员只能重置本租户用户的密码
	if !operator.GetIsPlatformAdmin() && u.GetTenantId() != operator.GetTenantId() {
		s.log.Errorf(ctx, "operator [%d] (tenant %d) has no permission to reset password of cross-tenant user [%d] (tenant %d)",
			operator.GetUserId(), operator.GetTenantId(), u.GetId(), u.GetTenantId())
		return nil, adminV1.ErrorForbidden("no permission to reset password of user in other tenant")
	}

	if err = s.userCredentialRepo.ResetCredential(ctx, &authenticationV1.ResetCredentialRequest{
		IdentityType:  authenticationV1.UserCredential_USERNAME,
		Identifier:    u.GetUsername(),
		NewCredential: req.GetNewPassword(),
		NeedDecrypt:   true, // 前端 AES 密文传输，解密后才走复杂度/哈希
	}); err != nil {
		s.log.Errorf(ctx, "reset user password err: %v", err)
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// createDefaultUser 创建默认用户，即超级用户
func (s *UserService) createDefaultUser(ctx context.Context) error {
	var err error

	// 创建默认用户
	for _, user := range constants.DefaultUsers {
		if _, err = s.userRepo.Create(ctx, &identityV1.CreateUserRequest{
			Data: user,
		}); err != nil {
			s.log.Errorf(ctx, "create default user err: %v", err)
			return err
		}
	}

	// 创建默认用户凭证
	for _, userCredential := range constants.DefaultUserCredentials {
		// 引导期默认管理员口令（admin/admin）不满足等保复杂度，走专用 Bootstrap 路径跳过校验，
		// 否则凭证无法写入，首次登录将恒报 INVALID_PASSWORD（用户不存在）。
		if err = s.userCredentialRepo.CreateBootstrapCredential(ctx, userCredential); err != nil {
			s.log.Errorf(ctx, "create default user credential err: %v", err)
			return err
		}
	}

	switch constants.DefaultUserTenantRelationType {
	default:
		fallthrough
	case constants.UserTenantRelationOneToOne:
		// 创建默认用户角色关联关系
		for _, userRole := range constants.DefaultUserRoles {
			if err = s.userRepo.AssignUserRole(ctx, userRole); err != nil {
				s.log.Errorf(ctx, "create default user role relation err: %v", err)
				return err
			}
		}

	case constants.UserTenantRelationOneToMany:
		// 创建默认用户租户关联关系
		for _, membership := range constants.DefaultMemberships {
			if err = s.membershipRepo.AssignTenantMembershipWith(ctx, membership); err != nil {
				s.log.Errorf(ctx, "create default user membership err: %v", err)
				return err
			}
		}
	}

	return err
}
