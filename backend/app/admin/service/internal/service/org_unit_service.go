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

type OrgUnitService struct {
	adminV1.OrgUnitServiceHTTPServer

	log *log.Helper

	orgUnitRepo *data.OrgUnitRepo
	userRepo    data.UserRepo
}

func NewOrgUnitService(
	ctx *bootstrap.Context,
	organizationRepo *data.OrgUnitRepo,
	userRepo data.UserRepo,
) *OrgUnitService {
	return &OrgUnitService{
		log:         ctx.NewLoggerHelper("org-unit/service/admin-service"),
		orgUnitRepo: organizationRepo,
		userRepo:    userRepo,
	}
}

func (s *OrgUnitService) List(ctx context.Context, req *paginationV1.PagingRequest) (*userV1.ListOrgUnitResponse, error) {
	resp, err := s.orgUnitRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	var userSet = make(name_set.UserNameSetMap)

	InitOrgUnitNameSetMap(resp.Items, &userSet)

	QueryUserInfoFromRepo(ctx, s.userRepo, &userSet)

	FillOrgUnitInfo(resp.Items, &userSet)

	return resp, nil
}

func (s *OrgUnitService) Get(ctx context.Context, req *userV1.GetOrgUnitRequest) (*userV1.OrgUnit, error) {
	resp, err := s.orgUnitRepo.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.LeaderId != nil {
		manager, err := s.userRepo.Get(ctx, &userV1.GetUserRequest{
			QueryBy: &userV1.GetUserRequest_Id{
				Id: resp.GetLeaderId(),
			},
		})
		if err == nil && manager != nil {
			resp.LeaderName = manager.Nickname
		} else {
			s.log.Warnf("Get orgUnit leader user failed: %v", err)
		}
	}

	return resp, nil
}

func (s *OrgUnitService) Create(ctx context.Context, req *userV1.CreateOrgUnitRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	if err = s.orgUnitRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *OrgUnitService) Update(ctx context.Context, req *userV1.UpdateOrgUnitRequest) (*emptypb.Empty, error) {
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

	if err = s.orgUnitRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *OrgUnitService) Delete(ctx context.Context, req *userV1.DeleteOrgUnitRequest) (*emptypb.Empty, error) {
	if err := s.orgUnitRepo.Delete(ctx, req); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
