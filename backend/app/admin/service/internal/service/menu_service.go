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
	resourceV1 "go-wind-admin/api/gen/go/resource/service/v1"

	"go-wind-admin/pkg/constants"
	appViewer "go-wind-admin/pkg/entgo/viewer"
	"go-wind-admin/pkg/middleware/auth"
)

type MenuService struct {
	adminV1.MenuServiceHTTPServer

	log *log.Helper

	menuRepo *data.MenuRepo
}

func NewMenuService(ctx *bootstrap.Context, menuRepo *data.MenuRepo) *MenuService {
	svc := &MenuService{
		log:      ctx.NewLoggerHelper("menu/service/admin-service"),
		menuRepo: menuRepo,
	}

	svc.init()

	return svc
}

func (s *MenuService) init() {
	ctx := appViewer.NewSystemViewerContext(context.Background())
	if count, _ := s.menuRepo.Count(ctx, nil); count == 0 {
		_ = s.createDefaultMenus(ctx)
	}
}

func (s *MenuService) List(ctx context.Context, req *paginationV1.PagingRequest) (*resourceV1.ListMenuResponse, error) {
	ret, err := s.menuRepo.List(ctx, req, false)
	if err != nil {

		return nil, err
	}

	return ret, nil
}

func (s *MenuService) Get(ctx context.Context, req *resourceV1.GetMenuRequest) (*resourceV1.Menu, error) {
	ret, err := s.menuRepo.Get(ctx, req)
	if err != nil {

		return nil, err
	}

	return ret, nil
}

func (s *MenuService) Create(ctx context.Context, req *resourceV1.CreateMenuRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	if err = s.menuRepo.Create(ctx, req); err != nil {

		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *MenuService) Update(ctx context.Context, req *resourceV1.UpdateMenuRequest) (*emptypb.Empty, error) {
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

	if err = s.menuRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *MenuService) Delete(ctx context.Context, req *resourceV1.DeleteMenuRequest) (*emptypb.Empty, error) {
	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.OperatorId = trans.Ptr(operator.UserId)

	if err := s.menuRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *MenuService) createDefaultMenus(ctx context.Context) error {
	for _, m := range constants.DefaultMenus {
		if err := s.menuRepo.Create(ctx, &resourceV1.CreateMenuRequest{Data: m}); err != nil {
			s.log.Errorf("create default menu err: %v", err)
			return err
		}
	}
	return nil
}
