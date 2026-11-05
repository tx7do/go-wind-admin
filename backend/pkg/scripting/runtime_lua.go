package scripting

import (
	"context"
	"fmt"
	"time"

	lua "github.com/yuin/gopher-lua"

	gsEngine "github.com/tx7do/go-scripts"
	"github.com/tx7do/go-scripts/hostmodule"
	gsLua "github.com/tx7do/go-scripts/lua"
	"github.com/tx7do/go-scripts/lua/convert"
	"github.com/tx7do/go-scripts/lua/host"

	"go-wind-admin/pkg/scripting/api"
)

// 本文件是 Lua 语言适配器，实现 runtime.go 中定义的语言无关接口。
// 迁移自原 engine_runtime.go + engine.go 的 Lua 专属逻辑。

func init() {
	RegisterBinder(&LuaBinder{})
}

// LuaBinder 实现 RuntimeBinder，负责将业务 API 注入 Lua VM。
type LuaBinder struct {
	holder *execCtxHolder // 执行上下文持有器（Bind 时由编排器注入）
	cfg    *Config
}

// Type 返回支持的引擎类型。
func (b *LuaBinder) Type() gsEngine.Type { return gsEngine.LuaType }

// PreInit 在引擎 Init 前配置沙箱（必须在 VM 创建前生效）。
func (b *LuaBinder) PreInit(eng gsEngine.Engine) {
	type openLibsSetter interface {
		SetOpenLibs(libs ...string)
	}
	if setter, ok := eng.(openLibsSetter); ok {
		setter.SetOpenLibs(
			gsLua.AllowedLibBase,
			gsLua.AllowedLibLoad,
			gsLua.AllowedLibTab,
			gsLua.AllowedLibStr,
			gsLua.AllowedLibMath,
			gsLua.AllowedLibCoroutine,
		)
	}
}

// WithContext 创建一个绑定了执行上下文持有器的新 LuaBinder 实例。
// 返回新实例而非修改单例，避免多引擎实例间的 holder 冲突。
func (b *LuaBinder) WithContext(holder *execCtxHolder, cfg *Config) RuntimeBinder {
	return &LuaBinder{
		holder: holder,
		cfg:    cfg,
	}
}

// httpOpts 从编排器配置取 http 护栏（binder 无 cfg 时 fail-closed 空白名单）。
func (b *LuaBinder) httpOpts() hostmodule.HTTPOptions {
	if b.cfg == nil {
		return hostmodule.HTTPOptions{}
	}
	return b.cfg.HTTPOptions
}

// Bind 在 Lua VM 上注入全部业务模块与执行上下文函数。
// 沙箱配置在 PreInit 中完成（必须在 VM 创建前生效）。
func (b *LuaBinder) Bind(eng gsEngine.Engine, deps *RuntimeDeps) error {
	// sleep 上限 = 单脚本执行超时（防御脚本用 sleep 绕过超时控制）
	sleepCap := 5 * time.Second
	if b.cfg != nil && b.cfg.VMTimeout > 0 {
		sleepCap = b.cfg.VMTimeout
	}

	// 注入业务模块（经 preloadAdapter 适配为 go-scripts 期望的 preload 风格）
	registrations := []struct {
		name    string
		builder lua.LGFunction
	}{
		{"kratos_logger", host.LoaderLogger(deps.Logger)},
		// P1 修复：用受限版替代库版 crypto——库版把全局主密钥的 decrypt 原语暴露给
		// 脚本（同一把钥匙加密着 AI key/SMTP 密码/MFA secret），见 api/crypto_safe.go。
		{"kratos_crypto", api.LoaderCryptoSafe(deps.Logger)},
		{"kratos_util", host.LoaderUtil(deps.Logger, sleepCap)},
		{"kratos_http", host.LoaderHTTP(b.httpOpts())},
	}

	// http 模块即便未配置白名单也注册（fail-closed：脚本调用即收到明确的白名单错误）

	if deps.Rdb != nil {
		registrations = append(registrations, struct {
			name    string
			builder lua.LGFunction
		}{"kratos_cache", api.LoaderCache(deps.Rdb, deps.Logger)})
	}
	if deps.EventBusManager != nil {
		registrations = append(registrations, struct {
			name    string
			builder lua.LGFunction
		}{"kratos_eventbus", api.LoaderEventBus(deps.EventBusManager, deps.Logger)})
	}
	if deps.OSSClient != nil {
		registrations = append(registrations, struct {
			name    string
			builder lua.LGFunction
		}{"kratos_oss", api.LoaderOSS(deps.OSSClient, deps.Logger)})
	}
	if deps.AI != nil {
		registrations = append(registrations, struct {
			name    string
			builder lua.LGFunction
		}{"kratos_ai", api.LoaderAI(deps.AI, deps.Logger)})
	}

	// hook 模块（脚本自注册 hook 回调）
	hookAdapter := &luaHookAdapter{orchestrator: deps.Orchestrator}
	registrations = append(registrations, struct {
		name    string
		builder lua.LGFunction
	}{"kratos_hook", host.LoaderHook(hookAdapter, deps.Logger)})

	// task 模块
	registrations = append(registrations, struct {
		name    string
		builder lua.LGFunction
	}{"task", api.LoaderTask(luaVMManagerAdapter{}, deps.Logger)})

	for _, reg := range registrations {
		if err := eng.RegisterModule(reg.name, luaPreloadAdapter(reg.builder)); err != nil {
			return err
		}
	}

	// 注册执行上下文访问函数
	b.registerContextFunctions(eng, deps.Orchestrator)

	return nil
}

