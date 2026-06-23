package lua

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	lua "github.com/yuin/gopher-lua"

	gsEngine "github.com/tx7do/go-scripts"
	gsSource "github.com/tx7do/go-scripts/source"

	"go-wind-admin/pkg/eventbus"
	"go-wind-admin/pkg/lua/hook"
	"go-wind-admin/pkg/lua/internal/convert"
	"go-wind-admin/pkg/oss"
)

// CallbackInfo 存储已注册的回调信息。
type CallbackInfo struct {
	L        *lua.LState    // 回调所属 VM（专用 VM，不归还池）
	Function *lua.LFunction // 回调函数
	HookName string         // hook 名称
}

// Engine 是 Hook 编排器，语言无关。
//
// 重构后的架构：
//   - Engine（本结构）：负责 Hook 注册、脚本按优先级/链式执行、回调管理、执行上下文
//   - gsEngine.Engine（脚本引擎）：负责 VM 生命周期、脚本编译/执行、沙箱、模块注册
//
// 引擎通过 ScriptEngineFactory 创建，当前默认为 LuaEngine，
// 未来可无缝替换为 JSEngine/PythonEngine 等（只需实现 go-scripts.Engine 接口）。
type Engine struct {
	config   *Config
	logger   *log.Helper
	registry *hook.Registry

	// 脚本引擎（语言无关，实现 go-scripts.Engine 接口）
	scriptEngine gsEngine.Engine

	// 业务依赖（可选，按需注入到引擎）
	rdb             *redis.Client
	eventbusManager *eventbus.Manager
	ossClient       *oss.MinIOClient

	// Hook 回调（hook 名称 -> 多个回调）
	callbacks    map[string][]*CallbackInfo
	callbacksMu  sync.RWMutex
	dedicatedVMs map[*lua.LState]bool // 注册了回调/task handler 的专用 VM
	mu           sync.RWMutex
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
		EngineType:     EngineTypeLua,
	}
}

// ScriptEngineFactory 创建脚本引擎实例。
// 通过注入工厂实现多语言切换；默认使用 LuaEngine。
type ScriptEngineFactory func(config *Config, logger log.Logger) (gsEngine.Engine, error)

// 默认引擎工厂：按 config.EngineType 选择引擎实现。
var defaultEngineFactory ScriptEngineFactory = func(config *Config, logger log.Logger) (gsEngine.Engine, error) {
	switch config.EngineType {
	case "", EngineTypeLua:
		return newLuaEngineAsGS(config, logger)
	// 未来扩展：
	// case EngineTypeJavaScript:
	//     return newJSEngineAsGS(config, logger)
	default:
		return nil, fmt.Errorf("unsupported engine type: %s", config.EngineType)
	}
}

// SetEngineFactory 覆盖默认引擎工厂（用于注入自定义引擎实现）。
func SetEngineFactory(f ScriptEngineFactory) {
	if f != nil {
		defaultEngineFactory = f
	}
}

// newLuaEngineAsGS 创建 LuaEngine 并使其满足 gsEngine.Engine 接口。
// 返回 (engine, orchestratorRef) 中的 orchestratorRef 在 NewEngine 中回填。
// newLuaEngineAsGS 创建 LuaEngine（未初始化）。
// 调用方需在注入 HookRegistrar/业务依赖后再调用 Init()，
// 以确保 VM 池创建时 hook API 等已就绪。
func newLuaEngineAsGS(config *Config, logger log.Logger) (gsEngine.Engine, error) {
	return NewLuaEngine(config, logger), nil
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
		eng, err = newLuaEngineAsGS(config, logger)
		if err != nil {
			l.Errorf("Fallback Lua engine also failed: %v", err)
		}
	}

	e := &Engine{
		config:       config,
		logger:       l,
		registry:     hook.NewRegistry(),
		scriptEngine: eng,
		callbacks:    make(map[string][]*CallbackInfo),
		dedicatedVMs: make(map[*lua.LState]bool),
	}

	// 注入业务依赖到引擎（在 Init 前注入，使 VM 池创建时业务 API 就绪）
	e.injectDepsToEngine()

	// 建立 Hook 编排器 ↔ 引擎的双向引用（在 Init 前绑定，
	// 确保 VM 池创建时 hook API 已注册，脚本可调用 hook.register）
	e.bindHookRegistrar()

	// 初始化引擎（创建 VM 池）—— 必须在依赖注入与 HookRegistrar 绑定之后
	if eng != nil {
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

	l.Infof("Engine initialized (type: %s, pool: %d, timeout: %s)",
		config.EngineType, config.PoolSize, config.VMTimeout)

	return e
}

// injectDepsToEngine 将已注入的业务依赖转发给脚本引擎。
func (e *Engine) injectDepsToEngine() {
	if e.scriptEngine == nil {
		return
	}
	le, ok := e.scriptEngine.(*LuaEngine)
	if !ok {
		return // 非 Lua 引擎暂不支持业务 API 注入
	}
	if e.rdb != nil {
		le.SetRedis(e.rdb)
	}
	if e.eventbusManager != nil {
		le.SetEventBus(e.eventbusManager)
	}
	if e.ossClient != nil {
		le.SetOSS(e.ossClient)
	}
}

