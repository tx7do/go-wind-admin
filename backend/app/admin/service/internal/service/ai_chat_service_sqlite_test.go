// AiChatService 的 SQLite 内存库 + 假 LLM（httptest SSE）测试（白盒，包内测试）。
//
// 覆盖目标：
//   - Chat 主链路可测段：入参守卫（无身份/空内容）、提供商解析（禁用/不存在/默认兜底）、
//     会话归属（复用自己的/他人会话 NotFound）、配额闸门（平台不限/未配置不限/超限拒绝）；
//   - 全链路往返（假 OpenAI 兼容 SSE 服务）：user/assistant 消息落库、用量流水落库、
//     会话 last_message_at 触碰、SSE chunk 帧经 publisher 推出（protojson camelCase）、
//     第二轮上下文组装（system+历史按 OpenAI 小写 role 拼装）；
//   - SSE 推送缝：publishChunk / publishToolEvent 的事件名与载荷、丢帧（TryPublish=false）
//     不炸主链路；
//   - newOpenAIClientForProvider 的密钥/模型类型守卫；
//   - 纯函数：aiRoleToOpenAI / modelTypeToProtoInt / truncateRunes / ptr 系列兜底。
//
// 跳过项：知识库 RAG 注入（searchKnowledgeBase 需向量化与 pgvector，PG 门控集成另有覆盖）。
package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/tx7do/kratos-transport/transport/sse"
	"google.golang.org/protobuf/encoding/protojson"

	entCrud "github.com/tx7do/go-crud/entgo"
	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/aiconversation"
	"go-wind-admin/app/admin/service/internal/data/ent/aimessage"
	"go-wind-admin/app/admin/service/internal/data/ent/aiprovider"
	"go-wind-admin/app/admin/service/internal/data/ent/planquota"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
	"go-wind-admin/pkg/sseevent"
)

// newAiChatServiceForTest 白盒复刻 NewAiChatService 的字段初始化：
// log 换 NopLogger，repo 走 data 层 ForTest 构造器，chatPublisher 留 noop（与生产一致）。
// knowledgeRepo 在 Chat 主链路仅在带 knowledge_base_id 时使用，这里置 nil 不影响。
func newAiChatServiceForTest(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *AiChatService {
	t.Helper()
	return &AiChatService{
		log:              bLogger.NewHelper(bLogger.NopLogger()),
		conversationRepo: data.NewAiConversationRepoForTest(entClient),
		messageRepo:      data.NewAiMessageRepoForTest(entClient),
		providerRepo:     data.NewAiProviderRepoForTest(entClient),
		usageLogRepo:     data.NewAiUsageLogRepoForTest(entClient),
		knowledgeRepo:    nil,
		chatPublisher:    noopAiChatPublisher{},
	}
}

// aiChatPublisherStub 记录全部推送帧的 publisher 桩；tryFail=true 时模拟
// 流不存在/缓冲满（TryPublish 返回 false，丢帧可容忍语义）。
type aiChatPublisherStub struct {
	mu      sync.Mutex
	events  []aiChatPublishedEvent
	tryFail bool
}

type aiChatPublishedEvent struct {
	streamId sse.StreamID
	name     string
	data     []byte
}

func (s *aiChatPublisherStub) Publish(_ context.Context, streamId sse.StreamID, event *sse.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, aiChatPublishedEvent{streamId: streamId, name: string(event.Event), data: event.Data})
}

func (s *aiChatPublisherStub) TryPublish(ctx context.Context, streamId sse.StreamID, event *sse.Event) bool {
	s.Publish(ctx, streamId, event)
	return !s.tryFail
}

func (s *aiChatPublisherStub) named(name string) []aiChatPublishedEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []aiChatPublishedEvent
	for _, e := range s.events {
		if e.name == name {
			out = append(out, e)
		}
	}
	return out
}

