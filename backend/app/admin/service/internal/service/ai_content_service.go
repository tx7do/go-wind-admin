package service

import (
	"database/sql"

	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"google.golang.org/protobuf/types/known/emptypb"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/pkg/middleware/auth"
)

const aiContentCallTimeout = 60 * time.Second

// 场景 → 系统提示词模板（%s = langName(lang)）
var aiContentScenePrompts = map[aiV1.ContentScene]string{
	aiV1.ContentScene_DESCRIPTION:  "你是企业后台的内容撰写助手。根据用户给出的主题，为后台管理系统的实体（角色/菜单/字典项/文件等）撰写一段简明的描述文字。要求：专业、简洁、准确，不超过 100 字。用%s输出，只输出正文。",
	aiV1.ContentScene_ANNOUNCEMENT: "你是企业后台的公告撰写助手。根据用户给出的主题，撰写一则面向全体员工的通知公告。要求：包含标题行、正文（背景/内容/要求），语气正式但不生硬，总长度不超过 300 字。用%s输出，只输出正文。",
	aiV1.ContentScene_REPLY:        "你是企业后台的客服回复助手。根据用户给出的问题或场景，撰写一段专业的回复文案。要求：语气友好、有条理、针对性强，不超过 200 字。用%s输出，只输出正文。",
	aiV1.ContentScene_GENERAL:      "你是企业后台的智能写作助手。根据用户的指令生成内容。要求：直接输出结果，不要解释你的思考过程。用%s输出。",
}

type AiContentService struct {
	adminV1.AiContentServiceHTTPServer
	log *bLogger.Helper

	providerRepo *data.AiProviderRepo
	usageRepo    *data.AiUsageLogRepo
	menuRepo     *data.MenuRepo
	roleRepo     interface {
		GetRolesPermissionMenuFullPaths(ctx context.Context, roleIDs []int64) ([]string, error)
	}
	userRepo interface {
		Get(ctx context.Context, req *identityV1.GetUserRequest) (*identityV1.User, error)
	}
	entClient *entCrud.EntClient[*ent.Client]
}

func NewAiContentService(
	ctx *bootstrap.Context,
	providerRepo *data.AiProviderRepo,
	usageRepo *data.AiUsageLogRepo,
	menuRepo *data.MenuRepo,
	roleRepo *data.RoleRepo,
	userRepo data.UserRepo,
	entClient *entCrud.EntClient[*ent.Client],
) *AiContentService {
	return &AiContentService{
		log:          ctx.NewLoggerHelper("ai_content/service/admin-service"),
		providerRepo: providerRepo,
		usageRepo:    usageRepo,
		menuRepo:     menuRepo,
		entClient:    entClient,
		roleRepo:     roleRepo,
		userRepo:     userRepo,
	}
}

