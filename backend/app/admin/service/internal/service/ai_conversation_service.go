package service

import (
	"context"

	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/pkg/middleware/auth"
)

type AiConversationService struct {
	adminV1.AiConversationServiceHTTPServer
	log  *bLogger.Helper
	repo *data.AiConversationRepo
}

func NewAiConversationService(ctx *bootstrap.Context, repo *data.AiConversationRepo) *AiConversationService {
	return &AiConversationService{
		log:  ctx.NewLoggerHelper("ai_conversation/service/admin-service"),
		repo: repo,
	}
}

// List 只返回当前用户自己的会话（Phase 1 聊天语义；管理员视角的全量会话属后续阶段）。
func (s *AiConversationService) List(ctx context.Context, req *paginationV1.PagingRequest) (*aiV1.ListAiConversationResponse, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.List(ctx, req, operator.UserId)
}

// Get 会话详情；归属校验失败一律 NotFound（不泄露他人会话的存在性）。
func (s *AiConversationService) Get(ctx context.Context, req *aiV1.GetAiConversationRequest) (*aiV1.AiConversation, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	dto, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, err
	}
	if dto.GetUserId() != operator.UserId {
		return nil, adminV1.ErrorNotFound("ai conversation not found")
	}
	return dto, nil
}

func (s *AiConversationService) Update(ctx context.Context, req *aiV1.UpdateAiConversationRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	entity, err := s.repo.GetEntityByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if entity.UserID == nil || *entity.UserID != operator.UserId {
		return nil, adminV1.ErrorNotFound("ai conversation not found")
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

func (s *AiConversationService) Delete(ctx context.Context, req *aiV1.DeleteAiConversationRequest) (*emptypb.Empty, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	entity, err := s.repo.GetEntityByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if entity.UserID == nil || *entity.UserID != operator.UserId {
		return nil, adminV1.ErrorNotFound("ai conversation not found")
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
