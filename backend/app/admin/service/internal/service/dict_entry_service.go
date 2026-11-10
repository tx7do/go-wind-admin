package service

import (
	"context"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	dictV1 "go-wind-admin/api/gen/go/dict/service/v1"

	"go-wind-admin/pkg/middleware/auth"
)

type DictEntryService struct {
	adminV1.DictEntryServiceHTTPServer

	log *bLogger.Helper

	dictEntryRepo *data.DictEntryRepo
	dictTypeRepo  *data.DictTypeRepo
}

func NewDictEntryService(
	ctx *bootstrap.Context,
	dictEntryRepo *data.DictEntryRepo,
	dictTypeRepo *data.DictTypeRepo,
) *DictEntryService {
	return &DictEntryService{
		log:           ctx.NewLoggerHelper("dict-entry/service/admin-service"),
		dictEntryRepo: dictEntryRepo,
		dictTypeRepo:  dictTypeRepo,
	}
}

// validateDictEntryType 字典类型归属校验（P2 修复）：dict_entry 行级隔离由
// TenantPrivacy 承担，但 type_id 此前不做归属校验——可把条目挂到他租户/平台
// 的 dict_type id 上（完整性破坏 + id 存在性探测，读取侧 edge 水合被租户过滤
// 形成悬挂引用）。经 tenant 过滤的 Get 查 0 行即拒绝。
func (s *DictEntryService) validateDictEntryType(ctx context.Context, typeId int64) error {
	if typeId == 0 {
		return nil
	}
	_, err := s.dictTypeRepo.Get(ctx, &dictV1.GetDictTypeRequest{
		QueryBy: &dictV1.GetDictTypeRequest_Id{Id: typeId},
	})
	if err != nil {
		return dictV1.ErrorBadRequest("dict type not found or not accessible")
	}
	return nil
}

func (s *DictEntryService) List(ctx context.Context, req *paginationV1.PagingRequest) (*dictV1.ListDictEntryResponse, error) {
	return s.dictEntryRepo.List(ctx, req)
}

func (s *DictEntryService) Create(ctx context.Context, req *dictV1.CreateDictEntryRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, dictV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err = s.validateDictEntryType(ctx, req.Data.GetTypeId()); err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	if err = s.dictEntryRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *DictEntryService) Update(ctx context.Context, req *dictV1.UpdateDictEntryRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, dictV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	// 仅在请求显式携带 typeId 时校验（与 repo SetDictTypeID 口径一致）
	if req.Data.TypeId != nil {
		if err = s.validateDictEntryType(ctx, req.Data.GetTypeId()); err != nil {
			return nil, err
		}
	}

	req.Data.Id = trans.Ptr(req.GetId())

	req.Data.UpdatedBy = trans.Ptr(operator.UserId)
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	if err = s.dictEntryRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *DictEntryService) Delete(ctx context.Context, req *dictV1.DeleteDictEntryRequest) (*emptypb.Empty, error) {
	if err := s.dictEntryRepo.BatchDelete(ctx, req.GetIds()); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *DictEntryService) ListByTypeCode(ctx context.Context, req *dictV1.ListDictEntryByTypeCodeRequest) (*dictV1.ListDictEntryByTypeCodeResponse, error) {
	if req.GetTypeCode() == "" {
		return nil, dictV1.ErrorBadRequest("invalid parameter")
	}

	return s.dictEntryRepo.ListByTypeCode(ctx, req)
}
