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
	userV1 "go-wind-admin/api/gen/go/user/service/v1"

	"go-wind-admin/pkg/middleware/auth"
	"go-wind-admin/pkg/utils/name_set"
)

type PositionService struct {
	adminV1.PositionServiceHTTPServer

	log *log.Helper

	positionRepo *data.PositionRepo
	orgUnitRepo  *data.OrgUnitRepo
}

func NewPositionService(
	ctx *bootstrap.Context,
	positionRepo *data.PositionRepo,
	orgUnitRepo *data.OrgUnitRepo,
) *PositionService {
	return &PositionService{
		log:          ctx.NewLoggerHelper("position/service/admin-service"),
		positionRepo: positionRepo,
		orgUnitRepo:  orgUnitRepo,
	}
}

func (s *PositionService) List(ctx context.Context, req *paginationV1.PagingRequest) (*userV1.ListPositionResponse, error) {
	resp, err := s.positionRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	var deptSet = make(name_set.UserNameSetMap)
	var orgSet = make(name_set.UserNameSetMap)

	InitPositionNameSetMap(resp.Items, &orgSet, &deptSet)

	QueryOrgUnitInfoFromRepo(ctx, s.orgUnitRepo, &orgSet)

	FillPositionOrgUnitInfo(resp.Items, &orgSet)

	return resp, nil
}

func (s *PositionService) Get(ctx context.Context, req *userV1.GetPositionRequest) (*userV1.Position, error) {
	resp, err := s.positionRepo.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.OrgUnitId != nil {
		organization, err := s.orgUnitRepo.Get(ctx, &userV1.GetOrgUnitRequest{QueryBy: &userV1.GetOrgUnitRequest_Id{Id: resp.GetOrgUnitId()}})
		if err == nil && organization != nil {
			resp.OrgUnitName = organization.Name
		} else {
			s.log.Warnf("Get position organization failed: %v", err)
		}
	}

	return resp, nil
}

func (s *PositionService) Create(ctx context.Context, req *userV1.CreatePositionRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	if err = s.positionRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *PositionService) Update(ctx context.Context, req *userV1.UpdatePositionRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.UpdatedBy = trans.Ptr(operator.UserId)
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	if err = s.positionRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *PositionService) Delete(ctx context.Context, req *userV1.DeletePositionRequest) (*emptypb.Empty, error) {
	if err := s.positionRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
