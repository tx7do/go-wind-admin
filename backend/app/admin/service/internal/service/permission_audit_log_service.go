package service

import (
	"context"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
)

type PermissionAuditLogService struct {
	adminV1.PermissionAuditLogServiceHTTPServer

	log *bLogger.Helper

	permissionAuditLogRepo *data.PermissionAuditLogRepo
}

func NewPermissionAuditLogService(
	ctx *bootstrap.Context,
	permissionAuditLogRepo *data.PermissionAuditLogRepo,
) *PermissionAuditLogService {
	return &PermissionAuditLogService{
		log:                    ctx.NewLoggerHelper("permission-audit-log/service/admin-service"),
		permissionAuditLogRepo: permissionAuditLogRepo,
	}
}

func (s *PermissionAuditLogService) List(ctx context.Context, req *paginationV1.PagingRequest) (*auditV1.ListPermissionAuditLogResponse, error) {
	resp, err := s.permissionAuditLogRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *PermissionAuditLogService) Get(ctx context.Context, req *auditV1.GetPermissionAuditLogRequest) (*auditV1.PermissionAuditLog, error) {
	resp, err := s.permissionAuditLogRepo.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *PermissionAuditLogService) Create(ctx context.Context, req *auditV1.CreatePermissionAuditLogRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	if err := s.permissionAuditLogRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