// aiChatSeedProvider 直插一个提供商行，按需覆盖启用/默认形态（密钥/模型名固定，
// 全局加密器在测试进程未初始化，api_key 明文直通）。
func aiChatSeedProvider(t *testing.T, entClient *entCrud.EntClient[*ent.Client], ctx context.Context, name, baseURL string, enabled, isDefault bool) uint64 {
	t.Helper()
	row, err := entClient.Client().AiProvider.Create().
		SetNillableName(trans.Ptr(name)).
		SetNillableModelType(trans.Ptr(aiprovider.ModelTypeCLOUD)).
		SetNillableModelName(trans.Ptr("test-model")).
		SetNillableBaseURL(trans.Ptr(baseURL)).
		SetNillableAPIKey(trans.Ptr("sk-plain-test-key")).
		SetNillableSystemPrompt(trans.Ptr("你是测试助手")).
		SetNillableIsEnabled(trans.Ptr(enabled)).
		SetNillableIsDefault(trans.Ptr(isDefault)).
		Save(ctx)
	require.NoError(t, err, "种子提供商应直插成功")
	return row.ID
}

// aiChatSeedConversation 直插一个归属指定用户的会话行。
func aiChatSeedConversation(t *testing.T, entClient *entCrud.EntClient[*ent.Client], ctx context.Context, title string, userID uint64) uint64 {
	t.Helper()
	row, err := entClient.Client().AiConversation.Create().
		SetNillableTitle(trans.Ptr(title)).
		SetNillableUserID(trans.Ptr(userID)).
		SetNillableTenantID(trans.Ptr(uint64(0))).
		Save(ctx)
	require.NoError(t, err, "种子会话应直插成功")
	return row.ID
}

// aiChatNewOperator 构造指定用户/租户的操作者上下文。
func aiChatNewOperator(ctx context.Context, userID uint64, tenantID ...uint64) context.Context {
	payload := &authenticationV1.UserTokenPayload{UserId: userID}
	if len(tenantID) > 0 {
		payload.TenantId = trans.Ptr(tenantID[0])
	}
	return auth.NewContext(ctx, payload)
}

// startAiChatFakeLLM 起一个 OpenAI 兼容的流式假服务（纯文本答案，不带工具调用）：
// 每次请求固定回两帧文本增量（"你好" + "，世界"）+ 一帧 usage + [DONE]。
// 返回的读取函数给出累计的请求 body 供上下文组装断言。
func startAiChatFakeLLM(t *testing.T) (*httptest.Server, func() []map[string]any) {
	t.Helper()
	var mu sync.Mutex
	var bodies []map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		bodies = append(bodies, body)
		mu.Unlock()

		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		writeChunk := func(payload map[string]any) {
			raw, _ := json.Marshal(payload)
			_, _ = w.Write([]byte("data: " + string(raw) + "\n\n"))
			flusher.Flush()
		}
		writeChunk(map[string]any{"choices": []any{map[string]any{
			"delta": map[string]any{"content": "你好"},
		}}})
		writeChunk(map[string]any{"choices": []any{map[string]any{
			"delta":         map[string]any{"content": "，世界"},
			"finish_reason": "stop",
		}}})
		// IncludeUsage=true 的契约：最后一帧只带 usage、choices 为空
		writeChunk(map[string]any{"choices": []any{}, "usage": map[string]any{
			"prompt_tokens": 11, "completion_tokens": 7, "total_tokens": 18,
		}})
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, func() []map[string]any {
		mu.Lock()
		defer mu.Unlock()
		return append([]map[string]any{}, bodies...)
	}
}

// aiChatBodyMessages 取假服务请求 body 里的 messages 数组（map 形态）。
func aiChatBodyMessages(t *testing.T, body map[string]any) []map[string]any {
	t.Helper()
	raw, ok := body["messages"].([]any)
	require.True(t, ok, "请求 body 应带 messages 数组")
	out := make([]map[string]any, 0, len(raw))
	for _, m := range raw {
		out = append(out, m.(map[string]any))
	}
	return out
}

