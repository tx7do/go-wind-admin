package api

import (
	"context"
	"time"

	"github.com/tx7do/go-scripts/hostmodule"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	lua "github.com/yuin/gopher-lua"
)

// aiChatCallTimeout 脚本内单次 AI 调用的兜底时长。
// 注意：脚本引擎的 VMTimeout（默认 5s）同样约束本次调用所在的整体执行，
// 脚本要调 AI 时应把该脚本运行环境的 VMTimeout 配大（任务类脚本已是独立超时）。
const aiChatCallTimeout = 60 * time.Second

// AICompleter 脚本 ai 模块依赖的最小对话接口。
// 由 internal/service 的 ScriptRuntime 实现（复用 provider 解析 / 密钥解密 / 用量记账），
// 经 Engine.SetAICompleter 注入；nil 时 ai 模块不注册（脚本调用即报未定义）。
type AICompleter interface {
	// ChatForScript 发起一轮（可选 system 前缀的）补全，返回回复文本。
	// providerId=0 表示使用默认启用的提供商。
	ChatForScript(ctx context.Context, providerId int64, systemPrompt, content string) (string, error)
}

// ModuleAI 构建语言无关的 ai 模块（JS 等基于 map[string]any 桥接的语言使用）。
// 约定：Go 函数返回 (T, error) 时，goja 会把非 nil error 转成 JS 异常，脚本用 try/catch 处理。
// completer 为 nil 时返回不含函数的空模块。
func ModuleAI(completer AICompleter, logger *bLogger.Helper) hostmodule.ModuleDef {
	if completer == nil {
		return hostmodule.ModuleDef{Name: "ai", Funcs: map[string]any{}}
	}

	bg := context.Background()
	logError := func(op string, err error) {
		if logger != nil {
			logger.Errorf(bg, "ai.%s error: %v", op, err)
		}
	}
	callCtx := func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(bg, aiChatCallTimeout)
	}

	return hostmodule.ModuleDef{
		Name: "ai",
		Funcs: map[string]any{
			// chat(content) → 回复文本；用默认启用的提供商
			"chat": func(content string) (string, error) {
				ctx, cancel := callCtx()
				defer cancel()
				reply, err := completer.ChatForScript(ctx, 0, "", content)
				if err != nil {
					logError("chat", err)
					return "", err
				}
				return reply, nil
			},
			// chatWith(providerId, content) → 回复文本；显式指定提供商
			"chatWith": func(providerId float64, content string) (string, error) {
				ctx, cancel := callCtx()
				defer cancel()
				reply, err := completer.ChatForScript(ctx, int64(providerId), "", content)
				if err != nil {
					logError("chatWith", err)
					return "", err
				}
				return reply, nil
			},
			// chatWithSystem(providerId, systemPrompt, content) → 回复文本；带 system 前缀
			"chatWithSystem": func(providerId float64, systemPrompt, content string) (string, error) {
				ctx, cancel := callCtx()
				defer cancel()
				reply, err := completer.ChatForScript(ctx, int64(providerId), systemPrompt, content)
				if err != nil {
					logError("chatWithSystem", err)
					return "", err
				}
				return reply, nil
			},
		},
	}
}

// RegisterAI registers the AI API for Lua as a requireable module.
func RegisterAI(L *lua.LState, completer AICompleter, logger *bLogger.Helper) {
	L.PreloadModule("kratos_ai", LoaderAI(completer, logger))
}

// LoaderAI 返回 ai 模块（kratos_ai）的 loader，供 go-scripts 引擎 RegisterModule 使用。
// completer 为 nil 时返回空模块。
func LoaderAI(completer AICompleter, logger *bLogger.Helper) lua.LGFunction {
	return func(L *lua.LState) int {
		if completer == nil {
			L.Push(L.NewTable())
			return 1
		}

		aiModule := L.NewTable()

		// ai.chat(content) → string | nil, err
		aiModule.RawSetString("chat", L.NewFunction(func(L *lua.LState) int {
			content := L.CheckString(1)
			ctx, cancel := context.WithTimeout(context.Background(), aiChatCallTimeout)
			defer cancel()
			reply, err := completer.ChatForScript(ctx, 0, "", content)
			if err != nil {
				if logger != nil {
					logger.Errorf(context.Background(), "ai.chat error: %v", err)
				}
				L.Push(lua.LNil)
				L.Push(lua.LString(err.Error()))
				return 2
			}
			L.Push(lua.LString(reply))
			return 1
		}))

		// ai.chat_with(providerId, content) → string | nil, err
		aiModule.RawSetString("chat_with", L.NewFunction(func(L *lua.LState) int {
			providerId := int64(L.CheckNumber(1))
			content := L.CheckString(2)
			ctx, cancel := context.WithTimeout(context.Background(), aiChatCallTimeout)
			defer cancel()
			reply, err := completer.ChatForScript(ctx, int64(providerId), "", content)
			if err != nil {
				if logger != nil {
					logger.Errorf(context.Background(), "ai.chat_with error: %v", err)
				}
				L.Push(lua.LNil)
				L.Push(lua.LString(err.Error()))
				return 2
			}
			L.Push(lua.LString(reply))
			return 1
		}))

		// ai.chat_with_system(providerId, systemPrompt, content) → string | nil, err
		aiModule.RawSetString("chat_with_system", L.NewFunction(func(L *lua.LState) int {
			providerId := int64(L.CheckNumber(1))
			systemPrompt := L.CheckString(2)
			content := L.CheckString(3)
			ctx, cancel := context.WithTimeout(context.Background(), aiChatCallTimeout)
			defer cancel()
			reply, err := completer.ChatForScript(ctx, int64(providerId), systemPrompt, content)
			if err != nil {
				if logger != nil {
					logger.Errorf(context.Background(), "ai.chat_with_system error: %v", err)
				}
				L.Push(lua.LNil)
				L.Push(lua.LString(err.Error()))
				return 2
			}
			L.Push(lua.LString(reply))
			return 1
		}))

		L.Push(aiModule)
		return 1
	}
}
