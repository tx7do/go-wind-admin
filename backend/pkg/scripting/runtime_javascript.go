package scripting

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	gsEngine "github.com/tx7do/go-scripts"
	"github.com/tx7do/go-scripts/hostmodule"

	"go-wind-admin/pkg/scripting/api"
)

// 本文件是 JavaScript（goja）语言适配器，实现 runtime.go 中的语言无关接口。
//
// goja 的 RegisterFunction(name, fn) 直接把 Go 函数设为 JS 全局变量，
// goja 自动通过反射把 Go 函数包装为 JS 可调用对象。
// 因此业务模块注入只需 RegisterModule(name, map[string]any) 即可。

// js 模块名常量（JS 全局对象名，与 Lua 的 require 模块名对应）。
const (
	jsModuleLog      = "log"
	jsModuleCrypto   = "crypto"
	jsModuleUtil     = "util"
	jsModuleCache    = "cache"
	jsModuleHook     = "hook"
	jsModuleTask     = "task"
	jsModuleEventBus = "eventbus"
	jsModuleOSS      = "oss"
	jsModuleAI       = "ai"
)

func init() {
	RegisterBinder(&JSBinder{})
}

// JSBinder 实现 RuntimeBinder，负责将业务 API 注入 goja JS 引擎。
type JSBinder struct {
	holder *execCtxHolder
	cfg    *Config
}

// Type 返回支持的引擎类型。
func (b *JSBinder) Type() gsEngine.Type { return gsEngine.JavaScriptType }

// WithContext 创建一个绑定了执行上下文持有器的新 JSBinder 实例。
func (b *JSBinder) WithContext(holder *execCtxHolder, cfg *Config) RuntimeBinder {
	return &JSBinder{
		holder: holder,
		cfg:    cfg,
	}
}

// httpOpts 从编排器配置取 http 护栏（binder 无 cfg 时 fail-closed 空白名单）。
func (b *JSBinder) httpOpts() hostmodule.HTTPOptions {
	if b.cfg == nil {
		return hostmodule.HTTPOptions{}
	}
	return b.cfg.HTTPOptions
}

// Bind 在 JS 引擎上注入全部业务模块与执行上下文函数。
func (b *JSBinder) Bind(eng gsEngine.Engine, deps *RuntimeDeps) error {
	// sleep 上限 = 单脚本执行超时（防御脚本用 sleep 绕过超时控制）
	sleepCap := 5 * time.Second
	if b.cfg != nil && b.cfg.VMTimeout > 0 {
		sleepCap = b.cfg.VMTimeout
	}

	// 注入无依赖模块
	modules := []hostmodule.ModuleDef{
		hostmodule.ModuleLogger(deps.Logger),
		// P1 修复：用受限版替代库版 crypto——库版把全局主密钥的 decrypt 原语暴露给
		// 脚本（同一把钥匙加密着 AI key/SMTP 密码/MFA secret），见 api/crypto_safe.go。
		{Name: "crypto", Funcs: api.ModuleCryptoSafe()},
		hostmodule.ModuleUtil(sleepCap),
		hostmodule.ModuleHTTP(b.httpOpts()), // fail-closed：未配置白名单时调用即报错
	}
	for _, m := range modules {
		if err := eng.RegisterModule(m.Name, m.Funcs); err != nil {
			return fmt.Errorf("js bind module %s: %w", m.Name, err)
		}
	}

	// 注入有依赖模块（真实现：与 Lua 侧 kratos_cache / kratos_eventbus / kratos_oss 功能对齐）
	if deps.Rdb != nil {
		if err := eng.RegisterModule(jsModuleCache, api.ModuleCache(deps.Rdb, deps.Logger).Funcs); err != nil {
			return fmt.Errorf("js bind module %s: %w", jsModuleCache, err)
		}
	}
	if deps.EventBusManager != nil {
		eventBusFuncs := api.ModuleEventBus(eng, deps.EventBusManager, deps.Logger).Funcs
		if err := eng.RegisterModule(jsModuleEventBus, eventBusFuncs); err != nil {
			return fmt.Errorf("js bind module %s: %w", jsModuleEventBus, err)
		}
	}
	if deps.OSSClient != nil {
		if err := eng.RegisterModule(jsModuleOSS, api.ModuleOSS(deps.OSSClient, deps.Logger).Funcs); err != nil {
			return fmt.Errorf("js bind module %s: %w", jsModuleOSS, err)
		}
	}
	if deps.AI != nil {
		if err := eng.RegisterModule(jsModuleAI, api.ModuleAI(deps.AI, deps.Logger).Funcs); err != nil {
			return fmt.Errorf("js bind module %s: %w", jsModuleAI, err)
		}
	}

	// task 模块：JS 侧 in-script 任务注册 API（对齐 Lua 的 task.register_handler）尚未实现，
	// JS 脚本暂无法注册任务处理器。asynq 任务桥与执行链路本身已通
	// （internal/service ScriptRuntime.RunScriptTaskHandler → Engine.ExecuteTaskHandler，Lua 处理器可注册可执行）；
	// 跨语言契约对齐时用真实实现替换此占位。
	_ = eng.RegisterModule(jsModuleTask, map[string]any{})

	// 注册执行上下文全局函数（goja 自动桥接 Go 函数）
	if err := b.registerContextFunctions(eng); err != nil {
		return err
	}

	// 注册 hook.register（捕获 JS 回调函数引用）
	if err := b.registerHookModule(eng, deps); err != nil {
		return err
	}

	return nil
}

