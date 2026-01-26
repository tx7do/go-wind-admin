package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/aggregator"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	userV1 "go-wind-admin/api/gen/go/user/service/v1"

	"go-wind-admin/pkg/middleware/auth"
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

func (s *PositionService) extractRelationIDs(
	positions []*userV1.Position,
	orgUnitSet aggregator.ResourceMap[uint32, *userV1.OrgUnit],
) {
	for _, p := range positions {
		if p.GetOrgUnitId() > 0 {
			orgUnitSet[p.GetOrgUnitId()] = nil
		}
	}
}

func (s *PositionService) fetchRelationInfo(
	ctx context.Context,
	orgUnitSet aggregator.ResourceMap[uint32, *userV1.OrgUnit],
) error {
	if len(orgUnitSet) > 0 {
		orgUnitIds := make([]uint32, 0, len(orgUnitSet))
		for id := range orgUnitSet {
			orgUnitIds = append(orgUnitIds, id)
		}

		orgUnits, err := s.orgUnitRepo.ListOrgUnitsByIds(ctx, orgUnitIds)
		if err != nil {
			s.log.Errorf("query orgUnits err: %v", err)
			return err
		}

		for _, orgUnit := range orgUnits {
			orgUnitSet[orgUnit.GetId()] = orgUnit
		}
	}

	return nil
}

func (s *PositionService) populateRelationInfos(
	positions []*userV1.Position,
	orgUnitSet aggregator.ResourceMap[uint32, *userV1.OrgUnit],
) {
	aggregator.Populate(
		positions,
		orgUnitSet,
		func(ou *userV1.Position) uint32 { return ou.GetOrgUnitId() },
		func(ou *userV1.Position, org *userV1.OrgUnit) {
			ou.OrgUnitName = org.Name
		},
	)
}

func (s *PositionService) List(ctx context.Context, req *paginationV1.PagingRequest) (*userV1.ListPositionResponse, error) {
	resp, err := s.positionRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	var orgUnitSet = make(aggregator.ResourceMap[uint32, *userV1.OrgUnit])
	s.extractRelationIDs(resp.Items, orgUnitSet)
	_ = s.fetchRelationInfo(ctx, orgUnitSet)
	s.populateRelationInfos(resp.Items, orgUnitSet)

	return resp, nil
}

func (s *PositionService) Get(ctx context.Context, req *userV1.GetPositionRequest) (*userV1.Position, error) {
	resp, err := s.positionRepo.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	fakeItems := []*userV1.Position{resp}

	var orgUnitSet = make(aggregator.ResourceMap[uint32, *userV1.OrgUnit])
	s.extractRelationIDs(fakeItems, orgUnitSet)
	_ = s.fetchRelationInfo(ctx, orgUnitSet)
	s.populateRelationInfos(fakeItems, orgUnitSet)

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
