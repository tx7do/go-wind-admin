package api

// 本文件针对 hook.go（kratos_hook 模块）做单元测试：
//   - RegisterHookAPI / LoaderHook 经 require "kratos_hook" 加载；
//   - hook.register：带回调、无回调、非函数回调、引擎 RegisterHook 报错仍继续注册回调各分支；
//   - hook.add_script：完整字段、缺省字段（enabled/priority/description 默认值）、
//     priority 非数值、引擎 AddScript 报错（返回 false + 错误字符串）；
//   - hook.list：引擎钩子列表转 Lua 表；
//   - engine 为 nil 时 loader 返回空模块。
//
// 引擎侧使用记录型 fakeHookEngine 捕获全部调用参数供断言。

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	lua "github.com/yuin/gopher-lua"
)

// fakeRegisterCall 记录一次 RegisterHook 调用。
type fakeRegisterCall struct {
	name        string
	description string
}

// fakeAddScriptCall 记录一次 AddScript 调用。
type fakeAddScriptCall struct {
	hookName string
	script   Script
}

// fakeCallbackCall 记录一次 RegisterCallback 调用。
type fakeCallbackCall struct {
	hookName string
	L        *lua.LState
	fn       *lua.LFunction
}

// fakeHookEngine 记录型 HookEngine 假实现（测试专用）。
type fakeHookEngine struct {
	registerCalls  []fakeRegisterCall
	addScriptCalls []fakeAddScriptCall
	callbackCalls  []fakeCallbackCall
	listResult     []string
	registerErr    error
	addScriptErr   error
}

func (f *fakeHookEngine) RegisterHook(name, description string) error {
	f.registerCalls = append(f.registerCalls, fakeRegisterCall{name, description})
	return f.registerErr
}

func (f *fakeHookEngine) AddScript(hookName string, script interface{}) error {
	if s, ok := script.(Script); ok {
		f.addScriptCalls = append(f.addScriptCalls, fakeAddScriptCall{hookName, s})
	}
	return f.addScriptErr
}

func (f *fakeHookEngine) ListHooks() []string { return f.listResult }

func (f *fakeHookEngine) RegisterCallback(hookName string, L *lua.LState, fn *lua.LFunction) {
	f.callbackCalls = append(f.callbackCalls, fakeCallbackCall{hookName, L, fn})
}

// newHookLuaState 构造注册了 kratos_hook 模块的 LState。
func newHookLuaState(engine HookEngine) *lua.LState {
	L := lua.NewState()
	RegisterHookAPI(L, engine, bLogger.NewHelper(bLogger.NopLogger()))
	return L
}

// TestRegisterHookAPI_ModuleLoads 模块可加载且暴露 register/add_script/list 三个函数。
func TestRegisterHookAPI_ModuleLoads(t *testing.T) {
	L := newHookLuaState(&fakeHookEngine{})
	defer L.Close()

	err := L.DoString(`
		local h = require "kratos_hook"
		assert(type(h) == "table", "module should be a table")
		assert(h.register ~= nil, "register should exist")
		assert(h.add_script ~= nil, "add_script should exist")
		assert(h.list ~= nil, "list should exist")
	`)
	require.NoError(t, err)
}

// TestHookAPI_NilEngineYieldsEmptyModule engine 为 nil 时 loader 返回空模块。
func TestHookAPI_NilEngineYieldsEmptyModule(t *testing.T) {
	L := newHookLuaState(nil)
	defer L.Close()

	err := L.DoString(`
		local h = require "kratos_hook"
		assert(type(h) == "table", "module should still be a table")
		assert(h.register == nil, "empty module should have no register")
		assert(h.add_script == nil, "empty module should have no add_script")
		assert(h.list == nil, "empty module should have no list")
	`)
	require.NoError(t, err)
}

// TestHookAPI_RegisterWithCallback 带回调注册：引擎收到钩子注册与回调注册，
// Lua 侧返回 true。
func TestHookAPI_RegisterWithCallback(t *testing.T) {
	engine := &fakeHookEngine{}
	L := newHookLuaState(engine)
	defer L.Close()

	err := L.DoString(`
		local h = require "kratos_hook"
		local ok = h.register("myhook", "hook-desc", function() end)
		assert(ok == true, "register should return true")
	`)
	require.NoError(t, err)

	require.Len(t, engine.registerCalls, 1)
	require.Equal(t, "myhook", engine.registerCalls[0].name)
	require.Equal(t, "hook-desc", engine.registerCalls[0].description)
	require.Len(t, engine.callbackCalls, 1)
	require.Equal(t, "myhook", engine.callbackCalls[0].hookName)
	require.NotNil(t, engine.callbackCalls[0].L)
	require.NotNil(t, engine.callbackCalls[0].fn)
}

// TestHookAPI_RegisterWithoutCallback 仅名称注册：description 取默认空串，不注册回调。
func TestHookAPI_RegisterWithoutCallback(t *testing.T) {
	engine := &fakeHookEngine{}
	L := newHookLuaState(engine)
	defer L.Close()

	err := L.DoString(`
		local h = require "kratos_hook"
		local ok = h.register("plainhook")
		assert(ok == true, "register should return true")
	`)
	require.NoError(t, err)

	require.Len(t, engine.registerCalls, 1)
	require.Equal(t, "plainhook", engine.registerCalls[0].name)
	require.Equal(t, "", engine.registerCalls[0].description)
	require.Empty(t, engine.callbackCalls)
}