// TestAiChatService_ChatGuards 验证 Chat 的入参守卫链：无身份、空内容、
// 提供商禁用/不存在。这些分支在任何模型调用之前就该被拦下。
func TestAiChatService_ChatGuards(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newAiChatServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := aiChatNewOperator(ctx, 81)
	disabledID := aiChatSeedProvider(t, entClient, ctx, "停用提供商", "http://127.0.0.1:9/v1", false, false)

	// 无身份：未走 auth.NewContext 的裸 ctx 必须拒绝
	_, err := svc.Chat(ctx, &aiV1.ChatRequest{Content: trans.Ptr("hi")})
	require.Error(t, err, "无操作者身份的对话必须被拒绝")

	// 空 content
	_, err = svc.Chat(opCtx, &aiV1.ChatRequest{})
	require.Error(t, err, "空 content 的对话必须被拒绝")
	require.Contains(t, err.Error(), "content is required")

	// 显式指定已禁用的提供商
	_, err = svc.Chat(opCtx, &aiV1.ChatRequest{Content: trans.Ptr("hi"), ProviderId: trans.Ptr(disabledID)})
	require.Error(t, err, "禁用提供商上的对话必须被拒绝")
	require.Contains(t, err.Error(), "ai provider is disabled")

	// 显式指定不存在的提供商
	_, err = svc.Chat(opCtx, &aiV1.ChatRequest{Content: trans.Ptr("hi"), ProviderId: trans.Ptr(uint64(424242))})
	require.Error(t, err, "不存在提供商上的对话必须被拒绝")
	require.Contains(t, err.Error(), "ai provider not found")
}

// TestAiChatService_ResolveProvider 验证提供商解析三分支：显式 ID 命中启用行、
// 未指定时回退"默认优先，其次任一启用"、全无启用行时报 NotFound。
func TestAiChatService_ResolveProvider(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newAiChatServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	// 空库：未指定 ID → no enabled ai provider
	_, err := svc.resolveProvider(ctx, 0)
	require.Error(t, err, "无任何提供商时默认解析应报错")
	require.Contains(t, err.Error(), "no enabled ai provider")

	// 只有一个非默认的启用提供商：未指定 ID → 兜底取任一启用
	anyID := aiChatSeedProvider(t, entClient, ctx, "普通启用提供商", "http://127.0.0.1:9/v1", true, false)
	entity, err := svc.resolveProvider(ctx, 0)
	require.NoError(t, err)
	require.Equal(t, anyID, entity.ID, "无默认时应兜底取任一启用提供商")

	// 再插一个"默认且启用"的：未指定 ID → 优先取默认
	defaultID := aiChatSeedProvider(t, entClient, ctx, "默认提供商", "http://127.0.0.1:9/v1", true, true)
	entity, err = svc.resolveProvider(ctx, 0)
	require.NoError(t, err)
	require.Equal(t, defaultID, entity.ID, "有默认时必须优先取默认提供商")

	// 显式 ID：命中即返回
	entity, err = svc.resolveProvider(ctx, anyID)
	require.NoError(t, err)
	require.Equal(t, anyID, entity.ID)
}