// bindHookRegistrar 建立编排器 ↔ 引擎双向引用。
// 引擎需感知编排器以便脚本调用 hook.register(name, fn) 时注册回调。
func (e *Engine) bindHookRegistrar() {
	if le, ok := e.scriptEngine.(*LuaEngine); ok {
		le.SetHookRegistrar(e)
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
		ID:          0,
		Name:        getStringField("Name"),
		Hook:        getStringField("Hook"),
		Source:      getStringField("Source"),
		Enabled:     getBoolField("Enabled"),
		Priority:    getIntField("Priority"),
		Description: getStringField("Description"),
		Version:     0,
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

// RegisterCallback 注册 Lua 回调函数到 hook（供脚本自注册调用）。
func (e *Engine) RegisterCallback(hookName string, L *lua.LState, fn *lua.LFunction) {
	e.callbacksMu.Lock()
	defer e.callbacksMu.Unlock()

	e.callbacks[hookName] = append(e.callbacks[hookName], &CallbackInfo{
		L:        L,
		Function: fn,
		HookName: hookName,
	})

	// 标记 VM 为专用（不归还池），保证回调函数引用持续有效
	e.mu.Lock()
	e.dedicatedVMs[L] = true
	e.mu.Unlock()

	// 同步标记到引擎
	if le, ok := e.scriptEngine.(*LuaEngine); ok {
		le.MarkVMDedicated(L)
	}

	e.logger.Infof("Registered callback for hook: %s (total: %d callbacks)",
		hookName, len(e.callbacks[hookName]))
}

// MarkVMDedicated 标记 VM 为专用（不归还池）。
func (e *Engine) MarkVMDedicated(L *lua.LState) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.dedicatedVMs[L] = true
	if le, ok := e.scriptEngine.(*LuaEngine); ok {
		le.MarkVMDedicated(L)
	}
	e.logger.Debugf("VM marked as dedicated")
}

////////////////////////////////////////////////////////////////////////////////
// 脚本执行
////////////////////////////////////////////////////////////////////////////////

// Execute 执行单个脚本（带执行上下文）。
// 语言无关：通过 scriptEngine.ExecuteWithResult 统一执行。
func (e *Engine) Execute(ctx context.Context, script *Script, execCtx *Context) error {
	le, ok := e.scriptEngine.(*LuaEngine)
	if !ok {
		// 非 Lua 引擎：退化为 ExecuteString
		_, err := e.scriptEngine.ExecuteString(ctx, script.Name, script.Source)
		return err
	}

	result, err := le.ExecuteWithResult(ctx, script.Name, script.Source, func(L *lua.LState) lua.LValue {
		return e.contextToLuaTable(L, execCtx)
	})

	if err != nil {
		return err
	}

	// 脚本返回 false 表示中止（保持原语义）
	if b, isBool := result.(bool); isBool && !b {
		return fmt.Errorf("script returned false")
	}

	// 检查 ctx.stop() 中止信号
	if execCtx != nil && execCtx.Stopped {
		return fmt.Errorf("execution stopped: %s", execCtx.StopReason)
	}
	return nil
}

// ExecuteHook 执行挂载在某个 hook 上的所有脚本与回调（按优先级/链式）。
func (e *Engine) ExecuteHook(ctx context.Context, hookName string, execCtx *Context) error {
	// 先执行回调
	e.callbacksMu.RLock()
	callbacks := e.callbacks[hookName]
	e.callbacksMu.RUnlock()

	for i, callback := range callbacks {
		start := time.Now()
		err := e.executeCallback(ctx, callback, execCtx)
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

// executeCallback 执行已注册的回调函数。
// 修复：超时后销毁损坏的专用 VM 并通知引擎（原实现未销毁，可能导致后续调用异常）。
func (e *Engine) executeCallback(ctx context.Context, callback *CallbackInfo, execCtx *Context) error {
	L := callback.L

	timeoutCtx, cancel := context.WithTimeout(ctx, e.config.VMTimeout)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		L.SetContext(timeoutCtx)
		L.Push(callback.Function)
		L.Push(e.contextToLuaTable(L, execCtx))

		if err := L.PCall(1, 1, nil); err != nil {
			errChan <- fmt.Errorf("callback execution error: %w", err)
			return
		}

		ret := L.Get(-1)
		L.Pop(1)

		if ret.Type() == lua.LTBool && !lua.LVAsBool(ret) {
			errChan <- fmt.Errorf("callback returned false")
			return
		}
		errChan <- nil
	}()

	select {
	case err := <-errChan:
		return err
	case <-timeoutCtx.Done():
		// 修复原 bug：超时后销毁损坏的专用 VM。
		// 专用 VM 无法重建（回调函数引用绑定在此 VM），只能关闭，
		// 后续对该回调的调用将返回错误（VM 已关闭），需调用方重新注册。
		func() {
			defer func() { recover() }()
			L.Close()
		}()
		e.mu.Lock()
		delete(e.dedicatedVMs, L)
		e.mu.Unlock()
		return fmt.Errorf("callback execution timeout after %s (VM destroyed)", e.config.VMTimeout)
	}
}

// contextToLuaTable 将 Context 转为 Lua table（含 get/set/stop 方法）。
func (e *Engine) contextToLuaTable(L *lua.LState, ctx *Context) *lua.LTable {
	table := L.NewTable()

	table.RawSetString("get", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		if val, ok := ctx.Data[key]; ok {
			L.Push(convert.ToLuaValue(L, val))
		} else {
			L.Push(lua.LNil)
		}
		return 1
	}))

	table.RawSetString("set", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		val := L.Get(2)
		ctx.Data[key] = convert.ToGoValue(val)
		return 0
	}))

	table.RawSetString("stop", L.NewFunction(func(L *lua.LState) int {
		reason := L.CheckString(1)
		ctx.Stopped = true
		ctx.StopReason = reason
		return 0
	}))

	return table
}

