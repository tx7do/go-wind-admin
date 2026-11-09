package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"

	"github.com/sashabaranov/go-openai"
	"github.com/tx7do/go-utils/id"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/tx7do/kratos-transport/transport/sse"
	"google.golang.org/protobuf/encoding/protojson"

	appCrypto "github.com/tx7do/go-utils/crypto"

	pkgAi "go-wind-admin/pkg/ai"
	"go-wind-admin/pkg/sseevent"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/aimessage"
	"go-wind-admin/app/admin/service/internal/data/ent/aiprovider"
	"go-wind-admin/pkg/middleware/auth"
)

const (
	// chatCallTimeout 单轮流式对话的兜底时长：覆盖整轮（历史拼装 + 模型生成 + 落库）。
	// 流式读取节奏靠它控制，而不是 http.Client.Timeout（那会掐掉整条流）。
	chatCallTimeout = 5 * time.Minute

	// chatContextMaxMessages 拼进 LLM 上下文的历史条数上限（含本轮 user 消息）。
	chatContextMaxMessages = 20

	// conversationTitleMaxRunes 新会话默认标题截断长度（按字符，中文友好）。
	conversationTitleMaxRunes = 30
)

// AiChatPublisher SSE 推送接口（与 InternalMessagePublisher 同型——同一个
// kratos sse.Server 实例同时实现两者，装配期注入）。
type AiChatPublisher interface {
	Publish(ctx context.Context, streamId sse.StreamID, event *sse.Event)
	// TryPublish 非阻塞推送：流不存在或缓冲已满时立即返回 false。
	// chunk 允许丢帧：同步 POST 响应携带完整回复作为最终事实。
	TryPublish(ctx context.Context, streamId sse.StreamID, event *sse.Event) bool
}

// noopAiChatPublisher SSE 未配置时的占位实现：对话仍完整可用（同步响应兜底），只是不推送 chunk。
type noopAiChatPublisher struct{}

func (noopAiChatPublisher) Publish(_ context.Context, _ sse.StreamID, _ *sse.Event) {}

func (noopAiChatPublisher) TryPublish(_ context.Context, _ sse.StreamID, _ *sse.Event) bool {
	return false
}

type AiChatService struct {
	adminV1.AiChatServiceHTTPServer
	log *bLogger.Helper

	conversationRepo *data.AiConversationRepo
	messageRepo      *data.AiMessageRepo
	providerRepo     *data.AiProviderRepo
	usageLogRepo     *data.AiUsageLogRepo
	knowledgeRepo    *data.AiKnowledgeRepo

	chatPublisher AiChatPublisher
}

func NewAiChatService(
	ctx *bootstrap.Context,
	conversationRepo *data.AiConversationRepo,
	messageRepo *data.AiMessageRepo,
	providerRepo *data.AiProviderRepo,
	usageLogRepo *data.AiUsageLogRepo,
	knowledgeRepo *data.AiKnowledgeRepo,
) *AiChatService {
	return &AiChatService{
		log:              ctx.NewLoggerHelper("ai_chat/service/admin-service"),
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		providerRepo:     providerRepo,
		usageLogRepo:     usageLogRepo,
		knowledgeRepo:    knowledgeRepo,
		chatPublisher:    noopAiChatPublisher{},
	}
}

// RegisterAiChatPublisher SSE 启动时注入真身（与 RegisterInternalMessagePublisher 同模式）。
func (s *AiChatService) RegisterAiChatPublisher(publisher AiChatPublisher) {
	if publisher == nil {
		return
	}
	s.chatPublisher = publisher
}

