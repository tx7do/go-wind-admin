package service

import (
	"context"
	"strings"

	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	appCrypto "github.com/tx7do/go-utils/crypto"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/pkg/middleware/auth"
)

type AiProviderService struct {
	adminV1.AiProviderServiceHTTPServer
	log  *bLogger.Helper
	repo *data.AiProviderRepo
}

func NewAiProviderService(ctx *bootstrap.Context, repo *data.AiProviderRepo) *AiProviderService {
	return &AiProviderService{
		log:  ctx.NewLoggerHelper("ai_provider/service/admin-service"),
		repo: repo,
	}
}

func (s *AiProviderService) List(ctx context.Context, req *paginationV1.PagingRequest) (*aiV1.ListAiProviderResponse, error) {
	return s.repo.List(ctx, req)
}

func (s *AiProviderService) Get(ctx context.Context, req *aiV1.GetAiProviderRequest) (*aiV1.AiProvider, error) {
	return s.repo.Get(ctx, req)
}

// Create 创建提供商；api_key 明文经全局加密器（GOWIND_CRYPTO_KEY，未配置则明文落库——
// 与全局加密既有语义一致）加密后落库，hint 只存脱敏形态。
func (s *AiProviderService) Create(ctx context.Context, req *aiV1.CreateAiProviderRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	req.Data.CreatedBy = trans.Ptr(operator.UserId)
	req.Data.UpdatedBy = trans.Ptr(operator.UserId)

	if err = s.sealApiKey(req.Data); err != nil {
		return nil, err
	}

	if err = s.repo.Create(ctx, req); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *AiProviderService) Update(ctx context.Context, req *aiV1.UpdateAiProviderRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	req.Data.Id = trans.Ptr(req.GetId())
	req.Data.UpdatedBy = trans.Ptr(operator.UserId)
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	// api_key 语义：DTO 里 key 明文非空 → 加密落库并刷新 hint（mask 已含 api_key，照常写）；
	// key 为空 → 本次请求不改 key，把 api_key/api_key_hint 摘出 mask 且清空 DTO
	//（否则空值会经 FilterByFieldMask 被当成"显式清空"写库，把已存的 key 抹掉）。
	hasNewKey := req.Data.ApiKey != nil && *req.Data.ApiKey != ""
	if hasNewKey {
		if err = s.sealApiKey(req.Data); err != nil {
			return nil, err
		}
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "api_key_hint")
	} else {
		req.Data.ApiKey = nil
		req.Data.ApiKeyHint = nil
		if req.UpdateMask != nil {
			req.UpdateMask.Paths = removeMaskPaths(req.UpdateMask.Paths, "api_key", "api_key_hint")
		}
	}

	if err = s.repo.Update(ctx, req); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *AiProviderService) Delete(ctx context.Context, req *aiV1.DeleteAiProviderRequest) (*emptypb.Empty, error) {
	if err := s.repo.Delete(ctx, req); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// sealApiKey 加密明文 key 并生成脱敏 hint（Create 路径）。
func (s *AiProviderService) sealApiKey(data *aiV1.AiProvider) error {
	if data == nil || data.ApiKey == nil || *data.ApiKey == "" {
		data.ApiKeyHint = nil
		return nil
	}
	plain := *data.ApiKey
	encrypted, err := appCrypto.EncryptIfNeeded(plain)
	if err != nil {
		s.log.Errorf(context.Background(), "encrypt ai provider api key failed: %v", err)
		return adminV1.ErrorInternalServerError("encrypt api key failed")
	}
	data.ApiKey = trans.Ptr(encrypted)
	data.ApiKeyHint = trans.Ptr(maskApiKey(plain))
	return nil
}

// maskApiKey 脱敏：保留前 3 后 4，中间以 *** 代替；过短则整体打码。
func maskApiKey(plain string) string {
	plain = strings.TrimSpace(plain)
	if len(plain) <= 7 {
		return "***"
	}
	return plain[:3] + "***" + plain[len(plain)-4:]
}

// removeMaskPaths 从 update mask 中移除指定路径（保持语义清晰，避免空值清写）。
func removeMaskPaths(paths []string, removed ...string) []string {
	drop := make(map[string]struct{}, len(removed))
	for _, p := range removed {
		drop[p] = struct{}{}
	}
	kept := make([]string, 0, len(paths))
	for _, p := range paths {
		if _, ok := drop[p]; ok {
			continue
		}
		kept = append(kept, p)
	}
	return kept
}
