package scripting

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	gsEngine "github.com/tx7do/go-scripts"
	gsSource "github.com/tx7do/go-scripts/source"

	// 空导入各语言引擎包：触发 init() 将引擎工厂注册到全局注册表。
	// 各引擎的适配器在对应的 runtime_*.go 中引用。
	_ "github.com/tx7do/go-scripts/javascript"
	_ "github.com/tx7do/go-scripts/lua"

	"go-wind-admin/pkg/eventbus"
	"go-wind-admin/pkg/oss"
	"go-wind-admin/pkg/scripting/api"
	"go-wind-admin/pkg/scripting/hook"
)

// Engine 是 Hook 编排器，语言无关。
//
// 架构（基于 go-scripts + 语言适配器）：
//   - Engine（本结构）：Hook 注册、脚本按优先级/链式执行、回调管理、执行上下文
//   - gsEngine.Engine（脚本引擎）：VM 生命周期、脚本编译/执行、模块注册、热更新
//   - RuntimeBinder（语言适配器）：把业务 API / hook.register / 执行上下文注入到特定语言 VM
//
// 切换语言只需改 Config.EngineType（LuaType / JavaScriptType），
// 编排器通过 RuntimeBinder 接口面向抽象，核心逻辑零语言耦合。
type Engine struct {
	config   *Config
	logger   *bLogger.Helper
	registry *hook.Registry

	// 脚本引擎（语言无关，实现 go-scripts.Engine 接口）
	scriptEngine gsEngine.Engine

	// 语言适配器（注入业务 API 到特定语言 VM）
	binder RuntimeBinder

	// 业务依赖（可选，经 binder 注入到 VM）
	rdb             *redis.Client
	eventbusManager *eventbus.Manager
	ossClient       *oss.MinIOClient

	// Hook 回调（hook 名称 -> 多个回调），由脚本 hook.register 注册
	callbacks   map[string][]ScriptCallback
	callbacksMu sync.RWMutex

	// 执行上下文持有器（执行期间 Set，执行后 Reset，供脚本 __get_ctx 等访问）
	execCtx execCtxHolder

	// execMu 串行化所有脚本执行路径（Execute / ExecuteHook / LoadScriptString）。
	//
	// 必须串行的原因：底层 go-scripts/lua 引擎是单 VM——
	//  1. ExecuteString 与后续 GetGlobal/CallFunction 之间引擎锁会释放，
	//     并发执行会让脚本间同名全局函数（如约定入口 execute）互相覆盖；
	//  2. execCtxHolder 是单槽，并发执行的上下文会互相污染。
	// 串行执行是当前引擎形态下的正确性前提，不是性能优化。
	execMu sync.Mutex

	mu sync.RWMutex
}

// Config 定义引擎配置。
type Config struct {
	MaxVMs         int           // 最大并发 VM 数（默认 10）
	VMTimeout      time.Duration // 单脚本执行超时（默认 5s）
	MaxMemory      int64         // 单 VM 内存上限（字节，默认 50MB）
	EnableDebug    bool          // 启用调试日志
	ScriptDir      string        // 文件脚本目录
	AllowedModules []string      // 允许的模块
	PoolSize       int           // VM 池大小（默认 5）
	EngineType     gsEngine.Type // 脚本引擎类型（默认 lua）

	// HTTPOptions http 出站模块护栏：域名白名单（空 = 全部拒绝）、超时、响应体上限。
	// 白名单来源：环境变量 SCRIPT_HTTP_ALLOWED_DOMAINS（逗号分隔）。
	HTTPOptions api.HTTPOptions
}

// DefaultConfig 返回默认配置。
func DefaultConfig() *Config {
	return &Config{
		MaxVMs:         10,
		VMTimeout:      5 * time.Second,
		MaxMemory:      50 * 1024 * 1024, // 50MB
		EnableDebug:    false,
		ScriptDir:      "scripts",
		AllowedModules: []string{},
		PoolSize:       5,
		EngineType:     gsEngine.LuaType,
	}
}

