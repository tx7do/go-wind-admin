package service

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sashabaranov/go-openai"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	gsEngine "github.com/tx7do/go-scripts"
	"github.com/tx7do/go-scripts/hostmodule"
	"github.com/tx7do/go-utils/trans"

	"github.com/tx7do/go-crud/viewer"
	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	scriptV1 "go-wind-admin/api/gen/go/script/service/v1"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/pkg/oss"
	"go-wind-admin/pkg/scripting"
	"go-wind-admin/pkg/scripting/api"
	"go-wind-admin/pkg/task"
)

// ScriptRuntime 是脚本引擎的平台运行时：按语言持有一组编排器实例，
// 负责依赖注入（Redis 等）、数据库脚本的启动加载与变更重同步。
//
// 语言分发模型：内核编排器（scripting.Engine）是单语言单 VM 的，
// 每种语言一个 Engine 实例；脚本记录携带 language 字段，
// ScriptRuntime 按其分发到对应 Engine。go-scripts 注册表里有适配器的语言
// （当前 lua / javascript）都会被实例化。
//
// 多实例部署注意：当前重同步由本进程的管理接口触发（事件驱动），
// 其他实例需等下一次 Resync 才能感知脚本变更；跨实例通知（Redis pub/sub）
// 是后续增强项。
type ScriptRuntime struct {
	mu      sync.RWMutex
	engines map[gsEngine.Type]*scripting.Engine
	cfg     *scripting.Config
	repo    *data.ScriptRepo
	log     *bLogger.Helper
	logger  bLogger.Logger // 原始 logger，供一次性引擎构造使用

	// 任务桥：taskRegistrar 由 asynq 侧注入（asynq 的 mux 支持启动后注册），
	// taskHandlerOwner 记录处理器名 → 注册它的引擎（决定执行走哪条引擎锁）。
	taskRegistrar    func(taskType string, fn func(taskType string, data *task.ScriptTaskData) error) error
	taskHandlerOwner map[string]gsEngine.Type

	// 工具市场：toolHandlerOwner 记录工具名 → 注册它的引擎（AI 工具循环消费）。
	toolHandlerOwner map[string]gsEngine.Type

	// scriptLog 脚本执行日志（可选：nil 时不落审计）
	scriptLog *data.ScriptLogRepo

	// ai 脚本模块依赖（可选：nil 时引擎不注册 ai 模块）
	aiProviderRepo *data.AiProviderRepo
	aiUsageRepo    *data.AiUsageLogRepo

	// ossClient 保存引用供 TestRun 沙箱重放依赖（与 redisClient 同理）
	ossClient *oss.MinIOClient

	// redisClient 用于跨实例 Resync 通知（可选：nil 时不通知）
	redisClient *redis.Client
	notifyStop  chan struct{}
	notifyOnce  sync.Once
}

// ScriptResyncNotifyChannel 脚本 Resync 的跨实例通知频道。
const ScriptResyncNotifyChannel = "gowind:script:resync"

// logExecution 落一条执行日志（nil logger 时 no-op）。
func (r *ScriptRuntime) logExecution(trigger, hookPoint string, scriptName, language string, version uint32, started time.Time, err error) {
	if r.scriptLog == nil {
		return
	}
	rec := data.ScriptLogRecord{
		ScriptName: scriptName,
		Language:   language,
		Trigger:    trigger,
		HookPoint:  hookPoint,
		Version:    version,
		Success:    err == nil,
		DurationMS: time.Since(started).Milliseconds(),
	}
	if err != nil {
		rec.Error = err.Error()
	}
	r.scriptLog.Record(context.Background(), rec)
}

// 脚本任务的 sys_tasks 约定：type=PERIODIC，type_name=task.ScriptTaskDispatchType
// （"script_task"，启动期注册的固定分发订阅），task_payload 携带 handler 与 params。

