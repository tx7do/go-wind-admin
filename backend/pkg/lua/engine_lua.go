package lua

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	lua "github.com/yuin/gopher-lua"

	gsEngine "github.com/tx7do/go-scripts"
	gsSource "github.com/tx7do/go-scripts/source"

	"go-wind-admin/pkg/eventbus"
	"go-wind-admin/pkg/lua/api"
	"go-wind-admin/pkg/oss"
)

// 引擎类型标识（对齐 go-scripts 的 Type 概念）。
const (
	EngineTypeLua = gsEngine.LuaType
	// 未来扩展：
	// EngineTypeJavaScript = gsEngine.JavaScriptType
	// EngineTypePython     = gsEngine.GPythonType
)

// 编译期断言：*LuaEngine 实现 go-scripts 的 Engine 接口（所有能力子接口之和）。
//
// 这是「多语言统一抽象」的关键 —— LuaEngine 与未来可能加入的
// JSEngine（goja）、PythonEngine（gpython）等实现同一接口，
// Hook 编排器只需面向 Engine 接口编程，即可透明切换脚本语言。
var _ gsEngine.Engine = (*LuaEngine)(nil)

// VMManager 提供 VM 管理能力（标记 VM 为专用，不归还池）。
// 供 api 包注册 task handler 时使用。
type VMManager interface {
	MarkVMDedicated(L *lua.LState)
}

// HookRegistrar 是 Hook 编排器暴露给引擎的回调接口。
// 引擎在执行脚本时，脚本可能调用 hook.register(name, fn) 自注册回调，
// 通过此接口将回调注册回编排器。
type HookRegistrar interface {
	RegisterCallback(hookName string, L *lua.LState, fn *lua.LFunction)
}

// LuaEngine 是一个安全的 Lua 脚本引擎，实现 go-scripts 的 Engine 接口。
//
// 与 go-scripts/lua 子模块的区别（为何不直接用它）：
//   - 本引擎使用沙箱（openSafeLibs），禁用 dofile/loadfile/load/os/io 等危险函数
//   - go-scripts/lua 调 OpenLibs() 全开，对插件系统不安全
//   - 本引擎自持 VM 池，与业务 API（cache/eventbus/oss/crypto）深度集成
//
// 实现 go-scripts.Engine 接口后，未来可平滑接入 JS/Python 等其他语言引擎。
type LuaEngine struct {
	config *Config
	pool   *vmPool
	logger *log.Helper
	source gsSource.Reader // 绑定的脚本源（go-scripts/source.Reader）

	// 业务依赖（可选，按需注入）
	rdb             *redis.Client     // Redis，供 cache API
	eventbusManager *eventbus.Manager // EventBus，供 eventbus API
	ossClient       *oss.MinIOClient  // OSS/MinIO，供 oss API

	// Hook 编排器回调（脚本自注册 hook 回调用）
	hookRegistrar HookRegistrar

	// 专用 VM 集合（注册了 callback/task handler 的 VM 不归还池）
	dedicatedVMs map[*lua.LState]bool
	mu           sync.RWMutex

	// 全局注册项：新创建的 VM 会重放这些注册，保证池中所有 VM 状态一致。
	// 解决池化架构下 RegisterFunction/RegisterGlobal/RegisterModule
	// 只作用于单个 VM 的问题。
	globalFuncs   map[string]lua.LGFunction // 全局宿主函数
	globalVars    map[string]any            // 全局变量
	globalModules map[string]lua.LGFunction // 模块 loader
	regMu         sync.RWMutex

	// 生命周期状态
	initialized bool
	lastError   error
	lastErrMu   sync.Mutex

	// 热更新
	watchers   map[string]context.CancelFunc
	watchersMu sync.Mutex
}

