package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
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
	"http": true, "script": true, "parallel": true, "subflow": true,
}

// agentMaxSubflowDepth 子图引用深度上限（防递归爆炸——A 引 B、B 引 A 的图对）。
const agentMaxSubflowDepth = 3

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
	Prompt     uint64 `json:"prompt"`
	Completion uint64 `json:"completion"`
	Total      uint64 `json:"total"`
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
		case "script":
			if err := validateAgentScriptNode(n); err != nil {
				return err
			}
		case "parallel":
			// parallel 节点允许 N 条出边（并发分支），但每条必须指向存在的节点（通用校验已覆盖）
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
		case "parallel":
			// parallel 节点允许 N 条出边（并发分支）；每条出边在下游汇聚即可
			if len(edges) == 0 {
				return fmt.Errorf("parallel node %q must have at least one outgoing edge", n.ID)
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

	// 环合法性（M2「循环」）：强连通分量 size>1 必须至少含一个 condition 节点
	// （即所有环都经过条件分流保证可终止）；单节点自环禁止。
	if err := validateCycleConditions(g, byID, out); err != nil {
		return err
	}

	return nil
}

// validateCycleConditions 环合法性：Tarjan 强连通分量分解——
// size>1 的分量（即含环）必须至少含一个 condition 节点，否则 400。
// 近似白名单：分量内可能同时含 condition 环与非 condition 环，v1 按分量粒度判定（文档注明）。
func validateCycleConditions(g *agentGraph, byID map[string]*agentGraphNode, out map[string][]*agentGraphEdge) error {
	index, low := map[string]int{}, map[string]int{}
	onStack := map[string]bool{}
	stack := []string{}
	curr := 0
	var sccs [][]string

	var strongconnect func(id string)
	strongconnect = func(id string) {
		index[id], low[id] = curr, curr
		curr++
		stack = append(stack, id)
		onStack[id] = true

		for _, e := range out[id] {
			if _, seen := index[e.Target]; !seen {
				strongconnect(e.Target)
				if low[e.Target] < low[id] {
					low[id] = low[e.Target]
				}
			} else if onStack[e.Target] {
				if index[e.Target] < low[id] {
					low[id] = index[e.Target]
				}
			}
		}

		if low[id] == index[id] {
			var component []string
			for {
				top := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[top] = false
				component = append(component, top)
				if top == id {
					break
				}
			}
			if len(component) > 1 {
				sccs = append(sccs, component)
			}
		}
	}

	for i := range g.Nodes {
		if _, seen := index[g.Nodes[i].ID]; !seen {
			strongconnect(g.Nodes[i].ID)
		}
	}

	for _, scc := range sccs {
		hasCondition := false
		for _, id := range scc {
			if byID[id].Type == "condition" {
				hasCondition = true
				break
			}
		}
		if !hasCondition {
			return fmt.Errorf("cycle through nodes %v has no condition node (loops must pass through a condition to guarantee termination)", scc)
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
	// runScript 执行 script 节点：一次性隔离沙箱跑完返回上下文数据（真实现 =
	// ScriptRuntime.TestRun，见 runScriptForAgent）；引擎本体不触沙箱，测试注桩。
	runScript func(ctx context.Context, node *agentGraphNode, vars map[string]map[string]string) (map[string]any, error)
	// loadSubflowGraph 加载子图（subflow 节点引用的目标应用图）：
	// 返回目标应用名与解析后的图；目标不存在/停用返回错误。
	// 真实现 = appRepo.GetEntityByID + parseAgentGraph；测试注桩。
	loadSubflowGraph func(ctx context.Context, appID uint64) (appName string, graph *agentGraph, err error)
	// onStep 节点执行阶段回调（node_start / node_end），SSE 流式观感用；nil = 不推。
	// 推帧失败（订阅端慢）不影响执行——回调实现方自己保证尽力而为语义。
	onStep func(phase, nodeID, summary, status string)
}

// agentNodeOutcome 单节点在一个执行波次内的完整结果。
// 波内并发执行时各 goroutine 只写本地 outcome，合并阶段统一落账（vars/trace/usage），
// 避免 Go map 并发写。
type agentNodeOutcome struct {
	step      agentTraceStep
	usage     agentUsage
	outputs   map[string]string   // 写回 vars[node.ID] 的命名空间（nil = 不写）
	nextEdges []*agentGraphEdge   // 选定的后续边（end = nil；parallel = 全部出边）
	endOutput *string             // end 节点的产出（非 end = nil；并行多 end 按序首个胜出）
	branch    string              // condition 的分支判定（"true"/"false"）
	err       error
}

// executeAgentNodeInWave 执行单个节点并产出本地结果。
// 波内并发安全：只读共享 vars（渲染模板/条件求值），一切产出写本地 outcome，
// 由合并阶段统一写回。
func executeAgentNodeInWave(ctx context.Context, node *agentGraphNode, vars map[string]map[string]string, deps *agentEngineDeps, inputCount int) *agentNodeOutcome {
	return executeAgentNodeWithDepth(ctx, node, vars, deps, inputCount, agentMaxSubflowDepth)
}

// executeAgentNodeWithDepth 带 subflow 深度预算的节点执行本体。
func executeAgentNodeWithDepth(ctx context.Context, node *agentGraphNode, vars map[string]map[string]string, deps *agentEngineDeps, inputCount, subflowDepthRemaining int) *agentNodeOutcome {
	stepStart := time.Now()
	oc := &agentNodeOutcome{}
	fail := func(err error) *agentNodeOutcome {
		oc.err = err
		return oc
	}

	var summary string

	switch node.Type {
	case "start":
		summary = fmt.Sprintf("%d input variable(s)", inputCount)

	case "knowledge":
		query, rerr := renderAgentTemplate(agentDataString(node.Data, "query"), vars)
		if rerr != nil {
			return fail(rerr)
		}
		contextText, hits, kerr := deps.searchKnowledge(ctx, node, query)
		if kerr != nil {
			return fail(kerr)
		}
		oc.outputs = map[string]string{"context": contextText, "hits": strconv.Itoa(hits)}
		summary = fmt.Sprintf("hits=%d, %d chars", hits, len(contextText))

	case "llm":
		system, rerr := renderAgentTemplate(agentDataString(node.Data, "systemPrompt"), vars)
		if rerr != nil {
			return fail(rerr)
		}
		user, rerr := renderAgentTemplate(agentDataString(node.Data, "userPrompt"), vars)
		if rerr != nil {
			return fail(rerr)
		}
		if strings.TrimSpace(user) == "" {
			return fail(fmt.Errorf("llm node %q has empty userPrompt after rendering", node.ID))
		}
		answer, llmUsage, lerr := deps.runLLM(ctx, node, system, user)
		if lerr != nil {
			return fail(lerr)
		}
		oc.usage = llmUsage
		oc.outputs = map[string]string{"output": answer}
		summary = truncateRunes(answer, 120)

	case "http":
		status, body, contentType, herr := runAgentHTTPNode(ctx, node, vars)
		if herr != nil {
			return fail(herr)
		}
		oc.outputs = map[string]string{
			"status":      status,
			"body":        body,
			"contentType": contentType,
		}
		summary = fmt.Sprintf("status=%s, %d bytes", status, len(body))

	case "condition":
		approved, cerr := evalAgentCondition(agentDataString(node.Data, "expression"), vars)
		if cerr != nil {
			return fail(cerr)
		}
		if approved {
			oc.branch = "true"
		} else {
			oc.branch = "false"
		}
		summary = fmt.Sprintf("branch=%t", approved)

	case "script":
		if deps.runScript == nil {
			return fail(fmt.Errorf("script node %q unavailable: script runtime not wired", node.ID))
		}
		data, serr := deps.runScript(ctx, node, vars)
		if serr != nil {
			return fail(serr)
		}
		oc.outputs = agentScriptOutputs(data)
		summary = agentScriptSummary(oc.outputs)

	case "parallel":
		summary = "pass-through (fan-out via outgoing edges)"

	case "subflow":
		if deps.loadSubflowGraph == nil {
			return fail(fmt.Errorf("subflow node %q unavailable: subflow loader not wired", node.ID))
		}
		if subflowDepthRemaining <= 0 {
			return fail(fmt.Errorf("subflow node %q exceeds max subflow depth %d (recursive reference?)", node.ID, agentMaxSubflowDepth))
		}
		subAppID := agentDataUint32(node.Data, "appId")
		if subAppID == 0 {
			return fail(fmt.Errorf("subflow node %q has no appId configured", node.ID))
		}
		subName, subGraph, lerr := deps.loadSubflowGraph(ctx, subAppID)
		if lerr != nil {
			return fail(lerr)
		}
		// 递归执行子图：输入 = inputMapping 渲染后的键值（键 = 子 start 变量名，值 = 父模板）；
		// 深度预算逐层递减（防 A 引 B、B 引 A 的递归爆炸）。
		subInput := map[string]string{}
		for _, raw := range agentDataSlice(node.Data, "inputMapping") {
			pair, _ := raw.(map[string]any)
			key, _ := pair["name"].(string)
			if key == "" {
				continue
			}
			val, rerr := renderAgentTemplate(agentDataString(pair, "value"), vars)
			if rerr != nil {
				return fail(rerr)
			}
			subInput[key] = val
		}
		subOutput, subTrace, subUsage, subErr := executeAgentGraphInner(ctx, subGraph, subInput, deps, subflowDepthRemaining-1)
		if subErr != nil {
			return fail(fmt.Errorf("subflow %q (%s): %w", node.ID, subName, subErr))
		}
		oc.usage = subUsage
		oc.outputs = map[string]string{"output": subOutput}
		summary = fmt.Sprintf("subflow %q done, %d steps", subName, len(subTrace))

	case "end":
		outText, rerr := renderAgentTemplate(agentDataString(node.Data, "output"), vars)
		if rerr != nil {
			return fail(rerr)
		}
		s := outText
		oc.endOutput = &s
		summary = "done"

	default:
		return fail(fmt.Errorf("unknown type %q", node.Type))
	}

	oc.step = agentTraceStep{
		NodeID: node.ID, Type: node.Type, Status: "ok",
		Ms: time.Since(stepStart).Milliseconds(), Summary: summary,
		Usage: func() *agentUsage {
			u := oc.usage
			if u.Total == 0 {
				return nil
			}
			return &u
		}(),
	}
	return oc
}

// executeAgentGraph 从 start 走图到 end（frontier 波次模型），返回产出、轨迹与累计用量。
// 对外入口（子图深度默认 agentMaxSubflowDepth）。
func executeAgentGraph(
	ctx context.Context,
	g *agentGraph,
	input map[string]string,
	deps *agentEngineDeps,
) (output string, trace []agentTraceStep, usage agentUsage, err error) {
	return executeAgentGraphInner(ctx, g, input, deps, agentMaxSubflowDepth)
}

// executeAgentGraphInner frontier 波次执行（可递归）：维护一个待执行节点集合
// （frontier），每轮波次把集合内全部节点并发执行（各 goroutine 只读共享 vars、写本地
// outcome，WaitGroup 栅栏后统一合并落账——vars 写入只发生在栅栏后的串行合并段，无
// Go map 并发写）。节点产出后选后继：condition 按 branch 取一条、parallel 取全部出边
// （fan-out）、其余取唯一出边；下一波按 id 排序去重后继续（同波次多上游汇聚到同一节点
// 时只入队一次——波栅栏即汇合点）。
//
// 失败语义：波内任一节点失败即整跑失败（一票失败，v1 不做部分失败聚合）；end 到达
// 即终止（并行多 end 按 frontier 排序首个胜出，其余分支同波产出仍落 trace）。
// 步数预算：agentMaxSteps 按节点执行次数计（波内每个节点各计 1），并发 LLM 调用
// 的墙钟收益与预算语义解耦。
// subflowDepthRemaining 子图递归深度预算（每进一层减一，0 = 拒绝继续下钻）。
func executeAgentGraphInner(
	ctx context.Context,
	g *agentGraph,
	input map[string]string,
	deps *agentEngineDeps,
	subflowDepthRemaining int,
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

	frontier := []*agentGraphNode{start}
	totalSteps := 0

	for len(frontier) > 0 && totalSteps < agentMaxSteps {
		// 确定性顺序：按节点 id 排序（trace/SSE 帧序可复现，并行多 end 首个胜出也可预期）
		sort.Slice(frontier, func(i, j int) bool { return frontier[i].ID < frontier[j].ID })

		if deps.onStep != nil {
			for _, node := range frontier {
				deps.onStep("node_start", node.ID, "", "")
			}
		}

		// 波内并发执行：各 goroutine 只读 vars、写本地 outcome。
		// deps 的实现须并发安全（entClient/openai 客户端线程安全；脚本工具走
		// 引擎 execMu 串行；TestRun 沙箱一次性隔离）。
		outcomes := make([]*agentNodeOutcome, len(frontier))
		var wg sync.WaitGroup
		for i := range frontier {
			wg.Add(1)
			go func(i int, node *agentGraphNode) {
				defer wg.Done()
				outcomes[i] = executeAgentNodeWithDepth(ctx, node, vars, deps, len(input), subflowDepthRemaining)
			}(i, frontier[i])
		}
		wg.Wait()

		// 合并阶段（串行）：写 vars / 累计 usage / 追加 trace / 推 SSE / 选后继。
		nextSet := map[string]*agentGraphNode{}
		nextOrder := []string{}
		for i, node := range frontier {
			oc := outcomes[i]
			if oc.err != nil {
				trace = append(trace, agentTraceStep{
					NodeID: node.ID, Type: node.Type, Status: "error",
					Ms: oc.step.Ms, Error: oc.err.Error(),
				})
				return "", trace, usage, agentStepError(node, oc.err)
			}
			totalSteps++
			if oc.outputs != nil {
				vars[node.ID] = oc.outputs
			}
			usage.add(oc.usage)
			trace = append(trace, oc.step)
			if deps.onStep != nil {
				deps.onStep("node_end", node.ID, oc.step.Summary, "ok")
			}

			if oc.endOutput != nil {
				// 首个 end 到达即终止（同波其他分支的产出已落 trace，但后继不再执行）
				return *oc.endOutput, trace, usage, nil
			}

			// 选后继：condition 按 branch 取；parallel/其余取全部出边
			for _, e := range out[node.ID] {
				if node.Type == "condition" && e.Branch != oc.branch {
					continue
				}
				target := byID[e.Target]
				if target == nil {
					return "", trace, usage, fmt.Errorf("edge target %q vanished", e.Target)
				}
				if _, queued := nextSet[target.ID]; !queued {
					nextSet[target.ID] = target
					nextOrder = append(nextOrder, target.ID)
				}
			}
		}

		// 组装下一波（按 id 排序保持确定性）
		frontier = frontier[:0]
		sort.Strings(nextOrder)
		for _, id := range nextOrder {
			frontier = append(frontier, nextSet[id])
		}
	}

	if len(frontier) > 0 {
		return "", trace, usage, fmt.Errorf(
			"execution exceeded %d node steps (loop budget; reduce cycle iterations or simplify the graph)",
			agentMaxSteps)
	}
	// frontier 清空但未触 end：某非 end 节点无出边（校验应已拦，防御分支）
	return "", trace, usage, fmt.Errorf("all branches terminated without reaching an end node")
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

func agentDataUint32(data map[string]any, key string) uint64 {
	if data == nil {
		return 0
	}
	switch v := data[key].(type) {
	case float64:
		if v <= 0 {
			return 0
		}
		return uint64(v)
	case string:
		n, _ := strconv.ParseUint(v, 10, 64)
		return n
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
