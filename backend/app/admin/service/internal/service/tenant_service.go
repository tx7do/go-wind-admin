package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	userV1 "go-wind-admin/api/gen/go/user/service/v1"

	"go-wind-admin/pkg/middleware/auth"
	"go-wind-admin/pkg/utils/name_set"
)

type TenantService struct {
	adminV1.TenantServiceHTTPServer

	log *log.Helper

	tenantRepo          *data.TenantRepo
	userRepo            data.UserRepo
	userCredentialsRepo *data.UserCredentialRepo
	roleRepo            *data.RoleRepo

	authorizer *data.Authorizer
}

func NewTenantService(
	ctx *bootstrap.Context,
	tenantRepo *data.TenantRepo,
	userRepo data.UserRepo,
	userCredentialsRepo *data.UserCredentialRepo,
	roleRepo *data.RoleRepo,
	authorizer *data.Authorizer,
) *TenantService {
	return &TenantService{
		log:                 ctx.NewLoggerHelper("tenant/service/admin-service"),
		tenantRepo:          tenantRepo,
		userRepo:            userRepo,
		userCredentialsRepo: userCredentialsRepo,
		roleRepo:            roleRepo,
		authorizer:          authorizer,
	}
}

func (s *TenantService) List(ctx context.Context, req *paginationV1.PagingRequest) (*userV1.ListTenantResponse, error) {
	resp, err := s.tenantRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	var userSet = make(name_set.UserNameSetMap)

	for _, v := range resp.Items {
		if v.AdminUserId != nil {
			userSet[v.GetAdminUserId()] = nil
		}
	}

	QueryUserInfoFromRepo(ctx, s.userRepo, &userSet)

	for _, v := range resp.Items {
		if v.AdminUserId != nil {
			if userInfo, ok := userSet[v.GetAdminUserId()]; ok && userInfo != nil {
				v.AdminUserName = &userInfo.UserName
			}
		}
	}

	return resp, nil
}

func (s *TenantService) Get(ctx context.Context, req *userV1.GetTenantRequest) (*userV1.Tenant, error) {
	resp, err := s.tenantRepo.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.AdminUserId != nil {
		userResp, err := s.userRepo.Get(ctx, &userV1.GetUserRequest{
			QueryBy: &userV1.GetUserRequest_Id{
				Id: resp.GetAdminUserId(),
			},
		})
		if err != nil {
			s.log.Errorf("failed to get admin user info: %v", err)
		} else {
			resp.AdminUserName = userResp.Username
		}
	}

	return resp, nil
}

func (s *TenantService) Create(ctx context.Context, req *userV1.CreateTenantRequest) (*emptypb.Empty, error) {
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

func (s *TenantService) Update(ctx context.Context, req *userV1.UpdateTenantRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.UpdatedBy = trans.Ptr(operator.UserId)

	if err = s.tenantRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *TenantService) Delete(ctx context.Context, req *userV1.DeleteTenantRequest) (*emptypb.Empty, error) {
	if err := s.tenantRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *TenantService) TenantExists(ctx context.Context, req *userV1.TenantExistsRequest) (*userV1.TenantExistsResponse, error) {
	return s.tenantRepo.TenantExists(ctx, req)
}

// CreateTenantWithAdminUser 创建租户及其管理员用户
func (s *TenantService) CreateTenantWithAdminUser(ctx context.Context, req *adminV1.CreateTenantWithAdminUserRequest) (*emptypb.Empty, error) {
	if req.Tenant == nil || req.User == nil {
		s.log.Error("invalid parameter: tenant or user is nil", req)
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Tenant.CreatedBy = trans.Ptr(operator.UserId)
	req.User.CreatedBy = trans.Ptr(operator.UserId)

	// Check if tenant code or admin username already exists
	if _, err = s.tenantRepo.TenantExists(ctx, &userV1.TenantExistsRequest{
		Code: req.GetTenant().GetCode(),
		Name: req.GetTenant().GetName(),
	}); err != nil {
		s.log.Errorf("check tenant code exists err: %v", err)
		return nil, err
	}

	// Check if admin user exists
	if _, err = s.userRepo.UserExists(ctx, &userV1.UserExistsRequest{
		QueryBy: &userV1.UserExistsRequest_Username{Username: req.GetUser().GetUsername()},
	}); err != nil {
		s.log.Errorf("check admin user exists err: %v", err)
		return nil, err
	}

	tx, cleanup, err := s.tenantRepo.BeginTx(ctx)
	if err != nil {
		s.log.Errorf("begin tx err: %v", err)
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
	var tenant *userV1.Tenant
	if tenant, err = s.tenantRepo.CreateWithTx(ctx, tx, req.Tenant); err != nil {
		s.log.Errorf("create tenant err: %v", err)
		return nil, err
	}

	req.User.TenantId = tenant.Id

	// copy tenant manager role to tenant
	var role *userV1.Role
	if role, err = s.roleRepo.CreateTenantRoleFromTemplate(ctx, tx, tenant.GetId(), operator.GetUserId()); err != nil {
		s.log.Errorf("copy tenant admin role template to tenant err: %v", err)
		return nil, err
	}

	// Create tenant admin user
	var adminUser *userV1.User
	req.User.RoleId = role.Id
	//req.User.Status = userV1.User_NORMAL.Enum()
	if adminUser, err = s.userRepo.CreateWithTx(ctx, tx, req.User); err != nil {
		s.log.Errorf("create tenant admin user err: %v", err)
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
		s.log.Errorf("create tenant admin user credential err: %v", err)
		return nil, err
	}

	// assign admin user id to tenant
	if err = s.tenantRepo.AssignTenantAdmin(ctx, tx, *tenant.Id, *adminUser.Id); err != nil {
		s.log.Errorf("assign admin user id to tenant err: %v", err)
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