// Chat 一轮对话：落 user 消息 → 流式调模型（chunk 经 SSE 推给发起用户）→
// 落 assistant 消息与用量流水 → 同步返回完整回复。
func (s *AiChatService) Chat(ctx context.Context, req *aiV1.ChatRequest) (*aiV1.ChatResponse, error) {
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	content := req.GetContent()
	if content == "" {
		return nil, adminV1.ErrorBadRequest("content is required")
	}

	// 1. 解析提供商（显式指定或默认）
	provider, err := s.resolveProvider(ctx, req.GetProviderId())
	if err != nil {
		return nil, err
	}

	// 2. 会话归属：复用或新建
	conversation, err := s.resolveConversation(ctx, operator, req.GetConversationId(), provider, content)
	if err != nil {
		return nil, err
	}

	// 3. 配额检查（AI_TOKENS 月度）
	if err = s.checkTokenQuota(ctx, operator); err != nil {
		return nil, err
	}

	// 4. 落 user 消息
	userEntity, err := s.messageRepo.Create(ctx, &aiV1.AiMessage{
		ConversationId: &conversation.ID,
		Role:           aiV1.AiRole_USER.Enum(),
		Content:        &content,
		UserId:         &operator.UserId,
		TenantId:       trans.Ptr(operator.GetTenantId()),
		CreatedBy:      &operator.UserId,
	})
	if err != nil {
		return nil, err
	}

	// 5. LLM 上下文：system（知识库检索注入 RAG + 提供商 system prompt）+ 历史 + 本轮
	messages := make([]openai.ChatCompletionMessage, 0, chatContextMaxMessages+2)
	systemParts := make([]string, 0, 2)
	if req.GetKnowledgeBaseId() > 0 {
		hits, searchErr := searchKnowledgeBase(ctx, s.knowledgeRepo, s.providerRepo, s.usageLogRepo, s.log, operator, req.GetKnowledgeBaseId(), content, ragTopK)
		if searchErr != nil {
			// 检索失败不阻断对话：降级为无知识库上下文
			s.log.Errorf(ctx, "knowledge search failed, degrade to plain chat: base=%d: %v", req.GetKnowledgeBaseId(), searchErr)
		} else if len(hits) > 0 {
			var kb strings.Builder
			kb.WriteString("请优先依据以下知识库片段回答用户问题，片段无关时可忽略：\n")
			for i, hit := range hits {
				kb.WriteString(fmt.Sprintf("[片段%d] %s\n", i+1, hit.Content))
			}
			systemParts = append(systemParts, kb.String())
		}
	}
	if provider.SystemPrompt != nil && *provider.SystemPrompt != "" {
		systemParts = append(systemParts, *provider.SystemPrompt)
	}
	if len(systemParts) > 0 {
		messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleSystem, Content: strings.Join(systemParts, "\n\n")})
	}
	// 多取一条：第 4 步刚落库的本轮消息必是表内最新一行，会被这次历史查询带出，
	// 循环里按 ID 剔除后恰好剩 N-1 条历史；本轮消息在下方显式 append，不剔就会重复计两次。
	history, err := s.messageRepo.ListRecentByConversation(ctx, conversation.ID, chatContextMaxMessages)
	if err != nil {
		return nil, err
	}
	for _, m := range history {
		if m.Role == nil || m.ID == userEntity.ID {
			continue
		}
		// ent 枚举是大写（USER/ASSISTANT/SYSTEM），OpenAI 协议要求小写 role——
		// 真实厂商（DeepSeek 等）会 422 拒绝大写值，必须映射。
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    aiRoleToOpenAI(*m.Role),
			Content: ptrStr(m.Content),
		})
	}
	messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: content})

	// 6. 客户端 + Function Call 多轮循环：模型请求工具 → 本地执行 → 结果回传，
	//    直到给出文本答案（最终答案轮才经 onDelta 推给前端）。
	client, err := newOpenAIClientForProvider(ctx, provider)
	if err != nil {
		return nil, err
	}

	streamCtx, cancel := context.WithTimeout(ctx, chatCallTimeout)
	defer cancel()

	streamId := strconv.FormatUint(uint64(operator.UserId), 10)
	var full strings.Builder
	var seq uint32
	var usage openai.Usage
	start := time.Now()

	runner := &aiToolLoop{
		client:    client,
		model:     ptrStrOr(provider.ModelName, "gpt-4o-mini"),
		tools:     aiBuiltinToolDefs(),
		exec:      execAiTool,
		maxRounds: aiToolMaxRounds,
		onDelta: func(delta string) {
			full.WriteString(delta)
			s.publishChunk(ctx, streamId, conversation.ID, seq, delta)
			seq++
		},
		onToolCall: func(name, arguments, result string) {
			s.publishToolEvent(ctx, streamId, conversation.ID, name, arguments, result)
		},
	}
	fullText, usage, toolErr := runner.run(streamCtx, messages)
	if toolErr != nil {
		s.log.Errorf(ctx, "ai chat stream failed: conversation=%d: %v", conversation.ID, toolErr)
		return nil, adminV1.ErrorInternalServerError("ai chat failed: %v", toolErr)
	}
	_ = fullText
	durationMs := uint32(time.Since(start).Milliseconds())

	// 8. 落 assistant 消息与用量流水
	assistantContent := fullText
	modelName := ptrStrOr(provider.ModelName, "")
	promptTokens := uint32(usage.PromptTokens)
	completionTokens := uint32(usage.CompletionTokens)
	totalTokens := uint32(usage.TotalTokens)

	assistantEntity, err := s.messageRepo.Create(ctx, &aiV1.AiMessage{
		ConversationId:   &conversation.ID,
		Role:             aiV1.AiRole_ASSISTANT.Enum(),
		Content:          &assistantContent,
		ModelName:        &modelName,
		PromptTokens:     &promptTokens,
		CompletionTokens: &completionTokens,
		DurationMs:       &durationMs,
		UserId:           &operator.UserId,
		TenantId:         trans.Ptr(operator.GetTenantId()),
		CreatedBy:        &operator.UserId,
	})
	if err != nil {
		return nil, err
	}

	if err = s.usageLogRepo.Create(ctx, &aiV1.AiUsageLog{
		ProviderId:       conversation.ProviderID,
		ConversationId:   &conversation.ID,
		UserId:           &operator.UserId,
		TenantId:         trans.Ptr(operator.GetTenantId()),
		ModelName:        &modelName,
		PromptTokens:     &promptTokens,
		CompletionTokens: &completionTokens,
		TotalTokens:      &totalTokens,
		DurationMs:       &durationMs,
	}); err != nil {
		// 记账失败不阻断对话返回：流水缺失只影响配额统计精度，记日志跟进。
		s.log.Errorf(ctx, "record ai usage log failed: conversation=%d: %v", conversation.ID, err)
	}

	if err = s.conversationRepo.TouchLastMessage(ctx, conversation.ID); err != nil {
		s.log.Errorf(ctx, "touch conversation last message failed: conversation=%d: %v", conversation.ID, err)
	}

	conversationDto := s.conversationRepo.ToDTO(conversation)
	messageDto := s.messageRepo.ToDTO(assistantEntity)
	if conversationDto == nil || messageDto == nil {
		return nil, adminV1.ErrorInternalServerError("map ai chat result failed")
	}

	return &aiV1.ChatResponse{
		Conversation: conversationDto,
		Message:      messageDto,
	}, nil
}