// GenerateContent 按场景生成内容：登录用户均可使用（复用 ChatForScript 的全链路）。
// 前端在表单字段旁放 ✨ 按钮，点击后传入场景+主题，生成结果由前端自行填入字段。
func (s *AiContentService) GenerateContent(ctx context.Context, req *aiV1.GenerateContentRequest) (*aiV1.GenerateContentResponse, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	topic := strings.TrimSpace(req.GetTopic())
	if topic == "" {
		return nil, adminV1.ErrorBadRequest("topic is required")
	}

	lang := req.GetLang()
	if lang == "" {
		lang = "zh-CN"
	}
	langDisplay := langName(lang)

	scene := req.GetScene()
	systemPrompt, ok := aiContentScenePrompts[scene]
	if !ok {
		systemPrompt = aiContentScenePrompts[aiV1.ContentScene_GENERAL]
	}
	systemPrompt = fmt.Sprintf(systemPrompt, langDisplay)

	// 补充上下文作为追加指令
	if extra := strings.TrimSpace(req.GetContext()); extra != "" {
		systemPrompt += "\n补充要求：" + extra
	}
	if req.GetMaxLength() > 0 {
		systemPrompt += fmt.Sprintf("\n输出长度不超过 %d 字。", req.GetMaxLength())
	}

	provider, err := s.providerRepo.GetEnabledDefault(ctx)
	if err != nil {
		return nil, err
	}
	client, err := newOpenAIClientForProvider(ctx, provider)
	if err != nil {
		return nil, err
	}

	callStart := time.Now()
	cCtx, cancel := context.WithTimeout(ctx, aiContentCallTimeout)
	defer cancel()
	resp, err := client.CreateChatCompletion(cCtx, openai.ChatCompletionRequest{
		Model: ptrStrOr(provider.ModelName, "gpt-4o-mini"),
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: topic},
		},
	})
	if err != nil {
		s.log.Errorf(ctx, "ai content generate failed: scene=%s: %v", scene.String(), err)
		return nil, adminV1.ErrorInternalServerError("ai content generate failed: %v", err)
	}

	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	tokens := uint32(resp.Usage.TotalTokens)
	durationMs := uint32(time.Since(callStart).Milliseconds())

	// 记用量（尽力而为，不阻断返回）
	if s.usageRepo != nil {
		modelName := ptrStrOr(provider.ModelName, "")
		promptTokens := uint32(resp.Usage.PromptTokens)
		completionTokens := uint32(resp.Usage.CompletionTokens)
		totalTokens := uint32(resp.Usage.TotalTokens)
		if uerr := s.usageRepo.Create(ctx, &aiV1.AiUsageLog{
			UserId:           trans.Ptr(operator.UserId),
			TenantId:         trans.Ptr(operator.GetTenantId()),
			ModelName:        &modelName,
			PromptTokens:     &promptTokens,
			CompletionTokens: &completionTokens,
			TotalTokens:      &totalTokens,
			DurationMs:       &durationMs,
		}); uerr != nil {
			s.log.Errorf(ctx, "ai content usage log failed: %v", uerr)
		}
	}

	return &aiV1.GenerateContentResponse{
		Content:     content,
		TotalTokens: tokens,
	}, nil
}

// ── 语义搜索（pgvector 升级全局搜索） ────────────────────────────────

// buildSearchIndexSQL 菜单语义搜索索引表。embedding 定维（见 data.AiEmbeddingDimensions）：
// 混维向量会让 <=> 比较在查询期整体报错，定维把失败提前到写入期。
var buildSearchIndexSQL = fmt.Sprintf(
	"CREATE TABLE IF NOT EXISTS sys_ai_search_index ("+
		"id BIGSERIAL PRIMARY KEY, "+
		"created_at TIMESTAMPTZ DEFAULT NOW(), "+
		"item_type VARCHAR(20) NOT NULL DEFAULT 'menu', "+
		"item_id BIGINT NOT NULL, "+
		"title TEXT NOT NULL, "+
		"route TEXT NOT NULL DEFAULT '', "+
		"embedding vector(%d))",
	data.AiEmbeddingDimensions)

// menuIndexDoc 待索引菜单：真实菜单 id + 标题 + 路由。
// item_id 落库为菜单 id（历史版本误存过数组下标，该列此前无消费方故无影响）。
type menuIndexDoc struct {
	menuId int64
	title  string
	route  string
}

// collectMenuIndexDocs 从菜单列表收集可索引文档：跳过无 id / 无 path /
// 无标题的菜单；标题取 meta.title，为空回退 name（与全局搜索原始行为一致）。
// 输出与输入顺序一致，item_id 落真实菜单 id。
func collectMenuIndexDocs(items []*permissionV1.Menu) []menuIndexDoc {
	docs := make([]menuIndexDoc, 0, len(items))
	for _, m := range items {
		if m.GetId() == 0 || m.GetPath() == "" {
			continue
		}
		title := ""
		if mt := m.GetMeta(); mt != nil {
			title = mt.GetTitle()
		}
		if title == "" {
			title = m.GetName()
		}
		if title == "" {
			continue
		}
		docs = append(docs, menuIndexDoc{menuId: m.GetId(), title: title, route: m.GetPath()})
	}
	return docs
}

// placeEmbeddingBatch 把一批 embedding 响应按其 index（对位请求输入序号）
// 写入 vectors 的 [batchStart, batchStart+len) 段：拒绝越界 index 与零维向量。
// 语义检索索引用于全局搜索导航，宁可整体失败也不落错位的行。
func placeEmbeddingBatch(batchStart int, data []openai.Embedding, vectors []string) error {
	for _, item := range data {
		if item.Index < 0 || batchStart+item.Index >= len(vectors) || len(item.Embedding) == 0 {
			return fmt.Errorf("embedding response malformed: batch=%d item=%d", batchStart, item.Index)
		}
		vectors[batchStart+item.Index] = vectorToLiteral(item.Embedding)
	}
	return nil
}