// EngineTypeLua 是 Lua 引擎类型标识（对齐 go-scripts）。
const EngineTypeLua = gsEngine.LuaType

// EnvHTTPAllowedDomains http 出站白名单环境变量名（逗号分隔域名）。
const EnvHTTPAllowedDomains = "SCRIPT_HTTP_ALLOWED_DOMAINS"

// HTTPAllowlistFromEnv 从环境变量读取域名白名单构造 http 护栏。
// 未设置 = 空 = 全部出站拒绝（fail-closed）。
func HTTPAllowlistFromEnv() api.HTTPOptions {
	raw := strings.TrimSpace(os.Getenv(EnvHTTPAllowedDomains))
	opts := api.HTTPOptions{}
	if raw == "" {
		return opts
	}
	for _, d := range strings.Split(raw, ",") {
		if d = strings.TrimSpace(d); d != "" {
			opts.AllowedDomains = append(opts.AllowedDomains, d)
		}
	}
	return opts
}

// ScriptEngineFactory 创建脚本引擎实例。
type ScriptEngineFactory func(config *Config, logger bLogger.Logger) (gsEngine.Engine, error)

// 默认引擎工厂：通过 go-scripts 全局工厂按 EngineType 创建。
var defaultEngineFactory ScriptEngineFactory = func(config *Config, _ bLogger.Logger) (gsEngine.Engine, error) {
	return gsEngine.NewScriptEngine(config.EngineType)
}

// SetEngineFactory 覆盖默认引擎工厂（用于注入自定义引擎实现）。
func SetEngineFactory(f ScriptEngineFactory) {
	if f != nil {
		defaultEngineFactory = f
	}
}

// NewEngine 创建一个新的 Hook 编排器（默认 Lua 引擎）。
func NewEngine(config *Config, logger bLogger.Logger) *Engine {
	return NewEngineWithFactory(config, logger, defaultEngineFactory)
}

// NewEngineWithFactory 使用指定工厂创建编排器（支持自定义引擎）。
func NewEngineWithFactory(config *Config, logger bLogger.Logger, factory ScriptEngineFactory) *Engine {
	if config == nil {
		config = DefaultConfig()
	}
	if factory == nil {
		factory = defaultEngineFactory
	}

	l := bLogger.NewHelper(logger.With("module", "lua/engine"))

	eng, err := factory(config, logger)
	if err != nil {
		l.Errorf(context.Background(), "Failed to create script engine (%s): %v", config.EngineType, err)
		// 降级：使用默认 Lua 引擎
		eng, err = gsEngine.NewScriptEngine(gsEngine.LuaType)
		if err != nil {
			l.Errorf(context.Background(), "Fallback Lua engine also failed: %v", err)
		}
	}

	e := &Engine{
		config:       config,
		logger:       l,
		registry:     hook.NewRegistry(),
		scriptEngine: eng,
		callbacks:    make(map[string][]ScriptCallback),
	}

	// 按引擎类型选择语言适配器，并注入执行上下文持有器。
	if eng != nil {
		binder := getBinder(eng.GetType())
		if binder != nil {
			// 各 binder 类型需要持有 holder 和 cfg，通过类型断言注入。
			// 这里用一个通用接口 WithContextHolder 来注入。
			if wh, ok := binder.(interface {
				WithContext(holder *execCtxHolder, cfg *Config) RuntimeBinder
			}); ok {
				binder = wh.WithContext(&e.execCtx, config)
			}
		}
		e.binder = binder

		// 注册 RuntimeHook：binder.Bind 会在 Init 后、Load/Execute 前被调用。
		if binder != nil {
			// PreInit：在 Init 前配置引擎（如 Lua 沙箱白名单，必须在 VM 创建前生效）
			if pre, ok := binder.(interface{ PreInit(eng gsEngine.Engine) }); ok {
				pre.PreInit(eng)
			}

			if registrar := gsEngine.AsRuntimeHookRegistrar(eng); registrar != nil {
				hook := e.buildBindHook(binder)
				if err := registrar.AddRuntimeHook(hook); err != nil {
					l.Errorf(context.Background(), "Failed to add runtime hook: %v", err)
				}
			}
		}

		if err := eng.Init(context.Background()); err != nil {
			l.Errorf(context.Background(), "Failed to init script engine: %v", err)
		}
	}

	// 自动加载脚本目录
	if config.ScriptDir != "" {
		if err := e.LoadScriptsFromDir(context.Background(), config.ScriptDir); err != nil {
			l.Errorf(context.Background(), "Failed to load scripts from %s: %v", config.ScriptDir, err)
		}
	}

	l.Infof(context.Background(), "Engine initialized (type: %s, timeout: %s)", config.EngineType, config.VMTimeout)

	return e
}