// registerContextFunctions 注册执行上下文的全局便捷函数。
// 脚本可调用 __get_ctx() / __set_ctx(k,v) / __stop(reason)。
func (b *LuaBinder) registerContextFunctions(eng gsEngine.Engine, orch *Engine) {
	var (
		getCtx lua.LGFunction = func(L *lua.LState) int {
			ctx := b.holder.get()
			if ctx == nil {
				L.Push(L.NewTable())
				return 1
			}
			L.Push(contextToLuaTable(L, ctx))
			return 1
		}
		setCtx lua.LGFunction = func(L *lua.LState) int {
			ctx := b.holder.get()
			if ctx != nil {
				key := L.CheckString(1)
				val := L.Get(2)
				ctx.Data[key] = convert.ToGoValue(val)
			}
			return 0
		}
		stopCtx lua.LGFunction = func(L *lua.LState) int {
			ctx := b.holder.get()
			if ctx != nil {
				ctx.Stopped = true
				ctx.StopReason = L.OptString(1, "stopped by script")
			}
			return 0
		}
	)

	_ = eng.RegisterFunction("__get_ctx", getCtx)
	_ = eng.RegisterFunction("__set_ctx", setCtx)
	_ = eng.RegisterFunction("__stop", stopCtx)
}

// contextToLuaTable 将 Context 转为 Lua table（含 get/set/stop 方法）。
func contextToLuaTable(L *lua.LState, ctx *Context) *lua.LTable {
	table := L.NewTable()

	table.RawSetString("get", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		if ctx != nil {
			if val, ok := ctx.Data[key]; ok {
				L.Push(convert.ToLuaValue(L, val))
				return 1
			}
		}
		L.Push(lua.LNil)
		return 1
	}))

	table.RawSetString("set", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		val := L.Get(2)
		if ctx != nil {
			ctx.Data[key] = convert.ToGoValue(val)
		}
		return 0
	}))

	table.RawSetString("stop", L.NewFunction(func(L *lua.LState) int {
		reason := L.OptString(1, "stopped by script")
		if ctx != nil {
			ctx.Stopped = true
			ctx.StopReason = reason
		}
		return 0
	}))

	return table
}

// luaPreloadAdapter 将 builder 风格的 loader 适配为 go-scripts/lua 的 preload 风格。
func luaPreloadAdapter(builder lua.LGFunction) lua.LGFunction {
	return func(L *lua.LState) int {
		name := L.CheckString(1)
		L.PreloadModule(name, builder)
		return 0
	}
}

// --- ScriptCallback 的 Lua 实现 ---

// luaCallback 封装 Lua 回调函数，实现 ScriptCallback 接口。
type luaCallback struct {
	L        *lua.LState
	Function *lua.LFunction
	hookName string
	owner    string // 注册时正在执行的脚本名（热更新幂等替换的依据）
	cfg      *Config
	holder   *execCtxHolder
}

// 确保 luaCallback 实现 ScriptCallback。
var _ ScriptCallback = (*luaCallback)(nil)

// CallbackOwner 返回回调归属（脚本名），空表示匿名回调（不去重）。
func (c *luaCallback) CallbackOwner() string { return c.owner }

// Source 返回回调来源信息。
func (c *luaCallback) Source() string {
	return fmt.Sprintf("lua:hook=%s", c.hookName)
}

// Call 执行 Lua 回调函数，传入执行上下文 table。
func (c *luaCallback) Call(ctx context.Context, execCtx *Context) (any, error) {
	L := c.L

	timeoutCtx, cancel := context.WithTimeout(ctx, c.cfg.VMTimeout)
	defer cancel()

	// 设置执行上下文
	prev := c.holder.set(execCtx)
	defer c.holder.reset(prev)

	errChan := make(chan error, 1)
	go func() {
		L.SetContext(timeoutCtx)
		defer L.SetContext(context.Background()) // 重置 context，避免 LState 残留已取消的 ctx

		L.Push(c.Function)
		L.Push(contextToLuaTable(L, execCtx))

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
		return nil, err
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("callback execution timeout after %s", c.cfg.VMTimeout)
	}
}

// --- 适配器（供 api 包的 Lua 专用接口使用） ---

// luaHookAdapter 将编排器适配为 go-scripts/lua/host 包的 HookEngine 接口。
type luaHookAdapter struct {
	orchestrator *Engine
}

var _ host.HookEngine = (*luaHookAdapter)(nil)

func (a *luaHookAdapter) RegisterHook(name, description string) error {
	return a.orchestrator.RegisterHook(name, description)
}

func (a *luaHookAdapter) AddScript(hookName string, script interface{}) error {
	return a.orchestrator.AddScript(hookName, script)
}

func (a *luaHookAdapter) ListHooks() []string {
	return a.orchestrator.ListHooks()
}

// RegisterCallback 捕获 Lua 回调函数，创建 luaCallback 并注册到编排器。
// 归属（owner）取注册时刻正在执行的脚本名——脚本热更新重执行
// hook.register 时，同归属旧回调会被替换而非追加。
func (a *luaHookAdapter) RegisterCallback(hookName string, L *lua.LState, fn *lua.LFunction) {
	cb := &luaCallback{
		L:        L,
		Function: fn,
		hookName: hookName,
		owner:    a.orchestrator.execCtx.currentScript(),
		cfg:      a.orchestrator.config,
		holder:   &a.orchestrator.execCtx,
	}
	a.orchestrator.registerCallback(hookName, cb)
}

// luaVMManagerAdapter 适配 api.VMManager（go-scripts 池自动管理，no-op）。
type luaVMManagerAdapter struct{}

func (luaVMManagerAdapter) MarkVMDedicated(L *lua.LState) {
	// go-scripts/lua 引擎的 statePool 管理 VM 生命周期，此处无需处理。
}