// NewLuaEngine 创建一个 Lua 引擎实例（未初始化，需调用 Init）。
func NewLuaEngine(config *Config, logger log.Logger) *LuaEngine {
	if config == nil {
		config = DefaultConfig()
	}
	l := log.NewHelper(log.With(logger, "module", "lua/engine_lua"))
	return &LuaEngine{
		config:        config,
		logger:        l,
		dedicatedVMs:  make(map[*lua.LState]bool),
		globalFuncs:   make(map[string]lua.LGFunction),
		globalVars:    make(map[string]any),
		globalModules: make(map[string]lua.LGFunction),
		watchers:      make(map[string]context.CancelFunc),
	}
}

// GetType 返回引擎类型标识。
func (e *LuaEngine) GetType() gsEngine.Type { return EngineTypeLua }

// Init 初始化引擎（创建 VM 池）。重复调用返回错误。
func (e *LuaEngine) Init(_ context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.initialized {
		e.setLastError(ErrEngineAlreadyInitialized)
		return ErrEngineAlreadyInitialized
	}

	e.pool = newVMPool(e.config.PoolSize, func() *lua.LState {
		return e.createVM()
	})
	e.initialized = true
	e.ClearError()

	e.logger.Infof("Lua engine initialized (pool: %d, timeout: %s)", e.config.PoolSize, e.config.VMTimeout)
	return nil
}

// Close 关闭引擎，释放所有 VM 与热更新 goroutine。
func (e *LuaEngine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.initialized {
		e.setLastError(ErrEngineNotInitialized)
		return ErrEngineNotInitialized
	}

	// 停止所有热更新
	e.stopAllWatchers()

	// 关闭专用 VM（注册了 callback/task handler 的）
	for vm := range e.dedicatedVMs {
		if vm != nil {
			func() {
				defer func() { recover() }()
				vm.Close()
			}()
		}
	}
	e.dedicatedVMs = make(map[*lua.LState]bool)

	// 关闭 VM 池
	if e.pool != nil {
		e.pool.Close()
	}

	e.initialized = false
	return nil
}

// IsInitialized 报告引擎是否已初始化。
func (e *LuaEngine) IsInitialized() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.initialized
}

// GetLastError 返回最后一次记录的错误。
func (e *LuaEngine) GetLastError() error {
	e.lastErrMu.Lock()
	defer e.lastErrMu.Unlock()
	return e.lastError
}

// ClearError 清除错误状态。
func (e *LuaEngine) ClearError() {
	e.lastErrMu.Lock()
	defer e.lastErrMu.Unlock()
	e.lastError = nil
}

func (e *LuaEngine) setLastError(err error) {
	e.lastErrMu.Lock()
	defer e.lastErrMu.Unlock()
	e.lastError = err
}

// SetHookRegistrar 注入 Hook 编排器回调，供脚本自注册 hook 回调使用。
// 必须在 Init 前调用。
func (e *LuaEngine) SetHookRegistrar(r HookRegistrar) {
	e.hookRegistrar = r
}

// SetRedis 注入 Redis 客户端，启用 cache API。
func (e *LuaEngine) SetRedis(rdb *redis.Client) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rdb = rdb
	e.logger.Info("Redis client configured for Lua cache API")
}

// SetEventBus 注入 EventBus 管理器，启用 eventbus API。
func (e *LuaEngine) SetEventBus(manager *eventbus.Manager) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.eventbusManager = manager
	e.logger.Info("EventBus manager configured for Lua eventbus API")
}

// SetOSS 注入 OSS 客户端，启用 oss API。
func (e *LuaEngine) SetOSS(client *oss.MinIOClient) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.ossClient = client
	e.logger.Info("OSS client configured for Lua OSS API")
}

////////////////////////////////////////////////////////////////////////////////
// VM 创建与沙箱（从原 engine.go 迁移，保持安全语义不变）
////////////////////////////////////////////////////////////////////////////////

// createVM 创建一个带沙箱与 API 绑定的 Lua VM。
// 新 VM 会重放已注册的全局函数/变量/模块，保证池中所有 VM 状态一致。
func (e *LuaEngine) createVM() *lua.LState {
	L := lua.NewState(lua.Options{
		CallStackSize:       120,
		RegistrySize:        120 * 20,
		SkipOpenLibs:        true, // 选择性开启安全库
		IncludeGoStackTrace: e.config.EnableDebug,
	})

	e.openSafeLibs(L)
	e.registerAPIs(L)
	e.replayRegistrations(L) // 重放全局注册项，保证池内 VM 一致

	return L
}

