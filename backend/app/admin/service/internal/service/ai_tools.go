package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

// execAiTool 执行一个内置工具调用，返回回传给模型的结果文本。
// 未知工具/参数非法返回错误文本（回传模型让它自行纠正，而非中断整轮）。
func execAiTool(_ context.Context, name, args string) (string, error) {
	switch name {
	case "get_current_time":
		return time.Now().Format("2006-01-02 15:04:05"), nil
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

// AI Function Call（协议层）：
//   - aiBuiltinToolDefs：暴露给模型的工具清单（OpenAI function 格式）；
//   - aiToolLoop：多轮工具调用循环——模型请求工具（finish_reason=tool_calls）→
//     本地执行 → 结果回传 → 继续生成，直到给出文本答案或轮数耗尽。
//
// 轮数耗尽前的最后一轮强制不带 tools（模型必须给文本答案，不允许无限工具循环）。
// 工具调用经 onToolCall 回调推 SSE（ai_chat_tool 事件）供前端展示"模型正在调工具"；
// 工具轮不进消息落库（落库的 assistant 消息只有最终答案）；
// 用量跨轮累加（每轮的 prompt/completion 计入同一条用量流水）。

// aiToolMaxRounds 工具调用循环的最大轮数。
const aiToolMaxRounds = 4

// aiBuiltinToolDefs 内置工具清单。新增工具：此处加定义 + execAiTool 加执行分支。
func aiBuiltinToolDefs() []openai.Tool {
	return []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_current_time",
				Description: "获取服务器当前时间（本地时区，YYYY-MM-DD HH:MM:SS）。回答与当前时间相关的问题前调用。",
				Parameters:  json.RawMessage(`{"type":"object","properties":{},"required":[]}`),
			},
		},
	}
}

// aiScriptToolDefs 把脚本工具注册表（工具市场，tool.register）翻成 OpenAI 工具定义。
// rt 为 nil 或无注册工具时返回空——脚本工具不出现在 defs 里（不静默报错的纪律
// 与 notify seam 一致）。
func aiScriptToolDefs(rt *ScriptRuntime) []openai.Tool {
	if rt == nil {
		return nil
	}
	handlers := rt.ScriptTools()
	if len(handlers) == 0 {
		return nil
	}
	defs := make([]openai.Tool, 0, len(handlers))
	for _, h := range handlers {
		props := map[string]any{}
		required := make([]string, 0, len(h.Required))
		for _, name := range h.Required {
			props[name] = map[string]any{"type": "string"}
			required = append(required, name)
		}
		for name, typ := range h.Optional {
			t := "string"
			switch typ.(type) {
			case float64, int:
				t = "number"
			case bool:
				t = "boolean"
			}
			props[name] = map[string]any{"type": t}
		}
		schema, _ := json.Marshal(map[string]any{
			"type":       "object",
			"properties": props,
			"required":   required,
		})
		defs = append(defs, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        h.Name,
				Description: h.Description,
				Parameters:  schema,
			},
		})
	}
	return defs
}

// aiIsScriptTool 报告工具名是否为脚本工具（路由 exec 用：脚本工具优先走脚本引擎）。
func aiIsScriptTool(rt *ScriptRuntime, name string) bool {
	return rt != nil && rt.HasScriptTool(name)
}

// execAiScriptTool 执行脚本工具并把产出收口为文本回给模型。
func execAiScriptTool(rt *ScriptRuntime, ctx context.Context, name, args string) (string, error) {
	argsMap := map[string]any{}
	if args != "" {
		if err := json.Unmarshal([]byte(args), &argsMap); err != nil {
			return "", fmt.Errorf("invalid tool arguments: %w", err)
		}
	}
	result, err := rt.ExecuteScriptTool(ctx, name, argsMap)
	if err != nil {
		return "", err
	}
	switch v := result.(type) {
	case string:
		return v, nil
	case nil:
		return "", nil
	default:
		if b, mErr := json.Marshal(v); mErr == nil {
			return string(b), nil
		}
		return fmt.Sprintf("%v", v), nil
	}
}

