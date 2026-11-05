package service

import (
	"context"
	"fmt"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/aggregator"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"

	"go-wind-admin/pkg/middleware/auth"
)

type OrgUnitService struct {
	adminV1.OrgUnitServiceHTTPServer

	log *bLogger.Helper

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

// extractRelationIDs 提取关联数据ID
func (s *OrgUnitService) extractRelationIDs(
	orgUnits []*identityV1.OrgUnit,
	userSet aggregator.ResourceMap[uint32, *identityV1.User],
) {
	for _, ou := range orgUnits {
		if ou == nil {
			continue
		}
		if id := ou.GetLeaderId(); id > 0 {
			userSet[id] = nil
		}
		if id := ou.GetContactUserId(); id > 0 {
			userSet[id] = nil
		}

		if len(ou.Children) > 0 {
			s.extractRelationIDs(ou.Children, userSet)
		}
	}
}

// fetchRelationInfo 获取关联数据
func (s *OrgUnitService) fetchRelationInfo(
	ctx context.Context,
	userSet aggregator.ResourceMap[uint32, *identityV1.User],
) error {
	if len(userSet) == 0 {
		return nil
	}

	userIds := make([]uint32, 0, len(userSet))
	for id := range userSet {
		userIds = append(userIds, id)
	}

	users, err := s.userRepo.ListUsersByIds(ctx, userIds)
	if err != nil {
		bLogger.GetLogger().Error(context.Background(), fmt.Sprintf("query users err: %v", err))
		return err
	}

	for _, user := range users {
		userSet[user.GetId()] = user
	}
	return nil
}

// bindRelations 绑定关联数据
func (s *OrgUnitService) bindRelations(
	orgUnits []*identityV1.OrgUnit,
	userSet aggregator.ResourceMap[uint32, *identityV1.User],
) {
	childrenFunc := func(ou *identityV1.OrgUnit) []*identityV1.OrgUnit { return ou.GetChildren() }

	// 回填 LeaderName
	aggregator.PopulateTree(
		orgUnits,
		userSet,
		func(ou *identityV1.OrgUnit) uint32 { return ou.GetLeaderId() },
		func(ou *identityV1.OrgUnit, user *identityV1.User) {
			ou.LeaderName = trans.Ptr(user.GetUsername())
		},
		childrenFunc,
	)

	// 回填 ContactUserName
	aggregator.PopulateTree(
		orgUnits,
		userSet,
		func(ou *identityV1.OrgUnit) uint32 { return ou.GetContactUserId() },
		func(ou *identityV1.OrgUnit, user *identityV1.User) {
			ou.ContactUserName = trans.Ptr(user.GetUsername())
		},
		childrenFunc,
	)
}

// enrichRelations 填充关联数据
func (s *OrgUnitService) enrichRelations(ctx context.Context, orgUnits []*identityV1.OrgUnit) error {
	var userSet = make(aggregator.ResourceMap[uint32, *identityV1.User])
	s.extractRelationIDs(orgUnits, userSet)
	if err := s.fetchRelationInfo(ctx, userSet); err != nil {
		return err
	}
	s.bindRelations(orgUnits, userSet)
	return nil
}

func (s *OrgUnitService) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListOrgUnitResponse, error) {
	resp, err := s.orgUnitRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	_ = s.enrichRelations(ctx, resp.Items)

	return resp, nil
}

func (s *OrgUnitService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.CountOrgUnitResponse, error) {
	count, err := s.orgUnitRepo.Count(ctx, req)
	if err != nil {
		return nil, err
	}

	return &identityV1.CountOrgUnitResponse{
		Count: uint64(count),
	}, nil
}

func (s *OrgUnitService) Get(ctx context.Context, req *identityV1.GetOrgUnitRequest) (*identityV1.OrgUnit, error) {
	resp, err := s.orgUnitRepo.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	fakeItems := []*identityV1.OrgUnit{resp}
	_ = s.enrichRelations(ctx, fakeItems)

	return resp, nil
}

// validateOrgUnitParent 父部门归属校验（P2 修复）：org_unit 行级隔离由
// TenantPrivacy 承担，但 parent_id 此前不做归属校验——本租户部门可挂到他租户
// 部门树下（数据范围 UNIT/UNIT_AND_CHILD 的 path 前缀语义被污染 + 外租户 id
// 存在性探测）。经 tenant 过滤的 Get 查 0 行即拒绝；与角色父子绑定同口径。
func (s *OrgUnitService) validateOrgUnitParent(ctx context.Context, parentId uint32) error {
	if parentId == 0 {
		return nil
	}
	_, err := s.orgUnitRepo.Get(ctx, &identityV1.GetOrgUnitRequest{
		QueryBy: &identityV1.GetOrgUnitRequest_Id{Id: parentId},
	})
	if err != nil {
		return adminV1.ErrorBadRequest("parent org unit not found or not accessible")
	}
	return nil
}

func (s *OrgUnitService) Create(ctx context.Context, req *identityV1.CreateOrgUnitRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err = s.validateOrgUnitParent(ctx, req.Data.GetParentId()); err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	if err = s.orgUnitRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *OrgUnitService) Update(ctx context.Context, req *identityV1.UpdateOrgUnitRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	// 仅在请求显式携带 parentId 时校验（repo SetNillableParentID 口径一致）
	if req.Data.ParentId != nil {
		if err = s.validateOrgUnitParent(ctx, req.Data.GetParentId()); err != nil {
			return nil, err
		}
	}

	req.Data.Id = trans.Ptr(req.GetId())

	req.Data.UpdatedBy = trans.Ptr(operator.UserId)
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	if err = s.orgUnitRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *OrgUnitService) Delete(ctx context.Context, req *identityV1.DeleteOrgUnitRequest) (*emptypb.Empty, error) {
	if err := s.orgUnitRepo.Delete(ctx, req); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
