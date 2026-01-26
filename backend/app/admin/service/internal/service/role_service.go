package service

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/aggregator"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	userV1 "go-wind-admin/api/gen/go/user/service/v1"

	"go-wind-admin/pkg/authorizer"
	"go-wind-admin/pkg/constants"
	appViewer "go-wind-admin/pkg/entgo/viewer"
	"go-wind-admin/pkg/middleware/auth"
)

type RoleService struct {
	adminV1.RoleServiceHTTPServer

	log *log.Helper

	authorizer *authorizer.Authorizer

	roleRepo   *data.RoleRepo
	tenantRepo *data.TenantRepo
}

func NewRoleService(
	ctx *bootstrap.Context,
	authorizer *authorizer.Authorizer,
	roleRepo *data.RoleRepo,
	tenantRepo *data.TenantRepo,
) *RoleService {
	svc := &RoleService{
		log:        ctx.NewLoggerHelper("role/service/admin-service"),
		authorizer: authorizer,
		roleRepo:   roleRepo,
		tenantRepo: tenantRepo,
	}

	svc.init()

	return svc
}

func (s *RoleService) init() {
	ctx := appViewer.NewSystemViewerContext(context.Background())
	if count, _ := s.roleRepo.Count(ctx, []func(s *sql.Selector){}); count == 0 {
		_ = s.createDefaultRoles(ctx)
	}
}

func (s *RoleService) extractRelationIDs(
	roles []*userV1.Role,
	tenantSet aggregator.ResourceMap[uint32, *userV1.Tenant],
) {
	for _, p := range roles {
		if p.GetTenantId() > 0 {
			tenantSet[p.GetTenantId()] = nil
		}
	}
}

func (s *RoleService) fetchRelationInfo(
	ctx context.Context,
	tenantSet aggregator.ResourceMap[uint32, *userV1.Tenant],
) error {
	if len(tenantSet) > 0 {
		tenantIds := make([]uint32, 0, len(tenantSet))
		for id := range tenantSet {
			tenantIds = append(tenantIds, id)
		}

		tenants, err := s.tenantRepo.ListTenantsByIds(ctx, tenantIds)
		if err != nil {
			s.log.Errorf("query tenants err: %v", err)
			return err
		}

		for _, tenant := range tenants {
			tenantSet[tenant.GetId()] = tenant
		}
	}

	return nil
}

func (s *RoleService) populateRelationInfos(
	roles []*userV1.Role,
	tenantSet aggregator.ResourceMap[uint32, *userV1.Tenant],
) {
	aggregator.Populate(
		roles,
		tenantSet,
		func(ou *userV1.Role) uint32 { return ou.GetTenantId() },
		func(ou *userV1.Role, r *userV1.Tenant) {
			ou.TenantName = r.Name
		},
	)
}

func (s *RoleService) List(ctx context.Context, req *paginationV1.PagingRequest) (*userV1.ListRoleResponse, error) {
	resp, err := s.roleRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	var tenantSet = make(aggregator.ResourceMap[uint32, *userV1.Tenant])
	s.extractRelationIDs(resp.Items, tenantSet)
	_ = s.fetchRelationInfo(ctx, tenantSet)
	s.populateRelationInfos(resp.Items, tenantSet)

	return resp, nil
}

func (s *RoleService) Get(ctx context.Context, req *userV1.GetRoleRequest) (*userV1.Role, error) {
	resp, err := s.roleRepo.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	fakeItems := []*userV1.Role{resp}
	var tenantSet = make(aggregator.ResourceMap[uint32, *userV1.Tenant])
	s.extractRelationIDs(fakeItems, tenantSet)
	_ = s.fetchRelationInfo(ctx, tenantSet)
	s.populateRelationInfos(fakeItems, tenantSet)

	return resp, nil
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

	if operator.GetTenantId() == 0 {
		req.Data.IsSystem = nil
	}

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
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	if operator.GetTenantId() == 0 {
		req.Data.IsSystem = nil
	}

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

	for _, d := range constants.DefaultRoles {
		err = s.roleRepo.Create(ctx, &userV1.CreateRoleRequest{
			Data: d,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
