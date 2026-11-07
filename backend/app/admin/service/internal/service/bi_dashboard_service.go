package service

import (
	"context"
	"fmt"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/id"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
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
	revisionRepo    *data.BiDashboardRevisionRepo
}

func NewBiDashboardService(
	ctx *bootstrap.Context,
	biDashboardRepo *data.BiDashboardRepo,
	revisionRepo *data.BiDashboardRevisionRepo,
) *BiDashboardService {
	svc := &BiDashboardService{
		log:             ctx.NewLoggerHelper("bi-dashboard/service/admin-service"),
		biDashboardRepo: biDashboardRepo,
		revisionRepo:    revisionRepo,
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

	// 版本快照（docs/bi_dashboard.md 后续项）：保存成功把**保存前**的
	// payload/name/version 留一行。快照失败不阻断保存（Errorf 留痕）——
	// 主功能不应被辅功能拖死，但丢快照必须可观测。
	current, getErr := s.biDashboardRepo.Get(ctx, &biV1.GetBiDashboardRequest{
		QueryBy: &biV1.GetBiDashboardRequest_Id{Id: req.GetId()},
	})
	if getErr != nil {
		s.log.Warnf(ctx, "snapshot before update: load dashboard %d failed: %s", req.GetId(), getErr.Error())
	}

	if err = s.biDashboardRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	if current != nil && s.revisionRepo != nil {
		snapshot := &biV1.BiDashboardRevision{
			DashboardId: req.GetId(),
			Version:     current.GetVersion(),
			Name:        current.Name,
			Payload:     current.Payload,
			Source:      trans.Ptr("SAVE"),
			CreatedBy:   trans.Ptr(operator.UserId),
		}
		if serr := s.revisionRepo.Create(ctx, snapshot); serr != nil {
			s.log.Errorf(ctx, "snapshot before update dashboard %d failed: %s", req.GetId(), serr.Error())
		}
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
		// 短随机后缀保证租户内唯一：时间戳后缀不可靠——秒级在"同一秒内复制两次"
		//（双击另存为、批量派生）必撞；纳秒级在 Windows 上因时钟粒度仍可能连续
		// 两次取到同一读数（实测整包并发跑测试时复现）。
		code = fmt.Sprintf("%s-copy-%s", source.GetCode(), id.NewShortUUID())
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

// ListRevisions 版本历史（新快照在前）。
func (s *BiDashboardService) ListRevisions(ctx context.Context, req *biV1.ListBiDashboardRevisionsRequest) (*biV1.ListBiDashboardRevisionsResponse, error) {
	if req == nil || req.GetDashboardId() == 0 {
		return nil, adminV1.ErrorBadRequest("dashboardId is required")
	}
	items, total, err := s.revisionRepo.List(ctx, req.GetDashboardId(), req.GetPage())
	if err != nil {
		return nil, err
	}
	return &biV1.ListBiDashboardRevisionsResponse{
		Items: items,
		Total: total,
	}, nil
}

// RestoreRevision 恢复版本：快照 payload/name 写回主表，版本号 +1。
// 恢复动作本身先把**当前版本**快照（RESTORE 来源），恢复可逆。
func (s *BiDashboardService) RestoreRevision(ctx context.Context, req *biV1.RestoreBiDashboardRevisionRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, adminV1.ErrorBadRequest("id is required")
	}
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	revision, err := s.revisionRepo.Get(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	dashboardId := revision.GetDashboardId()
	current, err := s.biDashboardRepo.Get(ctx, &biV1.GetBiDashboardRequest{
		QueryBy: &biV1.GetBiDashboardRequest_Id{Id: dashboardId},
	})
	if err != nil {
		return nil, err
	}

	// 恢复前留底当前版本（失败不阻断恢复，Errorf 留痕）
	if s.revisionRepo != nil {
		snapshot := &biV1.BiDashboardRevision{
			DashboardId: dashboardId,
			Version:     current.GetVersion(),
			Name:        current.Name,
			Payload:     current.Payload,
			Source:      trans.Ptr("RESTORE"),
			CreatedBy:   trans.Ptr(operator.UserId),
		}
		if serr := s.revisionRepo.Create(ctx, snapshot); serr != nil {
			s.log.Errorf(ctx, "snapshot before restore dashboard %d failed: %s", dashboardId, serr.Error())
		}
	}

	// 写回走普通更新（无 CAS——恢复是明确的人为决定，以快照为准整份覆盖），版本号 +1
	if err = s.biDashboardRepo.Update(ctx, &biV1.UpdateBiDashboardRequest{
		Id: dashboardId,
		Data: &biV1.BiDashboard{
			Name:      revision.Name,
			Payload:   revision.Payload,
			UpdatedBy: trans.Ptr(operator.UserId),
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name", "payload", "updated_by"}},
	}); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
