package service

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	"github.com/tx7do/go-crud/viewer"

	"github.com/tx7do/go-utils/doctext"
	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/pkg/middleware/auth"
	"go-wind-admin/pkg/task"
)

const (
	// ragChunkRunes 切片长度（字符）：中文友好。
	ragChunkRunes = 500
	// ragChunkOverlap 切片重叠（字符）：保证跨片语义连续。
	ragChunkOverlap = 50
	// ragTopK 检索注入上下文的默认片段数。
	ragTopK = 3
)

// AiKnowledgeService 知识库管理：文档入库（切片→向量化）+ 检索 + Chat 注入。
type AiKnowledgeService struct {
	adminV1.AiKnowledgeBaseServiceHTTPServer
	log  *bLogger.Helper
	repo *data.AiKnowledgeRepo

	providerRepo *data.AiProviderRepo
	usageLogRepo *data.AiUsageLogRepo
}

func NewAiKnowledgeService(
	ctx *bootstrap.Context,
	repo *data.AiKnowledgeRepo,
	providerRepo *data.AiProviderRepo,
	usageLogRepo *data.AiUsageLogRepo,
) *AiKnowledgeService {
	s := &AiKnowledgeService{
		log:          ctx.NewLoggerHelper("ai_knowledge/service/admin-service"),
		repo:         repo,
		providerRepo: providerRepo,
		usageLogRepo: usageLogRepo,
	}

	// 启动期补建 pgvector 扩展与 embedding 列（幂等；失败只降级 RAG，不阻断服务）
	if err := repo.MigrateVectorColumn(ctx.Context()); err != nil {
		ctx.GetLogger().Error(ctx.Context(), fmt.Sprintf("rag vector column migration failed: %v", err))
	}

	return s
}

// ── 知识库 CRUD ────────────────────────────────────────────────────

func (s *AiKnowledgeService) List(ctx context.Context, req *paginationV1.PagingRequest) (*aiV1.ListAiKnowledgeBaseResponse, error) {
	items, total, err := s.repo.List(ctx, req)
	if err != nil {
		return nil, err
	}
	s.repo.FillDocCounts(ctx, items)
	return &aiV1.ListAiKnowledgeBaseResponse{Items: items, Total: total}, nil
}

func (s *AiKnowledgeService) Get(ctx context.Context, req *aiV1.GetAiKnowledgeBaseRequest) (*aiV1.AiKnowledgeBase, error) {
	return s.repo.Get(ctx, req)
}

