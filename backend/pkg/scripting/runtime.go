package scripting

import (
	"context"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"

	gsEngine "github.com/tx7do/go-scripts"

	"go-wind-admin/pkg/eventbus"
	"go-wind-admin/pkg/oss"
)

// 本文件定义语言无关的运行时抽象层。
//
// 编排器（engine.go）面向这些接口编程，不直接依赖任何脚本语言的类型
// （如 gopher-lua 的 *lua.LState）。每门语言实现自己的 RuntimeBinder
// 和 ScriptCallback，即可接入编排器。
//
// 当前实现：
//   - Lua（runtime_lua.go）：LuaBinder + luaCallback
//   - JavaScript（runtime_js.go）：JSBinder + jsCallback

// ScriptCallback 是语言无关的脚本回调。
//
// 替代旧的 CallbackInfo{L *lua.LState, Function *lua.LFunction}。
// 由脚本通过 hook.register(name, fn) 注册，各语言适配器从自己的
// 函数引用类型（lua.LFunction / goja.Value）构造实现。
type ScriptCallback interface {
	// Call 在脚本引擎中调用此回调，传入执行上下文。
	// 返回 (result, error)：result 为 bool 且为 false 表示中止。
	Call(ctx context.Context, execCtx *Context) (any, error)
	// Source 返回回调来源信息（用于日志诊断）。
	Source() string
}

// RuntimeDeps 聚合编排器持有的业务依赖，传给 RuntimeBinder.Bind。
// 各语言适配器根据这些依赖在 VM 上注册对应的业务模块。
type RuntimeDeps struct {
	Logger          *log.Helper
	Rdb             *redis.Client     // 可为 nil，nil 时跳过 cache 模块
	EventBusManager *eventbus.Manager // 可为 nil，nil 时跳过 eventbus 模块
	OSSClient       *oss.MinIOClient  // 可为 nil，nil 时跳过 oss 模块
	Orchestrator    *Engine           // 用于 hook.register 反向回调注册
}

// RuntimeBinder 负责将业务 API、执行上下文函数、hook.register 等
// 注入到特定语言的 VM 中。每门语言实现一个。
//
// 生命周期：在 go-scripts 的 RuntimeHook 内被调用（Init 后、Load/Execute 前），
// 池化引擎复用 VM 时会重放。
type RuntimeBinder interface {
	// Type 返回此 binder 支持的 go-scripts 引擎类型（如 LuaType / JavaScriptType）。
	Type() gsEngine.Type
	// Bind 在引擎 VM 上注入全部业务依赖。
	// 在 RuntimeHook 回调内被调用。
	Bind(eng gsEngine.Engine, deps *RuntimeDeps) error
}

// execCtxHolder 在执行期间持有当前执行上下文。
// 并发安全：同一编排器实例串行执行 Hook，执行前 Set / 执行后 Reset。
// 供脚本通过 __get_ctx / __set_ctx / __stop 等全局函数访问。
type execCtxHolder struct {
	current *Context
	mu      sync.Mutex
}

// set 设置当前执行上下文，返回上一个（用于恢复）。
func (h *execCtxHolder) set(ctx *Context) *Context {
	h.mu.Lock()
	defer h.mu.Unlock()
	prev := h.current
	h.current = ctx
	return prev
}

// reset 恢复执行上下文。
func (h *execCtxHolder) reset(prev *Context) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.current = prev
}

// get 返回当前执行上下文（可能为 nil）。
func (h *execCtxHolder) get() *Context {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.current
}

// binderRegistry 注册所有语言适配器，按引擎类型查找。
var binderRegistry = struct {
	mu      sync.RWMutex
	binders map[gsEngine.Type]RuntimeBinder
}{
	binders: make(map[gsEngine.Type]RuntimeBinder),
}

// RegisterBinder 注册语言适配器。通常在各适配器文件的 init() 中调用。
func RegisterBinder(b RuntimeBinder) {
	binderRegistry.mu.Lock()
	defer binderRegistry.mu.Unlock()
	binderRegistry.binders[b.Type()] = b
}

// getBinder 按引擎类型查找适配器，找不到返回 nil。
func getBinder(typ gsEngine.Type) RuntimeBinder {
	binderRegistry.mu.RLock()
	defer binderRegistry.mu.RUnlock()
	return binderRegistry.binders[typ]
}
