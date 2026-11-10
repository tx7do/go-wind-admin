package service

import (
	"context"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"time"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	scriptV1 "go-wind-admin/api/gen/go/script/service/v1"

	"go-wind-admin/app/admin/service/internal/data"
)

// ScriptLogService 脚本执行日志管理服务（平台管理员，只读 + 清理）。
type ScriptLogService struct {
	adminV1.ScriptLogServiceHTTPServer

	log  *bLogger.Helper
	repo *data.ScriptLogRepo
}

func NewScriptLogService(ctx *bootstrap.Context, repo *data.ScriptLogRepo) *ScriptLogService {
	return &ScriptLogService{
		log:  ctx.NewLoggerHelper("script-log/service/admin-service"),
		repo: repo,
	}
}

func (s *ScriptLogService) List(ctx context.Context, req *paginationV1.PagingRequest) (*scriptV1.ListScriptLogsResponse, error) {
	return s.repo.List(ctx, req)
}

func (s *ScriptLogService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*scriptV1.CountScriptLogsResponse, error) {
	return s.repo.CountLog(ctx)
}

// Purge 清理指定时间之前的日志。before 为空时默认清理 90 天前。
func (s *ScriptLogService) Purge(ctx context.Context, req *scriptV1.PurgeScriptLogsRequest) (*scriptV1.PurgeScriptLogsResponse, error) {
	if req == nil {
		return nil, scriptV1.ErrorBadRequest("invalid parameter")
	}

	// P2 修复：sys_script_logs 无租户隔离（全平台共表），脚本执行台账是钩子/
	// 任务/test_run 的审计痕迹——清理入口此前无闸门，传 before=<未来> 即一键
	// 清空全平台脚本日志。显式挡在平台管理员之内（对齐服务注释自述的口径）。
	if err := requirePlatformAdmin(ctx, s.log, "script log purge"); err != nil {
		return nil, err
	}

	before := time.Now().AddDate(0, 0, -90)
	if req.GetBefore() != nil {
		before = req.GetBefore().AsTime()
	}

	// 钳制：before 至少保留最近 1 天，防误传未来时间清空全部台账
	if minBefore := time.Now().AddDate(0, 0, -1); before.After(minBefore) {
		before = minBefore
	}

	deleted, err := s.repo.Purge(ctx, before)
	if err != nil {
		return nil, err
	}

	return &scriptV1.PurgeScriptLogsResponse{Deleted: uint64(deleted)}, nil
}
