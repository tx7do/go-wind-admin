package service

import (
	"context"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

type PolicyEvaluationLogService struct {
	adminV1.PolicyEvaluationLogServiceHTTPServer

	log *bLogger.Helper

	policyEvaluationLogRepo *data.PolicyEvaluationLogRepo
}

func NewPolicyEvaluationLogService(
	ctx *bootstrap.Context,
	policyEvaluationLogRepo *data.PolicyEvaluationLogRepo,
) *PolicyEvaluationLogService {
	return &PolicyEvaluationLogService{
		log:                     ctx.NewLoggerHelper("policy-evaluation-log/service/admin-service"),
		policyEvaluationLogRepo: policyEvaluationLogRepo,
	}
}

func (s *PolicyEvaluationLogService) List(ctx context.Context, req *paginationV1.PagingRequest) (*permissionV1.ListPolicyEvaluationLogResponse, error) {
	resp, err := s.policyEvaluationLogRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *PolicyEvaluationLogService) Get(ctx context.Context, req *permissionV1.GetPolicyEvaluationLogRequest) (*permissionV1.PolicyEvaluationLog, error) {
	resp, err := s.policyEvaluationLogRepo.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *PolicyEvaluationLogService) Create(ctx context.Context, req *permissionV1.CreatePolicyEvaluationLogRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	if err := s.policyEvaluationLogRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
