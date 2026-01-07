package service

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	pagination "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	userV1 "go-wind-admin/api/gen/go/user/service/v1"

	"go-wind-admin/pkg/middleware/auth"
)

type RoleService struct {
	adminV1.RoleServiceHTTPServer

	log *log.Helper

	authorizer *data.Authorizer

	roleRepo *data.RoleRepo

	membershipOrgUnitRepo *data.MembershipOrgUnitRepo
	membershipRepo        *data.MembershipRepo
}

func NewRoleService(
	ctx *bootstrap.Context,
	authorizer *data.Authorizer,
	roleRepo *data.RoleRepo,
	membershipOrgUnitRepo *data.MembershipOrgUnitRepo,
	membershipRepo *data.MembershipRepo,
) *RoleService {
	svc := &RoleService{
		log:                   ctx.NewLoggerHelper("role/service/admin-service"),
		authorizer:            authorizer,
		roleRepo:              roleRepo,
		membershipOrgUnitRepo: membershipOrgUnitRepo,
		membershipRepo:        membershipRepo,
	}

	svc.init()

	return svc
}

func (s *RoleService) init() {
	ctx := context.Background()
	if count, _ := s.roleRepo.Count(ctx, []func(s *sql.Selector){}); count == 0 {
		_ = s.createDefaultRoles(ctx)
	}
}

func (s *RoleService) List(ctx context.Context, req *pagination.PagingRequest) (*userV1.ListRoleResponse, error) {
	return s.roleRepo.List(ctx, req)
}

func (s *RoleService) Get(ctx context.Context, req *userV1.GetRoleRequest) (*userV1.Role, error) {
	return s.roleRepo.Get(ctx, req)
}

func (s *RoleService) Create(ctx context.Context, req *userV1.CreateRoleRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	if err = s.roleRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	if err = s.authorizer.ResetPolicies(ctx); err != nil {
		s.log.Errorf("reset policies error: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *RoleService) Update(ctx context.Context, req *userV1.UpdateRoleRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		s.log.Errorf("get operator from context error: %v", err)
		return nil, err
	}

	req.Data.UpdatedBy = trans.Ptr(operator.UserId)

	if err = s.roleRepo.Update(ctx, req); err != nil {
		s.log.Errorf("update role error: %v", err)
		return nil, err
	}

	if err = s.authorizer.ResetPolicies(ctx); err != nil {
		s.log.Errorf("reset policies error: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *RoleService) Delete(ctx context.Context, req *userV1.DeleteRoleRequest) (*emptypb.Empty, error) {
	var err error

	if err = s.roleRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	if err = s.authorizer.ResetPolicies(ctx); err != nil {
		s.log.Errorf("reset policies error: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *RoleService) GetRoleCodesByRoleIds(ctx context.Context, req *userV1.GetRoleCodesByRoleIdsRequest) (*userV1.GetRoleCodesByRoleIdsResponse, error) {
	ids, err := s.roleRepo.ListRoleCodesByRoleIds(ctx, req.GetRoleIds())
	if err != nil {
		return nil, err
	}

	return &userV1.GetRoleCodesByRoleIdsResponse{
		RoleCodes: ids,
	}, nil
}

func (s *RoleService) GetRolesByRoleCodes(ctx context.Context, req *userV1.GetRolesByRoleCodesRequest) (*userV1.ListRoleResponse, error) {
	roles, err := s.roleRepo.ListRolesByRoleCodes(ctx, req.GetRoleCodes())
	if err != nil {
		return nil, err
	}

	return &userV1.ListRoleResponse{
		Items: roles,
		Total: uint64(len(roles)),
	}, nil
}

func (s *RoleService) GetRolesByRoleIds(ctx context.Context, req *userV1.GetRolesByRoleIdsRequest) (*userV1.ListRoleResponse, error) {
	roles, err := s.roleRepo.ListRolesByRoleIds(ctx, req.GetRoleIds())
	if err != nil {
		return nil, err
	}

	return &userV1.ListRoleResponse{
		Items: roles,
		Total: uint64(len(roles)),
	}, nil
}

// createDefaultRoles 创建默认角色(包括超级管理员)
func (s *RoleService) createDefaultRoles(ctx context.Context) error {
	var err error

	defaultRoles := []*userV1.CreateRoleRequest{
		{
			Data: &userV1.Role{
				Id:          trans.Ptr(uint32(1)),
				Name:        trans.Ptr("平台管理员"),
				Code:        trans.Ptr("platform_admin"),
				Status:      trans.Ptr(userV1.Role_ON),
				Description: trans.Ptr("拥有系统所有功能的操作权限，可管理租户、用户、角色及所有资源"),
				IsProtected: trans.Ptr(true),
				Permissions: []uint32{1},
			},
		},
	}

	for _, roleReq := range defaultRoles {
		err = s.roleRepo.Create(ctx, roleReq)
		if err != nil {
			return err
		}
	}

	return nil
}