// BuildMenuSearchIndex 向量化全部菜单标题并原子替换索引表里的菜单行。
// 顺序：先离线算完全部 embedding（不持事务调外部 API），再单事务 DELETE+分批
// 多行 INSERT——任一环节失败即回滚，旧索引原样保留，不会留下半空索引。
func (s *AiContentService) BuildMenuSearchIndex(ctx context.Context) (int64, error) {
	db := s.entClient.DB()
	if _, err := db.ExecContext(ctx, buildSearchIndexSQL); err != nil {
		return 0, fmt.Errorf("create search index table: %w", err)
	}
	rebuilt, err := data.EnsureVectorColumnDim(ctx, db, "sys_ai_search_index")
	if err != nil {
		return 0, fmt.Errorf("ensure embedding column: %w", err)
	}
	if rebuilt {
		s.log.Warnf(ctx, "sys_ai_search_index.embedding 与当前 embedding 模型维度不符，已整列重建，本次全量重写菜单索引")
	}
	operator, oErr := auth.FromContext(ctx)
	if oErr != nil {
		return 0, oErr
	}
	menus, err := s.menuRepo.List(ctx, paginationV1PagingNoLimit(), true)
	if err != nil {
		return 0, err
	}

	docs := collectMenuIndexDocs(menus.GetItems())
	if len(docs) == 0 {
		return 0, nil
	}

	provider, err := s.providerRepo.GetEnabledDefault(ctx)
	if err != nil {
		return 0, err
	}
	client, err := newOpenAIClientForProvider(ctx, provider)
	if err != nil {
		return 0, err
	}

	// 离线算完全部向量。响应按 OpenAI 规约携带 index（对位请求输入序号），
	// 对位与完整性校验在 placeEmbeddingBatch：缺号/错号/零维整体失败。
	// 向量化消耗与 chat 同口径记入用量流水（记量失败不阻断，仅记日志）。
	vectors := make([]string, len(docs))
	embCtx, embCancel := context.WithTimeout(ctx, 60*time.Second)
	defer embCancel()
	for start := 0; start < len(docs); start += 32 {
		end := min(start+32, len(docs))
		batch := make([]string, 0, end-start)
		for _, d := range docs[start:end] {
			batch = append(batch, d.title)
		}
		callStart := time.Now()
		resp, embErr := client.CreateEmbeddings(embCtx, openai.EmbeddingRequest{
			Model: openai.EmbeddingModel("text-embedding-3-small"),
			Input: batch,
		})
		if embErr != nil {
			return 0, fmt.Errorf("embed batch: %w", embErr)
		}
		if placeErr := placeEmbeddingBatch(start, resp.Data, vectors); placeErr != nil {
			return 0, placeErr
		}
		if s.usageRepo != nil {
			modelName := "text-embedding-3-small"
			promptTokens := uint32(resp.Usage.PromptTokens)
			totalTokens := uint32(resp.Usage.TotalTokens)
			durationMs := uint32(time.Since(callStart).Milliseconds())
			if uerr := s.usageRepo.Create(ctx, &aiV1.AiUsageLog{
				UserId:       trans.Ptr(operator.UserId),
				TenantId:     trans.Ptr(operator.GetTenantId()),
				ModelName:    &modelName,
				PromptTokens: &promptTokens,
				TotalTokens:  &totalTokens,
				DurationMs:   &durationMs,
			}); uerr != nil {
				s.log.Errorf(ctx, "record embedding usage failed: %v", uerr)
			}
		}
	}
	for i := range vectors {
		if vectors[i] == "" {
			return 0, fmt.Errorf("menu %d missing embedding response", docs[i].menuId)
		}
	}

	tx, txErr := db.BeginTx(ctx, nil)
	if txErr != nil {
		return 0, fmt.Errorf("begin tx: %w", txErr)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if _, delErr := tx.ExecContext(ctx, "DELETE FROM sys_ai_search_index WHERE item_type = 'menu'"); delErr != nil {
		return 0, fmt.Errorf("clear search index: %w", delErr)
	}

	const rowsPerStmt = 100
	for start := 0; start < len(docs); start += rowsPerStmt {
		end := min(start+rowsPerStmt, len(docs))
		var sb strings.Builder
		var args []any
		sb.WriteString("INSERT INTO sys_ai_search_index (item_type, item_id, title, route, embedding) VALUES ")
		for j := start; j < end; j++ {
			if j > start {
				sb.WriteByte(',')
			}
			base := (j - start) * 5
			_, _ = fmt.Fprintf(&sb, "($%d,$%d,$%d,$%d,$%d::vector)", base+1, base+2, base+3, base+4, base+5)
			args = append(args, "menu", int64(docs[j].menuId), docs[j].title, docs[j].route, vectors[j])
		}
		if _, insErr := tx.ExecContext(ctx, sb.String(), args...); insErr != nil {
			return 0, fmt.Errorf("insert search index rows: %w", insErr)
		}
	}

	if cErr := tx.Commit(); cErr != nil {
		return 0, fmt.Errorf("commit search index: %w", cErr)
	}
	committed = true
	return int64(len(docs)), nil
}

// SemanticSearch 语义搜索：向量化查询 → pgvector 余弦检索菜单索引。
// 查询向量化消耗与 chat 同口径记入用量流水（记量失败不阻断，仅记日志）。
func (s *AiContentService) SemanticSearch(ctx context.Context, req *aiV1.SemanticSearchRequest) (*aiV1.SemanticSearchResponse, error) {
	operator, oErr := auth.FromContext(ctx)
	if oErr != nil {
		return nil, oErr
	}
	query := strings.TrimSpace(req.GetQuery())
	if query == "" {
		return &aiV1.SemanticSearchResponse{}, nil
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 8
	}

	// 权限裁剪前置：非平台用户按其可见菜单集过滤索引行（item_id = menu_id）。
	// 平台管理员（tenantId==0）不过滤——与 GetNavigation 的口径一致。
	// 放在 embedding 之前：无可见菜单直接返回空结果，省一次向量调用。
	allowedMenuIDs, filterErr := s.visibleMenuIDs(ctx, operator)
	if filterErr != nil {
		s.log.Errorf(ctx, "semantic search: resolve visible menus failed: %v", filterErr)
		return nil, adminV1.ErrorInternalServerError("resolve accessible menus failed")
	}
	if operator.GetTenantId() != 0 && len(allowedMenuIDs) == 0 {
		return &aiV1.SemanticSearchResponse{Items: []*aiV1.SemanticSearchItem{}}, nil
	}

	provider, err := s.providerRepo.GetEnabledDefault(ctx)
	if err != nil {
		return nil, err
	}
	client, err := newOpenAIClientForProvider(ctx, provider)
	if err != nil {
		return nil, err
	}

	embCtx, embCancel := context.WithTimeout(ctx, 30*time.Second)
	defer embCancel()
	callStart := time.Now()
	embResp, embErr := client.CreateEmbeddings(embCtx, openai.EmbeddingRequest{
		Model: openai.EmbeddingModel("text-embedding-3-small"),
		Input: []string{query},
	})
	if embErr != nil {
		return nil, adminV1.ErrorInternalServerError("embed query failed: %v", embErr)
	}
	if len(embResp.Data) == 0 || len(embResp.Data[0].Embedding) == 0 {
		return nil, adminV1.ErrorInternalServerError("embed query returned empty embedding")
	}
	queryVec := vectorToLiteral(embResp.Data[0].Embedding)

	if s.usageRepo != nil {
		modelName := "text-embedding-3-small"
		promptTokens := uint32(embResp.Usage.PromptTokens)
		totalTokens := uint32(embResp.Usage.TotalTokens)
		durationMs := uint32(time.Since(callStart).Milliseconds())
		if uerr := s.usageRepo.Create(ctx, &aiV1.AiUsageLog{
			UserId:       trans.Ptr(operator.UserId),
			TenantId:     trans.Ptr(operator.GetTenantId()),
			ModelName:    &modelName,
			PromptTokens: &promptTokens,
			TotalTokens:  &totalTokens,
			DurationMs:   &durationMs,
		}); uerr != nil {
			s.log.Errorf(ctx, "record embedding usage failed: %v", uerr)
		}
	}

	db := s.entClient.DB()
	var rows *sql.Rows
	var dbErr error
	if operator.GetTenantId() == 0 {
		rows, dbErr = db.QueryContext(ctx,
			"SELECT title, route, 1 - (embedding <=> $1::vector) AS score FROM sys_ai_search_index WHERE item_type = 'menu' ORDER BY embedding <=> $1::vector LIMIT $2",
			queryVec, limit,
		)
	} else {
		// 动态 IN 占位：menuIDs 量级小（菜单树），直接展开
		placeholders := make([]string, len(allowedMenuIDs))
		args := make([]any, 0, len(allowedMenuIDs)+2)
		args = append(args, queryVec)
		for i, id := range allowedMenuIDs {
			placeholders[i] = fmt.Sprintf("$%d", i+2)
			args = append(args, id)
		}
		args = append(args, limit)
		sqlText := fmt.Sprintf(
			"SELECT title, route, 1 - (embedding <=> $1::vector) AS score FROM sys_ai_search_index WHERE item_type = 'menu' AND item_id IN (%s) ORDER BY embedding <=> $1::vector LIMIT $%d",
			strings.Join(placeholders, ", "), len(args),
		)
		rows, dbErr = db.QueryContext(ctx, sqlText, args...)
	}
	if dbErr != nil {
		s.log.Errorf(ctx, "semantic search query failed: %s", dbErr.Error())
		return nil, adminV1.ErrorInternalServerError("semantic search failed")
	}
	defer func() { _ = rows.Close() }()

	resp := &aiV1.SemanticSearchResponse{Items: make([]*aiV1.SemanticSearchItem, 0, limit)}
	for rows.Next() {
		var title, route string
		var score float64
		if err = rows.Scan(&title, &route, &score); err != nil {
			continue
		}
		resp.Items = append(resp.Items, &aiV1.SemanticSearchItem{
			Title: title, Route: route, ItemType: "menu", Score: float32(score),
		})
	}
	return resp, rows.Err()
}

// visibleMenuIDs 解析调用者可见的菜单 ID 集。
// 平台管理员返回 nil（调用方不过滤）；普通用户经 角色ID → 权限 → 菜单 链路解析。
// userRepo.Get 拿用户角色集；解析失败返回错误（fail-closed：AI 搜索不应泄露
// 越权菜单，解析不出来宁可不返回结果）。
func (s *AiContentService) visibleMenuIDs(ctx context.Context, operator *authenticationV1.UserTokenPayload) ([]int64, error) {
	if operator.GetTenantId() == 0 {
		return nil, nil // 平台管理员：不过滤
	}
	user, err := s.userRepo.Get(ctx, &identityV1.GetUserRequest{
		QueryBy: &identityV1.GetUserRequest_Id{Id: operator.GetUserId()},
	})
	if err != nil {
		return nil, err
	}
	if user == nil || len(user.GetRoleIds()) == 0 {
		return []int64{}, nil
	}
	// 菜单授权绑定存全路径快照：先取路径，再反解回当前菜单 id
	paths, err := s.roleRepo.GetRolesPermissionMenuFullPaths(ctx, user.GetRoleIds())
	if err != nil {
		return nil, err
	}
	return s.menuRepo.ResolveMenuIDsByFullPaths(ctx, paths)
}

// RebuildSearchIndex 重建搜索索引 RPC。
func (s *AiContentService) RebuildSearchIndex(ctx context.Context, _ *emptypb.Empty) (*aiV1.RebuildSearchIndexResponse, error) {
	count, err := s.BuildMenuSearchIndex(ctx)
	if err != nil {
		return nil, adminV1.ErrorInternalServerError("rebuild search index failed: %v", err)
	}
	return &aiV1.RebuildSearchIndexResponse{IndexedCount: uint32(count)}, nil
}

// paginationV1PagingNoLimit 构造不分页的 PagingRequest。
func paginationV1PagingNoLimit() *paginationV1.PagingRequest {
	return &paginationV1.PagingRequest{NoPaging: trans.Ptr(true)}
}