// NewScriptRuntime 创建脚本运行时并为每种已注册语言实例化编排器。
// ScriptDir 固定为空：平台脚本一律以数据库为事实源，不走文件目录。
func NewScriptRuntime(ctx *bootstrap.Context, repo *data.ScriptRepo, redisClient *redis.Client, ossClient *oss.MinIOClient, scriptLog *data.ScriptLogRepo, aiProviderRepo *data.AiProviderRepo, aiUsageRepo *data.AiUsageLogRepo) *ScriptRuntime {
	r := &ScriptRuntime{
		engines:          make(map[gsEngine.Type]*scripting.Engine),
		cfg:              scripting.DefaultConfig(),
		repo:             repo,
		log:              ctx.NewLoggerHelper("script/runtime"),
		logger:           ctx.GetLogger(),
		taskHandlerOwner: make(map[string]gsEngine.Type),
		toolHandlerOwner: make(map[string]gsEngine.Type),
		scriptLog:        scriptLog,
		redisClient:      redisClient,
		aiProviderRepo:   aiProviderRepo,
		aiUsageRepo:      aiUsageRepo,
		ossClient:        ossClient,
	}
	// http 出站护栏：域名白名单走环境变量 SCRIPT_HTTP_ALLOWED_DOMAINS
	// （逗号分隔，支持 *.example.com 通配一级子域；未设置 = 全部拒绝，fail-closed）。
	r.cfg.HTTPOptions = hostmodule.HTTPAllowlistFromEnv()
	// P2 修复：库版 HTTPAllowlistFromEnv/NormalizeHTTPOptions 都没补默认值，
	// DenyLoopback 注释声称"默认 true"但 bool 零值是 false——环回硬禁
	//（localhost/127.0.0.1/0.0.0.0/::1/169.254.169.254）在真实装配下从不生效。
	// 这里显式开启；确实需要打本机的联调场景在白名单里显式写明。
	r.cfg.HTTPOptions.DenyLoopback = true

	for _, t := range scripting.SupportedTypes() {
		cfg := *r.cfg
		cfg.EngineType = t
		cfg.ScriptDir = ""
		eng := scripting.NewEngine(&cfg, ctx.GetLogger())
		eng.SetRedis(redisClient)
		eng.SetOSS(ossClient)
		eng.SetAICompleter(r)
		r.engines[t] = eng
		r.log.Infof(context.Background(), "script engine initialized (type: %s)", t)
	}

	return r
}

// EngineFor 返回指定语言（不区分大小写，如 "lua" / "javascript"）的编排器。
func (r *ScriptRuntime) EngineFor(language string) *scripting.Engine {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.engines[gsEngine.Type(strings.ToLower(language))]
}

// Languages 返回支持的语言列表（小写字符串）。
func (r *ScriptRuntime) Languages() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.engines))
	for _, eng := range r.engines {
		out = append(out, string(eng.ScriptEngine().GetType()))
	}
	return out
}

// HookPoints 聚合全部语言引擎的钩子点（同名钩子合并计数）。
func (r *ScriptRuntime) HookPoints() []scripting.HookPointInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	merged := make(map[string]*scripting.HookPointInfo)
	for _, eng := range r.engines {
		for _, hp := range eng.HookPoints() {
			existing, ok := merged[hp.Name]
			if !ok {
				cp := hp
				merged[hp.Name] = &cp
				continue
			}
			existing.ScriptCount += hp.ScriptCount
			existing.CallbackCount += hp.CallbackCount
			if existing.Description == "" {
				existing.Description = hp.Description
			}
		}
	}

	out := make([]scripting.HookPointInfo, 0, len(merged))
	for _, hp := range merged {
		out = append(out, *hp)
	}
	return out
}

// ScriptTaskHandlers 返回当前注册的全部脚本任务处理器（供管理面/观测展示）。
func (r *ScriptRuntime) ScriptTaskHandlers() []api.LuaTaskHandler {
	handlers := api.GetRegisteredHandlers()
	out := make([]api.LuaTaskHandler, 0, len(handlers))
	for _, h := range handlers {
		out = append(out, *h)
	}
	return out
}

// ScriptTools 返回当前注册的全部脚本 LLM 工具（供 AI 工具循环的 defs 枚举）。
func (r *ScriptRuntime) ScriptTools() []api.LuaToolHandler {
	handlers := api.GetRegisteredTools()
	out := make([]api.LuaToolHandler, 0, len(handlers))
	for _, h := range handlers {
		out = append(out, *h)
	}
	return out
}

// HasScriptTool 报告工具名是否为脚本工具（AI 工具循环 exec 路由用）。
func (r *ScriptRuntime) HasScriptTool(name string) bool {
	_, ok := api.GetTool(name)
	return ok
}

