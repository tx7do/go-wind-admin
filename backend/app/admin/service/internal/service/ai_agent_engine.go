package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/expr-lang/expr"
)

// AI Agent 图执行引擎（M1）。
//
// 图 DSL、节点语义与校验规则见 docs/ai_agent_orchestration.md §2/§3。要点：
//   - 上下文按节点 id 分命名空间（map[节点id]map[变量]字符串值）：knowledge 写 context/hits、
//     llm 写 output、start 写运行入参。模板 `{{节点id.变量}}` 与条件表达式都读这份上下文；
//   - 条件表达式走 expr-lang（与 go-scripts 家族 expr 引擎同语言同版本）；不经它的 engine 包装——
//     那套 source/Load 管线面向"存储的脚本"，与"每次求值带运行时 env 的一次性表达式"不匹配；
//   - LLM 与知识检索是注入的函数（runLLM/searchKnowledge）：service 装配真实现，
//     单测注入桩，引擎本体不触网。

// agentMaxSteps 单次运行的节点执行步数上限。
// 正常图到不了（校验已保证无环）；这是条件表达式写出意外状况时的兜底。
const agentMaxSteps = 50

// agentNodeTypes 合法节点类型集。
var agentNodeTypes = map[string]bool{
	"start": true, "knowledge": true, "llm": true, "condition": true, "end": true,
	"http": true,
}

// agentGraphNode 图节点。data 保持无类型 map：五种节点各自的键在执行处按需取，
// 画布演进加键不必改这份结构（校验只关心 type 与 id）。
type agentGraphNode struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Position struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	} `json:"position"`
	Data map[string]any `json:"data"`
}

// agentGraphEdge 图边。branch 仅 condition 出边使用（"true"/"false"）。
type agentGraphEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Branch string `json:"branch,omitempty"`
}

type agentGraph struct {
	Nodes []agentGraphNode `json:"nodes"`
	Edges []agentGraphEdge `json:"edges"`
}

// agentUsage 单节点/单次运行的 token 用量。
type agentUsage struct {
	Prompt     uint32 `json:"prompt"`
	Completion uint32 `json:"completion"`
	Total      uint32 `json:"total"`
}

func (u *agentUsage) add(other agentUsage) {
	u.Prompt += other.Prompt
	u.Completion += other.Completion
	u.Total += other.Total
}