// buildBindHook 构造 RuntimeHook，在 VM 创建后调用 binder.Bind 注入业务依赖。
func (e *Engine) buildBindHook(binder RuntimeBinder) gsEngine.RuntimeHook {
	return func(_ context.Context) error {
		deps := &RuntimeDeps{
			Logger:          e.logger,
			Rdb:             e.rdb,
			EventBusManager: e.eventbusManager,
			OSSClient:       e.ossClient,
			Orchestrator:    e,
		}
		return binder.Bind(e.scriptEngine, deps)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Hook 注册与脚本管理
////////////////////////////////////////////////////////////////////////////////

// RegisterHook 注册一个 hook 点。
func (e *Engine) RegisterHook(name, description string) error {
	return e.registry.RegisterHook(name, description)
}

// AddScript 向 hook 添加脚本。接受 *Script 或任何兼容字段的结构体。
func (e *Engine) AddScript(hookName string, script interface{}) error {
	var hookScript *hook.Script

	switch s := script.(type) {
	case *Script:
		hookScript = &hook.Script{
			ID:          s.ID,
			Name:        s.Name,
			Hook:        s.Hook,
			Source:      s.Source,
			Enabled:     s.Enabled,
			Priority:    s.Priority,
			Description: s.Description,
			Version:     s.Version,
			Author:      s.Author,
			Critical:    s.Critical,
		}
	default:
		if scriptStruct, ok := extractScriptFields(script); ok {
			hookScript = scriptStruct
		} else {
			return fmt.Errorf("invalid script type: %T", script)
		}
	}

	return e.registry.AddScript(hookName, hookScript)
}

// extractScriptFields 通过反射从兼容结构体提取脚本字段（避免循环依赖）。
func extractScriptFields(script interface{}) (*hook.Script, bool) {
	v := reflect.ValueOf(script)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, false
	}

	getStringField := func(name string) string {
		field := v.FieldByName(name)
		if field.IsValid() && field.Kind() == reflect.String {
			return field.String()
		}
		return ""
	}
	getBoolField := func(name string) bool {
		field := v.FieldByName(name)
		if field.IsValid() && field.Kind() == reflect.Bool {
			return field.Bool()
		}
		return false
	}
	getIntField := func(name string) int {
		field := v.FieldByName(name)
		if field.IsValid() && field.Kind() == reflect.Int {
			return int(field.Int())
		}
		return 0
	}

	hookScript := &hook.Script{
		Name:        getStringField("Name"),
		Hook:        getStringField("Hook"),
		Source:      getStringField("Source"),
		Enabled:     getBoolField("Enabled"),
		Priority:    getIntField("Priority"),
		Description: getStringField("Description"),
		Author:      getStringField("Author"),
		Critical:    getBoolField("Critical"),
	}

	if hookScript.Name == "" || hookScript.Source == "" {
		return nil, false
	}
	return hookScript, true
}

// RemoveScript 从 hook 移除脚本。
func (e *Engine) RemoveScript(hookName, scriptName string) error {
	return e.registry.RemoveScript(hookName, scriptName)
}

// ListHooks 返回所有已注册 hook 名称。
func (e *Engine) ListHooks() []string {
	return e.registry.ListHooks()
}

// HookPointInfo 描述一个钩子点的注册情况（供管理面展示）。
type HookPointInfo struct {
	Name          string
	Description   string
	ScriptCount   int // 注册表挂载的脚本数
	CallbackCount int // hook.register 自注册的回调数
}

// HookPoints 返回全部钩子点及挂载数量（按名称排序）。
func (e *Engine) HookPoints() []HookPointInfo {
	hooks := e.registry.GetAllHooks()

	e.callbacksMu.RLock()
	defer e.callbacksMu.RUnlock()

	infos := make([]HookPointInfo, 0, len(hooks))
	for _, h := range hooks {
		infos = append(infos, HookPointInfo{
			Name:          h.Name,
			Description:   h.Description,
			ScriptCount:   len(h.Scripts),
			CallbackCount: len(e.callbacks[h.Name]),
		})
	}
	return infos
}

// ResetScriptRegistrations 清空脚本注册（registry + callbacks），供数据库脚本全量重同步。
// 注意：也会清掉从文件目录加载脚本的注册，调用方须在 Reset 后重新加载全部来源。
func (e *Engine) ResetScriptRegistrations() {
	e.execMu.Lock()
	defer e.execMu.Unlock()

	e.registry.Clear()

	e.callbacksMu.Lock()
	e.callbacks = make(map[string][]ScriptCallback)
	e.callbacksMu.Unlock()
}

// ExecuteTaskHandler 在引擎锁保护下执行脚本注册的任务处理器
// （asynq 桥的执行入口：与钩子/脚本执行互斥，共享同一 LState 是不安全的）。
func (e *Engine) ExecuteTaskHandler(ctx context.Context, name string, params map[string]any) error {
	e.execMu.Lock()
	defer e.execMu.Unlock()
	return api.InvokeHandler(ctx, name, params)
}

// registerCallback 注册语言无关的脚本回调到 hook（供 hook.register 经适配器调用）。
//
// 幂等语义：回调实现若提供 CallbackOwner()（非空），同 hook 下同归属的旧回调
// 会被替换——脚本热更新重执行 hook.register 时不会产生重复回调。
func (e *Engine) registerCallback(hookName string, cb ScriptCallback) {
	e.callbacksMu.Lock()
	defer e.callbacksMu.Unlock()

	owner := ""
	if o, ok := cb.(interface{ CallbackOwner() string }); ok {
		owner = o.CallbackOwner()
	}
	if owner != "" {
		kept := e.callbacks[hookName][:0]
		for _, existing := range e.callbacks[hookName] {
			if o, ok := existing.(interface{ CallbackOwner() string }); ok && o.CallbackOwner() == owner {
				continue // 同归属旧回调：替换
			}
			kept = append(kept, existing)
		}
		e.callbacks[hookName] = kept
	}

	e.callbacks[hookName] = append(e.callbacks[hookName], cb)

	e.logger.Infof(context.Background(), "Registered callback for hook: %s (owner: %q, total: %d callbacks)",
		hookName, owner, len(e.callbacks[hookName]))
}

// RegisterCallback 注册语言无关的脚本回调（公开方法，供适配器使用）。
func (e *Engine) RegisterCallback(hookName string, cb ScriptCallback) {
	e.registerCallback(hookName, cb)
}

////////////////////////////////////////////////////////////////////////////////
// 脚本执行
////////////////////////////////////////////////////////////////////////////////

// Execute 执行单个脚本（带执行上下文）。
// 语言无关：通过 scriptEngine.ExecuteString 执行脚本主体，
// 若脚本定义了 execute() 函数则调用它。
//
// 必须持有 execMu（ExecuteHook 已持有；单独调用时自行加锁）——
// 脚本执行与 execute() 全局函数探活/调用必须原子，否则并发下会被其他脚本覆盖。
func (e *Engine) Execute(ctx context.Context, script *Script, execCtx *Context) error {
	if e.scriptEngine == nil {
		return fmt.Errorf("no script engine available")
	}

	e.execMu.Lock()
	defer e.execMu.Unlock()

	return e.executeLocked(ctx, script, execCtx)
}

// executeLocked 在已持有 execMu 的前提下执行单个脚本。
func (e *Engine) executeLocked(ctx context.Context, script *Script, execCtx *Context) error {
	// 超时 ctx 按常规 defer cancel() 释放即可：上游 go-scripts/javascript
	// v0.0.9 起对带期限 ctx 改用本地计时器（父函数先行 timer.Stop 再
	// close(done)，迟到中断被构造性排除），立即取消不再制造"双就绪"
	// 窗口。旧版曾需把取消推迟到预算期限规避监视器迟到毒化（见
	// TestJSEngine_HookRegister 历史），随 v0.0.9 升版已退役。
	timeoutCtx, cancel := context.WithTimeout(ctx, e.config.VMTimeout)
	defer cancel()

	// 记录当前脚本名（hook.register 回调归属、诊断日志用）
	prevScript := e.execCtx.setScript(script.Name)
	defer e.execCtx.setScript(prevScript)

	// 设置执行上下文（供脚本 __get_ctx 等访问）
	prev := e.execCtx.set(execCtx)
	defer e.execCtx.reset(prev)

	// execute() 入口调用策略（哨兵法）：
	// go-scripts/lua 的 GetGlobal 把 LFunction convert 成 nil，无法直接探知脚本
	// 是否定义了 execute（未定义时读回也是 nil）。因此执行主体前先把全局 execute
	// 预写为哨兵字符串（可读回）：主体执行后哨兵原样保留 = 脚本未定义 execute，
	// 跳过调用；哨兵消失/变化（被函数覆盖，函数读回即 nil）= 已定义，调用之。
	// 哨兵不得含 NUL：luar/convert 对含 NUL 字符串的往返会失败（读回 nil）。
	const sentinel = "__go_wind_no_execute__"
	_ = e.scriptEngine.RegisterGlobal("execute", sentinel)

	// 执行脚本主体（定义函数、注册 hook 等）
	if _, err := e.scriptEngine.ExecuteString(timeoutCtx, script.Name, script.Source); err != nil {
		return err
	}

	// 哨兵被覆盖 → 脚本定义了 execute()，调用它
	//（函数经 GetGlobal 读出为 nil，因此 nil 同样视为「已定义」）
	if v, err := e.scriptEngine.GetGlobal("execute"); err != nil || v != sentinel {
		result, callErr := e.scriptEngine.CallFunction(timeoutCtx, "execute")
		if callErr != nil {
			// 正常情况到不了这里（哨兵仍在即未定义已跳过）；防御性透出
			return fmt.Errorf("execute function error: %w", callErr)
		}
		// execute 返回 false 表示中止
		if b, isBool := result.(bool); isBool && !b {
			return fmt.Errorf("script returned false")
		}
	}

	// 检查 __stop() 中止信号
	if execCtx != nil && execCtx.Stopped {
		return fmt.Errorf("execution stopped: %s", execCtx.StopReason)
	}
	return nil
}

// ExecuteHook 执行挂载在某个 hook 上的所有脚本与回调（按优先级/链式）。
// 整个 hook 链在 execMu 下串行执行：单 VM 形态下的正确性前提，
// 同时保证 execCtxHolder 单槽不会被并发 hook 互相污染。
func (e *Engine) ExecuteHook(ctx context.Context, hookName string, execCtx *Context) error {
	e.execMu.Lock()
	defer e.execMu.Unlock()

	// 先执行回调（语言无关：通过 ScriptCallback.Call）
	e.callbacksMu.RLock()
	callbacks := e.callbacks[hookName]
	e.callbacksMu.RUnlock()

	for i, callback := range callbacks {
		start := time.Now()
		_, err := callback.Call(ctx, execCtx)
		duration := time.Since(start)

		if err != nil {
			e.logger.Errorf(ctx, "Callback %d failed (hook: %s, duration: %s): %v",
				i+1, hookName, duration, err)
			return fmt.Errorf("callback %d failed: %w", i+1, err)
		}
		e.logger.Debugf(ctx, "Callback %d completed (hook: %s, duration: %s)",
			i+1, hookName, duration)
	}

	// 再执行注册的脚本
	hookScripts := e.registry.GetScripts(hookName)
	if len(hookScripts) == 0 && len(callbacks) == 0 {
		e.logger.Debugf(ctx, "No scripts or callbacks registered for hook: %s", hookName)
		return nil
	}

	e.logger.Debugf(ctx, "Executing %d scripts for hook: %s", len(hookScripts), hookName)

	for _, hookScript := range hookScripts {
		if !hookScript.Enabled {
			continue
		}

		script := &Script{
			ID:          hookScript.ID,
			Name:        hookScript.Name,
			Hook:        hookScript.Hook,
			Source:      hookScript.Source,
			Enabled:     hookScript.Enabled,
			Priority:    hookScript.Priority,
			Description: hookScript.Description,
			Version:     hookScript.Version,
			Author:      hookScript.Author,
			Critical:    hookScript.Critical,
		}

		start := time.Now()
		// 已持有 execMu，走 executeLocked（Execute 会重复加锁自死锁）
		err := e.executeLocked(ctx, script, execCtx)
		duration := time.Since(start)

		if err != nil {
			e.logger.Errorf(ctx, "Script '%s' failed (hook: %s, duration: %s): %v",
				script.Name, hookName, duration, err)
			return fmt.Errorf("script '%s' failed: %w", script.Name, err)
		}

		e.logger.Debugf(ctx, "Script '%s' completed (hook: %s, duration: %s)",
			script.Name, hookName, duration)
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// 脚本加载（基于 source 抽象）
////////////////////////////////////////////////////////////////////////////////

// SetSource 绑定脚本源（go-scripts/source.Reader）。
func (e *Engine) SetSource(src gsSource.Reader) {
	if e.scriptEngine != nil {
		e.scriptEngine.SetSource(src)
	}
}

// LoadScript 按 key 从绑定的 Source 加载脚本。
func (e *Engine) LoadScript(ctx context.Context, key string) error {
	if e.scriptEngine == nil {
		return fmt.Errorf("no script engine available")
	}
	return e.scriptEngine.Load(ctx, key)
}

// WatchScript 对 key 启动热更新监听。
func (e *Engine) WatchScript(ctx context.Context, key string) error {
	if e.scriptEngine == nil {
		return fmt.Errorf("no script engine available")
	}
	return e.scriptEngine.StartWatch(ctx, key)
}

// LoadScriptsFromDir 从目录加载所有脚本文件。
// 根据引擎类型匹配扩展名（.lua / .js）。
func (e *Engine) LoadScriptsFromDir(ctx context.Context, dir string) error {
	e.logger.Infof(ctx, "📂 Loading scripts from directory: %s", dir)

	if _, err := os.Stat(dir); err != nil {
		e.logger.Warnf(ctx, "Scripts directory does not exist: %s", dir)
		return nil
	}

	ext := scriptExtForType(e.config.EngineType)

	var loadedCount int
	walkErr := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ext) {
			return nil
		}
		e.logger.Infof(ctx, "📄 Loading script: %s", path)
		if err := e.LoadScriptFile(ctx, path); err != nil {
			e.logger.Errorf(ctx, "❌ Failed to load script %s: %v", path, err)
			return nil
		}
		e.logger.Infof(ctx, "✅ Successfully loaded script: %s", path)
		loadedCount++
		return nil
	})
	if walkErr != nil {
		return fmt.Errorf("failed to walk scripts directory: %w", walkErr)
	}

	e.logger.Infof(ctx, "Loaded %d scripts from %s", loadedCount, dir)
	return nil
}

