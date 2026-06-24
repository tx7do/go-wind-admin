package scripting

import (
	"context"
	"fmt"
	"sync/atomic"

	gsEngine "github.com/tx7do/go-scripts"

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

// Bind 在 JS 引擎上注入全部业务模块与执行上下文函数。
func (b *JSBinder) Bind(eng gsEngine.Engine, deps *RuntimeDeps) error {
	// 注入无依赖模块
	modules := []api.ModuleDef{
		api.ModuleLogger(deps.Logger),
		api.ModuleCrypto(),
		api.ModuleUtil(),
	}
	for _, m := range modules {
		if err := eng.RegisterModule(m.Name, m.Funcs); err != nil {
			return fmt.Errorf("js bind module %s: %w", m.Name, err)
		}
	}

	// 注入有依赖模块（当前用空 map 占位，后续扩展 ModuleDef）
	if deps.Rdb != nil {
		_ = eng.RegisterModule(jsModuleCache, map[string]any{})
	}
	if deps.EventBusManager != nil {
		_ = eng.RegisterModule(jsModuleEventBus, map[string]any{})
	}
	if deps.OSSClient != nil {
		_ = eng.RegisterModule(jsModuleOSS, map[string]any{})
	}

	// task 模块
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
	cfg       *Config
	holder    *execCtxHolder
}

var _ ScriptCallback = (*jsCallback)(nil)

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