// TestAiChatService_ResolveConversation 验证会话归属：复用自己的会话、
// 他人的会话一律 NotFound（不泄露存在性）、新建时标题按 30 字符截断并落归属。
func TestAiChatService_ResolveConversation(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newAiChatServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	providerID := aiChatSeedProvider(t, entClient, ctx, "归属校验提供商", "http://127.0.0.1:9/v1", true, false)
	operator := &authenticationV1.UserTokenPayload{UserId: 81, TenantId: trans.Ptr(uint64(5))}

	// 他人会话：归属校验失败必须 NotFound（把越权读藏进 404，不暴露会话存在性）
	othersConv := aiChatSeedConversation(t, entClient, ctx, "张三的会话", 44)
	_, err := svc.resolveConversation(ctx, operator, othersConv, nil, "hi")
	require.Error(t, err, "复用他人会话必须被拒绝")
	require.Contains(t, err.Error(), "ai conversation not found")

	// 自己的会话：复用返回原行
	ownConv := aiChatSeedConversation(t, entClient, ctx, "李四的会话", 81)
	entity, err := svc.resolveConversation(ctx, operator, ownConv, nil, "hi")
	require.NoError(t, err)
	require.Equal(t, ownConv, entity.ID, "复用自己的会话应返回原会话")
	require.Equal(t, "李四的会话", *entity.Title, "复用路径不得改写标题")

	// 新建：标题取首条消息摘要并按字符截断（中文友好，30 runes）
	long := ""
	for i := 0; i < 40; i++ {
		long += "测"
	}
	created, err := svc.resolveConversation(ctx, operator, 0, &ent.AiProvider{ID: providerID}, long)
	require.NoError(t, err)
	require.NotZero(t, created.ID, "新建会话应落库并拿到 ID")
	require.Len(t, []rune(*created.Title), conversationTitleMaxRunes, "标题应按 30 字符截断")
	require.Equal(t, providerID, *created.ProviderID, "新会话应绑定本次解析出的提供商")
	require.Equal(t, uint64(81), *created.UserID, "会话必须归属操作者")
	require.Equal(t, uint64(5), *created.TenantID, "会话必须归属操作者租户")
}

// TestAiChatService_CheckAiTokenQuota 验证 AI_TOKENS 月度配额闸门四态：
// 平台用户（tenant 0）不限、未配置不限、用量低于限额放行、达到/超过限额拒绝。
// 闸门算错一次就是"多扣或饿死一个租户"。
func TestAiChatService_CheckAiTokenQuota(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	usageRepo := data.NewAiUsageLogRepoForTest(entClient)
	log := bLogger.NewHelper(bLogger.NopLogger())
	ctx := enttest.NewSystemContext(context.Background())

	// 平台用户（tenant 0）：直接放行，无需任何种子
	require.NoError(t, checkAiTokenQuota(ctx, usageRepo, log, &authenticationV1.UserTokenPayload{UserId: 1}),
		"平台用户（tenantId=0）应不限量")

	// 租户 A：套餐配了 AI_TOKENS=100，本月已用 100（>= 限额）→ 拒绝
	planFull, err := entClient.Client().Plan.Create().SetNillableName(trans.Ptr("满额套餐")).Save(ctx)
	require.NoError(t, err)
	_, err = entClient.Client().PlanQuota.Create().
		SetPlan(planFull).
		SetQuotaType(planquota.QuotaTypeAiTokens).
		SetQuotaValue(100).
		Save(ctx)
	require.NoError(t, err)
	tenantFull, err := entClient.Client().Tenant.Create().
		SetNillableName(trans.Ptr("满额租户")).
		SetPlan(planFull).
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, usageRepo.Create(ctx, &aiV1.AiUsageLog{
		TenantId: trans.Ptr(tenantFull.ID), TotalTokens: trans.Ptr(uint32(60)), ModelName: trans.Ptr("m"),
	}), "种子用量行应落库")
	require.NoError(t, usageRepo.Create(ctx, &aiV1.AiUsageLog{
		TenantId: trans.Ptr(tenantFull.ID), TotalTokens: trans.Ptr(uint32(40)), ModelName: trans.Ptr("m"),
	}))
	// 别家租户的大额流水不得算进租户 A 的账
	_, err = entClient.Client().AiUsageLog.Create().
		SetTenantID(999).
		SetTotalTokens(999999).
		SetModelName("m").
		Save(ctx)
	require.NoError(t, err)

	err = checkAiTokenQuota(ctx, usageRepo, log, &authenticationV1.UserTokenPayload{
		UserId: 2, TenantId: trans.Ptr(tenantFull.ID),
	})
	require.Error(t, err, "达到限额的租户应被拒绝发起")
	require.Contains(t, err.Error(), "ai token quota exceeded")

	// 租户 B：套餐配 AI_TOKENS=100，已用 99（< 限额）→ 放行
	planHeadroom, err := entClient.Client().Plan.Create().SetNillableName(trans.Ptr("有余量套餐")).Save(ctx)
	require.NoError(t, err)
	_, err = entClient.Client().PlanQuota.Create().
		SetPlan(planHeadroom).
		SetQuotaType(planquota.QuotaTypeAiTokens).
		SetQuotaValue(100).
		Save(ctx)
	require.NoError(t, err)
	tenantHeadroom, err := entClient.Client().Tenant.Create().
		SetNillableName(trans.Ptr("有余量租户")).
		SetPlan(planHeadroom).
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, usageRepo.Create(ctx, &aiV1.AiUsageLog{
		TenantId: trans.Ptr(tenantHeadroom.ID), TotalTokens: trans.Ptr(uint32(99)), ModelName: trans.Ptr("m"),
	}))
	require.NoError(t, checkAiTokenQuota(ctx, usageRepo, log, &authenticationV1.UserTokenPayload{
		UserId: 3, TenantId: trans.Ptr(tenantHeadroom.ID),
	}), "低于限额的租户应放行")

	// 租户 C：无套餐 → 未配置即不限量（"没配 = 不限"的缺省语义）
	tenantBare, err := entClient.Client().Tenant.Create().
		SetNillableName(trans.Ptr("裸租户")).
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, checkAiTokenQuota(ctx, usageRepo, log, &authenticationV1.UserTokenPayload{
		UserId: 4, TenantId: trans.Ptr(tenantBare.ID),
	}), "无套餐租户应视为不限量")
}

