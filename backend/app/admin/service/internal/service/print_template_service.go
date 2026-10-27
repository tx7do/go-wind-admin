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
	printV1 "go-wind-admin/api/gen/go/print/service/v1"

	"github.com/tx7do/go-crud/viewer"
	"go-wind-admin/pkg/constants"
	"go-wind-admin/pkg/middleware/auth"
)

type PrintTemplateService struct {
	adminV1.PrintTemplateServiceHTTPServer

	log             *bLogger.Helper
	printTemplateRepo *data.PrintTemplateRepo
}

func NewPrintTemplateService(
	ctx *bootstrap.Context,
	printTemplateRepo *data.PrintTemplateRepo,
) *PrintTemplateService {
	svc := &PrintTemplateService{
		log:               ctx.NewLoggerHelper("print-template/service/admin-service"),
		printTemplateRepo: printTemplateRepo,
	}

	svc.init()

	return svc
}

// init 播种内置打印模板（通用单据/标签基线版式）。与其他默认数据一致，
// 在服务构造（进程启动）时执行一次；SeedDefaults 按编码缺一补一、不覆盖既有行。
func (s *PrintTemplateService) init() {
	ctx := viewer.WithSystemContext(context.Background())
	if err := s.printTemplateRepo.SeedDefaults(ctx, constants.DefaultPrintTemplates); err != nil {
		s.log.Errorf(ctx, "seed default print templates failed: %s", err.Error())
	}
}

func (s *PrintTemplateService) List(ctx context.Context, req *paginationV1.PagingRequest) (*printV1.ListPrintTemplateResponse, error) {
	return s.printTemplateRepo.List(ctx, req)
}

func (s *PrintTemplateService) Get(ctx context.Context, req *printV1.GetPrintTemplateRequest) (*printV1.PrintTemplate, error) {
	return s.printTemplateRepo.Get(ctx, req)
}

func (s *PrintTemplateService) Create(ctx context.Context, req *printV1.CreatePrintTemplateRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	if err = s.printTemplateRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *PrintTemplateService) Update(ctx context.Context, req *printV1.UpdatePrintTemplateRequest) (*emptypb.Empty, error) {
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

	if err = s.printTemplateRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *PrintTemplateService) Delete(ctx context.Context, req *printV1.DeletePrintTemplateRequest) (*emptypb.Empty, error) {
	if err := s.printTemplateRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