////////////////////////////////////////////////////////////////////////////////
// 脚本加载（基于 source 抽象，保留向后兼容）
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

// LoadScriptsFromDir 从目录加载所有 .lua 文件（向后兼容，内部转用 source 抽象）。
func (e *Engine) LoadScriptsFromDir(ctx context.Context, dir string) error {
	e.logger.Infof("📂 Loading scripts from directory: %s", dir)

	if _, err := osStat(dir); err != nil {
		e.logger.Warnf("Scripts directory does not exist: %s", dir)
		return nil
	}

	var loadedCount int
	walkErr := walkLuaFiles(dir, func(path string) error {
		e.logger.Infof("📄 Loading script: %s", path)
		if err := e.LoadScriptFile(ctx, path); err != nil {
			e.logger.Errorf("❌ Failed to load script %s: %v", path, err)
			return nil // 继续加载其他脚本
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

// LoadScriptFile 加载并预执行单个脚本文件（触发 hook.register 自注册）。
func (e *Engine) LoadScriptFile(ctx context.Context, filePath string) error {
	// filePath 已是完整路径，不传搜索路径以避免 resolveKey 再次拼接前缀
	src := NewFileSource()
	code, err := src.Load(ctx, filePath)
	if err != nil {
		return fmt.Errorf("failed to read script file: %w", err)
	}
	return e.LoadScriptString(ctx, filePath, code)
}

// LoadScriptString 加载并预执行脚本字符串。
func (e *Engine) LoadScriptString(ctx context.Context, scriptName, source string) error {
	if e.scriptEngine == nil {
		return fmt.Errorf("no script engine available")
	}
	return e.scriptEngine.LoadString(ctx, scriptName, source)
}

////////////////////////////////////////////////////////////////////////////////
// 业务依赖注入
////////////////////////////////////////////////////////////////////////////////

// SetRedis 注入 Redis 客户端，启用 cache API。
func (e *Engine) SetRedis(rdb *redis.Client) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rdb = rdb
	if le, ok := e.scriptEngine.(*LuaEngine); ok {
		le.SetRedis(rdb)
	}
}

// SetEventBus 注入 EventBus 管理器，启用 eventbus API。
func (e *Engine) SetEventBus(manager *eventbus.Manager) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.eventbusManager = manager
	if le, ok := e.scriptEngine.(*LuaEngine); ok {
		le.SetEventBus(manager)
	}
}

// SetOSS 注入 OSS 客户端，启用 oss API。
func (e *Engine) SetOSS(client *oss.MinIOClient) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.ossClient = client
	if le, ok := e.scriptEngine.(*LuaEngine); ok {
		le.SetOSS(client)
	}
}

// ScriptEngine 返回底层脚本引擎（用于高级场景直接操作引擎）。
func (e *Engine) ScriptEngine() gsEngine.Engine {
	return e.scriptEngine
}

// Close 关闭编排器及底层引擎。
func (e *Engine) Close() error {
	e.logger.Info("Closing engine...")

	// 关闭专用 VM（Lua 引擎的 Close 会处理，此处兼容旧路径）
	e.mu.Lock()
	for vm := range e.dedicatedVMs {
		if vm != nil {
			func() {
				defer func() {
					if r := recover(); r != nil {
						e.logger.Warnf("Recovered from panic while closing VM: %v", r)
					}
				}()
				vm.Close()
			}()
		}
	}
	e.dedicatedVMs = make(map[*lua.LState]bool)
	e.mu.Unlock()

	// 关闭底层脚本引擎（含 VM 池、热更新 goroutine）
	if e.scriptEngine != nil {
		if err := e.scriptEngine.Close(); err != nil {
			e.logger.Errorf("Error closing script engine: %v", err)
		}
	}

	return nil
}