// ExecuteScriptTool 在工具所属引擎的锁保护下执行脚本工具，返回工具产出
// （AI 工具循环 execAiTool 的转发入口）。
func (r *ScriptRuntime) ExecuteScriptTool(ctx context.Context, name string, args map[string]any) (any, error) {
	r.mu.RLock()
	owner, ok := r.toolHandlerOwner[name]
	r.mu.RUnlock()
	if !ok {
		return nil, scriptV1.ErrorNotFound("script tool not found: %s", name)
	}

	eng := r.EngineFor(string(owner))
	if eng == nil {
		return nil, scriptV1.ErrorInternalServerError("engine %s unavailable", owner)
	}

	r.log.Infof(ctx, "execute script tool %q (engine: %s)", name, owner)

	started := time.Now()
	result, err := eng.ExecuteToolHandler(ctx, name, args)
	r.logExecution("tool", "", "tool:"+name, string(owner), 0, started, err)
	return result, err
}

// AttachScriptTaskRegistrar 注入 asynq 订阅注册回调（asynq 的 mux 拒绝 Start 后注册，
// 因此只在启动期调用一次；分发模型见 RunScriptTask）。
func (r *ScriptRuntime) AttachScriptTaskRegistrar(
	registrar func(taskType string, fn func(taskType string, data *task.ScriptTaskData) error) error,
) error {
	r.mu.Lock()
	r.taskRegistrar = registrar
	r.mu.Unlock()
	return nil
}

// NotifyResync 向其他实例广播 Resync 通知（Redis pub/sub；redisClient 为 nil 时 no-op）。
// 本实例的 Resync 由调用方直接执行，不经此通知。
func (r *ScriptRuntime) NotifyResync(ctx context.Context) {
	if r.redisClient == nil {
		return
	}
	if err := r.redisClient.Publish(ctx, ScriptResyncNotifyChannel, "resync").Err(); err != nil {
		r.log.Errorf(ctx, "publish script resync notify failed: %v", err)
	}
}

// StartResyncListener 订阅跨实例 Resync 通知并在本实例执行 Resync
// （redisClient 为 nil 时 no-op）。阻塞 goroutine 运行至 Stop 被调用，
// 由 wiring 注册 cleanup。
func (r *ScriptRuntime) StartResyncListener(ctx context.Context) {
	if r.redisClient == nil {
		return
	}

	r.notifyStop = make(chan struct{})
	go func() {
		for {
			select {
			case <-r.notifyStop:
				return
			default:
			}

			sub := r.redisClient.Subscribe(ctx, ScriptResyncNotifyChannel)
			for {
				select {
				case <-r.notifyStop:
					_ = sub.Close()
					return
				case msg, ok := <-sub.Channel():
					if !ok {
						// 连接断开：退避后重建订阅
						select {
						case <-r.notifyStop:
						case <-time.After(3 * time.Second):
						}
						break
					}
					if msg == nil {
						continue
					}
					r.log.Infof(ctx, "script resync notify received from channel, resyncing")
					if err := r.Resync(ctx); err != nil {
						r.log.Errorf(ctx, "resync after notify failed: %v", err)
					}
				}
			}
		}
	}()
}

// Stop 停止后台 goroutine（通知监听等）。幂等。
func (r *ScriptRuntime) StopResyncListener() {
	r.notifyOnce.Do(func() {
		if r.notifyStop != nil {
			close(r.notifyStop)
		}
	})
}

// RegisterScriptTaskSubscriber 注册固定分发类型的订阅（启动期一次）。
// 运行期脚本变更只影响处理器注册表，不需要新的 asynq 订阅。
func (r *ScriptRuntime) RegisterScriptTaskSubscriber() {
	r.mu.RLock()
	registrar := r.taskRegistrar
	r.mu.RUnlock()
	if registrar == nil {
		return
	}

	if err := registrar(task.ScriptTaskDispatchType, func(_ string, data *task.ScriptTaskData) error {
		return r.RunScriptTaskHandler(context.Background(), data.Handler, data.Params)
	}); err != nil {
		r.log.Errorf(context.Background(), "register script task subscriber %q failed: %v", task.ScriptTaskDispatchType, err)
		return
	}
	r.log.Infof(context.Background(), "script task subscriber registered (type: %s)", task.ScriptTaskDispatchType)
}

