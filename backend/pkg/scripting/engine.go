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

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"

	gsEngine "github.com/tx7do/go-scripts"
	gsSource "github.com/tx7do/go-scripts/source"

	// 空导入各语言引擎包：触发 init() 将引擎工厂注册到全局注册表。
	// 各引擎的适配器在对应的 runtime_*.go 中引用。
	_ "github.com/tx7do/go-scripts/javascript"
	_ "github.com/tx7do/go-scripts/lua"

	"go-wind-admin/pkg/eventbus"
	"go-wind-admin/pkg/oss"
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
	logger   *log.Helper
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

// ScriptEngineFactory 创建脚本引擎实例。
type ScriptEngineFactory func(config *Config, logger log.Logger) (gsEngine.Engine, error)

// 默认引擎工厂：通过 go-scripts 全局工厂按 EngineType 创建。
var defaultEngineFactory ScriptEngineFactory = func(config *Config, _ log.Logger) (gsEngine.Engine, error) {
	return gsEngine.NewScriptEngine(config.EngineType)
}

// SetEngineFactory 覆盖默认引擎工厂（用于注入自定义引擎实现）。
func SetEngineFactory(f ScriptEngineFactory) {
	if f != nil {
		defaultEngineFactory = f
	}
}

// NewEngine 创建一个新的 Hook 编排器（默认 Lua 引擎）。
func NewEngine(config *Config, logger log.Logger) *Engine {
	return NewEngineWithFactory(config, logger, defaultEngineFactory)
}

// NewEngineWithFactory 使用指定工厂创建编排器（支持自定义引擎）。
func NewEngineWithFactory(config *Config, logger log.Logger, factory ScriptEngineFactory) *Engine {
	if config == nil {
		config = DefaultConfig()
	}
	if factory == nil {
		factory = defaultEngineFactory
	}

	l := log.NewHelper(log.With(logger, "module", "lua/engine"))

	eng, err := factory(config, logger)
	if err != nil {
		l.Errorf("Failed to create script engine (%s): %v", config.EngineType, err)
		// 降级：使用默认 Lua 引擎
		eng, err = gsEngine.NewScriptEngine(gsEngine.LuaType)
		if err != nil {
			l.Errorf("Fallback Lua engine also failed: %v", err)
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
					l.Errorf("Failed to add runtime hook: %v", err)
				}
			}
		}

		if err := eng.Init(context.Background()); err != nil {
			l.Errorf("Failed to init script engine: %v", err)
		}
	}

	// 自动加载脚本目录
	if config.ScriptDir != "" {
		if err := e.LoadScriptsFromDir(context.Background(), config.ScriptDir); err != nil {
			l.Errorf("Failed to load scripts from %s: %v", config.ScriptDir, err)
		}
	}

	l.Infof("Engine initialized (type: %s, timeout: %s)", config.EngineType, config.VMTimeout)

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
	if v.Kind() == reflect.Ptr {
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

// registerCallback 注册语言无关的脚本回调到 hook（供 hook.register 经适配器调用）。
func (e *Engine) registerCallback(hookName string, cb ScriptCallback) {
	e.callbacksMu.Lock()
	defer e.callbacksMu.Unlock()

	e.callbacks[hookName] = append(e.callbacks[hookName], cb)

	e.logger.Infof("Registered callback for hook: %s (total: %d callbacks)",
		hookName, len(e.callbacks[hookName]))
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
func (e *Engine) Execute(ctx context.Context, script *Script, execCtx *Context) error {
	if e.scriptEngine == nil {
		return fmt.Errorf("no script engine available")
	}

	// 设置执行上下文（供脚本 __get_ctx 等访问）
	prev := e.execCtx.set(execCtx)
	defer e.execCtx.reset(prev)

	// 执行脚本主体（定义函数、注册 hook 等）
	_, err := e.scriptEngine.ExecuteString(ctx, script.Name, script.Source)
	if err != nil {
		return err
	}

	// 若脚本定义了 execute() 函数，调用它
	if result, callErr := e.scriptEngine.CallFunction(ctx, "execute"); callErr == nil {
		// execute 返回 false 表示中止
		if b, isBool := result.(bool); isBool && !b {
			return fmt.Errorf("script returned false")
		}
	} else if !isMissingFunctionErr(callErr) {
		return fmt.Errorf("execute function error: %w", callErr)
	}

	// 检查 __stop() 中止信号
	if execCtx != nil && execCtx.Stopped {
		return fmt.Errorf("execution stopped: %s", execCtx.StopReason)
	}
	return nil
}

// isMissingFunctionErr 判断错误是否为「函数不存在」（即脚本未定义 execute）。
func isMissingFunctionErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "not a function") ||
		strings.Contains(msg, "nil") ||
		strings.Contains(msg, "attempt to call") ||
		strings.Contains(msg, "not found")
}

