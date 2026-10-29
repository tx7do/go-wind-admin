package service

import (
	"context"
	"fmt"
	"time"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	biV1 "go-wind-admin/api/gen/go/bi/service/v1"

	"github.com/tx7do/go-crud/viewer"
	"go-wind-admin/pkg/constants"
	"go-wind-admin/pkg/middleware/auth"
)

type BiDashboardService struct {
	adminV1.BiDashboardServiceHTTPServer

	log             *bLogger.Helper
	biDashboardRepo *data.BiDashboardRepo
}

func NewBiDashboardService(
	ctx *bootstrap.Context,
	biDashboardRepo *data.BiDashboardRepo,
) *BiDashboardService {
	svc := &BiDashboardService{
		log:             ctx.NewLoggerHelper("bi-dashboard/service/admin-service"),
		biDashboardRepo: biDashboardRepo,
	}

	svc.init()

	return svc
}

// init 播种内置 BI 模板（基线仪表盘版式）。与其他默认数据一致，
// 在服务构造（进程启动）时执行一次；按编码缺一补一、不覆盖既有行。
func (s *BiDashboardService) init() {
	ctx := viewer.WithSystemContext(context.Background())
	if err := s.biDashboardRepo.SeedDefaults(ctx, constants.DefaultBiDashboards); err != nil {
		s.log.Errorf(ctx, "seed default bi dashboards failed: %s", err.Error())
	}
}

func (s *BiDashboardService) List(ctx context.Context, req *paginationV1.PagingRequest) (*biV1.ListBiDashboardResponse, error) {
	return s.biDashboardRepo.List(ctx, req)
}

func (s *BiDashboardService) Get(ctx context.Context, req *biV1.GetBiDashboardRequest) (*biV1.BiDashboard, error) {
	return s.biDashboardRepo.Get(ctx, req)
}

func (s *BiDashboardService) Create(ctx context.Context, req *biV1.CreateBiDashboardRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	if err = s.biDashboardRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *BiDashboardService) Update(ctx context.Context, req *biV1.UpdateBiDashboardRequest) (*emptypb.Empty, error) {
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

	if err = s.biDashboardRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *BiDashboardService) Delete(ctx context.Context, req *biV1.DeleteBiDashboardRequest) (*emptypb.Empty, error) {
	if err := s.biDashboardRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// Copy 把源仪表盘整份复制成一行新记录（模板派生 / 另存为）。
// 副本恒为非内置、版本归 1；名称/编码缺省自动派生，编码需租户内唯一。
func (s *BiDashboardService) Copy(ctx context.Context, req *biV1.CopyBiDashboardRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	source, err := s.biDashboardRepo.Get(ctx, &biV1.GetBiDashboardRequest{
		QueryBy: &biV1.GetBiDashboardRequest_Id{Id: req.GetId()},
	})
	if err != nil {
		return nil, err
	}

	name := req.GetName()
	if name == "" {
		name = source.GetName() + "-copy"
	}
	code := req.GetCode()
	if code == "" {
		// 时间戳后缀保证租户内唯一，避免"复制两次同名冲突"
		code = fmt.Sprintf("%s-copy-%d", source.GetCode(), time.Now().Unix())
	}

	createReq := &biV1.CreateBiDashboardRequest{
		Data: &biV1.BiDashboard{
			Name:       trans.Ptr(name),
			Code:       trans.Ptr(code),
			Kind:       source.Kind,
			Remark:     source.Remark,
			Payload:    source.Payload,
			ChartCount: source.ChartCount,
			ThemeKey:   source.ThemeKey,
			Thumbnail:  source.Thumbnail,
			// 副本恒非内置：内置模板只读，派生出来的才是租户自有资产
			IsBuiltIn: trans.Ptr(false),
			IsPublic:  source.IsPublic,
			IsEnabled: trans.Ptr(true),
			SortOrder: source.SortOrder,
			CreatedBy: trans.Ptr(operator.UserId),
		},
	}

	if err = s.biDashboardRepo.Create(ctx, createReq); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