// replayRegistrations 将已注册的全局函数/变量/模块应用到指定 VM。
func (e *LuaEngine) replayRegistrations(L *lua.LState) {
	e.regMu.RLock()
	defer e.regMu.RUnlock()

	for name, fn := range e.globalFuncs {
		L.SetGlobal(name, L.NewFunction(fn))
	}
	for name, val := range e.globalVars {
		L.SetGlobal(name, toLValue(L, val))
	}
	for name, loader := range e.globalModules {
		L.PreloadModule(name, loader)
	}
}

// applyToAllPoolVMs 对池中所有现存 VM 应用一次变更。
// 通过排空池、应用、再填回的方式实现，保证池内 VM 状态一致。
func (e *LuaEngine) applyToAllPoolVMs(apply func(L *lua.LState)) {
	if e.pool == nil {
		return
	}
	// 排空池
	var vms []*lua.LState
	for {
		L := e.pool.TryGet()
		if L == nil {
			break
		}
		vms = append(vms, L)
	}
	// 应用变更
	for _, L := range vms {
		func() {
			defer func() { recover() }()
			apply(L)
		}()
	}
	// 填回
	for _, L := range vms {
		e.pool.Put(L)
	}
}

// openSafeLibs 仅开启安全的 Lua 标准库，禁用危险函数。
func (e *LuaEngine) openSafeLibs(L *lua.LState) {
	lua.OpenBase(L)
	lua.OpenTable(L)
	lua.OpenString(L)
	lua.OpenMath(L)

	// 禁用危险函数
	L.SetGlobal("dofile", lua.LNil)
	L.SetGlobal("loadfile", lua.LNil)
	L.SetGlobal("load", lua.LNil)
	L.SetGlobal("loadstring", lua.LNil)

	// 不开启完整 package 库（不安全），仅创建 package.preload 供模块系统使用
	packageTable := L.NewTable()
	preloadTable := L.NewTable()
	packageTable.RawSetString("preload", preloadTable)
	L.SetGlobal("package", packageTable)

	// 最小化 require() 实现
	L.SetGlobal("require", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)

		pkg := L.GetGlobal("package")
		if pkg == lua.LNil {
			L.RaiseError("package table not found")
			return 0
		}
		pkgTable := pkg.(*lua.LTable)

		loaded := pkgTable.RawGetString("loaded")
		if loaded == lua.LNil {
			loaded = L.NewTable()
			pkgTable.RawSetString("loaded", loaded)
		}
		loadedTable := loaded.(*lua.LTable)

		cached := loadedTable.RawGetString(name)
		if cached != lua.LNil {
			L.Push(cached)
			return 1
		}

		preload := pkgTable.RawGetString("preload")
		if preload == lua.LNil {
			L.RaiseError("module '%s' not found in package.preload", name)
			return 0
		}
		preloadTable := preload.(*lua.LTable)

		loader := preloadTable.RawGetString(name)
		if loader == lua.LNil {
			L.RaiseError("module '%s' not found", name)
			return 0
		}

		L.Push(loader)
		L.Call(0, 1)

		result := L.Get(-1)
		loadedTable.RawSetString(name, result)

		L.Push(result)
		return 1
	}))
}

