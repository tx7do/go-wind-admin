package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	pagination "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

type PermissionAuditLogService struct {
	adminV1.PermissionAuditLogServiceHTTPServer

	log *log.Helper

	policyEvaluationLogRepo *data.PermissionAuditLogRepo
}

func NewPermissionAuditLogService(
	ctx *bootstrap.Context,
	policyEvaluationLogRepo *data.PermissionAuditLogRepo,
) *PermissionAuditLogService {
	return &PermissionAuditLogService{
		log:                     ctx.NewLoggerHelper("permission-audit-log/service/admin-service"),
		policyEvaluationLogRepo: policyEvaluationLogRepo,
	}
}

func (s *PermissionAuditLogService) List(ctx context.Context, req *pagination.PagingRequest) (*permissionV1.ListPermissionAuditLogResponse, error) {
	resp, err := s.policyEvaluationLogRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *PermissionAuditLogService) Get(ctx context.Context, req *permissionV1.GetPermissionAuditLogRequest) (*permissionV1.PermissionAuditLog, error) {
	resp, err := s.policyEvaluationLogRepo.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *PermissionAuditLogService) Create(ctx context.Context, req *permissionV1.CreatePermissionAuditLogRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	if err := s.policyEvaluationLogRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