// scriptExtForType 返回引擎类型对应的脚本文件扩展名。
func scriptExtForType(t gsEngine.Type) string {
	switch t {
	case gsEngine.LuaType:
		return ".lua"
	case gsEngine.JavaScriptType:
		return ".js"
	default:
		return ".lua"
	}
}

// LoadScriptFile 加载并预执行单个脚本文件（触发 hook.register 自注册）。
func (e *Engine) LoadScriptFile(ctx context.Context, filePath string) error {
	src := NewFileSource()
	code, err := src.Load(ctx, filePath)
	if err != nil {
		return fmt.Errorf("failed to read script file: %w", err)
	}
	return e.LoadScriptString(ctx, filePath, code)
}

// LoadScriptString 加载并预执行脚本字符串（触发 hook.register 自注册与全局副作用）。
func (e *Engine) LoadScriptString(ctx context.Context, scriptName, source string) error {
	if e.scriptEngine == nil {
		return fmt.Errorf("no script engine available")
	}
	e.execMu.Lock()
	defer e.execMu.Unlock()

	// 超时 ctx 按常规 defer cancel() 释放即可：上游 go-scripts/javascript
	// v0.0.9 起对带期限 ctx 改用本地计时器（父函数先行 timer.Stop 再
	// close(done)，迟到中断被构造性排除），立即取消不再制造"双就绪"
	// 窗口。旧版曾需把取消推迟到预算期限规避监视器迟到毒化（见
	// TestJSEngine_HookRegister 历史），随 v0.0.9 升版已退役。
	timeoutCtx, cancel := context.WithTimeout(ctx, e.config.VMTimeout)
	defer cancel()

	prevScript := e.execCtx.setScript(scriptName)
	defer e.execCtx.setScript(prevScript)

	_, err := e.scriptEngine.ExecuteString(timeoutCtx, scriptName, source)
	return err
}