// registerAPIs 注册业务 API 模块（logger/cache/eventbus/oss/crypto/hook/task/util）。
func (e *LuaEngine) registerAPIs(L *lua.LState) {
	api.RegisterLogger(L, e.logger)

	if e.rdb != nil {
		api.RegisterCache(L, e.rdb, e.logger)
	}
	if e.eventbusManager != nil {
		api.RegisterEventBus(L, e.eventbusManager, e.logger)
	}
	if e.ossClient != nil {
		api.RegisterOSS(L, e.ossClient, e.logger)
	}

	api.RegisterCrypto(L, e.logger)

	// hook API 需要 HookRegistrar；若未注入则跳过（脚本无法自注册 hook）
	if e.hookRegistrar != nil {
		api.RegisterHookAPI(L, &hookEngineAdapter{registrar: e.hookRegistrar, lister: e}, e.logger)
	}

	api.RegisterTask(L, e, e.logger)
	api.RegisterUtilAPI(L, e.logger)

	// 模块别名与全局便捷变量
	preloadTable := L.GetGlobal("package").(*lua.LTable).RawGetString("preload").(*lua.LTable)

	if logLoader := preloadTable.RawGetString("kratos_logger"); logLoader != lua.LNil {
		preloadTable.RawSetString("logger", logLoader)
	}
	if hookLoader := preloadTable.RawGetString("kratos_hook"); hookLoader != lua.LNil {
		preloadTable.RawSetString("hook", hookLoader)
	}
	if utilLoader := preloadTable.RawGetString("kratos_util"); utilLoader != lua.LNil {
		preloadTable.RawSetString("util", utilLoader)
	}

	if logLoader, ok := preloadTable.RawGetString("logger").(*lua.LFunction); ok {
		L.Push(logLoader)
		if err := L.PCall(0, 1, nil); err == nil {
			L.SetGlobal("log", L.Get(-1))
			L.Pop(1)
		}
	}

	if hookLoader, ok := preloadTable.RawGetString("hook").(*lua.LFunction); ok {
		L.Push(hookLoader)
		if err := L.PCall(0, 1, nil); err == nil {
			L.SetGlobal("hook", L.Get(-1))
			L.Pop(1)
		}
	}
}

// MarkVMDedicated 将 VM 标记为专用（不归还池）。
// 用于注册了 task handler / hook callback 的 VM，保证其函数引用持续有效。
func (e *LuaEngine) MarkVMDedicated(L *lua.LState) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.dedicatedVMs[L] = true
	e.logger.Debugf("VM marked as dedicated")
}

// ListHooks 返回所有已注册的 hook 名称（供 hook API 的 list 函数）。
func (e *LuaEngine) ListHooks() []string {
	// 委托给 Hook 编排器；若未注入返回空
	if hr, ok := e.hookRegistrar.(hookLister); ok {
		return hr.ListHooks()
	}
	return nil
}

// isDedicated 报告 VM 是否为专用 VM。
func (e *LuaEngine) isDedicated(L *lua.LState) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.dedicatedVMs[L]
}

////////////////////////////////////////////////////////////////////////////////
// go-scripts Engine 接口实现
////////////////////////////////////////////////////////////////////////////////

// --- ScriptLoader ---

// SetSource 绑定脚本源。传 nil 清除绑定。
func (e *LuaEngine) SetSource(src gsSource.Reader) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.source = src
}

// GetSource 返回当前绑定的脚本源。
func (e *LuaEngine) GetSource() gsSource.Reader {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.source
}

// Load 从绑定的 Source 按 key 加载脚本（仅编译/缓存，不执行）。
func (e *LuaEngine) Load(ctx context.Context, key string) error {
	if !e.IsInitialized() {
		e.setLastError(ErrEngineNotInitialized)
		return ErrEngineNotInitialized
	}
	src := e.GetSource()
	if src == nil {
		err := fmt.Errorf("lua engine: no source bound")
		e.setLastError(err)
		return err
	}
	code, err := src.Load(ctx, key)
	if err != nil {
		e.setLastError(err)
		return err
	}
	// 在专用 VM 上预执行脚本（触发 hook.register 等自注册）
	return e.preloadScript(ctx, key, code)
}

// LoadMulti 批量加载脚本，遇首个错误中止。
func (e *LuaEngine) LoadMulti(ctx context.Context, keys []string) error {
	for _, k := range keys {
		if err := e.Load(ctx, k); err != nil {
			return err
		}
	}
	return nil
}