// ExecuteHook 执行挂载在某个 hook 上的所有脚本与回调（按优先级/链式）。
func (e *Engine) ExecuteHook(ctx context.Context, hookName string, execCtx *Context) error {
	// 先执行回调（语言无关：通过 ScriptCallback.Call）
	e.callbacksMu.RLock()
	callbacks := e.callbacks[hookName]
	e.callbacksMu.RUnlock()

	for i, callback := range callbacks {
		start := time.Now()
		_, err := callback.Call(ctx, execCtx)
		duration := time.Since(start)

		if err != nil {
			e.logger.Errorf("Callback %d failed (hook: %s, duration: %s): %v",
				i+1, hookName, duration, err)
			return fmt.Errorf("callback %d failed: %w", i+1, err)
		}
		e.logger.Debugf("Callback %d completed (hook: %s, duration: %s)",
			i+1, hookName, duration)
	}

	// 再执行注册的脚本
	hookScripts := e.registry.GetScripts(hookName)
	if len(hookScripts) == 0 && len(callbacks) == 0 {
		e.logger.Debugf("No scripts or callbacks registered for hook: %s", hookName)
		return nil
	}

	e.logger.Debugf("Executing %d scripts for hook: %s", len(hookScripts), hookName)

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
		err := e.Execute(ctx, script, execCtx)
		duration := time.Since(start)

		if err != nil {
			e.logger.Errorf("Script '%s' failed (hook: %s, duration: %s): %v",
				script.Name, hookName, duration, err)
			return fmt.Errorf("script '%s' failed: %w", script.Name, err)
		}

		e.logger.Debugf("Script '%s' completed (hook: %s, duration: %s)",
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
	e.logger.Infof("📂 Loading scripts from directory: %s", dir)

	if _, err := os.Stat(dir); err != nil {
		e.logger.Warnf("Scripts directory does not exist: %s", dir)
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
		e.logger.Infof("📄 Loading script: %s", path)
		if err := e.LoadScriptFile(ctx, path); err != nil {
			e.logger.Errorf("❌ Failed to load script %s: %v", path, err)
			return nil
		}
		e.logger.Infof("✅ Successfully loaded script: %s", path)
		loadedCount++
		return nil
	})
	if walkErr != nil {
		return fmt.Errorf("failed to walk scripts directory: %w", walkErr)
	}

	e.logger.Infof("Loaded %d scripts from %s", loadedCount, dir)
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
	_, err := e.scriptEngine.ExecuteString(ctx, scriptName, source)
	return err
}

////////////////////////////////////////////////////////////////////////////////
// 业务依赖注入（暂存于编排器，binder.Bind 执行时注入到 VM）
////////////////////////////////////////////////////////////////////////////////

// SetRedis 注入 Redis 客户端，启用 cache API。
func (e *Engine) SetRedis(rdb *redis.Client) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rdb = rdb
	e.logger.Info("Redis client configured for cache API")
}

// SetEventBus 注入 EventBus 管理器，启用 eventbus API。
func (e *Engine) SetEventBus(manager *eventbus.Manager) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.eventbusManager = manager
	e.logger.Info("EventBus manager configured for eventbus API")
}

// SetOSS 注入 OSS 客户端，启用 oss API。
func (e *Engine) SetOSS(client *oss.MinIOClient) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.ossClient = client
	e.logger.Info("OSS client configured for oss API")
}

// ScriptEngine 返回底层脚本引擎（用于高级场景直接操作引擎）。
func (e *Engine) ScriptEngine() gsEngine.Engine {
	return e.scriptEngine
}

// Close 关闭编排器及底层引擎。
func (e *Engine) Close() error {
	e.logger.Info("Closing engine...")

	if e.scriptEngine != nil {
		if err := e.scriptEngine.Close(); err != nil {
			e.logger.Errorf("Error closing script engine: %v", err)
		}
	}

	e.callbacksMu.Lock()
	e.callbacks = make(map[string][]ScriptCallback)
	e.callbacksMu.Unlock()

	return nil
}