// agentTraceStep 一步执行的轨迹记录（落 sys_ai_agent_runs.trace JSON 数组）。
type agentTraceStep struct {
	NodeID  string      `json:"nodeId"`
	Type    string      `json:"type"`
	Status  string      `json:"status"` // ok | error
	Ms      int64       `json:"ms"`
	Summary string      `json:"summary,omitempty"`
	Usage   *agentUsage `json:"usage,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// parseAgentGraph 解析图 DSL（JSON → 结构）。
func parseAgentGraph(raw string) (*agentGraph, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("graph is empty")
	}
	var g agentGraph
	if err := json.Unmarshal([]byte(raw), &g); err != nil {
		return nil, fmt.Errorf("graph is not valid JSON: %w", err)
	}
	return &g, nil
}

// validateAgentGraph Run 前的静态校验；规则清单见设计文档 §2.3。
func validateAgentGraph(g *agentGraph) error {
	if len(g.Nodes) == 0 {
		return fmt.Errorf("graph has no nodes")
	}

	byID := make(map[string]*agentGraphNode, len(g.Nodes))
	idPattern := func(id string) bool {
		if id == "" || len(id) > 64 {
			return false
		}
		for i := 0; i < len(id); i++ {
			c := id[i]
			switch {
			case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c == '_':
			case c >= '0' && c <= '9' && i > 0:
			default:
				return false
			}
		}
		return true
	}

	startCount, endCount := 0, 0
	for i := range g.Nodes {
		n := &g.Nodes[i]
		if !agentNodeTypes[n.Type] {
			return fmt.Errorf("node %q has unknown type %q", n.ID, n.Type)
		}
		if !idPattern(n.ID) {
			return fmt.Errorf("node id %q is invalid (letters/digits/underscore, must not start with a digit)", n.ID)
		}
		if _, dup := byID[n.ID]; dup {
			return fmt.Errorf("duplicate node id %q", n.ID)
		}
		byID[n.ID] = n
		switch n.Type {
		case "start":
			startCount++
		case "end":
			endCount++
		case "http":
			if err := validateAgentHTTPNode(n); err != nil {
				return err
			}
		}
	}
	if startCount != 1 {
		return fmt.Errorf("graph must have exactly one start node, got %d", startCount)
	}
	if endCount == 0 {
		return fmt.Errorf("graph must have at least one end node")
	}

	edgeSeen := make(map[string]bool, len(g.Edges))
	out := make(map[string][]*agentGraphEdge, len(g.Nodes))
	for i := range g.Edges {
		e := &g.Edges[i]
		if _, ok := byID[e.Source]; !ok {
			return fmt.Errorf("edge %q references unknown source node %q", e.ID, e.Source)
		}
		if _, ok := byID[e.Target]; !ok {
			return fmt.Errorf("edge %q references unknown target node %q", e.ID, e.Target)
		}
		key := e.Source + "->" + e.Target
		if edgeSeen[key] {
			return fmt.Errorf("duplicate edge %s", key)
		}
		edgeSeen[key] = true
		out[e.Source] = append(out[e.Source], e)
	}

	for i := range g.Nodes {
		n := &g.Nodes[i]
		edges := out[n.ID]
		switch n.Type {
		case "end":
			if len(edges) > 0 {
				return fmt.Errorf("end node %q must not have outgoing edges", n.ID)
			}
		case "condition":
			if len(edges) != 2 {
				return fmt.Errorf("condition node %q must have exactly two outgoing edges, got %d", n.ID, len(edges))
			}
			branches := map[string]bool{}
			for _, e := range edges {
				if e.Branch != "true" && e.Branch != "false" {
					return fmt.Errorf("condition node %q outgoing edge must carry branch=true|false", n.ID)
				}
				branches[e.Branch] = true
			}
			if !branches["true"] || !branches["false"] {
				return fmt.Errorf("condition node %q must have one true and one false branch", n.ID)
			}
		default:
			if len(edges) > 1 {
				return fmt.Errorf("node %q (%s) must have at most one outgoing edge", n.ID, n.Type)
			}
		}
	}

	// 可达性：从 start 出发沿全部边走，所有节点必须可达。
	reachable := map[string]bool{}
	var visit func(id string)
	visit = func(id string) {
		if reachable[id] {
			return
		}
		reachable[id] = true
		for _, e := range out[id] {
			visit(e.Target)
		}
	}
	for _, e := range out[byIDStart(g).ID] {
		visit(e.Target)
	}
	reachable[byIDStart(g).ID] = true
	for i := range g.Nodes {
		if !reachable[g.Nodes[i].ID] {
			return fmt.Errorf("node %q is not reachable from start", g.Nodes[i].ID)
		}
	}
	return nil
}

func byIDStart(g *agentGraph) *agentGraphNode {
	for i := range g.Nodes {
		if g.Nodes[i].Type == "start" {
			return &g.Nodes[i]
		}
	}
	return nil
}

// renderAgentTemplate 单趟替换 {{node.var}}（与 webhook 载荷模板同一套防递归思路：
// 单趟扫描让每个值只被消费一次）。未知占位符报错——拼错的名字悄悄变空串，
// 用户只会看到"模型没收到资料"，排障方向整个是反的。
func renderAgentTemplate(tpl string, vars map[string]map[string]string) (string, error) {
	var out strings.Builder
	rest := tpl
	for len(rest) > 0 {
		open := strings.Index(rest, "{{")
		if open < 0 {
			out.WriteString(rest)
			break
		}
		closing := strings.Index(rest[open+2:], "}}")
		if closing < 0 {
			out.WriteString(rest)
			break
		}
		name := strings.TrimSpace(rest[open+2 : open+2+closing])
		dot := strings.Index(name, ".")
		if dot <= 0 || dot == len(name)-1 {
			return "", fmt.Errorf("template uses invalid placeholder {{%s}} (expected {{node.var}})", name)
		}
		nodeID, varName := name[:dot], name[dot+1:]
		ns, ok := vars[nodeID]
		if !ok {
			return "", fmt.Errorf("template references unknown node %q", nodeID)
		}
		value, ok := ns[varName]
		if !ok {
			return "", fmt.Errorf("template references unknown variable %q of node %q", varName, nodeID)
		}
		out.WriteString(rest[:open])
		out.WriteString(value)
		rest = rest[open+2+closing+2:]
	}
	return out.String(), nil
}

// evalAgentCondition 求值条件节点表达式：env 是节点命名空间，表达式必须产出布尔。
func evalAgentCondition(expression string, vars map[string]map[string]string) (bool, error) {
	if strings.TrimSpace(expression) == "" {
		return false, fmt.Errorf("condition expression is empty")
	}
	program, err := expr.Compile(expression, expr.Env(vars), expr.AsBool())
	if err != nil {
		return false, fmt.Errorf("condition expression compile failed: %w", err)
	}
	result, err := expr.Run(program, vars)
	if err != nil {
		return false, fmt.Errorf("condition expression eval failed: %w", err)
	}
	approved, _ := result.(bool)
	return approved, nil
}

// agentEngineDeps 引擎的外呼依赖（service 装配真实现；测试注入桩）。
type agentEngineDeps struct {
	// searchKnowledge 执行知识检索节点，返回拼好的上下文与命中条数。
	searchKnowledge func(ctx context.Context, node *agentGraphNode, query string) (contextText string, hits int, err error)
	// runLLM 执行 LLM 节点（含工具调用循环），返回最终答案与用量。
	runLLM func(ctx context.Context, node *agentGraphNode, system, user string) (string, agentUsage, error)
	// onStep 节点执行阶段回调（node_start / node_end），SSE 流式观感用；nil = 不推。
	// 推帧失败（订阅端慢）不影响执行——回调实现方自己保证尽力而为语义。
	onStep func(phase, nodeID, summary, status string)
}

// executeAgentGraph 从 start 走图到 end，返回产出、轨迹与累计用量。
// 任一节点执行失败即返回错误（trace 保留到失败步为止，由调用方落 FAILED 运行行）。
func executeAgentGraph(
	ctx context.Context,
	g *agentGraph,
	input map[string]string,
	deps *agentEngineDeps,
) (output string, trace []agentTraceStep, usage agentUsage, err error) {
	byID := make(map[string]*agentGraphNode, len(g.Nodes))
	out := make(map[string][]*agentGraphEdge)
	for i := range g.Nodes {
		byID[g.Nodes[i].ID] = &g.Nodes[i]
	}
	for i := range g.Edges {
		out[g.Edges[i].Source] = append(out[g.Edges[i].Source], &g.Edges[i])
	}

	start := byIDStart(g)
	// 命名空间：节点 id → 变量集。start 的输入变量按声明必填校验。
	vars := map[string]map[string]string{}
	inputVars := map[string]bool{}
	for _, raw := range agentDataSlice(start.Data, "inputVars") {
		m, _ := raw.(map[string]any)
		name, _ := m["name"].(string)
		if name == "" {
			continue
		}
		inputVars[name] = true
		required, _ := m["required"].(bool)
		if required {
			if _, ok := input[name]; !ok {
				return "", trace, usage, fmt.Errorf("missing required input variable %q", name)
			}
		}
	}
	startVars := map[string]string{}
	for k, v := range input {
		startVars[k] = v
	}
	vars[start.ID] = startVars

	current := start
	for step := 0; step < agentMaxSteps; step++ {
		stepStart := time.Now()
		node := current
		stepUsage := agentUsage{}
		summary := ""
		branchTaken := "" // condition 节点的分支判定（"true"/"false"），选边用

		if deps.onStep != nil {
			deps.onStep("node_start", node.ID, "", "")
		}

		switch node.Type {
		case "start":
			summary = fmt.Sprintf("%d input variable(s)", len(startVars))

		case "knowledge":
			query, rerr := renderAgentTemplate(agentDataString(node.Data, "query"), vars)
			if rerr != nil {
				return "", trace, usage, agentStepError(node, rerr)
			}
			contextText, hits, kerr := deps.searchKnowledge(ctx, node, query)
			if kerr != nil {
				return "", trace, usage, agentStepError(node, kerr)
			}
			vars[node.ID] = map[string]string{"context": contextText, "hits": strconv.Itoa(hits)}
			summary = fmt.Sprintf("hits=%d, %d chars", hits, len(contextText))

		case "llm":
			system, rerr := renderAgentTemplate(agentDataString(node.Data, "systemPrompt"), vars)
			if rerr != nil {
				return "", trace, usage, agentStepError(node, rerr)
			}
			user, rerr := renderAgentTemplate(agentDataString(node.Data, "userPrompt"), vars)
			if rerr != nil {
				return "", trace, usage, agentStepError(node, rerr)
			}
			if strings.TrimSpace(user) == "" {
				return "", trace, usage, agentStepError(node, fmt.Errorf("llm node %q has empty userPrompt after rendering", node.ID))
			}
			answer, llmUsage, lerr := deps.runLLM(ctx, node, system, user)
			if lerr != nil {
				return "", trace, usage, agentStepError(node, lerr)
			}
			stepUsage = llmUsage
			usage.add(llmUsage)
			vars[node.ID] = map[string]string{"output": answer}
			summary = truncateRunes(answer, 120)

		case "http":
			status, body, contentType, herr := runAgentHTTPNode(ctx, node, vars)
			if herr != nil {
				return "", trace, usage, agentStepError(node, herr)
			}
			vars[node.ID] = map[string]string{
				"status":      status,
				"body":        body,
				"contentType": contentType,
			}
			summary = fmt.Sprintf("status=%s, %d bytes", status, len(body))

		case "condition":
			approved, cerr := evalAgentCondition(agentDataString(node.Data, "expression"), vars)
			if cerr != nil {
				return "", trace, usage, agentStepError(node, cerr)
			}
			if approved {
				branchTaken = "true"
			} else {
				branchTaken = "false"
			}
			summary = fmt.Sprintf("branch=%t", approved)

		case "end":
			outText, rerr := renderAgentTemplate(agentDataString(node.Data, "output"), vars)
			if rerr != nil {
				return "", trace, usage, agentStepError(node, rerr)
			}
			trace = append(trace, agentTraceStep{
				NodeID: node.ID, Type: node.Type, Status: "ok",
				Ms: time.Since(stepStart).Milliseconds(), Summary: "done",
			})
			if deps.onStep != nil {
				deps.onStep("node_end", node.ID, "done", "ok")
			}
			return outText, trace, usage, nil

		default:
			return "", trace, usage, fmt.Errorf("node %q has unknown type %q", node.ID, node.Type)
		}

		trace = append(trace, agentTraceStep{
			NodeID: node.ID, Type: node.Type, Status: "ok",
			Ms: time.Since(stepStart).Milliseconds(), Summary: summary,
			Usage: func() *agentUsage {
				u := stepUsage
				if u.Total == 0 {
					return nil
				}
				return &u
			}(),
		})
		if deps.onStep != nil {
			deps.onStep("node_end", node.ID, summary, "ok")
		}

		// 选下一条边：condition 按 true/false 分支取，其余取唯一出边。
		var next *agentGraphEdge
		for _, e := range out[node.ID] {
			if node.Type == "condition" {
				if e.Branch == branchTaken {
					next = e
					break
				}
			} else {
				next = e
				break
			}
		}
		if next == nil {
			return "", trace, usage, fmt.Errorf("node %q has no outgoing edge to continue", node.ID)
		}
		current = byID[next.Target]
		if current == nil {
			return "", trace, usage, fmt.Errorf("edge target %q vanished", next.Target)
		}
	}
	return "", trace, usage, fmt.Errorf("execution exceeded %d steps", agentMaxSteps)
}

func agentStepError(node *agentGraphNode, err error) error {
	return fmt.Errorf("node %s[%s]: %w", node.ID, node.Type, err)
}

// ---- agentGraphNode.Data 的取值辅助（JSON 反序列化后数字一律 float64） ----

func agentDataString(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	v, _ := data[key].(string)
	return v
}

func agentDataUint32(data map[string]any, key string) uint32 {
	if data == nil {
		return 0
	}
	switch v := data[key].(type) {
	case float64:
		if v <= 0 {
			return 0
		}
		return uint32(v)
	case string:
		n, _ := strconv.ParseUint(v, 10, 32)
		return uint32(n)
	default:
		return 0
	}
}

func agentDataInt(data map[string]any, key string, def int) int {
	if data == nil {
		return def
	}
	if v, ok := data[key].(float64); ok && v > 0 {
		return int(v)
	}
	return def
}

func agentDataFloat(data map[string]any, key string) (float64, bool) {
	if data == nil {
		return 0, false
	}
	v, ok := data[key].(float64)
	return v, ok
}

func agentDataSlice(data map[string]any, key string) []any {
	if data == nil {
		return nil
	}
	v, _ := data[key].([]any)
	return v
}