// LoadString 编译内联脚本字符串（不经 Source）。
func (e *LuaEngine) LoadString(_ context.Context, name, code string) error {
	if !e.IsInitialized() {
		e.setLastError(ErrEngineNotInitialized)
		return ErrEngineNotInitialized
	}
	return e.preloadScript(context.Background(), name, code)
}

// preloadScript 在专用 VM 上预执行脚本源码。
// 这使脚本能够调用 hook.register(name, fn) 自注册回调。
func (e *LuaEngine) preloadScript(ctx context.Context, name, code string) error {
	L := e.pool.Get()
	defer func() {
		// 专用 VM 不归还池
		if !e.isDedicated(L) {
			e.pool.Put(L)
		}
	}()

	L.SetContext(ctx)
	if err := L.DoString(code); err != nil {
		return fmt.Errorf("failed to preload script %q: %w", name, err)
	}
	e.logger.Debugf("Preloaded script: %s", name)
	return nil
}

// --- ScriptExecutor ---

// Execute 执行一个脚本字符串（name 用于诊断），返回结果。
func (e *LuaEngine) ExecuteString(ctx context.Context, name, code string) (any, error) {
	if !e.IsInitialized() {
		e.setLastError(ErrEngineNotInitialized)
		return nil, ErrEngineNotInitialized
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, e.config.VMTimeout)
	defer cancel()

	L := e.pool.Get()
	defer func() {
		if !e.isDedicated(L) {
			e.pool.Put(L)
		}
	}()

	errChan := make(chan error, 1)
	go func() {
		L.SetContext(timeoutCtx)
		if err := L.DoString(code); err != nil {
			errChan <- err
			return
		}
		errChan <- nil
	}()

	select {
	case err := <-errChan:
		if err != nil {
			e.setLastError(err)
			return nil, err
		}
		return nil, nil
	case <-timeoutCtx.Done():
		// 超时销毁损坏的 VM，创建新 VM 补位
		L.Close()
		newL := e.createVM()
		e.pool.Replace(L, newL)
		err := fmt.Errorf("script %q execution timeout after %s", name, e.config.VMTimeout)
		e.setLastError(err)
		return nil, err
	}
}

// Execute 执行此前 Load 的脚本。因 gopher-lua 单脚本语义，此处重新执行入口。
func (e *LuaEngine) Execute(ctx context.Context) (any, error) {
	return nil, fmt.Errorf("lua engine: Execute requires a source key; use ExecuteFromKey or ExecuteString")
}

// ExecuteFromKey 从 Source 加载并立即执行 key 对应的脚本。
func (e *LuaEngine) ExecuteFromKey(ctx context.Context, key string) (any, error) {
	if !e.IsInitialized() {
		e.setLastError(ErrEngineNotInitialized)
		return nil, ErrEngineNotInitialized
	}
	src := e.GetSource()
	if src == nil {
		err := fmt.Errorf("lua engine: no source bound")
		e.setLastError(err)
		return nil, err
	}
	code, err := src.Load(ctx, key)
	if err != nil {
		e.setLastError(err)
		return nil, err
	}
	return e.ExecuteString(ctx, key, code)
}

