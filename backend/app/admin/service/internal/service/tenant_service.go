package service

import (
	"context"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/aggregator"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"

	"go-wind-admin/pkg/authorizer"
	"go-wind-admin/pkg/middleware/auth"
)

type TenantService struct {
	adminV1.TenantServiceHTTPServer

	log *bLogger.Helper

	tenantRepo          *data.TenantRepo
	tenantUsageRepo     *data.TenantUsageRepo
	userRepo            data.UserRepo
	userCredentialsRepo *data.UserCredentialRepo
	roleRepo            *data.RoleRepo

	authorizer *authorizer.Authorizer
}

func NewTenantService(
	ctx *bootstrap.Context,
	tenantRepo *data.TenantRepo,
	tenantUsageRepo *data.TenantUsageRepo,
	userRepo data.UserRepo,
	userCredentialsRepo *data.UserCredentialRepo,
	roleRepo *data.RoleRepo,
	authorizer *authorizer.Authorizer,
) *TenantService {
	return &TenantService{
		log:                 ctx.NewLoggerHelper("tenant/service/admin-service"),
		tenantRepo:          tenantRepo,
		tenantUsageRepo:     tenantUsageRepo,
		userRepo:            userRepo,
		userCredentialsRepo: userCredentialsRepo,
		roleRepo:            roleRepo,
		authorizer:          authorizer,
	}
}

func (s *TenantService) extractRelationIDs(
	tenants []*identityV1.Tenant,
	userSet aggregator.ResourceMap[uint32, *identityV1.User],
) {
	for _, t := range tenants {
		if t.GetAdminUserId() > 0 {
			userSet[t.GetAdminUserId()] = nil
		}
	}
}

func (s *TenantService) fetchRelationInfo(
	ctx context.Context,
	userSet aggregator.ResourceMap[uint32, *identityV1.User],
) error {
	if len(userSet) > 0 {
		userIds := make([]uint32, 0, len(userSet))
		for id := range userSet {
			userIds = append(userIds, id)
		}

		users, err := s.userRepo.ListUsersByIds(ctx, userIds)
		if err != nil {
			s.log.Errorf(context.Background(), "query users err: %v", err)
			return err
		}

		for _, u := range users {
			userSet[u.GetId()] = u
		}
	}

	return nil
}

func (s *TenantService) bindRelations(
	tenants []*identityV1.Tenant,
	userSet aggregator.ResourceMap[uint32, *identityV1.User],
) {
	aggregator.Populate(
		tenants,
		userSet,
		func(ou *identityV1.Tenant) uint32 { return ou.GetAdminUserId() },
		func(ou *identityV1.Tenant, r *identityV1.User) {
			ou.AdminUserName = r.Username
		},
	)
}

func (s *TenantService) enrichRelations(ctx context.Context, tenants []*identityV1.Tenant) error {
	var userSet = make(aggregator.ResourceMap[uint32, *identityV1.User])
	s.extractRelationIDs(tenants, userSet)
	if err := s.fetchRelationInfo(ctx, userSet); err != nil {
		return err
	}
	s.bindRelations(tenants, userSet)

	// 回填 member_count：按各租户 ID 批量统计用户数
	tenantIDs := make([]uint32, 0, len(tenants))
	for _, t := range tenants {
		if t != nil && t.Id != nil && *t.Id > 0 {
			tenantIDs = append(tenantIDs, *t.Id)
		}
	}
	counts, err := s.userRepo.CountByTenantIDs(ctx, tenantIDs)
	if err != nil {
		s.log.Errorf(ctx, "enrich member_count failed: %s", err.Error())
	} else {
		for _, t := range tenants {
			if t != nil && t.Id != nil {
				if cnt, ok := counts[*t.Id]; ok {
					val := int32(cnt)
					t.MemberCount = &val
				}
			}
		}
	}

	return nil
}

func (s *TenantService) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListTenantResponse, error) {
	resp, err := s.tenantRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	_ = s.enrichRelations(ctx, resp.Items)

	return resp, nil
}

func (s *TenantService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.CountTenantResponse, error) {
	count, err := s.tenantRepo.Count(ctx, req)
	if err != nil {
		return nil, err
	}

	return &identityV1.CountTenantResponse{
		Count: uint64(count),
	}, nil
}

func (s *TenantService) Get(ctx context.Context, req *identityV1.GetTenantRequest) (*identityV1.Tenant, error) {
	resp, err := s.tenantRepo.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	fakeItems := []*identityV1.Tenant{resp}
	_ = s.enrichRelations(ctx, fakeItems)

	return resp, nil
}

