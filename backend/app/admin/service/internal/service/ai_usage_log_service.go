package service

import (
	"context"
	"time"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/emptypb"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/pkg/middleware/auth"
)

// AiUsageLogService 用量流水查询（配额记账事实源的只读视图）。
// 租户隔离由 mixin 谓词自动生效：租户管理员只见本租户流水，平台管理员见全量。
type AiUsageLogService struct {
	adminV1.AiUsageLogServiceHTTPServer
	log  *bLogger.Helper
	repo *data.AiUsageLogRepo
}

func NewAiUsageLogService(ctx *bootstrap.Context, repo *data.AiUsageLogRepo) *AiUsageLogService {
	return &AiUsageLogService{
		log:  ctx.NewLoggerHelper("ai_usage_log/service/admin-service"),
		repo: repo,
	}
}

// GetUsageSummary 当月用量汇总：tokens/调用次数 + 套餐配额上限。
// 统计范围与本表流水列表同口径（见 MonthStats）：租户用户按本租户，平台管理员按全量；
// 配额只对租户成立，平台侧不限量。
func (s *AiUsageLogService) GetUsageSummary(ctx context.Context, _ *emptypb.Empty) (*aiV1.UsageSummaryResponse, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	tenantId := operator.GetTenantId()
	monthStart := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Local)
	tokens, calls, err := s.repo.MonthStats(ctx, tenantId, monthStart)
	if err != nil {
		return nil, err
	}

	resp := &aiV1.UsageSummaryResponse{
		MonthTokens: uint32(tokens),
		MonthCalls:  uint32(calls),
	}

	if tenantId != 0 {
		limit, configured, err := s.repo.FetchTenantTokenQuotaLimit(ctx, tenantId)
		if err != nil {
			return nil, err
		}
		resp.QuotaConfigured = configured
		resp.QuotaLimit = uint64(limit)
	}

	return resp, nil
}

func (s *AiUsageLogService) List(ctx context.Context, req *paginationV1.PagingRequest) (*aiV1.ListAiUsageLogResponse, error) {
	return s.repo.List(ctx, req)
}

func (s *AiUsageLogService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*aiV1.CountAiUsageLogResponse, error) {
	return s.repo.Count(ctx, req)
}