// RunScriptTaskHandler 执行指定脚本任务处理器（asynq worker 的最终落点）。
func (r *ScriptRuntime) RunScriptTaskHandler(ctx context.Context, name string, params map[string]any) error {
	r.mu.RLock()
	owner, ok := r.taskHandlerOwner[name]
	r.mu.RUnlock()
	if !ok {
		return scriptV1.ErrorNotFound("script task handler not found: %s", name)
	}

	eng := r.EngineFor(string(owner))
	if eng == nil {
		return scriptV1.ErrorInternalServerError("engine %s unavailable", owner)
	}

	r.log.Infof(ctx, "run script task handler %q (engine: %s)", name, owner)

	started := time.Now()
	err := eng.ExecuteTaskHandler(ctx, name, params)
	r.logExecution("task", task.ScriptTaskDispatchType, "task:"+name, string(owner), 0, started, err)
	return err
}

// Resync 全量重同步：清空各语言引擎的脚本注册，重新加载数据库中全部已启用脚本。
// 挂载了 hook_point 的脚本走注册表路径（钩子触发时执行）；
// 未挂载、在源码顶层自行 hook.register 的脚本走自注册路径（加载时执行一次）。
// 任一脚本失败不中断整体同步（记日志继续），保证一个坏脚本不拖垮其他脚本。
func (r *ScriptRuntime) Resync(ctx context.Context) error {
	r.mu.Lock()
	for _, eng := range r.engines {
		eng.ResetScriptRegistrations()
	}
	r.mu.Unlock()

	// 任务处理器代际：本轮重同步内未被重新注册的处理器视为已禁用/删除，随后清理
	generation := api.NextTaskGeneration()

	rows, err := r.repo.ListEnabledScripts(ctx)
	if err != nil {
		r.log.Errorf(ctx, "script resync: list enabled scripts failed: %v", err)
		return err
	}

	loaded, failed := 0, 0
	for _, row := range rows {
		lang := strings.ToLower(row.GetLanguage().String())
		eng := r.EngineFor(lang)
		if eng == nil {
			r.log.Errorf(ctx, "script resync: script %q has unsupported language %q, skipped", row.GetName(), lang)
			failed++
			continue
		}

		record := &scripting.Script{
			ID:          row.GetId(),
			Name:        row.GetName(),
			Hook:        row.GetHookPoint(),
			Language:    lang,
			Source:      row.GetSource(),
			Enabled:     row.GetIsEnabled(),
			Priority:    int(row.GetPriority()),
			Description: row.GetDescription(),
			Version:     int(row.GetVersion()),
			Critical:    row.GetCritical(),
		}

		// 挂载脚本进注册表；未挂载的脚本执行一次以触发 hook.register / task.register_handler 自注册
		if record.Hook != "" {
			if err := eng.AddScript(record.Hook, record); err != nil {
				r.log.Errorf(ctx, "script resync: add script %q failed: %v", record.Name, err)
				failed++
				continue
			}
		} else if err := eng.LoadScriptString(ctx, record.Name, record.Source); err != nil {
			r.log.Errorf(ctx, "script resync: self-register script %q failed: %v", record.Name, err)
			failed++
			continue
		}

		// 归属本轮新注册的任务处理器到对应引擎（决定执行时走哪条引擎锁）
		for name, h := range api.GetRegisteredHandlers() {
			if h.Generation >= generation {
				r.taskHandlerOwner[name] = gsEngine.Type(lang)
			}
		}
		// 归属本轮新注册的脚本工具到对应引擎（AI 工具循环消费）
		for name, h := range api.GetRegisteredTools() {
			if h.Generation >= generation {
				r.toolHandlerOwner[name] = gsEngine.Type(lang)
			}
		}
		loaded++
	}

	pruned := api.PruneStaleTaskHandlers(generation)
	prunedTools := api.PruneStaleToolHandlers(generation)

	r.log.Infof(ctx, "script resync done: %d loaded, %d failed, %d total enabled, %d task handlers pruned, %d script tools pruned",
		loaded, failed, len(rows), pruned, prunedTools)
	return nil
}