// ExecuteFromKeys 批量执行，结果顺序与 keys 一致。
func (e *LuaEngine) ExecuteFromKeys(ctx context.Context, keys []string) ([]any, error) {
	results := make([]any, 0, len(keys))
	for _, k := range keys {
		res, err := e.ExecuteFromKey(ctx, k)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	return results, nil
}

// --- GlobalAccessor ---

// RegisterGlobal 注册全局变量。
// 记录到全局注册表，后续新建 VM 会自动重放；同时应用到当前池中所有 VM。
func (e *LuaEngine) RegisterGlobal(name string, value any) error {
	if !e.IsInitialized() {
		e.setLastError(ErrEngineNotInitialized)
		return ErrEngineNotInitialized
	}
	e.regMu.Lock()
	e.globalVars[name] = value
	e.regMu.Unlock()

	e.applyToAllPoolVMs(func(L *lua.LState) {
		L.SetGlobal(name, toLValue(L, value))
	})
	return nil
}

// GetGlobal 读取全局变量。
func (e *LuaEngine) GetGlobal(name string) (any, error) {
	if !e.IsInitialized() {
		e.setLastError(ErrEngineNotInitialized)
		return nil, ErrEngineNotInitialized
	}
	L := e.pool.Get()
	defer e.pool.Put(L)
	return toGoValue(L.GetGlobal(name)), nil
}

// --- FunctionRegistrar ---

// RegisterFunction 注册宿主函数。fn 必须为 lua.LGFunction 类型。
// 记录到全局注册表，后续新建 VM 会自动重放；同时应用到当前池中所有 VM。
func (e *LuaEngine) RegisterFunction(name string, fn any) error {
	if !e.IsInitialized() {
		e.setLastError(ErrEngineNotInitialized)
		return ErrEngineNotInitialized
	}
	lf, ok := fn.(lua.LGFunction)
	if !ok {
		err := fmt.Errorf("function must be of type lua.LGFunction, got %T", fn)
		e.setLastError(err)
		return err
	}
	e.regMu.Lock()
	e.globalFuncs[name] = lf
	e.regMu.Unlock()

	e.applyToAllPoolVMs(func(L *lua.LState) {
		L.SetGlobal(name, L.NewFunction(lf))
	})
	return nil
}

// CallFunction 调用脚本中的命名函数。
func (e *LuaEngine) CallFunction(ctx context.Context, name string, args ...any) (any, error) {
	if !e.IsInitialized() {
		e.setLastError(ErrEngineNotInitialized)
		return nil, ErrEngineNotInitialized
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, e.config.VMTimeout)
	defer cancel()

	type result struct {
		val any
		err error
	}
	done := make(chan result, 1)

	L := e.pool.Get()
	go func() {
		defer func() {
			if !e.isDedicated(L) {
				e.pool.Put(L)
			}
		}()

		var lArgs []lua.LValue
		for _, a := range args {
			lArgs = append(lArgs, toLValue(L, a))
		}
		if err := L.CallByParam(lua.P{
			Fn:      L.GetGlobal(name),
			NRet:    1,
			Protect: true,
		}, lArgs...); err != nil {
			done <- result{nil, err}
			return
		}
		ret := L.Get(-1)
		L.Pop(1)
		done <- result{toGoValue(ret), nil}
	}()

	select {
	case <-timeoutCtx.Done():
		L.Close()
		newL := e.createVM()
		e.pool.Replace(L, newL)
		err := timeoutCtx.Err()
		e.setLastError(err)
		return nil, err
	case r := <-done:
		if r.err != nil {
			e.setLastError(r.err)
		}
		return r.val, r.err
	}
}

// --- ModuleRegistrar ---

// RegisterModule 注册 Lua 模块。module 必须为 lua.LGFunction（loader）。
// 记录到全局注册表，后续新建 VM 会自动重放；同时应用到当前池中所有 VM。
func (e *LuaEngine) RegisterModule(name string, module any) error {
	if !e.IsInitialized() {
		e.setLastError(ErrEngineNotInitialized)
		return ErrEngineNotInitialized
	}
	mod, ok := module.(lua.LGFunction)
	if !ok {
		err := fmt.Errorf("module must be of type lua.LGFunction, got %T", module)
		e.setLastError(err)
		return err
	}
	e.regMu.Lock()
	e.globalModules[name] = mod
	e.regMu.Unlock()

	e.applyToAllPoolVMs(func(L *lua.LState) {
		L.PreloadModule(name, mod)
	})
	return nil
}

// --- ScriptWatcher ---

// StartWatch 对 key 启动热更新监听。变更时重新 Load。
func (e *LuaEngine) StartWatch(ctx context.Context, key string) error {
	if !e.IsInitialized() {
		e.setLastError(ErrEngineNotInitialized)
		return ErrEngineNotInitialized
	}
	src := e.GetSource()
	if src == nil {
		err := fmt.Errorf("lua engine: no source bound")
		e.setLastError(err)
		return err
	}
	watcher, ok := src.(gsSource.Watcher)
	if !ok {
		err := fmt.Errorf("lua engine: source %T does not implement Watcher", src)
		e.setLastError(err)
		return err
	}

	e.StopWatch(key)

	watchCtx, cancel := context.WithCancel(ctx)
	e.watchersMu.Lock()
	e.watchers[key] = cancel
	e.watchersMu.Unlock()

	go func() {
		ch, err := watcher.Watch(watchCtx, key)
		if err != nil {
			cancel()
			return
		}
		for range ch {
			if loadErr := e.Load(watchCtx, key); loadErr != nil {
				e.logger.Errorf("hot reload failed for %q: %v", key, loadErr)
			} else {
				e.logger.Infof("hot reloaded script: %s", key)
			}
		}
	}()

	return nil
}

// StopWatch 停止 key 的热更新监听。
func (e *LuaEngine) StopWatch(key string) error {
	e.watchersMu.Lock()
	defer e.watchersMu.Unlock()
	if cancel, ok := e.watchers[key]; ok {
		cancel()
		delete(e.watchers, key)
	}
	return nil
}

func (e *LuaEngine) stopAllWatchers() {
	e.watchersMu.Lock()
	defer e.watchersMu.Unlock()
	for key, cancel := range e.watchers {
		cancel()
		delete(e.watchers, key)
	}
}

// ExecuteWithResult 执行脚本并捕获其返回值（供 Hook 编排器使用）。
// 返回 (returnValue, error)：脚本返回 false 时 returnValue 为 false 表示中止。
func (e *LuaEngine) ExecuteWithResult(ctx context.Context, name, code string, argTable func(L *lua.LState) lua.LValue) (any, error) {
	if !e.IsInitialized() {
		e.setLastError(ErrEngineNotInitialized)
		return nil, ErrEngineNotInitialized
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, e.config.VMTimeout)
	defer cancel()

	L := e.pool.Get()

	type result struct {
		val any
		err error
	}
	errChan := make(chan result, 1)

	go func() {
		L.SetContext(timeoutCtx)

		if err := L.DoString(code); err != nil {
			errChan <- result{nil, fmt.Errorf("script %q execution error: %w", name, err)}
			return
		}

		// 若存在 execute 函数则调用，传入 argTable
		if fn := L.GetGlobal("execute"); fn.Type() == lua.LTFunction {
			L.Push(fn)
			nArg := 0
			if argTable != nil {
				L.Push(argTable(L))
				nArg = 1
			}
			if err := L.PCall(nArg, 1, nil); err != nil {
				errChan <- result{nil, fmt.Errorf("execute function error: %w", err)}
				return
			}
			ret := L.Get(-1)
			L.Pop(1)
			errChan <- result{toGoValue(ret), nil}
			return
		}

		errChan <- result{nil, nil}
	}()

	select {
	case r := <-errChan:
		if !e.isDedicated(L) {
			e.pool.Put(L)
		}
		if r.err != nil {
			e.setLastError(r.err)
		}
		return r.val, r.err
	case <-timeoutCtx.Done():
		L.Close()
		newL := e.createVM()
		e.pool.Replace(L, newL)
		err := fmt.Errorf("script %q execution timeout after %s", name, e.config.VMTimeout)
		e.setLastError(err)
		return nil, err
	}
}

// GetVM 从池获取一个 VM（供编排器执行回调时使用）。
func (e *LuaEngine) GetVM() *lua.LState {
	return e.pool.Get()
}

// PutVM 归还 VM（若是专用 VM 则不归还）。
func (e *LuaEngine) PutVM(L *lua.LState) {
	if !e.isDedicated(L) {
		e.pool.Put(L)
	}
}

// ReplaceVM 销毁损坏的 VM 并补位（超时/异常后使用）。
func (e *LuaEngine) ReplaceVM(old *lua.LState) {
	newL := e.createVM()
	e.pool.Replace(old, newL)
}

// Config 返回引擎配置。
func (e *LuaEngine) Config() *Config { return e.config }

// Logger 返回日志器。
func (e *LuaEngine) Logger() *log.Helper { return e.logger }
