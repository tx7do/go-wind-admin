package service

import (
	"context"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"

	"go-wind-admin/pkg/middleware/auth"
)

type LoginPolicyService struct {
	adminV1.LoginPolicyServiceHTTPServer

	log *bLogger.Helper

	repo *data.LoginPolicyRepo
}

func NewLoginPolicyService(ctx *bootstrap.Context, repo *data.LoginPolicyRepo) *LoginPolicyService {
	return &LoginPolicyService{
		log:  ctx.NewLoggerHelper("login-policy/service/admin-service"),
		repo: repo,
	}
}

func (s *LoginPolicyService) List(ctx context.Context, req *paginationV1.PagingRequest) (*authenticationV1.ListLoginPolicyResponse, error) {
	return s.repo.List(ctx, req)
}

func (s *LoginPolicyService) Get(ctx context.Context, req *authenticationV1.GetLoginPolicyRequest) (*authenticationV1.LoginPolicy, error) {
	return s.repo.Get(ctx, req)
}

func (s *LoginPolicyService) Create(ctx context.Context, req *authenticationV1.CreateLoginPolicyRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid request")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	// 值格式校验：配错格式会让匹配器静默不命中（白名单 = 全员被锁）
	if verr := data.ValidateLoginPolicyValue(req.Data.GetMethod().String(), req.Data.GetValue()); verr != nil {
		return nil, adminV1.ErrorBadRequest("%s", verr.Error())
	}

	if err = s.repo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *LoginPolicyService) Update(ctx context.Context, req *authenticationV1.UpdateLoginPolicyRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid request")
	}

	// 值格式校验（理由同 Create）
	if verr := data.ValidateLoginPolicyValue(req.Data.GetMethod().String(), req.Data.GetValue()); verr != nil {
		return nil, adminV1.ErrorBadRequest("%s", verr.Error())
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.Id = trans.Ptr(req.GetId())

	req.Data.UpdatedBy = trans.Ptr(operator.UserId)
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	if err = s.repo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *LoginPolicyService) Delete(ctx context.Context, req *authenticationV1.DeleteLoginPolicyRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid request")
	}

	if err := s.repo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