////////////////////////////////////////////////////////////////////////////////
// 业务依赖注入（暂存于编排器，binder.Bind 执行时注入到 VM）
////////////////////////////////////////////////////////////////////////////////

// rebind 在依赖注入变化后重放语言适配器绑定。
//
// 必须显式重放的原因：go-scripts 两个引擎的 RuntimeHook 都在 Init 时一次性回放完毕，
// 而业务依赖（Redis/EventBus/OSS）只能经 Set* 在构造之后注入——不重放则模块注册
// 发生在依赖为 nil 的时刻，cache/eventbus/oss 模块会是空的。
// RegisterModule 为覆盖注册语义，重复 Bind 幂等。
func (e *Engine) rebind() {
	if e.scriptEngine == nil || e.binder == nil {
		return
	}
	if err := e.buildBindHook(e.binder)(context.Background()); err != nil {
		e.logger.Errorf(context.Background(), "rebind script modules failed: %v", err)
	}
}

// SetRedis 注入 Redis 客户端，启用 cache API。
func (e *Engine) SetRedis(rdb *redis.Client) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rdb = rdb
	e.rebind()
	e.logger.Info(context.Background(), "Redis client configured for cache API")
}

// SetEventBus 注入 EventBus 管理器，启用 eventbus API。
func (e *Engine) SetEventBus(manager *eventbus.Manager) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.eventbusManager = manager
	e.rebind()
	e.logger.Info(context.Background(), "EventBus manager configured for eventbus API")
}

// SetOSS 注入 OSS 客户端，启用 oss API。
func (e *Engine) SetOSS(client *oss.MinIOClient) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.ossClient = client
	e.rebind()
	e.logger.Info(context.Background(), "OSS client configured for oss API")
}

// ScriptEngine 返回底层脚本引擎（用于高级场景直接操作引擎）。
func (e *Engine) ScriptEngine() gsEngine.Engine {
	return e.scriptEngine
}

// Close 关闭编排器及底层引擎。
func (e *Engine) Close() error {
	e.logger.Info(context.Background(), "Closing engine...")

	if e.scriptEngine != nil {
		if err := e.scriptEngine.Close(); err != nil {
			e.logger.Errorf(context.Background(), "Error closing script engine: %v", err)
		}
	}

	e.callbacksMu.Lock()
	e.callbacks = make(map[string][]ScriptCallback)
	e.callbacksMu.Unlock()

	return nil
}