// aiToolLoop 单次流式对话的多轮工具调用循环。
type aiToolLoop struct {
	client     *openai.Client
	model      string
	tools      []openai.Tool
	exec       func(ctx context.Context, name, args string) (string, error)
	onDelta    func(delta string)                   // 最终答案的文本增量回调（SSE 推送）
	onToolCall func(name, arguments, result string) // 每次工具执行完成后的可见化回调（SSE 推送）
	maxRounds  int
	// temperature 采样温度；nil = 不传该参数（chat 链路维持不设，agent 节点按画布配置传）。
	temperature *float32
}

// run 执行循环，返回最终答案全文与跨轮累计用量。
func (r *aiToolLoop) run(ctx context.Context, messages []openai.ChatCompletionMessage) (string, openai.Usage, error) {
	var totalUsage openai.Usage
	var full strings.Builder

	for round := 0; round < r.maxRounds; round++ {
		lastRound := round == r.maxRounds-1
		req := openai.ChatCompletionRequest{
			Model:    r.model,
			Messages: messages,
			StreamOptions: &openai.StreamOptions{
				IncludeUsage: true,
			},
		}
		if r.temperature != nil {
			req.Temperature = *r.temperature
		}
		// 最后一轮不带 tools：强制模型给文本答案，不允许无限工具循环。
		if !lastRound {
			req.Tools = r.tools
		}

		stream, err := r.client.CreateChatCompletionStream(ctx, req)
		if err != nil {
			return "", totalUsage, err
		}

		var roundFull strings.Builder
		var toolCalls []openai.ToolCall
		finishToolCalls := false

	streamDone:
		for {
			chunk, recvErr := stream.Recv()
			if recvErr != nil {
				_ = stream.Close()
				if errors.Is(recvErr, io.EOF) {
					break streamDone
				}
				return "", totalUsage, recvErr
			}
			if chunk.Usage != nil {
				totalUsage.PromptTokens += chunk.Usage.PromptTokens
				totalUsage.CompletionTokens += chunk.Usage.CompletionTokens
				totalUsage.TotalTokens += chunk.Usage.TotalTokens
			}
			if len(chunk.Choices) == 0 {
				continue
			}
			choice := chunk.Choices[0]
			if choice.FinishReason == openai.FinishReasonToolCalls {
				finishToolCalls = true
			}
			// 文本增量：只对最终答案轮推给前端（工具轮的 content 通常为空）
			if choice.Delta.Content != "" {
				roundFull.WriteString(choice.Delta.Content)
				if r.onDelta != nil {
					r.onDelta(choice.Delta.Content)
				}
			}
			// 工具调用增量：按 index 累积分片
			for _, tc := range choice.Delta.ToolCalls {
				toolCalls = appendToolCallDelta(toolCalls, tc)
			}
		}
		_ = stream.Close()

		// 本轮没有请求工具 → 答案完成
		if !finishToolCalls || len(toolCalls) == 0 {
			full.WriteString(roundFull.String())
			return full.String(), totalUsage, nil
		}

		// 有工具调用：把 assistant(tool_calls) 与各 tool 结果追加进消息，进入下一轮。
		assistantMsg := openai.ChatCompletionMessage{Role: openai.ChatMessageRoleAssistant, ToolCalls: toolCalls}
		messages = append(messages, assistantMsg)
		for _, tc := range toolCalls {
			result, execErr := r.exec(ctx, tc.Function.Name, tc.Function.Arguments)
			if execErr != nil {
				// 错误文本回传模型：让它知道工具失败并自行调整，而不是中断对话
				result = fmt.Sprintf("tool error: %v", execErr)
			}
			if r.onToolCall != nil {
				r.onToolCall(tc.Function.Name, tc.Function.Arguments, result)
			}
			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}

	return "", totalUsage, fmt.Errorf("ai tool loop exhausted %d rounds without a final answer", r.maxRounds)
}

// appendToolCallDelta 把流式工具调用分片按 Index 累积进片段集。
func appendToolCallDelta(acc []openai.ToolCall, delta openai.ToolCall) []openai.ToolCall {
	idx := 0
	if delta.Index != nil {
		idx = *delta.Index
	}
	for len(acc) <= idx {
		acc = append(acc, openai.ToolCall{})
	}
	target := &acc[idx]
	if delta.ID != "" {
		target.ID = delta.ID
	}
	if delta.Type != "" {
		target.Type = delta.Type
	}
	if delta.Function.Name != "" {
		target.Function.Name += delta.Function.Name
	}
	if delta.Function.Arguments != "" {
		target.Function.Arguments += delta.Function.Arguments
	}
	return acc
}