// TestAiChatService_ChatRoundTrip 全链路往返：假 LLM（httptest SSE）+ 推送桩，
// 钉住一次成功对话的全部副作用——user/assistant 消息落库、用量流水落库、
// 会话 last_message_at 触碰、SSE chunk 帧推送、同步响应 DTO、第二轮上下文组装。
func TestAiChatService_ChatRoundTrip(t *testing.T) {
	fake, bodies := startAiChatFakeLLM(t)
	entClient := enttest.NewEntClientForTest(t)
	svc := newAiChatServiceForTest(t, entClient)
	stub := &aiChatPublisherStub{}
	svc.RegisterAiChatPublisher(stub)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := aiChatNewOperator(ctx, 81)

	providerID := aiChatSeedProvider(t, entClient, ctx, "往返测试提供商", fake.URL+"/v1", true, false)

	resp, err := svc.Chat(opCtx, &aiV1.ChatRequest{Content: trans.Ptr("打个招呼"), ProviderId: &providerID})
	require.NoError(t, err, "假 LLM 下的一轮对话应完整成功")
	require.Equal(t, "你好，世界", resp.GetMessage().GetContent(), "同步响应应携带拼接后的完整回复")
	require.Equal(t, "test-model", resp.GetMessage().GetModelName(), "assistant 消息应快照模型名")
	conversationID := resp.GetConversation().GetId()
	require.NotZero(t, conversationID, "首轮对话应自动新建会话")

	// 消息落库：user + assistant 各一条，token/耗时快照完整
	msgs, err := entClient.Client().AiMessage.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, msgs, 2, "一轮对话应落 user+assistant 两条消息")
	var userRow, assistantRow *ent.AiMessage
	for _, m := range msgs {
		switch *m.Role {
		case aimessage.RoleUSER:
			userRow = m
		case aimessage.RoleASSISTANT:
			assistantRow = m
		}
	}
	require.NotNil(t, userRow, "应存在 USER 消息行")
	require.Equal(t, "打个招呼", *userRow.Content)
	require.Equal(t, uint64(81), *userRow.UserID, "消息必须归属操作者：消息列表的越权兜底全靠它")
	require.NotNil(t, assistantRow, "应存在 ASSISTANT 消息行")
	require.Equal(t, "你好，世界", *assistantRow.Content)
	require.Equal(t, uint32(11), *assistantRow.PromptTokens, "assistant 消息应快照 usage 的输入 token")
	require.Equal(t, uint32(7), *assistantRow.CompletionTokens)
	require.NotNil(t, assistantRow.DurationMs, "assistant 消息应快照耗时")

	// 用量流水：一行，token 三件套与耗时齐全（配额记账的事实源）
	usageRows, err := entClient.Client().AiUsageLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, usageRows, 1, "一轮对话应记一行用量流水")
	require.Equal(t, uint32(18), *usageRows[0].TotalTokens, "用量流水的 total_tokens 应取自 usage 帧累计")
	require.NotNil(t, usageRows[0].ConversationID)
	require.Equal(t, conversationID, *usageRows[0].ConversationID)

	// 会话触碰：last_message_at 被尽力刷新
	convRow, err := entClient.Client().AiConversation.Query().
		Where(aiconversation.IDEQ(conversationID)).Only(ctx)
	require.NoError(t, err)
	require.NotNil(t, convRow.LastMessageAt, "对话结束后会话的 last_message_at 应被刷新")

	// SSE chunk 帧：两帧增量，seq 从 0 单调递增，streamId 为操作者 UID 字符串
	chunks := stub.named(sseevent.AIChatChunk)
	require.Len(t, chunks, 2, "两个文本增量应各推一帧 chunk")
	var gotSeq []uint32
	var gotDelta []string
	for _, e := range chunks {
		require.Equal(t, sse.StreamID("81"), e.streamId, "chunk 帧的 streamId 应为操作者 UID")
		var payload aiV1.ChatChunkEvent
		require.NoError(t, protojson.Unmarshal(e.data, &payload), "chunk 载荷应为合法 protojson（camelCase 键）")
		require.Equal(t, conversationID, payload.GetConversationId())
		gotSeq = append(gotSeq, payload.GetSeq())
		gotDelta = append(gotDelta, payload.GetDelta())
	}
	require.Equal(t, []uint32{0, 1}, gotSeq, "chunk 序号应从 0 单调递增")
	require.Equal(t, []string{"你好", "，世界"}, gotDelta, "chunk 增量应逐帧对应文本片段")

	// 第二轮（复用会话）：上下文必须按序组装 system+历史+本轮，role 全小写
	resp2, err := svc.Chat(opCtx, &aiV1.ChatRequest{Content: trans.Ptr("再来一轮"), ConversationId: &conversationID})
	require.NoError(t, err, "复用会话的第二轮对话应成功")
	require.Equal(t, conversationID, resp2.GetConversation().GetId(), "第二轮应复用传入会话")

	all := bodies()
	require.Len(t, all, 2, "两轮对话应各发起一次 LLM 请求")
	msgs2 := aiChatBodyMessages(t, all[1])
	var roles []string
	for _, m := range msgs2 {
		role, _ := m["role"].(string)
		roles = append(roles, role)
	}
	require.Equal(t, []string{"system", "user", "assistant", "user"}, roles,
		"第二轮上下文应为 system+首轮 user+首轮 assistant+本轮 user，role 全小写（真实厂商拒绝大写）")
	require.Contains(t, msgs2[0]["content"], "你是测试助手", "system 消息应注入提供商 system prompt")
	require.Equal(t, "打个招呼", msgs2[1]["content"], "首轮 user 消息应进历史")
	require.Equal(t, "你好，世界", msgs2[2]["content"], "首轮 assistant 回复应进历史")
	require.Equal(t, "再来一轮", msgs2[3]["content"], "本轮 user 消息应排最后")
	require.Contains(t, all[1], "tools", "非最后一轮请求应携带内置工具清单")
}

