package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	pagination "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
)

type DataAccessAuditLogService struct {
	adminV1.DataAccessAuditLogServiceHTTPServer

	log *log.Helper

	repo *data.DataAccessAuditLogRepo
}

func NewDataAccessAuditLogService(ctx *bootstrap.Context, repo *data.DataAccessAuditLogRepo) *DataAccessAuditLogService {
	return &DataAccessAuditLogService{
		log:  ctx.NewLoggerHelper("data-access-audit-log/service/admin-service"),
		repo: repo,
	}
}

func (s *DataAccessAuditLogService) List(ctx context.Context, req *pagination.PagingRequest) (*auditV1.ListDataAccessAuditLogResponse, error) {
	return s.repo.List(ctx, req)
}

func (s *DataAccessAuditLogService) Get(ctx context.Context, req *auditV1.GetDataAccessAuditLogRequest) (*auditV1.DataAccessAuditLog, error) {
	return s.repo.Get(ctx, req)
}

func (s *DataAccessAuditLogService) Create(ctx context.Context, req *auditV1.CreateDataAccessAuditLogRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	if err := s.repo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
