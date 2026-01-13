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

type OperationAuditLogService struct {
	adminV1.OperationAuditLogServiceHTTPServer

	log *log.Helper

	repo *data.OperationAuditLogRepo
}

func NewOperationAuditLogService(ctx *bootstrap.Context, repo *data.OperationAuditLogRepo) *OperationAuditLogService {
	return &OperationAuditLogService{
		log:  ctx.NewLoggerHelper("operation-audit-log/service/admin-service"),
		repo: repo,
	}
}

func (s *OperationAuditLogService) List(ctx context.Context, req *pagination.PagingRequest) (*auditV1.ListOperationAuditLogResponse, error) {
	return s.repo.List(ctx, req)
}

func (s *OperationAuditLogService) Get(ctx context.Context, req *auditV1.GetOperationAuditLogRequest) (*auditV1.OperationAuditLog, error) {
	return s.repo.Get(ctx, req)
}

func (s *OperationAuditLogService) Create(ctx context.Context, req *auditV1.CreateOperationAuditLogRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	if err := s.repo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
