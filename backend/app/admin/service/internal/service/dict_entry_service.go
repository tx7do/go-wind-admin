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
	dictV1 "go-wind-admin/api/gen/go/dict/service/v1"

	"go-wind-admin/pkg/middleware/auth"
)

type DictEntryService struct {
	adminV1.DictEntryServiceHTTPServer

	log *log.Helper

	dictEntryRepo *data.DictEntryRepo
}

func NewDictEntryService(
	ctx *bootstrap.Context,
	dictEntryRepo *data.DictEntryRepo,
) *DictEntryService {
	return &DictEntryService{
		log:           ctx.NewLoggerHelper("dict-entry/service/admin-service"),
		dictEntryRepo: dictEntryRepo,
	}
}

func (s *DictEntryService) ListDictEntry(ctx context.Context, req *paginationV1.PagingRequest) (*dictV1.ListDictEntryResponse, error) {
	return s.dictEntryRepo.List(ctx, req)
}

func (s *DictEntryService) CreateDictEntry(ctx context.Context, req *dictV1.CreateDictEntryRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, dictV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	if err = s.dictEntryRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *DictEntryService) UpdateDictEntry(ctx context.Context, req *dictV1.UpdateDictEntryRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, dictV1.ErrorBadRequest("invalid parameter")
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

	if err = s.dictEntryRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *DictEntryService) DeleteDictEntry(ctx context.Context, req *dictV1.DeleteDictEntryRequest) (*emptypb.Empty, error) {
	if err := s.dictEntryRepo.BatchDelete(ctx, req.GetIds()); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
