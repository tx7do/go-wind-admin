package lua

import (
	"errors"

	lua "github.com/yuin/gopher-lua"

	"go-wind-admin/pkg/lua/api"
	"go-wind-admin/pkg/lua/internal/convert"
)

// 引擎生命周期哨兵错误（对齐 go-scripts/lua 的错误命名风格）。
var (
	// ErrEngineNotInitialized 操作在 Init 之前或 Close 之后调用时返回。
	ErrEngineNotInitialized = errors.New("lua engine not initialized")

	// ErrEngineAlreadyInitialized 对已初始化的引擎再次 Init 时返回。
	ErrEngineAlreadyInitialized = errors.New("lua engine already initialized")
)

// toLValue 将 Go 值转为 Lua 值（委托 internal/convert）。
func toLValue(L *lua.LState, val any) lua.LValue {
	return convert.ToLuaValue(L, val)
}

// toGoValue 将 Lua 值转为 Go 值（委托 internal/convert）。
func toGoValue(val lua.LValue) any {
	return convert.ToGoValue(val)
}

// hookLister 由 Hook 编排器实现，提供已注册 hook 名称列表。
type hookLister interface {
	ListHooks() []string
}

// hookEngineAdapter 将 HookRegistrar + hookLister 适配为 api.HookEngine 接口。
// 引擎在注册 hook API 时，Lua 脚本调用 hook.register(name, fn)，
// 经此适配器转发到 Hook 编排器。
type hookEngineAdapter struct {
	registrar HookRegistrar // 回调注册
	lister    hookLister    // hook 列表查询
}

// 确保 hookEngineAdapter 实现 api.HookEngine。
var _ api.HookEngine = (*hookEngineAdapter)(nil)

// RegisterHook 注册一个 hook 点（委托编排器）。
func (a *hookEngineAdapter) RegisterHook(name, description string) error {
	if rl, ok := a.registrar.(interface {
		RegisterHook(name, description string) error
	}); ok {
		return rl.RegisterHook(name, description)
	}
	return nil
}

// AddScript 向 hook 添加脚本（委托编排器）。
func (a *hookEngineAdapter) AddScript(hookName string, script interface{}) error {
	if al, ok := a.registrar.(interface {
		AddScript(hookName string, script interface{}) error
	}); ok {
		return al.AddScript(hookName, script)
	}
	return nil
}

// ListHooks 返回所有已注册 hook 名称（委托编排器）。
func (a *hookEngineAdapter) ListHooks() []string {
	if a.lister != nil {
		return a.lister.ListHooks()
	}
	return nil
}

// RegisterCallback 注册 Lua 回调函数（委托编排器）。
func (a *hookEngineAdapter) RegisterCallback(hookName string, L *lua.LState, fn *lua.LFunction) {
	a.registrar.RegisterCallback(hookName, L, fn)
}