// TestAiChatService_PublishEvents 验证 SSE 推送缝：chunk/tool 帧的事件名与载荷、
// 以及丢帧容忍——TryPublish=false（流不存在/缓冲满）不得影响主链路。
func TestAiChatService_PublishEvents(t *testing.T) {
	stub := &aiChatPublisherStub{}
	svc := &AiChatService{log: bLogger.NewHelper(bLogger.NopLogger()), chatPublisher: stub}
	ctx := context.Background()

	svc.publishChunk(ctx, "81", 7, 0, "第一帧")
	svc.publishChunk(ctx, "81", 7, 1, "第二帧")
	svc.publishToolEvent(ctx, "81", 7, "get_current_time", "{}", "2026-01-01 00:00:00")

	chunks := stub.named(sseevent.AIChatChunk)
	require.Len(t, chunks, 2)

	var first aiV1.ChatChunkEvent
	require.NoError(t, protojson.Unmarshal(chunks[0].data, &first))
	require.Equal(t, uint64(7), first.GetConversationId())
	require.Equal(t, uint32(0), first.GetSeq())
	require.Equal(t, "第一帧", first.GetDelta())

	var second aiV1.ChatChunkEvent
	require.NoError(t, protojson.Unmarshal(chunks[1].data, &second))
	require.Equal(t, uint32(1), second.GetSeq(), "chunk 序号应逐帧递增")
	require.Equal(t, "第二帧", second.GetDelta())

	tools := stub.named(sseevent.AIChatTool)
	require.Len(t, tools, 1, "工具执行完成应推一帧 tool 事件")
	var tool aiV1.ChatToolEvent
	require.NoError(t, protojson.Unmarshal(tools[0].data, &tool))
	require.Equal(t, uint64(7), tool.GetConversationId())
	require.Equal(t, "get_current_time", tool.GetName())
	require.Equal(t, "{}", tool.GetArguments())
	require.Equal(t, "2026-01-01 00:00:00", tool.GetResult())

	// 丢帧容忍：publisher 一律拒绝时推送静默降级，不得 panic
	svc.chatPublisher = &aiChatPublisherStub{tryFail: true}
	svc.publishChunk(ctx, "81", 7, 2, "没人听")
	svc.publishToolEvent(ctx, "81", 7, "get_current_time", "{}", "x")
}