// registerContextFunctions 注册 __get_ctx / __set_ctx / __stop 全局函数。
// goja 自动把 Go 函数桥接为 JS 可调用。
func (b *JSBinder) registerContextFunctions(eng gsEngine.Engine) error {
	// __get_ctx() 返回 {get, set, stop} 对象
	if err := eng.RegisterFunction("__get_ctx", func() map[string]any {
		ctx := b.holder.get()
		if ctx == nil {
			return map[string]any{}
		}
		return jsContextObject(ctx)
	}); err != nil {
		return err
	}

	// __set_ctx(key, value)
	if err := eng.RegisterFunction("__set_ctx", func(key string, val any) {
		ctx := b.holder.get()
		if ctx != nil {
			ctx.Data[key] = val
		}
	}); err != nil {
		return err
	}

	// __stop(reason)
	if err := eng.RegisterFunction("__stop", func(reason ...string) {
		ctx := b.holder.get()
		if ctx != nil {
			ctx.Stopped = true
			r := "stopped by script"
			if len(reason) > 0 {
				r = reason[0]
			}
			ctx.StopReason = r
		}
	}); err != nil {
		return err
	}

	return nil
}

// registerHookModule 注册 hook.register 全局函数。
//
// JS 脚本调用 hook.register(name, desc, fn) 时，fn 是 goja 的 JS 函数。
// 由于 go-scripts/js 引擎的 RegisterFunction 把 Go 函数设为全局，
// goja 会把 JS 函数参数传给 Go 函数的参数（类型为 goja.Value 或 func）。
//
// 策略：Go 函数接收 fn 后，将其注册为一个唯一命名的全局函数（__hook_cb_N），
// 然后创建 jsCallback（持有引擎引用 + 函数名）注册到编排器。
func (b *JSBinder) registerHookModule(eng gsEngine.Engine, deps *RuntimeDeps) error {
	hookObj := map[string]any{
		"register": func(name string, descAndFn ...any) {
			var desc string
			if len(descAndFn) > 0 {
				if s, ok := descAndFn[0].(string); ok {
					desc = s
				}
			}
			_ = deps.Orchestrator.RegisterHook(name, desc)

			// 捕获回调：若有函数参数，暂存原始 JS 函数值（不调用引擎 API，
			// 避免 goja execMu 重入死锁）。调用时再经 RegisterGlobal + CallFunction 执行。
			if len(descAndFn) > 1 {
				cbName := fmt.Sprintf("__hook_cb_%d", atomic.AddInt64(&jsCallbackCounter, 1))
				cb := &jsCallback{
					engine:    eng,
					funcName:  cbName,
					funcValue: descAndFn[1], // 原始 goja 函数值，调用时注册为全局
					hookName:  name,
					owner:     b.holder.currentScript(),
					cfg:       b.cfg,
					holder:    b.holder,
				}
				deps.Orchestrator.registerCallback(name, cb)
			}
		},
		"list": func() []string {
			return deps.Orchestrator.ListHooks()
		},
	}
	return eng.RegisterModule(jsModuleHook, hookObj)
}

// jsContextObject 构建 JS 侧的上下文对象 {get, set, stop}。
func jsContextObject(ctx *Context) map[string]any {
	return map[string]any{
		"get": func(key string) any {
			if val, ok := ctx.Data[key]; ok {
				return val
			}
			return nil
		},
		"set": func(key string, val any) {
			ctx.Data[key] = val
		},
		"stop": func(reason ...string) {
			ctx.Stopped = true
			r := "stopped by script"
			if len(reason) > 0 {
				r = reason[0]
			}
			ctx.StopReason = r
		},
	}
}

// --- ScriptCallback 的 JS 实现 ---

// jsCallback 封装 JS 回调函数，实现 ScriptCallback 接口。
//
// funcValue 是脚本注册时捕获的原始 goja 函数值（任意类型）。
// 调用时通过 RegisterGlobal 设为全局命名函数，再经 CallFunction 执行。
// 注册阶段（脚本执行中）不调用引擎 API，避免 goja execMu 重入死锁；
// 调用阶段（ExecuteHook 中）无正在进行的 ExecuteString，可安全获取锁。
type jsCallback struct {
	engine    gsEngine.Engine
	funcName  string // 回调函数的全局名（__hook_cb_N）
	funcValue any    // 原始 JS 函数值（注册时捕获，调用时注册为全局）
	hookName  string
	owner     string // 注册时正在执行的脚本名（热更新幂等替换的依据）
	cfg       *Config
	holder    *execCtxHolder
}

var _ ScriptCallback = (*jsCallback)(nil)

// CallbackOwner 返回回调归属（脚本名），空表示匿名回调（不去重）。
func (c *jsCallback) CallbackOwner() string { return c.owner }

// Source 返回回调来源信息。
func (c *jsCallback) Source() string {
	return fmt.Sprintf("js:hook=%s", c.hookName)
}

// Call 通过引擎的 CallFunction 调用 JS 回调，传入上下文对象。
func (c *jsCallback) Call(ctx context.Context, execCtx *Context) (any, error) {
	prev := c.holder.set(execCtx)
	defer c.holder.reset(prev)

	// 调用阶段：把暂存的 JS 函数值注册为全局命名函数（此时无 execMu 重入）
	if err := c.engine.RegisterGlobal(c.funcName, c.funcValue); err != nil {
		return nil, fmt.Errorf("js callback register: %w", err)
	}

	ctxObj := jsContextObject(execCtx)
	result, err := c.engine.CallFunction(ctx, c.funcName, ctxObj)
	if err != nil {
		return nil, fmt.Errorf("js callback error: %w", err)
	}

	// 返回 false 表示中止
	if b, ok := result.(bool); ok && !b {
		return nil, fmt.Errorf("callback returned false")
	}
	return result, nil
}

// jsCallbackCounter 用于生成唯一回调函数名。
var jsCallbackCounter int64
