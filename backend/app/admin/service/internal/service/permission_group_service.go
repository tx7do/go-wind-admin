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
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"

	"go-wind-admin/pkg/constants"
	"go-wind-admin/pkg/middleware/auth"
)

type PermissionGroupService struct {
	adminV1.PermissionGroupServiceHTTPServer

	log *log.Helper

	permissionGroupRepo *data.PermissionGroupRepo
	permissionRepo      *data.PermissionRepo
}

func NewPermissionGroupService(
	ctx *bootstrap.Context,
	permissionGroupRepo *data.PermissionGroupRepo,
	permissionRepo *data.PermissionRepo,
) *PermissionGroupService {
	svc := &PermissionGroupService{
		log:                 ctx.NewLoggerHelper("permission-group/service/admin-service"),
		permissionGroupRepo: permissionGroupRepo,
		permissionRepo:      permissionRepo,
	}

	svc.init()

	return svc
}

func (s *PermissionGroupService) init() {
	ctx := context.Background()
	if count, _ := s.permissionGroupRepo.Count(ctx, []func(s *sql.Selector){}); count == 0 {
		_ = s.createDefaultPermissionGroups(ctx)
	}
}

func (s *PermissionGroupService) List(ctx context.Context, req *pagination.PagingRequest) (*permissionV1.ListPermissionGroupResponse, error) {
	return s.permissionGroupRepo.List(ctx, req, true)
}

func (s *PermissionGroupService) Get(ctx context.Context, req *permissionV1.GetPermissionGroupRequest) (*permissionV1.PermissionGroup, error) {
	return s.permissionGroupRepo.Get(ctx, req)
}

func (s *PermissionGroupService) Create(ctx context.Context, req *permissionV1.CreatePermissionGroupRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	if _, err = s.permissionGroupRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *PermissionGroupService) Update(ctx context.Context, req *permissionV1.UpdatePermissionGroupRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.UpdatedBy = trans.Ptr(operator.UserId)

	if err = s.permissionGroupRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *PermissionGroupService) Delete(ctx context.Context, req *permissionV1.DeletePermissionGroupRequest) (*emptypb.Empty, error) {
	var err error

	if err = s.permissionGroupRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	if err = s.permissionRepo.Delete(ctx, &permissionV1.DeletePermissionRequest{
		DeleteBy: &permissionV1.DeletePermissionRequest_GroupId{GroupId: req.GetId()},
	}); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *PermissionGroupService) createDefaultPermissionGroups(ctx context.Context) error {
	var err error
	for _, d := range constants.DefaultPermissionGroups {
		if _, err = s.permissionGroupRepo.Create(ctx, &permissionV1.CreatePermissionGroupRequest{
			Data: d,
		}); err != nil {
			s.log.Errorf("create default permission group error: %v", err)
			return err
		}
	}

	return nil
}