// TestAiChatService_NewOpenAIClientForProvider 验证客户端构造守卫：
// 云端缺密钥拒绝、密文解密通道（测试内全局加密器未初始化=明文直通）、
// 本地类型免密钥、未知模型类型拒绝。
func TestAiChatService_NewOpenAIClientForProvider(t *testing.T) {
	ctx := context.Background()

	// 云端：api_key 未配置 → 明确拒绝（空 key 客户端调用必 401，不如早失败）
	_, err := newOpenAIClientForProvider(ctx, &ent.AiProvider{ModelType: trans.Ptr(aiprovider.ModelTypeCLOUD)})
	require.Error(t, err, "云端提供商缺 api_key 应拒绝")
	require.Contains(t, err.Error(), "ai provider api key is not configured")

	_, err = newOpenAIClientForProvider(ctx, &ent.AiProvider{
		ModelType: trans.Ptr(aiprovider.ModelTypeCLOUD), APIKey: trans.Ptr(""),
	})
	require.Error(t, err, "云端提供商空串 api_key 应视为未配置")

	// 云端 + 密钥（全局加密器未初始化时 DecryptIfNeeded 明文直通）：构造成功
	client, err := newOpenAIClientForProvider(ctx, &ent.AiProvider{
		ModelType: trans.Ptr(aiprovider.ModelTypeCLOUD),
		APIKey:    trans.Ptr("sk-plain"),
		BaseURL:   trans.Ptr("http://127.0.0.1:9/v1"),
	})
	require.NoError(t, err, "云端提供商密钥齐备应构造成功")
	require.NotNil(t, client)

	// 本地（Ollama）：免密钥
	client, err = newOpenAIClientForProvider(ctx, &ent.AiProvider{
		ModelType: trans.Ptr(aiprovider.ModelTypeLOCAL),
		LocalHost: trans.Ptr("127.0.0.1"),
		LocalPort: trans.Ptr(11434),
	})
	require.NoError(t, err, "本地提供商应免密钥构造成功")
	require.NotNil(t, client)

	// 模型类型缺失（UNSPECIFIED）→ 工厂拒绝
	_, err = newOpenAIClientForProvider(ctx, &ent.AiProvider{})
	require.Error(t, err, "未知模型类型应拒绝")
	require.Contains(t, err.Error(), "create ai client failed")
}