func (s *TenantService) Create(ctx context.Context, req *identityV1.CreateTenantRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	if _, err = s.tenantRepo.Create(ctx, req.Data); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *TenantService) Update(ctx context.Context, req *identityV1.UpdateTenantRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.Id = trans.Ptr(req.GetId())

	req.Data.UpdatedBy = trans.Ptr(operator.UserId)
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	if err = s.tenantRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *TenantService) Delete(ctx context.Context, req *identityV1.DeleteTenantRequest) (*emptypb.Empty, error) {
	if err := s.tenantRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *TenantService) TenantExists(ctx context.Context, req *identityV1.TenantExistsRequest) (*identityV1.TenantExistsResponse, error) {
	return s.tenantRepo.TenantExists(ctx, req)
}

// CreateTenantWithAdminUser 创建租户及其管理员用户
func (s *TenantService) CreateTenantWithAdminUser(ctx context.Context, req *identityV1.CreateTenantWithAdminUserRequest) (*emptypb.Empty, error) {
	if req.Tenant == nil || req.User == nil {
		s.log.Error(ctx, "invalid parameter: tenant or user is nil", req)
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Tenant.CreatedBy = trans.Ptr(operator.UserId)
	req.User.CreatedBy = trans.Ptr(operator.UserId)

	// Check if tenant code or name already exists
	// 此前只检查 err、丢弃 Exist：重复 code/name 不会被拦，直到 DB 唯一约束报原始 500。
	tenantExistsResp, err := s.tenantRepo.TenantExists(ctx, &identityV1.TenantExistsRequest{
		Code: req.GetTenant().GetCode(),
		Name: req.GetTenant().GetName(),
	})
	if err != nil {
		s.log.Errorf(ctx, "check tenant exists err: %v", err)
		return nil, err
	}
	if tenantExistsResp.GetExist() {
		return nil, adminV1.ErrorBadRequest("tenant with given code or name already exists")
	}

	// 注意：此处不再做 admin username 的全局查重。
	// username 仅在 (tenant_id, username) 维度唯一（见 ent/schema/user.go 唯一索引），
	// 不同租户允许使用相同 username。此前按 username 跨租户全局查重会误判冲突、
	// 且在平台上下文(tid=0)下跨租户泄露用户名存在性。租户内唯一性由 DB 唯一索引保证。

	tx, cleanup, err := s.tenantRepo.BeginTx(ctx)
	if err != nil {
		s.log.Errorf(ctx, "begin tx err: %v", err)
		return nil, err
	}
	defer func() {
		if cleanup != nil {
			cleanup()
		}

		if err == nil {
			_ = s.authorizer.ResetPolicies(ctx)
		}
	}()

	// Create tenant
	var tenant *identityV1.Tenant
	if tenant, err = s.tenantRepo.CreateWithTx(ctx, tx, req.Tenant); err != nil {
		s.log.Errorf(ctx, "create tenant err: %v", err)
		return nil, err
	}

	req.User.TenantId = tenant.Id

	// copy tenant manager role to tenant
	var role *permissionV1.Role
	if role, err = s.roleRepo.CreateTenantRoleFromTemplate(ctx, tx, tenant.GetId(), operator.GetUserId()); err != nil {
		s.log.Errorf(ctx, "copy tenant admin role template to tenant err: %v", err)
		return nil, err
	}

	// Create tenant admin user
	var adminUser *identityV1.User
	req.User.RoleId = role.Id
	//req.User.Status = identityV1.User_NORMAL.Enum()
	if adminUser, err = s.userRepo.CreateWithTx(ctx, tx, req.User); err != nil {
		s.log.Errorf(ctx, "create tenant admin user err: %v", err)
		return nil, err
	}

	// Create user credential
	if err = s.userCredentialsRepo.CreateWithTx(ctx, tx, &authenticationV1.UserCredential{
		UserId:         adminUser.Id,
		TenantId:       tenant.Id,
		IdentityType:   authenticationV1.UserCredential_USERNAME.Enum(),
		Identifier:     adminUser.Username,
		CredentialType: authenticationV1.UserCredential_PASSWORD_HASH.Enum(),
		Credential:     trans.Ptr(req.GetPassword()),
		IsPrimary:      trans.Ptr(true),
		Status:         authenticationV1.UserCredential_ENABLED.Enum(),
	}); err != nil {
		s.log.Errorf(ctx, "create tenant admin user credential err: %v", err)
		return nil, err
	}

	// assign admin user id to tenant
	if err = s.tenantRepo.AssignTenantAdmin(ctx, tx, *tenant.Id, *adminUser.Id); err != nil {
		s.log.Errorf(ctx, "assign admin user id to tenant err: %v", err)
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// GetUsage 查询租户用量与配额
func (s *TenantService) GetUsage(ctx context.Context, req *identityV1.GetTenantUsageRequest) (*identityV1.TenantUsage, error) {
	if req == nil || req.GetId() == 0 {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}
	return s.tenantUsageRepo.GetUsage(ctx, req.GetId())
}

// CleanupData 清理租户数据（保留租户记录，状态改为 OFF）
func (s *TenantService) CleanupData(ctx context.Context, req *identityV1.CleanupTenantDataRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}
	if err := s.tenantUsageRepo.CleanupTenantData(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