// TestRun 在一次性隔离引擎中试运行脚本：不污染常驻引擎的 VM 与 hook 注册。
// input 为执行上下文初始数据；返回执行后的完整上下文数据。
func (r *ScriptRuntime) TestRun(ctx context.Context, language, name, source string, input map[string]any) (map[string]any, error) {
	eng := r.EngineFor(language)
	if eng == nil {
		return nil, scriptV1.ErrorBadRequest("unsupported script language: %s", language)
	}

	// 一次性引擎：与常驻引擎同配置，独立 VM，用完即毁。
	// 可选业务依赖（redis/oss/ai）必须在此重放，否则沙箱里对应模块缺失
	// （此前 cache/oss 在试运行中同样缺席，属既有缺口，一并补上）。
	cfg := *r.cfg
	cfg.EngineType = eng.ScriptEngine().GetType()
	cfg.ScriptDir = ""
	sandbox := scripting.NewEngine(&cfg, r.logger)
	sandbox.SetRedis(r.redisClient)
	sandbox.SetOSS(r.ossClient)
	sandbox.SetAICompleter(r)
	defer sandbox.Close()

	if name == "" {
		name = "test_run_draft"
	}

	execCtx := scripting.NewContext("test_run")
	for k, v := range input {
		execCtx.Set(k, v)
	}

	script := &scripting.Script{
		Name:     name,
		Language: strings.ToLower(language),
		Source:   source,
		Enabled:  true,
	}
	started := time.Now()
	err := sandbox.Execute(ctx, script, execCtx)
	r.logExecution("test_run", "", name, strings.ToLower(language), 0, started, err)
	if err != nil {
		return execCtx.Data, err
	}
	return execCtx.Data, nil
}

// Close 关闭全部语言引擎。
func (r *ScriptRuntime) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for t, eng := range r.engines {
		if err := eng.Close(); err != nil {
			r.log.Errorf(context.Background(), "close script engine %s failed: %v", t, err)
		}
	}
	r.engines = make(map[gsEngine.Type]*scripting.Engine)
}

// ChatForScript 实现 scripting/api.AICompleter：脚本 ai 模块的对话入口。
// 复用 chat 主链路的 provider 解析 / 密钥解密 / 客户端工厂；每次调用记一条
// 用量流水（tenant/user/conversation 均为 0 = 系统脚本发起），记账失败不阻断调用。
func (r *ScriptRuntime) ChatForScript(ctx context.Context, providerId uint32, systemPrompt, content string) (string, error) {
	if r.aiProviderRepo == nil {
		return "", adminV1.ErrorInternalServerError("ai module is not available")
	}

	// 解析提供商：0 = 默认启用项
	var provider *ent.AiProvider
	var err error
	if providerId > 0 {
		provider, err = r.aiProviderRepo.GetEntityByID(ctx, providerId)
	} else {
		provider, err = r.aiProviderRepo.GetEnabledDefault(ctx)
	}
	if err != nil {
		return "", err
	}
	if provider.IsEnabled == nil || !*provider.IsEnabled {
		return "", adminV1.ErrorBadRequest("ai provider is disabled")
	}

	client, err := newOpenAIClientForProvider(ctx, provider)
	if err != nil {
		return "", err
	}

	messages := make([]openai.ChatCompletionMessage, 0, 2)
	if systemPrompt != "" {
		messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleSystem, Content: systemPrompt})
	}
	messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: content})

	started := time.Now()
	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    ptrStrOr(provider.ModelName, "gpt-4o-mini"),
		Messages: messages,
	})
	if err != nil {
		r.log.Errorf(ctx, "script ai chat failed: provider=%d: %v", provider.ID, err)
		return "", err
	}
	durationMs := uint32(time.Since(started).Milliseconds())

	reply := ""
	if len(resp.Choices) > 0 {
		reply = resp.Choices[0].Message.Content
	}

	// 用量记账（尽力而为：脚本调用以返回值为准，流水缺失只影响配额统计）。
	// 走系统查看器：脚本调用无请求上下文，裸 context 过不了租户隔离 mixin 的写检查。
	if r.aiUsageRepo != nil {
		ctx = viewer.WithSystemContext(ctx)
		modelName := ptrStrOr(provider.ModelName, "")
		promptTokens := uint32(resp.Usage.PromptTokens)
		completionTokens := uint32(resp.Usage.CompletionTokens)
		totalTokens := uint32(resp.Usage.TotalTokens)
		if uerr := r.aiUsageRepo.Create(ctx, &aiV1.AiUsageLog{
			ProviderId:       &provider.ID,
			UserId:           trans.Ptr(uint32(0)),
			TenantId:         trans.Ptr(uint32(0)),
			ModelName:        &modelName,
			PromptTokens:     &promptTokens,
			CompletionTokens: &completionTokens,
			TotalTokens:      &totalTokens,
			DurationMs:       &durationMs,
		}); uerr != nil {
			r.log.Errorf(ctx, "script ai usage log failed: %v", uerr)
		}
	}

	return reply, nil
}
