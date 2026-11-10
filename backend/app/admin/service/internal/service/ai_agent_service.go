package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"

	"github.com/tx7do/kratos-transport/transport/sse"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/aiagentrun"
	"strconv"

	"github.com/tx7do/go-utils/id"
	"go-wind-admin/pkg/middleware/auth"
	"go-wind-admin/pkg/sseevent"
)

// AiAgentService AI Agent 编排（可视化画布）：
// 应用 CRUD + 同步试运行（图引擎）+ 运行台账。
// 模型调用/知识检索/配额用量全部复用 AI 模块既有链路（见 docs/ai_agent_orchestration.md §1）。
type AiAgentRunPublisher interface {
	Publish(ctx context.Context, streamId sse.StreamID, event *sse.Event)
	TryPublish(ctx context.Context, streamId sse.StreamID, event *sse.Event) bool
}

// noopAiAgentRunPublisher 未接线（测试/旧装配）时的空实现：流式观感静默降级。
type noopAiAgentRunPublisher struct{}

func (noopAiAgentRunPublisher) Publish(_ context.Context, _ sse.StreamID, _ *sse.Event) {}
func (noopAiAgentRunPublisher) TryPublish(_ context.Context, _ sse.StreamID, _ *sse.Event) bool {
	return false
}

type AiAgentService struct {
	log           *bLogger.Helper
	appRepo       *data.AiAgentAppRepo
	runRepo       *data.AiAgentRunRepo
	providerRepo  *data.AiProviderRepo
	knowledgeRepo *data.AiKnowledgeRepo
	usageLogRepo  *data.AiUsageLogRepo
	revisionRepo  *data.AiAgentRevisionRepo
	runPublisher  AiAgentRunPublisher

	// scriptRuntime script 节点的沙箱执行能力（wiring 装配；nil = script 节点不可用并报错）。
	scriptRuntime *ScriptRuntime

	// 定时触发任务桥（agent_run）的结果回传：站内信创建与投递内核。
	// internalMessages 经 setter 装配（wiring 内构造顺序晚于本服务）；未装配时 NotifyUserId 不通知。
	internalMessages *InternalMessageService
	internalMsgRepo  *data.InternalMessageRepo
}

// RegisterInternalMessageNotifier 注入站内信服务（定时触发结果回传用；幂等覆盖）。
func (s *AiAgentService) RegisterInternalMessageNotifier(im *InternalMessageService) {
	if im != nil {
		s.internalMessages = im
	}
}

// RegisterAiAgentRunPublisher 接线 SSE 发布者（sse_server 装配期调用；幂等覆盖）。
func (s *AiAgentService) RegisterAiAgentRunPublisher(p AiAgentRunPublisher) {
	if p != nil {
		s.runPublisher = p
	}
}

func NewAiAgentService(
	ctx *bootstrap.Context,
	appRepo *data.AiAgentAppRepo,
	runRepo *data.AiAgentRunRepo,
	providerRepo *data.AiProviderRepo,
	knowledgeRepo *data.AiKnowledgeRepo,
	usageLogRepo *data.AiUsageLogRepo,
	scriptRuntime *ScriptRuntime,
	revisionRepo *data.AiAgentRevisionRepo,
	internalMsgRepo *data.InternalMessageRepo,
) *AiAgentService {
	return &AiAgentService{
		log:            ctx.NewLoggerHelper("ai-agent/service/admin-service"),
		appRepo:        appRepo,
		runRepo:        runRepo,
		providerRepo:   providerRepo,
		knowledgeRepo:  knowledgeRepo,
		usageLogRepo:   usageLogRepo,
		scriptRuntime:  scriptRuntime,
		revisionRepo:   revisionRepo,
		internalMsgRepo: internalMsgRepo,
	}
}

// List 查询应用列表（登录即可用，数据由租户隔离 mixin 收口）。
func (s *AiAgentService) List(ctx context.Context, req *paginationV1.PagingRequest) (*aiV1.ListAiAgentAppsResponse, error) {
	return s.appRepo.List(ctx, req)
}

