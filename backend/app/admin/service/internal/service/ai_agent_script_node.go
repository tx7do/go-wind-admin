package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Agent 编排 script 节点（docs/ai_agent_orchestration.md §2.1 节点集扩展）：
// 把全图上下文交给一次性隔离沙箱（go-scripts 引擎，与脚本系统试运行 TestRun 同款
// 依赖面：redis/oss/ai 模块可用）执行，脚本返回的 table 合并写回节点上下文。
//
// data 键：language("lua"|"javascript")、source(脚本源码)。
// source 是代码、不是模板——代码里的 {{...}} 是脚本自身语法，
// 绝不能过 renderAgentTemplate（与 http 节点"字段值过模板"的关键差异）。
// 超时沿用引擎 VMTimeout 缺省 5s，节点不带 timeout 字段（TestRun 形态定死）。

const (
	// agentScriptMaxSource 脚本源码上限：script 节点语义是"小段数据加工"，
	// 不是承载完整应用的载体；超限请在脚本系统建正式脚本（平台管理员面）。
	agentScriptMaxSource = 32 << 10
)

// agentScriptAllowedLanguages 语言白名单：与脚本系统的两种引擎对齐。
var agentScriptAllowedLanguages = map[string]bool{
	"lua":        true,
	"javascript": true,
}

// validateAgentScriptNode Run 前静态校验（source 是代码，只校验字面结构）。
func validateAgentScriptNode(n *agentGraphNode) error {
	lang := agentScriptLanguage(n)
	if !agentScriptAllowedLanguages[lang] {
		return fmt.Errorf("script node %q has unsupported language %q (lua|javascript)", n.ID, lang)
	}
	source := agentDataString(n.Data, "source")
	if strings.TrimSpace(source) == "" {
		return fmt.Errorf("script node %q has empty source", n.ID)
	}
	if len(source) > agentScriptMaxSource {
		return fmt.Errorf("script node %q source exceeds %d bytes", n.ID, agentScriptMaxSource)
	}
	return nil
}

func agentScriptLanguage(n *agentGraphNode) string {
	return strings.ToLower(strings.TrimSpace(agentDataString(n.Data, "language")))
}

// agentScriptNodeInput 节点入参：全图上下文逐命名空间透传（脚本里按节点 id 取表）。
func agentScriptNodeInput(vars map[string]map[string]string) map[string]any {
	input := make(map[string]any, len(vars))
	for k, v := range vars {
		input[k] = v
	}
	return input
}

// agentScriptOutputs 把脚本执行后的上下文数据收口为字符串命名空间：
// 字符串直通、标量 fmt 转写，其余（嵌套表等）JSON 编码——上下文值统一是 string，
// 下游模板 {{节点id.变量}} 与条件表达式都按字符串消费。
// nil 值跳过（脚本里显式 set nil 即"不产出该变量"）。
func agentScriptOutputs(data map[string]any) map[string]string {
	outs := make(map[string]string, len(data))
	for k, v := range data {
		if v == nil {
			continue
		}
		switch val := v.(type) {
		case string:
			outs[k] = val
		case bool:
			outs[k] = fmt.Sprintf("%t", val)
		case float64:
			outs[k] = strconv.FormatFloat(val, 'f', -1, 64)
		case int:
			outs[k] = fmt.Sprintf("%d", val)
		case int64:
			outs[k] = fmt.Sprintf("%d", val)
		default:
			encoded, err := json.Marshal(val)
			if err != nil {
				outs[k] = fmt.Sprintf("%v", val)
				continue
			}
			outs[k] = string(encoded)
		}
	}
	return outs
}

// agentScriptSummary 节点摘要：产出的变量名清单（键序稳定，截断到 120 rune）。
func agentScriptSummary(outs map[string]string) string {
	keys := make([]string, 0, len(outs))
	for k := range outs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return truncateRunes("keys=" + strings.Join(keys, ","), 120)
}

// runScriptForAgent script 节点的真实现：一次性隔离沙箱执行（ScriptRuntime.TestRun），
// 与常驻脚本引擎/钩子互不污染；执行留 script_log 审计行（name 带节点定位）。
// scriptRuntime 未装配（单测/极简部署）时报错而不是静默跳过——
// 图里摆了 script 节点却静默不执行，用户只会看到"输出是空的"。
func (s *AiAgentService) runScriptForAgent(ctx context.Context, node *agentGraphNode, vars map[string]map[string]string) (map[string]any, error) {
	if s.scriptRuntime == nil {
		return nil, fmt.Errorf("script node %q unavailable: script runtime not wired", node.ID)
	}
	language := agentScriptLanguage(node)
	source := agentDataString(node.Data, "source")
	name := "agent:" + node.ID
	input := agentScriptNodeInput(vars)
	data, err := s.scriptRuntime.TestRun(ctx, language, name, source, input)
	if err != nil {
		return nil, err
	}
	// 剔除输入键回声：TestRun 返回的是执行后完整上下文（含注入的输入命名空间），
	// 节点产出只收脚本新写入的键——输入本就按自己的节点 id 在上下文里，回声既污染
	// 下游命名空间又让 trace 摘要失真（keys= 里混进一堆 start_1）。
	for k := range input {
		delete(data, k)
	}
	return data, nil
}