// resolveProvider 解析本次对话使用的提供商。
func (s *AiChatService) resolveProvider(ctx context.Context, providerId uint64) (*ent.AiProvider, error) {
	if providerId > 0 {
		entity, err := s.providerRepo.GetEntityByID(ctx, providerId)
		if err != nil {
			return nil, err
		}
		if entity.IsEnabled == nil || !*entity.IsEnabled {
			return nil, adminV1.ErrorBadRequest("ai provider is disabled")
		}
		return entity, nil
	}
	return s.providerRepo.GetEnabledDefault(ctx)
}

// resolveConversation 会话复用或新建（新建时标题取首条消息摘要）。
func (s *AiChatService) resolveConversation(ctx context.Context, operator *authenticationV1.UserTokenPayload, conversationId uint64, provider *ent.AiProvider, content string) (*ent.AiConversation, error) {
	if conversationId > 0 {
		entity, err := s.conversationRepo.GetEntityByID(ctx, conversationId)
		if err != nil {
			return nil, err
		}
		if entity.UserID == nil || *entity.UserID != operator.UserId {
			return nil, adminV1.ErrorNotFound("ai conversation not found")
		}
		return entity, nil
	}

	title := truncateRunes(content, conversationTitleMaxRunes)
	created, err := s.conversationRepo.Create(ctx, &aiV1.AiConversation{
		Title:      &title,
		ProviderId: &provider.ID,
		UserId:     &operator.UserId,
		TenantId:   trans.Ptr(operator.GetTenantId()),
		CreatedBy:  &operator.UserId,
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// checkTokenQuota 套餐 AI_TOKENS 月度配额检查；未配置该维度即不限量。
// checkAiTokenQuota 租户级月度 token 配额检查（chat 与 agent 运行共用同一口径）：
// 平台用户（tenant 0）不限；未配置限额或限额为 0 不限；当月累计达到限额拒绝发起。
func checkAiTokenQuota(ctx context.Context, usageLogRepo *data.AiUsageLogRepo, log *bLogger.Helper, operator *authenticationV1.UserTokenPayload) error {
	tenantId := operator.GetTenantId()
	if tenantId == 0 {
		return nil // 平台用户不限（运营侧自用）
	}

	limit, configured, err := usageLogRepo.FetchTenantTokenQuotaLimit(ctx, tenantId)
	if err != nil {
		return err
	}
	if !configured || limit == 0 {
		return nil
	}

	monthStart := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Local)
	used, err := usageLogRepo.SumTokensByTenantSince(ctx, tenantId, monthStart)
	if err != nil {
		return err
	}
	if used >= limit {
		log.Warnf(ctx, "ai token quota exceeded: tenant=%d used=%d limit=%d", tenantId, used, limit)
		return adminV1.ErrorBadRequest("ai token quota exceeded for this month")
	}
	return nil
}

func (s *AiChatService) checkTokenQuota(ctx context.Context, operator *authenticationV1.UserTokenPayload) error {
	return checkAiTokenQuota(ctx, s.usageLogRepo, s.log, operator)
}

// newOpenAIClientForProvider 从 provider 表行构造客户端（api_key 密文就地解密，
// 不出 service 层）；chat 与 knowledge（RAG 向量化）共用。
func newOpenAIClientForProvider(ctx context.Context, provider *ent.AiProvider) (*openai.Client, error) {
	cfg := &pkgAi.ClientConfig{
		ModelType:      modelTypeToProtoInt(provider.ModelType),
		ModelName:      ptrStrOr(provider.ModelName, ""),
		BaseUrl:        ptrStrOr(provider.BaseURL, ""),
		Organization:   ptrStrOr(provider.Organization, ""),
		LocalHost:      ptrStrOr(provider.LocalHost, ""),
		LocalPort:      int32(ptrIntOr(provider.LocalPort, 0)),
		TimeoutSeconds: int32(ptrIntOr(provider.TimeoutSeconds, 0)),
	}

	var apiKey string
	if pkgAi.ModelType(cfg.ModelType) == pkgAi.ModelTypeCloud {
		encrypted := ptrStrOr(provider.APIKey, "")
		if encrypted == "" {
			return nil, adminV1.ErrorBadRequest("ai provider api key is not configured")
		}
		plain, err := appCrypto.DecryptIfNeeded(encrypted)
		if err != nil {
			bLogger.GetLogger().Error(ctx, fmt.Sprintf("decrypt ai provider api key failed: provider=%d: %v", provider.ID, err))
			return nil, adminV1.ErrorInternalServerError("decrypt api key failed")
		}
		apiKey = plain
	}

	client, err := pkgAi.NewClient(cfg, apiKey)
	if err != nil {
		bLogger.GetLogger().Error(ctx, fmt.Sprintf("create ai client failed: provider=%d: %v", provider.ID, err))
		return nil, adminV1.ErrorInternalServerError("create ai client failed")
	}
	return client, nil
}

// modelTypeToProtoInt ent 枚举（string 底层）→ proto ModelType 数值。
// 两端枚举命名逐字一致（LOCAL/CLOUD），这里只做值域对齐。
func modelTypeToProtoInt(mt *aiprovider.ModelType) int32 {
	if mt == nil {
		return int32(aiV1.AiProvider_MODEL_TYPE_UNSPECIFIED)
	}
	switch *mt {
	case aiprovider.ModelTypeLOCAL:
		return int32(aiV1.AiProvider_LOCAL)
	case aiprovider.ModelTypeCLOUD:
		return int32(aiV1.AiProvider_CLOUD)
	default:
		return int32(aiV1.AiProvider_MODEL_TYPE_UNSPECIFIED)
	}
}

// publishChunk 尽力而为地推送一个增量片段（protojson 序列化，camelCase 键）。
func (s *AiChatService) publishChunk(ctx context.Context, streamId string, conversationId uint64, seq uint32, delta string) {
	event := &aiV1.ChatChunkEvent{
		ConversationId: conversationId,
		Seq:            seq,
		Delta:          delta,
	}
	data, err := protojson.Marshal(event)
	if err != nil {
		s.log.Errorf(ctx, "marshal ai chat chunk failed, skip push: %v", err)
		return
	}
	if ok := s.chatPublisher.TryPublish(ctx, sse.StreamID(streamId), &sse.Event{
		ID:    []byte(id.NewGUIDv4(false)),
		Data:  data,
		Event: []byte(sseevent.AIChatChunk),
	}); !ok {
		// 丢帧可容忍：同步响应携带完整回复；不刷日志避免慢客户端制造噪音。
	}
}

// publishToolEvent 尽力而为地推送一帧工具调用可见化事件（protojson 序列化，camelCase 键）。
// 与 publishChunk 同为丢帧可容忍：同步响应携带完整回复，工具帧只服务实时观感。
func (s *AiChatService) publishToolEvent(ctx context.Context, streamId string, conversationId uint64, name, arguments, result string) {
	event := &aiV1.ChatToolEvent{
		ConversationId: conversationId,
		Name:           name,
		Arguments:      arguments,
		Result:         result,
	}
	data, err := protojson.Marshal(event)
	if err != nil {
		s.log.Errorf(ctx, "marshal ai chat tool event failed, skip push: %v", err)
		return
	}
	if ok := s.chatPublisher.TryPublish(ctx, sse.StreamID(streamId), &sse.Event{
		ID:    []byte(id.NewGUIDv4(false)),
		Data:  data,
		Event: []byte(sseevent.AIChatTool),
	}); !ok {
		// 丢帧可容忍：同步响应携带完整回复；不刷日志避免慢客户端制造噪音。
	}
}

// aiRoleToOpenAI ent 大写枚举 → OpenAI 小写 role。
func aiRoleToOpenAI(role aimessage.Role) string {
	switch role {
	case aimessage.RoleUSER:
		return openai.ChatMessageRoleUser
	case aimessage.RoleASSISTANT:
		return openai.ChatMessageRoleAssistant
	case aimessage.RoleSYSTEM:
		return openai.ChatMessageRoleSystem
	default:
		return openai.ChatMessageRoleUser
	}
}

// truncateRunes 按字符截断（中文友好）。
func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func ptrStrOr(p *string, def string) string {
	if p == nil || *p == "" {
		return def
	}
	return *p
}

func ptrIntOr(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}