func (s *AiAgentService) Get(ctx context.Context, req *aiV1.GetAiAgentAppRequest) (*aiV1.AiAgentApp, error) {
	if req == nil || req.GetId() == 0 {
		return nil, adminV1.ErrorBadRequest("id is required")
	}
	return s.appRepo.Get(ctx, req.GetId())
}

func (s *AiAgentService) Create(ctx context.Context, req *aiV1.CreateAiAgentAppRequest) (*aiV1.AiAgentApp, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}
	if err := s.validateGraphOrBadRequest(req.Data.GetGraph()); err != nil {
		return nil, err
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	req.Data.CreatedBy = trans.Ptr(operator.UserId)
	req.Data.TenantId = trans.Ptr(operator.GetTenantId())

	id, err := s.appRepo.Create(ctx, req.Data, operator.UserId)
	if err != nil {
		return nil, err
	}
	return s.appRepo.Get(ctx, id)
}

func (s *AiAgentService) Update(ctx context.Context, req *aiV1.UpdateAiAgentAppRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}
	if err := s.validateGraphOrBadRequest(req.Data.GetGraph()); err != nil {
		return nil, err
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	req.Data.UpdatedBy = trans.Ptr(operator.UserId)

	// 版本快照（docs/ai_agent_orchestration.md「版本化」，BI 同款口径）：更新成功把
	// **更新前**的 graph 留一行（SAVE，仅 graph 实际变化时）。快照失败不阻断更新
	// （Errorf 留痕）——主功能不应被辅功能拖死，但丢快照必须可观测。
	current, getErr := s.appRepo.GetEntityByID(ctx, req.GetId())
	if getErr != nil {
		s.log.Warnf(ctx, "snapshot before update: load agent app %d failed: %s", req.GetId(), getErr.Error())
	}

	var mask []string
	if req.UpdateMask != nil {
		mask = req.UpdateMask.Paths
	}
	if err = s.appRepo.Update(ctx, req.GetId(), req.Data, mask, operator.UserId); err != nil {
		return nil, err
	}

	if current != nil && s.revisionRepo != nil && ptrStrOr(current.Graph, "") != req.Data.GetGraph() {
		snapshot := &aiV1.AiAgentRevision{
			AppId:     trans.Ptr(req.GetId()),
			Graph:     current.Graph,
			Source:    trans.Ptr("SAVE"),
			CreatedBy: trans.Ptr(operator.UserId),
		}
		if serr := s.revisionRepo.Create(ctx, snapshot); serr != nil {
			s.log.Errorf(ctx, "snapshot before update agent app %d failed: %s", req.GetId(), serr.Error())
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *AiAgentService) Delete(ctx context.Context, req *aiV1.DeleteAiAgentAppRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, adminV1.ErrorBadRequest("id is required")
	}
	if err := s.appRepo.Delete(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// validateGraphOrBadRequest 图非空时必须能解析且过静态校验：存一张画布都画不出的图，
// 试运行时才炸等于把配置错误藏到使用时刻。
func (s *AiAgentService) validateGraphOrBadRequest(raw string) error {
	if raw == "" {
		return nil // 允许先建壳后画图
	}
	g, err := parseAgentGraph(raw)
	if err != nil {
		return adminV1.ErrorBadRequest("%s", err.Error())
	}
	if err = validateAgentGraph(g); err != nil {
		return adminV1.ErrorBadRequest("%s", err.Error())
	}
	return nil
}

// Run 试运行：校验（不过则 400、不建行）→ 建 RUNNING 行 → 同步执行 → 回写终态。
func (s *AiAgentService) Run(ctx context.Context, req *aiV1.RunAiAgentAppRequest) (*aiV1.AiAgentRun, error) {
	if req == nil || req.GetId() == 0 {
		return nil, adminV1.ErrorBadRequest("id is required")
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	app, err := s.appRepo.GetEntityByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if !app.Enabled {
		return nil, adminV1.ErrorBadRequest("ai agent app is disabled")
	}

	g, err := parseAgentGraph(ptrStrOr(app.Graph, ""))
	if err != nil {
		return nil, adminV1.ErrorBadRequest("%s", err.Error())
	}
	if err = validateAgentGraph(g); err != nil {
		return nil, adminV1.ErrorBadRequest("%s", err.Error())
	}

	if err = checkAiTokenQuota(ctx, s.usageLogRepo, s.log, operator); err != nil {
		return nil, err
	}

	inputJSON, _ := json.Marshal(req.GetInput())
	runID, err := s.runRepo.CreateRunning(ctx, app.ID, string(inputJSON), operator.UserId, operator.GetTenantId())
	if err != nil {
		return nil, err
	}

	start := time.Now()
	seq := uint32(0)
	publishFrame := func(phase, nodeID, summary, status string) {
		data, err := protojson.Marshal(&aiV1.AgentRunEvent{
			RunId:   runID,
			AppId:   app.ID,
			Seq:     seq,
			Phase:   phase,
			NodeId:  nodeID,
			Summary: summary,
			Status:  status,
		})
		seq++
		if err != nil {
			s.log.Errorf(ctx, "marshal agent run event failed, skip push: %v", err)
			return
		}
		// 尽力而为：订阅端慢/断线即丢帧；Run 同步响应携带完整 trace，以响应为准。
		s.runPublisher.TryPublish(ctx, sse.StreamID(strconv.FormatInt(operator.UserId, 10)), &sse.Event{
			ID:    []byte(id.NewGUIDv4(false)),
			Data:  data,
			Event: []byte(sseevent.AgentRun),
		})
	}

	output, trace, usage, execErr := executeAgentGraph(ctx, g, req.GetInput(), &agentEngineDeps{
		searchKnowledge:  s.searchKnowledgeForAgent(operator),
		runLLM:           s.runLLMForAgent(operator),
		runScript:        s.runScriptForAgent,
		loadSubflowGraph: s.loadSubflowGraph,
		onStep:           publishFrame,
	})
	durationMs := uint32(time.Since(start).Milliseconds())

	traceJSON, _ := json.Marshal(trace)

	status := aiagentrun.StatusSucceeded
	errMsg := ""
	if execErr != nil {
		status = aiagentrun.StatusFailed
		errMsg = execErr.Error()
		publishFrame("error", "", errMsg, "")
		s.log.Errorf(ctx, "ai agent run [%d] app [%d] failed: %s", runID, app.ID, execErr.Error())
	} else {
		publishFrame("done", "", output, "")
	}

	if err = s.runRepo.SettleResult(ctx, runID, status, output, errMsg, string(traceJSON),
		data.AgentRunTokens{Prompt: usage.Prompt, Completion: usage.Completion, Total: usage.Total},
		durationMs); err != nil {
		return nil, err
	}

	s.log.Infof(ctx, "ai agent run [%d] app [%d] %s in %dms (tokens %d) by user [%d]",
		runID, app.ID, status.String(), durationMs, usage.Total, operator.UserId)

	return s.runRepo.Get(ctx, runID)
}

func (s *AiAgentService) ListRuns(ctx context.Context, req *paginationV1.PagingRequest) (*aiV1.ListAiAgentRunsResponse, error) {
	return s.runRepo.List(ctx, req)
}

// loadSubflowGraph subflow 节点的真实现：加载目标应用（enabled 校验）并解析其图。
// 同租户隔离（ent viewer 租户谓词）：跨租户引用查不到即报错，不越权。
func (s *AiAgentService) loadSubflowGraph(ctx context.Context, appID int64) (string, *agentGraph, error) {
	app, err := s.appRepo.GetEntityByID(ctx, appID)
	if err != nil {
		return "", nil, err
	}
	if !app.Enabled {
		return "", nil, fmt.Errorf("subflow target app [%d] is disabled", appID)
	}
	g, perr := parseAgentGraph(ptrStrOr(app.Graph, ""))
	if perr != nil {
		return "", nil, fmt.Errorf("subflow target app [%d] has invalid graph: %w", appID, perr)
	}
	if err = validateAgentGraph(g); err != nil {
		return "", nil, fmt.Errorf("subflow target app [%d] graph failed validation: %w", appID, err)
	}
	return app.Name, g, nil
}

func (s *AiAgentService) GetRun(ctx context.Context, req *aiV1.GetAiAgentRunRequest) (*aiV1.AiAgentRun, error) {
	if req == nil || req.GetId() == 0 {
		return nil, adminV1.ErrorBadRequest("id is required")
	}
	return s.runRepo.Get(ctx, req.GetId())
}

// ListRevisions 版本历史（新快照在前）。
func (s *AiAgentService) ListRevisions(ctx context.Context, req *aiV1.ListAiAgentRevisionsRequest) (*aiV1.ListAiAgentRevisionsResponse, error) {
	if req == nil || req.GetAppId() == 0 {
		return nil, adminV1.ErrorBadRequest("appId is required")
	}
	items, total, err := s.revisionRepo.List(ctx, req.GetAppId(), req.GetPage())
	if err != nil {
		return nil, err
	}
	return &aiV1.ListAiAgentRevisionsResponse{Items: items, Total: int64(total)}, nil
}

// RestoreRevision 恢复版本：快照 graph 写回主表。恢复动作本身先把**当前 graph**
// 快照（RESTORE 来源），恢复可逆（BI 同款口径）。
// 快照 graph 先过静态校验，非法 400 且不落任何痕迹——历史快照可能早于校验规则演进。
func (s *AiAgentService) RestoreRevision(ctx context.Context, req *aiV1.RestoreAiAgentRevisionRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, adminV1.ErrorBadRequest("id is required")
	}
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	revision, err := s.revisionRepo.Get(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if err = s.validateGraphOrBadRequest(revision.GetGraph()); err != nil {
		return nil, err
	}

	appID := revision.GetAppId()
	current, err := s.appRepo.GetEntityByID(ctx, appID)
	if err != nil {
		return nil, err
	}

	// 恢复前留底当前 graph（失败不阻断恢复，Errorf 留痕）
	if s.revisionRepo != nil {
		snapshot := &aiV1.AiAgentRevision{
			AppId:     trans.Ptr(appID),
			Graph:     current.Graph,
			Source:    trans.Ptr("RESTORE"),
			CreatedBy: trans.Ptr(operator.UserId),
		}
		if serr := s.revisionRepo.Create(ctx, snapshot); serr != nil {
			s.log.Errorf(ctx, "snapshot before restore agent app %d failed: %s", appID, serr.Error())
		}
	}

	// 写回走普通更新（mask 只点 graph；恢复是明确的人为决定，以快照为准整份覆盖 graph）
	if err = s.appRepo.Update(ctx, appID, &aiV1.AiAgentApp{Graph: revision.Graph},
		[]string{"graph"}, operator.UserId); err != nil {
		return nil, err
	}

	s.log.Infof(ctx, "ai agent app [%d] graph restored to revision [%d] by user [%d]", appID, req.GetId(), operator.UserId)
	return &emptypb.Empty{}, nil
}

// searchKnowledgeForAgent 知识检索节点的真实现：复用 chat 同一条 RAG 链路
// （embed + 余弦检索 + embed 用量记账），命中片段按序拼接为上下文。
func (s *AiAgentService) searchKnowledgeForAgent(operator *authenticationV1.UserTokenPayload) func(ctx context.Context, node *agentGraphNode, query string) (string, int, error) {
	return func(ctx context.Context, node *agentGraphNode, query string) (string, int, error) {
		baseID := agentDataUint32(node.Data, "baseId")
		if baseID == 0 {
			return "", 0, fmt.Errorf("knowledge node has no baseId configured")
		}
		topK := agentDataInt(node.Data, "topK", 4)
		hits, err := searchKnowledgeBase(ctx, s.knowledgeRepo, s.providerRepo, s.usageLogRepo, s.log, operator, baseID, query, topK)
		if err != nil {
			return "", 0, err
		}
		var sb strings.Builder
		for i, h := range hits {
			if i > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString(h.Content)
		}
		return sb.String(), len(hits), nil
	}
}

// runLLMForAgent LLM 节点的真实现：provider 解析（节点点名，留空取默认启用）→
// 客户端构造（密文解密就地）→ aiToolLoop 多轮工具调用 → 逐节点落用量流水
// （conversation_id=0 的非对话调用，与 chat 共用 AI_TOKENS 配额口径）。
func (s *AiAgentService) runLLMForAgent(operator *authenticationV1.UserTokenPayload) func(ctx context.Context, node *agentGraphNode, system, user string) (string, agentUsage, error) {
	return func(ctx context.Context, node *agentGraphNode, system, user string) (string, agentUsage, error) {
		var provider *ent.AiProvider
		var err error
		if providerID := agentDataUint32(node.Data, "providerId"); providerID > 0 {
			provider, err = s.providerRepo.GetEntityByID(ctx, providerID)
			if err != nil {
				return "", agentUsage{}, err
			}
			if provider.IsEnabled != nil && !*provider.IsEnabled {
				return "", agentUsage{}, fmt.Errorf("ai provider [%d] is disabled", providerID)
			}
		} else {
			provider, err = s.providerRepo.GetEnabledDefault(ctx)
			if err != nil {
				return "", agentUsage{}, err
			}
		}

		client, err := newOpenAIClientForProvider(ctx, provider)
		if err != nil {
			return "", agentUsage{}, err
		}

		model := agentDataString(node.Data, "model")
		if model == "" {
			model = ptrStrOr(provider.ModelName, "")
		}

		var temperature *float32
		if t, ok := agentDataFloat(node.Data, "temperature"); ok {
			t32 := float32(t)
			temperature = &t32
		}

		var messages []openai.ChatCompletionMessage
		if system != "" {
			messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleSystem, Content: system})
		}
		messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: user})

		// P3 加固：Agent 画布执行补 per-call deadline（上游无超时兜底时，
		// provider 端点挂起会让请求 goroutine 悬挂累积）。
		callCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		// 工具清单 = 内置 + 脚本工具（工具市场：脚本经 tool.register 注册的 LLM 工具）；
		// exec 按工具名路由：脚本工具走脚本引擎（hasScriptTool 优先），内置走 execAiTool。
		tools := aiBuiltinToolDefs()
		tools = append(tools, aiScriptToolDefs(s.scriptRuntime)...)
		exec := func(ctx context.Context, name, args string) (string, error) {
			if aiIsScriptTool(s.scriptRuntime, name) {
				return execAiScriptTool(s.scriptRuntime, ctx, name, args)
			}
			return execAiTool(ctx, name, args)
		}

		loop := &aiToolLoop{
			client:      client,
			model:       model,
			tools:       tools,
			exec:        exec,
			maxRounds:   aiToolMaxRounds,
			temperature: temperature,
		}

		start := time.Now()
		answer, usage, err := loop.run(callCtx, messages)
		durationMs := uint32(time.Since(start).Milliseconds())

		// 逐 LLM 节点记一条用量流水：agent 运行在用量台账里是"非对话调用"，
		// model_name 记节点实际用的模型，配额由 Run 入口的统一检查把关。
		if usage.TotalTokens > 0 {
			promptTokens := uint32(usage.PromptTokens)
			completionTokens := uint32(usage.CompletionTokens)
			totalTokens := uint32(usage.TotalTokens)
			if uerr := s.usageLogRepo.Create(ctx, &aiV1.AiUsageLog{
				ProviderId:       trans.Ptr(provider.ID),
				UserId:           &operator.UserId,
				TenantId:         trans.Ptr(operator.GetTenantId()),
				ModelName:        &model,
				PromptTokens:     &promptTokens,
				CompletionTokens: &completionTokens,
				TotalTokens:      &totalTokens,
				DurationMs:       &durationMs,
			}); uerr != nil {
				// 记账失败不回滚答案：运行本身已成功，台账缺一行可补，答案丢了不可补。
				s.log.Errorf(ctx, "ai agent run: write usage log failed (model %s): %s", model, uerr.Error())
			}
		}

		if err != nil {
			return "", agentUsage{}, err
		}
		return answer, agentUsage{
			Prompt:     int64(usage.PromptTokens),
			Completion: int64(usage.CompletionTokens),
			Total:      int64(usage.TotalTokens),
		}, nil
	}
}