// TestHookAPI_RegisterWithNonFunctionCallback 第三参非函数：跳过回调注册，仍返回 true。
func TestHookAPI_RegisterWithNonFunctionCallback(t *testing.T) {
	engine := &fakeHookEngine{}
	L := newHookLuaState(engine)
	defer L.Close()

	err := L.DoString(`
		local h = require "kratos_hook"
		local ok = h.register("h3", "d3", 42)
		assert(ok == true, "register should still return true")
	`)
	require.NoError(t, err)

	require.Len(t, engine.registerCalls, 1)
	require.Equal(t, "h3", engine.registerCalls[0].name)
	require.Empty(t, engine.callbackCalls)
}

// TestHookAPI_RegisterEngineErrorStillRegistersCallback 引擎 RegisterHook 报错：
// 仅记录日志并继续注册回调，Lua 侧仍返回 true。
func TestHookAPI_RegisterEngineErrorStillRegistersCallback(t *testing.T) {
	engine := &fakeHookEngine{registerErr: errors.New("hook register failed")}
	L := newHookLuaState(engine)
	defer L.Close()

	err := L.DoString(`
		local h = require "kratos_hook"
		local ok = h.register("boom", "bd", function() end)
		assert(ok == true, "register should return true despite engine error")
	`)
	require.NoError(t, err)

	require.Len(t, engine.registerCalls, 1)
	require.Len(t, engine.callbackCalls, 1)
	require.Equal(t, "boom", engine.callbackCalls[0].hookName)
}

// TestHookAPI_AddScript_FullFields 完整字段表：引擎收到的 Script 携带全部显式字段。
func TestHookAPI_AddScript_FullFields(t *testing.T) {
	engine := &fakeHookEngine{}
	L := newHookLuaState(engine)
	defer L.Close()

	err := L.DoString(`
		local h = require "kratos_hook"
		local ok = h.add_script("targethook", {
			name = "s1",
			source = "print(1)",
			enabled = false,
			priority = 7,
			description = "d1"
		})
		assert(ok == true, "add_script should return true")
	`)
	require.NoError(t, err)

	require.Len(t, engine.addScriptCalls, 1)
	call := engine.addScriptCalls[0]
	require.Equal(t, "targethook", call.hookName)
	require.Equal(t, "targethook", call.script.Hook)
	require.Equal(t, "s1", call.script.Name)
	require.Equal(t, "print(1)", call.script.Source)
	require.False(t, call.script.Enabled)
	require.Equal(t, 7, call.script.Priority)
	require.Equal(t, "d1", call.script.Description)
}

// TestHookAPI_AddScript_DefaultFields 缺省字段表：enabled 默认 true、
// priority 默认 0、description 默认空串，其余字段按传入值。
func TestHookAPI_AddScript_DefaultFields(t *testing.T) {
	engine := &fakeHookEngine{}
	L := newHookLuaState(engine)
	defer L.Close()

	err := L.DoString(`
		local h = require "kratos_hook"
		local ok = h.add_script("t2", {name = "s2", source = "x"})
		assert(ok == true, "add_script should return true")
	`)
	require.NoError(t, err)

	require.Len(t, engine.addScriptCalls, 1)
	call := engine.addScriptCalls[0]
	require.Equal(t, "t2", call.hookName)
	require.Equal(t, "t2", call.script.Hook)
	require.Equal(t, "s2", call.script.Name)
	require.Equal(t, "x", call.script.Source)
	require.True(t, call.script.Enabled)
	require.Equal(t, 0, call.script.Priority)
	require.Equal(t, "", call.script.Description)
}

// TestHookAPI_AddScript_PriorityNotNumber priority 为字符串时类型断言失败，
// 保持默认值 0。
func TestHookAPI_AddScript_PriorityNotNumber(t *testing.T) {
	engine := &fakeHookEngine{}
	L := newHookLuaState(engine)
	defer L.Close()

	err := L.DoString(`
		local h = require "kratos_hook"
		local ok = h.add_script("t3", {name = "s3", source = "x", priority = "high"})
		assert(ok == true, "add_script should return true")
	`)
	require.NoError(t, err)

	require.Len(t, engine.addScriptCalls, 1)
	require.Equal(t, 0, engine.addScriptCalls[0].script.Priority)
}

// TestHookAPI_AddScript_EngineError 引擎 AddScript 报错：
// Lua 侧返回 (false, 错误字符串)，引擎侧仍收到调用。
func TestHookAPI_AddScript_EngineError(t *testing.T) {
	engine := &fakeHookEngine{addScriptErr: errors.New("nope")}
	L := newHookLuaState(engine)
	defer L.Close()

	err := L.DoString(`
		local h = require "kratos_hook"
		local ok, e = h.add_script("t4", {name = "s4", source = "x"})
		assert(ok == false, "add_script should return false on engine error")
		assert(type(e) == "string", "error should be surfaced as string")
	`)
	require.NoError(t, err)

	require.Len(t, engine.addScriptCalls, 1)
	require.Equal(t, "t4", engine.addScriptCalls[0].hookName)
}

// TestHookAPI_List 引擎钩子列表转换为 Lua 数组表（1 基）。
func TestHookAPI_List(t *testing.T) {
	engine := &fakeHookEngine{listResult: []string{"a", "b"}}
	L := newHookLuaState(engine)
	defer L.Close()

	err := L.DoString(`
		local h = require "kratos_hook"
		local l = h.list()
		assert(#l == 2, "list length should be 2, got " .. tostring(#l))
		assert(l[1] == "a", "first entry should be a")
		assert(l[2] == "b", "second entry should be b")
	`)
	require.NoError(t, err)
}