// TestAiChatService_PureHelpers 钉住 chat 链路的纯函数语义：
// 大写枚举→OpenAI 小写 role 的映射（含兜底）、ent 模型类型→proto 数值对齐、
// 标题按字符截断（中文不受字节截断坑害）、ptr 兜底系列。
func TestAiChatService_PureHelpers(t *testing.T) {
	// aiRoleToOpenAI：ent 大写 → OpenAI 小写；未知值兜底为 user
	require.Equal(t, openai.ChatMessageRoleUser, aiRoleToOpenAI(aimessage.RoleUSER))
	require.Equal(t, openai.ChatMessageRoleAssistant, aiRoleToOpenAI(aimessage.RoleASSISTANT))
	require.Equal(t, openai.ChatMessageRoleSystem, aiRoleToOpenAI(aimessage.RoleSYSTEM))
	require.Equal(t, openai.ChatMessageRoleUser, aiRoleToOpenAI(aimessage.Role("weird")),
		"未知角色应兜底为 user，不能把大写原样发给厂商")

	// modelTypeToProtoInt：值域对齐（nil→UNSPECIFIED，LOCAL/CLOUD 逐字映射）
	require.EqualValues(t, aiV1.AiProvider_MODEL_TYPE_UNSPECIFIED, modelTypeToProtoInt(nil))
	require.EqualValues(t, aiV1.AiProvider_LOCAL, modelTypeToProtoInt(trans.Ptr(aiprovider.ModelTypeLOCAL)))
	require.EqualValues(t, aiV1.AiProvider_CLOUD, modelTypeToProtoInt(trans.Ptr(aiprovider.ModelTypeCLOUD)))
	require.EqualValues(t, aiV1.AiProvider_MODEL_TYPE_UNSPECIFIED, modelTypeToProtoInt(trans.Ptr(aiprovider.ModelType("GHOST"))),
		"未知模型类型应落 UNSPECIFIED 而非脏值")

	// truncateRunes：按字符截断，中文不会拦腰截出半个字符
	require.Equal(t, "你好世", truncateRunes("你好世界", 3))
	require.Equal(t, "你好世界", truncateRunes("你好世界", 4), "恰好等长时原样返回")
	require.Equal(t, "你好世界", truncateRunes("你好世界", 100))
	require.Equal(t, "", truncateRunes("", 5))

	// ptr 兜底系列
	require.Equal(t, "", ptrStr(nil))
	require.Equal(t, "v", ptrStr(trans.Ptr("v")))
	require.Equal(t, "d", ptrStrOr(nil, "d"))
	require.Equal(t, "d", ptrStrOr(trans.Ptr(""), "d"), "空串应触发默认值（模型名缺省语义）")
	require.Equal(t, "v", ptrStrOr(trans.Ptr("v"), "d"))
	require.Equal(t, 5, ptrIntOr(nil, 5))
	require.Equal(t, 3, ptrIntOr(trans.Ptr(3), 5))
}
