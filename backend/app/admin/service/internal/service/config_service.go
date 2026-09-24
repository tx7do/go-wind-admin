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
	configV1 "go-wind-admin/api/gen/go/config/service/v1"

	"go-wind-admin/pkg/constants"
	appViewer "go-wind-admin/pkg/entgo/viewer"
	"go-wind-admin/pkg/middleware/auth"
)

type ConfigService struct {
	adminV1.ConfigServiceHTTPServer

	log *bLogger.Helper

	configRepo *data.ConfigRepo
}

func NewConfigService(
	ctx *bootstrap.Context,
	configRepo *data.ConfigRepo,
) *ConfigService {
	svc := &ConfigService{
		log:        ctx.NewLoggerHelper("config/service/admin-service"),
		configRepo: configRepo,
	}

	svc.init()

	return svc
}

// init 播种内置平台参数（等保口令策略阈值）。与其他默认数据一致，
// 在服务构造（进程启动）时执行一次；SeedDefaults 按键缺一补一、不覆盖既有值。
func (s *ConfigService) init() {
	ctx := appViewer.NewSystemViewerContext(context.Background())
	if err := s.configRepo.SeedDefaults(ctx, constants.DefaultConfigs); err != nil {
		s.log.Errorf(ctx, "seed default configs failed: %s", err.Error())
	}
}

func (s *ConfigService) List(ctx context.Context, req *paginationV1.PagingRequest) (*configV1.ListConfigResponse, error) {
	return s.configRepo.List(ctx, req)
}

func (s *ConfigService) Get(ctx context.Context, req *configV1.GetConfigRequest) (*configV1.Config, error) {
	return s.configRepo.Get(ctx, req)
}

func (s *ConfigService) Create(ctx context.Context, req *configV1.CreateConfigRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	if err = s.configRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *ConfigService) Update(ctx context.Context, req *configV1.UpdateConfigRequest) (*emptypb.Empty, error) {
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

	if err = s.configRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *ConfigService) Delete(ctx context.Context, req *configV1.DeleteConfigRequest) (*emptypb.Empty, error) {
	if err := s.configRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
