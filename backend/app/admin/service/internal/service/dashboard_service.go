package service

import (
	"context"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
)

// DashboardService 为后台首页分析页提供只读聚合统计。
// 多租户隔离由 ent Policy + auth 中间件注入的 viewer 自动完成。
type DashboardService struct {
	adminV1.DashboardServiceHTTPServer

	log *bLogger.Helper

	dashboardRepo *data.DashboardRepo
}

func NewDashboardService(
	ctx *bootstrap.Context,
	dashboardRepo *data.DashboardRepo,
) *DashboardService {
	return &DashboardService{
		log:           ctx.NewLoggerHelper("dashboard/service/admin-service"),
		dashboardRepo: dashboardRepo,
	}
}

// GetOverview 返回四张概览卡的计数。
func (s *DashboardService) GetOverview(ctx context.Context, _ *emptypb.Empty) (*adminV1.DashboardOverviewResponse, error) {
	userCount, err := s.dashboardRepo.CountActiveUsers(ctx)
	if err != nil {
		return nil, err
	}
	roleCount, err := s.dashboardRepo.CountRoles(ctx)
	if err != nil {
		return nil, err
	}
	todayLoginCount, err := s.dashboardRepo.CountTodayLogins(ctx)
	if err != nil {
		return nil, err
	}
	todayOperationCount, err := s.dashboardRepo.CountTodayOperations(ctx)
	if err != nil {
		return nil, err
	}

	return &adminV1.DashboardOverviewResponse{
		UserCount:           uint32(userCount),
		RoleCount:           uint32(roleCount),
		TodayLoginCount:     uint32(todayLoginCount),
		TodayOperationCount: uint32(todayOperationCount),
	}, nil
}

// GetLoginTrend 返回近 days 天每日登录次数趋势，按日期升序、缺日补零。
func (s *DashboardService) GetLoginTrend(ctx context.Context, req *adminV1.GetLoginTrendRequest) (*adminV1.LoginTrendResponse, error) {
	days := int(req.GetDays())
	rows, err := s.dashboardRepo.LoginTrend(ctx, days)
	if err != nil {
		return nil, err
	}

	points := make([]*adminV1.TrendPoint, 0, len(rows))
	for _, r := range rows {
		points = append(points, &adminV1.TrendPoint{
			Date:  r.Date,
			Count: uint32(r.Count),
		})
	}
	return &adminV1.LoginTrendResponse{Points: points}, nil
}

// GetOperationActionDistribution 返回操作审计按 action 的分布。
func (s *DashboardService) GetOperationActionDistribution(ctx context.Context, _ *emptypb.Empty) (*adminV1.ActionDistributionResponse, error) {
	rows, err := s.dashboardRepo.OperationActionDistribution(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]*adminV1.DistributionItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, &adminV1.DistributionItem{
			Label: r.Action,
			Count: uint32(r.Count),
		})
	}
	return &adminV1.ActionDistributionResponse{Items: items}, nil
}

// GetLoginStatusDistribution 返回登录审计按 status 的分布。
func (s *DashboardService) GetLoginStatusDistribution(ctx context.Context, _ *emptypb.Empty) (*adminV1.StatusDistributionResponse, error) {
	rows, err := s.dashboardRepo.LoginStatusDistribution(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]*adminV1.DistributionItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, &adminV1.DistributionItem{
			Label: r.Status,
			Count: uint32(r.Count),
		})
	}
	return &adminV1.StatusDistributionResponse{Items: items}, nil
}
