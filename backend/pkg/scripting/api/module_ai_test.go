// 本文件针对 module_ai.go（脚本 ai 模块）做单元测试。
// AICompleter 是单方法接口，用 fake 直接打桩（零 infra）；验证：
//   - JS 版 chat / chatWith / chatWithSystem 的参数传递与错误传播；
//   - 调用上下文带 aiChatCallTimeout 兜底 deadline；
//   - nil completer 的 fail-closed 空模块（Lua 与 JS 两条路线）；
//   - Lua 版成功返回 1 值、失败返回 (nil, errstr) 两值的约定。
package api

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	lua "github.com/yuin/gopher-lua"
)

// fakeAICompleter 记录入参并按注入的函数返回。
type fakeAICompleter struct {
	fn func(ctx context.Context, providerId int64, systemPrompt, content string) (string, error)
}

func (f *fakeAICompleter) ChatForScript(ctx context.Context, providerId int64, systemPrompt, content string) (string, error) {
	return f.fn(ctx, providerId, systemPrompt, content)
}

func TestModuleAI_NilCompleter(t *testing.T) {
	mod := ModuleAI(nil, nil)
	require.Equal(t, "ai", mod.Name)
	require.Empty(t, mod.Funcs, "nil completer must produce an empty (fail-closed) module")

	// Lua 路线：require 得到空表。
	L := lua.NewState()
	t.Cleanup(L.Close)
	L.PreloadModule("kratos_ai", LoaderAI(nil, nil))
	require.NoError(t, L.DoString(`
		local ai = require "kratos_ai"
		assert(type(ai) == "table", "module should be a table")
		assert(ai.chat == nil, "nil completer must not expose chat")
	`))
}

func TestModuleAI_JSChatVariants(t *testing.T) {
	var gotProvider int64
	var gotSystem, gotContent string
	completer := &fakeAICompleter{fn: func(_ context.Context, providerId int64, systemPrompt, content string) (string, error) {
		gotProvider, gotSystem, gotContent = providerId, systemPrompt, content
		return "回复文本", nil
	}}

	funcs := ModuleAI(completer, nil).Funcs
	chat := funcs["chat"].(func(string) (string, error))
	chatWith := funcs["chatWith"].(func(float64, string) (string, error))
	chatWithSystem := funcs["chatWithSystem"].(func(float64, string, string) (string, error))

	reply, err := chat("你好")
	require.NoError(t, err)
	require.Equal(t, "回复文本", reply)
	require.Zero(t, gotProvider, "chat() must use default provider (0)")
	require.Empty(t, gotSystem)
	require.Equal(t, "你好", gotContent)

	reply, err = chatWith(7, "指定提供商")
	require.NoError(t, err)
	require.Equal(t, "回复文本", reply)
	require.EqualValues(t, 7, gotProvider)

	reply, err = chatWithSystem(7, "你是巡检助手", "巡检")
	require.NoError(t, err)
	require.Equal(t, "回复文本", reply)
	require.Equal(t, "你是巡检助手", gotSystem)
	require.Equal(t, "巡检", gotContent)
}

func TestModuleAI_JSDeadline(t *testing.T) {
	completer := &fakeAICompleter{fn: func(ctx context.Context, _ int64, _ string, _ string) (string, error) {
		deadline, ok := ctx.Deadline()
		require.True(t, ok, "call context must carry a timeout deadline")
		require.Equal(t, aiChatCallTimeout, time.Until(deadline).Round(time.Minute), "deadline must be aiChatCallTimeout")
		return "ok", nil
	}}
	chat := ModuleAI(completer, nil).Funcs["chat"].(func(string) (string, error))
	_, err := chat("x")
	require.NoError(t, err)
}

func TestModuleAI_JSErrorPropagates(t *testing.T) {
	wantErr := errors.New("provider unavailable")
	completer := &fakeAICompleter{fn: func(context.Context, int64, string, string) (string, error) {
		return "", wantErr
	}}
	chat := ModuleAI(completer, nil).Funcs["chat"].(func(string) (string, error))
	reply, err := chat("x")
	require.ErrorIs(t, err, wantErr)
	require.Empty(t, reply, "error path must return empty reply for goja to raise")
}

// TestModuleAI_LuaContract Lua 路线约定：成功返回 1 值（字符串），
// 失败返回 (nil, errstr) 两值，脚本侧可 `local reply, err = ai.chat(...)`。
func TestModuleAI_LuaContract(t *testing.T) {
	completer := &fakeAICompleter{fn: func(_ context.Context, providerId int64, _, content string) (string, error) {
		if content == "boom" {
			return "", errors.New("boom")
		}
		require.EqualValues(t, 3, providerId)
		return "lua回复", nil
	}}

	L := lua.NewState()
	t.Cleanup(L.Close)
	L.PreloadModule("kratos_ai", LoaderAI(completer, bLogger.NewHelper(bLogger.NopLogger())))

	require.NoError(t, L.DoString(`
		local ai = require "kratos_ai"

		local reply = ai.chat_with(3, "普通提问")
		assert(reply == "lua回复", "success should return the reply string, got " .. tostring(reply))

		local reply2, err2 = ai.chat("boom")
		assert(reply2 == nil, "failure should return nil as the first value")
		assert(err2 == "boom", "failure should return the error string as the second value, got " .. tostring(err2))
	`))
}

func TestModuleAI_LuaMissingArg(t *testing.T) {
	completer := &fakeAICompleter{fn: func(context.Context, int64, string, string) (string, error) {
		return "should not be called", nil
	}}
	L := lua.NewState()
	t.Cleanup(L.Close)
	L.PreloadModule("kratos_ai", LoaderAI(completer, nil))

	// 参数缺失：CheckString/CheckNumber 抛错，且 completer 不被调用。
	require.Error(t, L.DoString(`require "kratos_ai".chat()`))
	require.Error(t, L.DoString(`require "kratos_ai".chat_with(1)`))
}
