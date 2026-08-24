package service

import (
	"context"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"

	"go-wind-admin/pkg/middleware/auth"
)

type PlanService struct {
	adminV1.PlanServiceHTTPServer

	log      *bLogger.Helper
	planRepo *data.PlanRepo
}

func NewPlanService(
	ctx *bootstrap.Context,
	planRepo *data.PlanRepo,
) *PlanService {
	return &PlanService{
		log:      ctx.NewLoggerHelper("plan/service/admin-service"),
		planRepo: planRepo,
	}
}

func (s *PlanService) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListPlanResponse, error) {
	return s.planRepo.List(ctx, req)
}

func (s *PlanService) Get(ctx context.Context, req *identityV1.GetPlanRequest) (*identityV1.Plan, error) {
	return s.planRepo.Get(ctx, req)
}

func (s *PlanService) Create(ctx context.Context, req *identityV1.CreatePlanRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	if err = s.planRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *PlanService) Update(ctx context.Context, req *identityV1.UpdatePlanRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
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

	if err = s.planRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *PlanService) Delete(ctx context.Context, req *identityV1.DeletePlanRequest) (*emptypb.Empty, error) {
	if err := s.planRepo.Delete(ctx, req.GetId()); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
