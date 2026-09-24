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

	before := time.Now().AddDate(0, 0, -90)
	if req.GetBefore() != nil {
		before = req.GetBefore().AsTime()
	}

	deleted, err := s.repo.Purge(ctx, before)
	if err != nil {
		return nil, err
	}

	return &scriptV1.PurgeScriptLogsResponse{Deleted: deleted}, nil
}
