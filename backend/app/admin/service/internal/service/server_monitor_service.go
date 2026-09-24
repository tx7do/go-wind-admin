package service

import (
	"context"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	serverMonitorV1 "go-wind-admin/api/gen/go/server_monitor/service/v1"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"

	"go-wind-admin/app/admin/service/internal/data"
)

// ServerMonitorService 提供服务运行时只读监控视图的 HTTP 接口。
// 纯透传至 ServerMonitorRepo，不做任何业务加工——与 RedisCacheMonitorService 同型的极简形态。
type ServerMonitorService struct {
	adminV1.ServerMonitorServiceHTTPServer

	log  *bLogger.Helper
	repo *data.ServerMonitorRepo
}

// NewServerMonitorService 构造服务监控服务。
func NewServerMonitorService(
	ctx *bootstrap.Context,
	repo *data.ServerMonitorRepo,
) *ServerMonitorService {
	return &ServerMonitorService{
		log:  ctx.NewLoggerHelper("server-monitor/service/admin-service"),
		repo: repo,
	}
}

// Get 返回服务监控聚合信息（Go 运行时 / 数据库 / 主机）。
func (s *ServerMonitorService) Get(ctx context.Context, _ *serverMonitorV1.GetServerMonitorRequest) (*serverMonitorV1.ServerMonitorInfo, error) {
	return s.repo.GetInfo(ctx)
}