func (s *AiKnowledgeService) Create(ctx context.Context, req *aiV1.CreateAiKnowledgeBaseRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	req.Data.CreatedBy = trans.Ptr(operator.UserId)
	req.Data.UserId = trans.Ptr(operator.UserId)
	req.Data.TenantId = trans.Ptr(operator.GetTenantId())

	// provider 必须存在且启用（向量化要走它的端点）
	if req.Data.ProviderId == nil || req.Data.EmbeddingModel == nil || *req.Data.EmbeddingModel == "" {
		return nil, adminV1.ErrorBadRequest("provider_id and embedding_model are required")
	}
	if _, err = s.providerRepo.GetEntityByID(ctx, req.Data.GetProviderId()); err != nil {
		return nil, err
	}

	if _, err = s.repo.Create(ctx, req.Data); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *AiKnowledgeService) Update(ctx context.Context, req *aiV1.UpdateAiKnowledgeBaseRequest) (*emptypb.Empty, error) {
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
	if err = s.repo.Update(ctx, req); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *AiKnowledgeService) Delete(ctx context.Context, req *aiV1.DeleteAiKnowledgeBaseRequest) (*emptypb.Empty, error) {
	if err := s.repo.Delete(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// ── 文档入库：切片 → embedding → 落库 ───────────────────────────────

// UploadDoc 纯文本入库入口。
func (s *AiKnowledgeService) UploadDoc(ctx context.Context, req *aiV1.UploadAiDocRequest) (*aiV1.UploadAiDocResponse, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetBaseId() == 0 || req.GetName() == "" || req.GetContent() == "" {
		return nil, adminV1.ErrorBadRequest("base_id, name and content are required")
	}

	base, err := s.repo.GetEntityByID(ctx, req.GetBaseId())
	if err != nil {
		return nil, err
	}

	doc, err := s.ingestDoc(ctx, operator, base, req.GetName(), req.GetContent())
	if err != nil {
		return nil, err
	}
	return &aiV1.UploadAiDocResponse{Doc: doc, ChunkCount: doc.GetChunkCount()}, nil
}

// maxDocxEntryUncompressedSize docx 单条目解压后大小上限。
const maxDocxEntryUncompressedSize = 64 << 20 // 64MB

// checkDocxEntrySizes 预检 docx（zip）各条目解压后大小，超限即拒。
// 非 zip/损坏包返回原错误交给 doctext.Extract 给出语义化报错。
func checkDocxEntrySizes(data []byte) error {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	for _, f := range zr.File {
		if f.UncompressedSize64 > maxDocxEntryUncompressedSize {
			return fmt.Errorf("docx entry %q uncompressed size %d exceeds limit %d",
				f.Name, f.UncompressedSize64, maxDocxEntryUncompressedSize)
		}
	}
	return nil
}

// UploadDocFile 文件入库入口：按扩展名抽取文本（txt/md/.../docx/pdf）后走同一入库链。
func (s *AiKnowledgeService) UploadDocFile(ctx context.Context, req *aiV1.UploadAiDocFileRequest) (*aiV1.UploadAiDocResponse, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetBaseId() == 0 || req.GetFileName() == "" || len(req.GetContentBase64()) == 0 {
		return nil, adminV1.ErrorBadRequest("base_id, fileName and contentBase64 are required")
	}

	base, err := s.repo.GetEntityByID(ctx, req.GetBaseId())
	if err != nil {
		return nil, err
	}

	// P1 修复：docx(zip bomb) 预检——10MB 压缩包可声称解出数 GB 的
	// word/document.xml，而 doctext.Extract 内部 io.ReadAll 无上限。
	// 这里在解压前用 zip 目录头的 Uncompressed64（无需解压即可读）预判拒收。
	if strings.EqualFold(filepath.Ext(req.GetFileName()), ".docx") {
		if zerr := checkDocxEntrySizes(req.GetContentBase64()); zerr != nil {
			return nil, adminV1.ErrorBadRequest("extract text failed: %v", zerr)
		}
	}

	content, err := doctext.Extract(req.GetFileName(), req.GetContentBase64())
	if err != nil {
		return nil, adminV1.ErrorBadRequest("extract text failed: %v", err)
	}

	docName := req.GetDocName()
	if docName == "" {
		docName = strings.TrimSuffix(req.GetFileName(), filepath.Ext(req.GetFileName()))
	}

	doc, err := s.ingestDoc(ctx, operator, base, docName, content)
	if err != nil {
		return nil, err
	}
	return &aiV1.UploadAiDocResponse{Doc: doc, ChunkCount: doc.GetChunkCount()}, nil
}

// ingestDoc 切片 → 向量化 → 落库的共用主链（纯文本与文件入口共用）。
// 失败时文档行标 FAILED（错误信息保留），不让一次 embedding 故障污染整个请求语义。
func (s *AiKnowledgeService) ingestDoc(ctx context.Context, operator *authenticationV1.UserTokenPayload, base *ent.AiKnowledgeBase, docName, content string) (*aiV1.AiDoc, error) {
	if base.EmbeddingModel == nil || base.ProviderID == nil {
		return nil, adminV1.ErrorBadRequest("knowledge base embedding config is incomplete")
	}

	// P1 修复：入库与对话同受 AI_TOKENS 月度配额约束——此前 chat 有配额闸而入库没有
	//（记账≠限额），单次大文档即可在平台 API key 上打出百万级 token 计费且可循环刷量。
	if err := checkAiTokenQuota(ctx, s.usageLogRepo, s.log, operator); err != nil {
		return nil, err
	}

	// 1. 切片（带数量上限：防大文档把向量库/费用打爆）
	chunks := splitIntoChunks(content, ragChunkRunes, ragChunkOverlap)
	if len(chunks) > maxIngestChunkCount {
		return nil, adminV1.ErrorBadRequest("document too large: %d chunks exceeds limit %d", len(chunks), maxIngestChunkCount)
	}

	// 2. 建文档行（先 READY 假设成功，失败置 FAILED）
	doc, err := s.repo.CreateDoc(ctx, &aiV1.AiDoc{
		BaseId:   &base.ID,
		Name:     trans.Ptr(docName),
		Status:   trans.Ptr("READY"),
		UserId:   &operator.UserId,
		TenantId: trans.Ptr(operator.GetTenantId()),
	})
	if err != nil {
		return nil, err
	}

	// 3. 向量化（逐批对齐 reindex 口径：此前一次性提交全部切片，OpenAI 兼容
	// /v1/embeddings；消耗记入用量流水）
	vectors := make([]string, 0, len(chunks))
	for start := 0; start < len(chunks); start += reindexEmbedBatchSize {
		end := start + reindexEmbedBatchSize
		if end > len(chunks) {
			end = len(chunks)
		}
		batchVectors, embedErr := embedTextsForBase(ctx, s.providerRepo, s.usageLogRepo, s.log, operator.UserId, operator.GetTenantId(), base, chunks[start:end])
		if embedErr != nil {
			errMsg := embedErr.Error()
			_ = s.repo.UpdateDocStatus(ctx, doc.ID, "FAILED", uint32(0), errMsg)
			return nil, adminV1.ErrorInternalServerError("embed doc failed: %v", embedErr)
		}
		vectors = append(vectors, batchVectors...)
	}

	// 4. 切片落库
	if err = s.repo.InsertChunks(ctx, operator.GetTenantId(), doc.ID, chunks, vectors); err != nil {
		errMsg := err.Error()
		_ = s.repo.UpdateDocStatus(ctx, doc.ID, "FAILED", uint32(0), errMsg)
		return nil, err
	}

	chunkCount := uint32(len(chunks))
	if err = s.repo.UpdateDocStatus(ctx, doc.ID, "READY", chunkCount, ""); err != nil {
		return nil, err
	}

	doc, err = s.repo.GetDocByID(ctx, doc.ID)
	if err != nil {
		return nil, err
	}
	return s.repo.ToDocDTO(doc), nil
}

func (s *AiKnowledgeService) ListDocs(ctx context.Context, req *aiV1.ListAiDocsRequest) (*aiV1.ListAiDocsResponse, error) {
	if req.GetBaseId() == 0 {
		return nil, adminV1.ErrorBadRequest("base_id is required")
	}
	entities, err := s.repo.ListDocs(ctx, req.GetBaseId())
	if err != nil {
		return nil, err
	}
	dtos := make([]*aiV1.AiDoc, 0, len(entities))
	for _, entity := range entities {
		dtos = append(dtos, s.repo.ToDocDTO(entity))
	}
	return &aiV1.ListAiDocsResponse{Items: dtos, Total: uint64(len(dtos))}, nil
}

func (s *AiKnowledgeService) DeleteDoc(ctx context.Context, req *aiV1.DeleteAiDocRequest) (*emptypb.Empty, error) {
	if err := s.repo.DeleteDoc(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// ── 检索 ───────────────────────────────────────────────────────────

func (s *AiKnowledgeService) Search(ctx context.Context, req *aiV1.SearchAiKnowledgeRequest) (*aiV1.SearchAiKnowledgeResponse, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetBaseId() == 0 || req.GetQuery() == "" {
		return nil, adminV1.ErrorBadRequest("base_id and query are required")
	}

	topK := int(req.GetTopK())
	hits, err := s.searchBase(ctx, operator, req.GetBaseId(), req.GetQuery(), topK)
	if err != nil {
		return nil, err
	}

	resp := &aiV1.SearchAiKnowledgeResponse{Hits: make([]*aiV1.KnowledgeHit, 0, len(hits))}
	for _, hit := range hits {
		resp.Hits = append(resp.Hits, &aiV1.KnowledgeHit{
			Doc:        &aiV1.AiDoc{Id: &hit.DocID},
			ChunkIndex: hit.ChunkIndex,
			Content:    hit.Content,
			Score:      hit.Score,
		})
	}
	return resp, nil
}

// searchBase 向量化 query 并检索知识库（service 包装，供 Search RPC）。
func (s *AiKnowledgeService) searchBase(ctx context.Context, operator *authenticationV1.UserTokenPayload, baseId uint32, query string, topK int) ([]*data.ChunkHit, error) {
	return searchKnowledgeBase(ctx, s.repo, s.providerRepo, s.usageLogRepo, s.log, operator, baseId, query, topK)
}

// searchKnowledgeBase 包级检索：向量化 query 并按余弦相似度取 topK 片段。
// tenantId=0（平台用户）可检索任意库，租户用户经 SQL 的 tenant 过滤兜底；
// chat 主链路（RAG 注入）与本 service 共用。向量化消耗与 chat 同口径记入
// 用量流水（usageLogRepo），配额体系对 embedding 消耗可见。
func searchKnowledgeBase(ctx context.Context, knowledgeRepo *data.AiKnowledgeRepo, providerRepo *data.AiProviderRepo, usageLogRepo *data.AiUsageLogRepo, log *bLogger.Helper, operator *authenticationV1.UserTokenPayload, baseId uint32, query string, topK int) ([]*data.ChunkHit, error) {
	base, err := knowledgeRepo.GetEntityByID(ctx, baseId)
	if err != nil {
		return nil, err
	}

	vecLiteral, err := embedOneForBase(ctx, providerRepo, usageLogRepo, log, operator.UserId, operator.GetTenantId(), base, query)
	if err != nil {
		return nil, adminV1.ErrorInternalServerError("embed query failed: %v", err)
	}

	tenantId := operator.GetTenantId()
	return knowledgeRepo.SearchChunks(ctx, tenantId, baseId, vecLiteral, topK)
}

// ── 向量化 ─────────────────────────────────────────────────────────

// embedOneForBase 单文本向量化，返回 pgvector 字面量（包级，供检索与 chat 复用）。
func embedOneForBase(ctx context.Context, providerRepo *data.AiProviderRepo, usageLogRepo *data.AiUsageLogRepo, log *bLogger.Helper, userId, tenantId uint32, base *ent.AiKnowledgeBase, text string) (string, error) {
	vectors, err := embedTextsForBase(ctx, providerRepo, usageLogRepo, log, userId, tenantId, base, []string{text})
	if err != nil {
		return "", err
	}
	return vectors[0], nil
}

// embedTextsForBase 批量向量化，返回与 texts 对齐的 pgvector 字面量数组。
// userId/tenantId 为消耗归属（重索引等无操作者场景按知识库租户归属、userId=0）。
// embedding 消耗与 chat 同口径记入用量流水；记量失败不阻断向量化，仅记日志。
func embedTextsForBase(ctx context.Context, providerRepo *data.AiProviderRepo, usageLogRepo *data.AiUsageLogRepo, log *bLogger.Helper, userId, tenantId uint32, base *ent.AiKnowledgeBase, texts []string) ([]string, error) {
	provider, err := providerRepo.GetEntityByID(ctx, derefUint32(base.ProviderID))
	if err != nil {
		return nil, err
	}

	client, err := newOpenAIClientForProvider(ctx, provider)
	if err != nil {
		return nil, err
	}

	callStart := time.Now()
	// P3 加固：embeddings 调用补 per-call deadline（此前继承请求 ctx，Agent/
	// 重索引等无 deadline 场景下 provider 挂起会永久占用 goroutine）。
	callCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	resp, err := client.CreateEmbeddings(callCtx, openai.EmbeddingRequest{
		Model: openai.EmbeddingModel(ptrStrOr(base.EmbeddingModel, "text-embedding-3-small")),
		Input: texts,
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Data) != len(texts) {
		return nil, fmt.Errorf("embedding count mismatch: got %d want %d", len(resp.Data), len(texts))
	}

	if usageLogRepo != nil {
		modelName := ptrStrOr(base.EmbeddingModel, "text-embedding-3-small")
		promptTokens := uint32(resp.Usage.PromptTokens)
		totalTokens := uint32(resp.Usage.TotalTokens)
		durationMs := uint32(time.Since(callStart).Milliseconds())
		if uerr := usageLogRepo.Create(ctx, &aiV1.AiUsageLog{
			UserId:       trans.Ptr(userId),
			TenantId:     trans.Ptr(tenantId),
			ModelName:    &modelName,
			PromptTokens: &promptTokens,
			TotalTokens:  &totalTokens,
			DurationMs:   &durationMs,
		}); uerr != nil {
			log.Errorf(ctx, "record embedding usage failed: %v", uerr)
		}
	}

	literals := make([]string, 0, len(resp.Data))
	for _, item := range resp.Data {
		literals = append(literals, vectorToLiteral(item.Embedding))
	}
	return literals, nil
}

// vectorToLiteral float32 向量 → pgvector 字面量 '[0.1,0.2,...]'。
func vectorToLiteral(vec []float32) string {
	var sb strings.Builder
	sb.WriteByte('[')
	for i, v := range vec {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(fmt.Sprintf("%g", v))
	}
	sb.WriteByte(']')
	return sb.String()
}

// splitIntoChunks 按字符切片，带重叠保证跨片语义连续。
func splitIntoChunks(content string, size, overlap int) []string {
	runes := []rune(content)
	if len(runes) == 0 {
		return nil
	}
	step := size - overlap
	if step <= 0 {
		step = size
	}
	chunks := make([]string, 0, len(runes)/step+1)
	for start := 0; start < len(runes); start += step {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
		if end == len(runes) {
			break
		}
	}
	return chunks
}

// derefUint32 nil 安全取值。
func derefUint32(p *uint32) uint32 {
	if p == nil {
		return 0
	}
	return *p
}

// ── 重索引任务（asynq：ai_doc_reindex） ─────────────────────────────

// reindexEmbedBatchSize 每批送 /v1/embeddings 的切片数。
const reindexEmbedBatchSize = 32

// maxIngestChunkCount 单文档入库切片数上限：ragChunkRunes=500 字符/片，4000 片
// ≈2M 字符 ≈50 万 token 一次入库。此前无上限（10MB 文本 ≈2.3 万片）且一次请求
// 全量送 embeddings，单文档即可在平台 API key 上打出百万级 token 计费。
const maxIngestChunkCount = 4000

// AsyncAiDocReindex 重索引任务 handler：对指定（或全部）知识库的既有切片
// 用当前 embedding 模型重算向量。切片文本不变，只换 embedding 列——
// 场景是管理员更换了知识库的 embedding 模型（或 provider 端点）后需要重建。
func (s *AiKnowledgeService) AsyncAiDocReindex(taskType string, data *task.AiDocReindexTaskData) error {
	// asynq ctx 不携带 viewer，租户隔离 mixin 会拒绝无 viewer 查询；重索引是平台操作，用系统查看器。
	ctx := viewer.WithSystemContext(context.Background())

	var baseIds []uint32
	if data != nil && data.BaseID > 0 {
		baseIds = append(baseIds, data.BaseID)
	} else {
		bases, err := s.repo.ListBases(ctx)
		if err != nil {
			return err
		}
		for _, b := range bases {
			baseIds = append(baseIds, b.ID)
		}
	}

	for _, baseId := range baseIds {
		updated, err := s.reindexBase(ctx, baseId)
		if err != nil {
			s.log.Errorf(ctx, "reindex base %d failed: %v", baseId, err)
			continue // 单库失败不阻断其余库
		}
		s.log.Infof(ctx, "reindex base %d done: %d chunks re-embedded", baseId, updated)
	}
	return nil
}

// reindexBase 对单个知识库分批重算全部切片向量，返回更新的切片数。
func (s *AiKnowledgeService) reindexBase(ctx context.Context, baseId uint32) (int, error) {
	base, err := s.repo.GetEntityByID(ctx, baseId)
	if err != nil {
		return 0, err
	}

	refs, err := s.repo.ListChunkRefsByBase(ctx, baseId)
	if err != nil {
		return 0, err
	}
	if len(refs) == 0 {
		return 0, nil
	}

	updated := 0
	for start := 0; start < len(refs); start += reindexEmbedBatchSize {
		end := start + reindexEmbedBatchSize
		if end > len(refs) {
			end = len(refs)
		}
		batch := refs[start:end]

		texts := make([]string, 0, len(batch))
		for _, ref := range batch {
			texts = append(texts, ref.Content)
		}
		// 重索引是平台维护操作（无操作者）：消耗按知识库归属租户计量、userId=0。
		vectors, err := embedTextsForBase(ctx, s.providerRepo, s.usageLogRepo, s.log, 0, derefUint32(base.TenantID), base, texts)
		if err != nil {
			return updated, err
		}
		for i, ref := range batch {
			if err = s.repo.UpdateChunkEmbedding(ctx, ref.ID, vectors[i]); err != nil {
				return updated, err
			}
			updated++
		}
	}
	return updated, nil
}
