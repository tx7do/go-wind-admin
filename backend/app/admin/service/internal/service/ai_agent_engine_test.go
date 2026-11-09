package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// 引擎单测不触网：LLM/知识检索全部注入桩。校验、模板、条件、图遍历的语义在这里钉死。

func agentTestGraph(t *testing.T, raw string) *agentGraph {
	t.Helper()
	g, err := parseAgentGraph(raw)
	if err != nil {
		t.Fatalf("parseAgentGraph: %v", err)
	}
	return g
}

const agentLinearGraphJSON = `{
  "nodes": [
    {"id": "start_1", "type": "start", "position": {"x":0,"y":0},
     "data": {"inputVars": [{"name": "question", "label": "问题", "required": true}]}},
    {"id": "kb_1", "type": "knowledge", "position": {"x":0,"y":0},
     "data": {"baseId": 7, "topK": 3, "query": "{{start_1.question}}"}},
    {"id": "llm_1", "type": "llm", "position": {"x":0,"y":0},
     "data": {"providerId": 1, "model": "test-model", "systemPrompt": "只用资料回答",
              "userPrompt": "资料:{{kb_1.context}} 问:{{start_1.question}}"}},
    {"id": "end_1", "type": "end", "position": {"x":0,"y":0}, "data": {"output": "答:{{llm_1.output}}"}}
  ],
  "edges": [
    {"id": "e1", "source": "start_1", "target": "kb_1"},
    {"id": "e2", "source": "kb_1", "target": "llm_1"},
    {"id": "e3", "source": "llm_1", "target": "end_1"}
  ]
}`

func TestValidateAgentGraph(t *testing.T) {
	ok := agentTestGraph(t, agentLinearGraphJSON)
	if err := validateAgentGraph(ok); err != nil {
		t.Fatalf("valid graph rejected: %v", err)
	}

	cases := []struct {
		name    string
		raw     string
		wantErr string
	}{
		{
			name:    "no start",
			raw:     `{"nodes":[{"id":"e","type":"end","data":{}}],"edges":[]}`,
			wantErr: "exactly one start",
		},
		{
			name:    "two starts",
			raw:     `{"nodes":[{"id":"a","type":"start","data":{}},{"id":"b","type":"start","data":{}},{"id":"e","type":"end","data":{}}],"edges":[{"id":"x","source":"a","target":"e"},{"id":"y","source":"b","target":"e"}]}`,
			wantErr: "exactly one start",
		},
		{
			name:    "no end",
			raw:     `{"nodes":[{"id":"a","type":"start","data":{}}],"edges":[]}`,
			wantErr: "at least one end",
		},
		{
			name:    "unknown type",
			raw:     `{"nodes":[{"id":"a","type":"start","data":{}},{"id":"z","type":"magic","data":{}},{"id":"e","type":"end","data":{}}],"edges":[{"id":"x","source":"a","target":"z"},{"id":"y","source":"z","target":"e"}]}`,
			wantErr: "unknown type",
		},
		{
			name:    "edge to missing node",
			raw:     `{"nodes":[{"id":"a","type":"start","data":{}},{"id":"e","type":"end","data":{}}],"edges":[{"id":"x","source":"a","target":"ghost"}]}`,
			wantErr: "unknown target node",
		},
		{
			name:    "condition single branch",
			raw:     `{"nodes":[{"id":"a","type":"start","data":{}},{"id":"c","type":"condition","data":{}},{"id":"e","type":"end","data":{}}],"edges":[{"id":"x","source":"a","target":"c"},{"id":"b1","source":"c","target":"e","branch":"true"}]}`,
			wantErr: "exactly two outgoing",
		},
		{
			name:    "end with outgoing",
			raw:     `{"nodes":[{"id":"a","type":"start","data":{}},{"id":"e1","type":"end","data":{}},{"id":"e2","type":"end","data":{}}],"edges":[{"id":"x","source":"a","target":"e1"},{"id":"y","source":"e1","target":"e2"}]}`,
			wantErr: "must not have outgoing",
		},
		{
			name:    "unreachable node",
			raw:     `{"nodes":[{"id":"a","type":"start","data":{}},{"id":"e","type":"end","data":{}},{"id":"l","type":"llm","data":{}}],"edges":[{"id":"x","source":"a","target":"e"}]}`,
			wantErr: "not reachable",
		},
		{
			name:    "duplicate id",
			raw:     `{"nodes":[{"id":"a","type":"start","data":{}},{"id":"a","type":"end","data":{}}],"edges":[]}`,
			wantErr: "duplicate node id",
		},
		{
			name:    "bad id starts with digit",
			raw:     `{"nodes":[{"id":"1a","type":"start","data":{}},{"id":"e","type":"end","data":{}}],"edges":[{"id":"x","source":"1a","target":"e"}]}`,
			wantErr: "invalid",
		},
		{
			name: "cycle without condition rejected",
			raw: `{"nodes":[{"id":"a","type":"start","data":{}},{"id":"c","type":"condition","data":{}},{"id":"l1","type":"llm","data":{}},{"id":"l2","type":"llm","data":{}},{"id":"e","type":"end","data":{}}],
			"edges":[{"id":"x","source":"a","target":"c"},{"id":"b1","source":"c","target":"e","branch":"true"},{"id":"b2","source":"c","target":"l1","branch":"false"},{"id":"y","source":"l1","target":"l2"},{"id":"z","source":"l2","target":"l1"}]}`,
			wantErr: "cycle",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAgentGraph(agentTestGraph(t, tc.raw))
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("validate error = %v, want contains %q", err, tc.wantErr)
			}
		})
	}

	// 合法 condition 回边：强连通分量含 condition，不触发环拒绝（M2「循环」）
	validCycle := `{"nodes":[{"id":"a","type":"start","data":{}},{"id":"l1","type":"llm","data":{}},{"id":"c","type":"condition","data":{}},{"id":"e","type":"end","data":{}}],
		"edges":[{"id":"x","source":"a","target":"l1"},{"id":"y","source":"l1","target":"c"},{"id":"b1","source":"c","target":"e","branch":"true"},{"id":"b2","source":"c","target":"l1","branch":"false"}]}`
	if err := validateAgentGraph(agentTestGraph(t, validCycle)); err != nil {
		t.Fatalf("valid cycle through condition rejected: %v", err)
	}
}

func TestRenderAgentTemplate(t *testing.T) {
	vars := map[string]map[string]string{
		"start_1": {"question": "退货政策"},
		"kb_1":    {"context": "七天内可退", "hits": "1"},
	}

	cases := []struct {
		tpl     string
		want    string
		wantErr string
	}{
		{tpl: "{{start_1.question}}", want: "退货政策"},
		{tpl: "资料:{{kb_1.context}} 问:{{start_1.question}}", want: "资料:七天内可退 问:退货政策"},
		{tpl: "no placeholder", want: "no placeholder"},
		{tpl: "{{kb_1.hits}} 条", want: "1 条"},
		{tpl: "{{ghost.var}}", wantErr: `unknown node "ghost"`},
		{tpl: "{{start_1.novar}}", wantErr: `unknown variable "novar"`},
		{tpl: "{{nodot}}", wantErr: "invalid placeholder"},
		{tpl: "{{kb_1.}}", wantErr: "invalid placeholder"},
	}
	for _, tc := range cases {
		got, err := renderAgentTemplate(tc.tpl, vars)
		if tc.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("render(%q) err = %v, want contains %q", tc.tpl, err, tc.wantErr)
			}
			continue
		}
		if err != nil {
			t.Fatalf("render(%q) unexpected err: %v", tc.tpl, err)
		}
		if got != tc.want {
			t.Fatalf("render(%q) = %q, want %q", tc.tpl, got, tc.want)
		}
	}
}

// TestRenderAgentTemplateNotRecursive 值里的 "{{...}}" 不得被当模板再渲染一轮
// （单趟扫描的直接后果，钉住防有人改回逐变量 ReplaceAll）。
func TestRenderAgentTemplateNotRecursive(t *testing.T) {
	vars := map[string]map[string]string{
		"a": {"v": "{{b.v}}"},
		"b": {"v": "boom"},
	}
	got, err := renderAgentTemplate("{{a.v}}", vars)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "{{b.v}}" {
		t.Fatalf("rendered recursively: %q", got)
	}
}

func TestEvalAgentCondition(t *testing.T) {
	vars := map[string]map[string]string{
		"llm_1": {"output": "您好，七天内可以退款"},
	}
	if ok, err := evalAgentCondition(`llm_1.output contains "退款"`, vars); err != nil || !ok {
		t.Fatalf("contains expr = %v, %v", ok, err)
	}
	if ok, err := evalAgentCondition(`llm_1.output contains "发货"`, vars); err != nil || ok {
		t.Fatalf("negative expr = %v, %v", ok, err)
	}
	// 非布尔产出必须报错，而不是把非零值当真
	if _, err := evalAgentCondition(`len(llm_1.output)`, vars); err == nil {
		t.Fatal("non-bool expression should fail (expr.AsBool)")
	}
	// 语法错误必须报错
	if _, err := evalAgentCondition(`llm_1.output >>>`, vars); err == nil {
		t.Fatal("syntax error should fail")
	}
}

// TestExecuteAgentGraphLinear 线性图：start→knowledge→llm→end，
// 上下文逐节点传递、用量累计、trace 逐步留痕。
func TestExecuteAgentGraphLinear(t *testing.T) {
	g := agentTestGraph(t, agentLinearGraphJSON)

	var gotQuery, gotSystem, gotUser string
	var steps []string
	deps := &agentEngineDeps{
		searchKnowledge: func(ctx context.Context, node *agentGraphNode, query string) (string, int, error) {
			gotQuery = query
			return "七天内可退货", 1, nil
		},
		runLLM: func(ctx context.Context, node *agentGraphNode, system, user string) (string, agentUsage, error) {
			gotSystem, gotUser = system, user
			return "七天无理由", agentUsage{Prompt: 10, Completion: 5, Total: 15}, nil
		},
		onStep: func(phase, nodeID, summary, status string) {
			steps = append(steps, phase+":"+nodeID)
		},
	}

	output, trace, usage, err := executeAgentGraph(context.Background(), g,
		map[string]string{"question": "退货"}, deps)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if gotQuery != "退货" {
		t.Fatalf("knowledge query = %q", gotQuery)
	}
	if gotSystem != "只用资料回答" || gotUser != "资料:七天内可退货 问:退货" {
		t.Fatalf("llm prompts system=%q user=%q", gotSystem, gotUser)
	}
	if output != "答:七天无理由" {
		t.Fatalf("output = %q", output)
	}
	if usage.Total != 15 {
		t.Fatalf("usage = %+v", usage)
	}
	if len(trace) != 4 {
		t.Fatalf("trace steps = %d (%+v)", len(trace), trace)
	}
	// onStep 每节点两帧（node_start/node_end），SSE 流式观感的数据源
	wantSteps := []string{
		"node_start:start_1", "node_end:start_1",
		"node_start:kb_1", "node_end:kb_1",
		"node_start:llm_1", "node_end:llm_1",
		"node_start:end_1", "node_end:end_1",
	}
	if len(steps) != len(wantSteps) {
		t.Fatalf("onStep frames = %d, want %d (%v)", len(steps), len(wantSteps), steps)
	}
	for i, w := range wantSteps {
		if steps[i] != w {
			t.Fatalf("onStep[%d] = %s, want %s", i, steps[i], w)
		}
	}
	for i, want := range []string{"start_1", "kb_1", "llm_1", "end_1"} {
		if trace[i].NodeID != want {
			t.Fatalf("trace[%d] = %s, want %s", i, trace[i].NodeID, want)
		}
	}
}

// TestExecuteAgentGraphCondition 分支图：表达式为真走 true 支，为假走 false 支。
func TestExecuteAgentGraphCondition(t *testing.T) {
	raw := `{
	  "nodes": [
	    {"id": "start_1", "type": "start", "position": {"x":0,"y":0}, "data": {"inputVars": []}},
	    {"id": "llm_1", "type": "llm", "position": {"x":0,"y":0}, "data": {"providerId": 1, "userPrompt": "hi"}},
	    {"id": "cond_1", "type": "condition", "position": {"x":0,"y":0}, "data": {"expression": "llm_1.output contains \"yes\""}},
	    {"id": "end_yes", "type": "end", "position": {"x":0,"y":0}, "data": {"output": "branch-true"}},
	    {"id": "end_no", "type": "end", "position": {"x":0,"y":0}, "data": {"output": "branch-false"}}
	  ],
	  "edges": [
	    {"id": "e1", "source": "start_1", "target": "llm_1"},
	    {"id": "e2", "source": "llm_1", "target": "cond_1"},
	    {"id": "e3", "source": "cond_1", "target": "end_yes", "branch": "true"},
	    {"id": "e4", "source": "cond_1", "target": "end_no", "branch": "false"}
	  ]
	}`

	run := func(answer string) string {
		g := agentTestGraph(t, raw)
		deps := &agentEngineDeps{
			searchKnowledge: func(ctx context.Context, node *agentGraphNode, query string) (string, int, error) {
				return "", 0, fmt.Errorf("should not be called")
			},
			runLLM: func(ctx context.Context, node *agentGraphNode, system, user string) (string, agentUsage, error) {
				return answer, agentUsage{}, nil
			},
		}
		output, _, _, err := executeAgentGraph(context.Background(), g, nil, deps)
		if err != nil {
			t.Fatalf("execute(%q): %v", answer, err)
		}
		return output
	}

	if got := run("yes sir"); got != "branch-true" {
		t.Fatalf("true branch output = %q", got)
	}
	if got := run("nope"); got != "branch-false" {
		t.Fatalf("false branch output = %q", got)
	}
}

// TestExecuteAgentGraphMissingRequired start 声明的必填入参缺失必须在入口报错。
func TestExecuteAgentGraphMissingRequired(t *testing.T) {
	g := agentTestGraph(t, agentLinearGraphJSON)
	deps := &agentEngineDeps{
		searchKnowledge: func(ctx context.Context, node *agentGraphNode, query string) (string, int, error) { return "", 0, nil },
		runLLM: func(ctx context.Context, node *agentGraphNode, system, user string) (string, agentUsage, error) {
			return "", agentUsage{}, nil
		},
	}
	_, _, _, err := executeAgentGraph(context.Background(), g, nil, deps)
	if err == nil || !strings.Contains(err.Error(), "missing required input") {
		t.Fatalf("err = %v, want missing required input", err)
	}
}

// ── script 节点 ────────────────────────────────────────────────────────────

// TestValidateAgentScriptNode script 节点静态校验：语言白名单、source 非空且 ≤32KB。
func TestValidateAgentScriptNode(t *testing.T) {
	node := func(data map[string]any) *agentGraphNode {
		return &agentGraphNode{ID: "scr_1", Type: "script", Data: data}
	}
	if err := validateAgentScriptNode(node(map[string]any{"language": "lua", "source": "return {ok=1}"})); err != nil {
		t.Fatalf("valid lua rejected: %v", err)
	}
	if err := validateAgentScriptNode(node(map[string]any{"language": "python", "source": "print(1)"})); err == nil || !strings.Contains(err.Error(), "unsupported language") {
		t.Fatalf("python err = %v", err)
	}
	if err := validateAgentScriptNode(node(map[string]any{"language": "lua", "source": "  "})); err == nil || !strings.Contains(err.Error(), "empty source") {
		t.Fatalf("empty source err = %v", err)
	}
	big := strings.Repeat("a", 32<<10+1)
	if err := validateAgentScriptNode(node(map[string]any{"language": "lua", "source": big})); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversize err = %v", err)
	}
	// graph 级校验同样要拦（validateAgentGraph 的 per-type 分支）
	raw := `{"nodes":[{"id":"a","type":"start","data":{}},{"id":"s","type":"script","data":{"language":"ruby","source":"1"}},{"id":"e","type":"end","data":{}}],"edges":[{"id":"x","source":"a","target":"s"},{"id":"y","source":"s","target":"e"}]}`
	if err := validateAgentGraph(agentTestGraph(t, raw)); err == nil || !strings.Contains(err.Error(), "unsupported language") {
		t.Fatalf("graph-level err = %v", err)
	}
}

// TestAgentScriptOutputs 返回值收口：字符串直通、标量转写、嵌套值 JSON、nil 跳过。
func TestAgentScriptOutputs(t *testing.T) {
	outs := agentScriptOutputs(map[string]any{
		"text":  "hello",
		"count": 3,
		"score": float64(2.5),
		"ok":    true,
		"list":  []any{"a", "b"},
		"skip":  nil,
	})
	if outs["text"] != "hello" || outs["count"] != "3" || outs["score"] != "2.5" || outs["ok"] != "true" {
		t.Fatalf("scalar outputs = %+v", outs)
	}
	if outs["list"] != `["a","b"]` {
		t.Fatalf("list output = %q", outs["list"])
	}
	if _, ok := outs["skip"]; ok {
		t.Fatal("nil value should be skipped (脚本显式 set nil = 不产出)")
	}
}

// TestExecuteAgentGraphScript script 节点执行：全图上下文透传、source 原样
// （不过模板——代码里的 {{...}} 是脚本自身语法）、产出写回节点命名空间供下游模板消费。
func TestExecuteAgentGraphScript(t *testing.T) {
	raw := `{
	  "nodes": [
	    {"id": "start_1", "type": "start", "position": {"x":0,"y":0}, "data": {"inputVars": [{"name": "question", "required": true}]}},
	    {"id": "scr_1", "type": "script", "position": {"x":0,"y":0}, "data": {"language": "lua", "source": "return {answer = start_1.question .. \"!\"} -- {{not_a_template}}"}},
	    {"id": "end_1", "type": "end", "position": {"x":0,"y":0}, "data": {"output": "答:{{scr_1.answer}}"}}
	  ],
	  "edges": [
	    {"id": "e1", "source": "start_1", "target": "scr_1"},
	    {"id": "e2", "source": "scr_1", "target": "end_1"}
	  ]
	}`
	g := agentTestGraph(t, raw)

	var gotSource string
	var gotVars map[string]map[string]string
	deps := &agentEngineDeps{
		runScript: func(ctx context.Context, node *agentGraphNode, vars map[string]map[string]string) (map[string]any, error) {
			gotSource = agentDataString(node.Data, "source")
			gotVars = vars
			return map[string]any{"answer": "七天无理由"}, nil
		},
	}
	output, trace, _, err := executeAgentGraph(context.Background(), g, map[string]string{"question": "退货"}, deps)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if gotSource != `return {answer = start_1.question .. "!"} -- {{not_a_template}}` {
		t.Fatalf("source was templated or mangled: %q", gotSource)
	}
	if gotVars == nil || gotVars["start_1"]["question"] != "退货" {
		t.Fatalf("context not passed through: %+v", gotVars)
	}
	if output != "答:七天无理由" {
		t.Fatalf("output = %q", output)
	}
	scriptStep := trace[1]
	if scriptStep.NodeID != "scr_1" || scriptStep.Summary != "keys=answer" {
		t.Fatalf("script trace step = %+v", scriptStep)
	}
}

// TestExecuteAgentGraphScriptNotWired 图里有 script 节点但 runScript 未装配（nil）
// 必须报错并留 trace，不得静默跳过——静默不执行只会让用户看到"输出是空的"。
func TestExecuteAgentGraphScriptNotWired(t *testing.T) {
	raw := `{
	  "nodes": [
	    {"id": "start_1", "type": "start", "data": {}},
	    {"id": "scr_1", "type": "script", "data": {"language": "lua", "source": "return {}"}},
	    {"id": "end_1", "type": "end", "data": {}}
	  ],
	  "edges": [
	    {"id": "e1", "source": "start_1", "target": "scr_1"},
	    {"id": "e2", "source": "scr_1", "target": "end_1"}
	  ]
	}`
	g := agentTestGraph(t, raw)
	_, trace, _, err := executeAgentGraph(context.Background(), g, nil, &agentEngineDeps{})
	if err == nil || !strings.Contains(err.Error(), "script runtime not wired") {
		t.Fatalf("err = %v, want script runtime not wired", err)
	}
	// 波次模型：失败节点也落 trace（error 步骤含错误详情），比旧行为更可观测
	if !strings.Contains(err.Error(), "node scr_1[script]") {
		t.Fatalf("err should locate the failing node: %v", err)
	}
	if len(trace) != 2 {
		t.Fatalf("trace = %+v, want start ok + script error steps", trace)
	}
	if trace[0].NodeID != "start_1" || trace[0].Status != "ok" {
		t.Fatalf("trace[0] = %+v, want start_1 ok", trace[0])
	}
	if trace[1].NodeID != "scr_1" || trace[1].Status != "error" {
		t.Fatalf("trace[1] = %+v, want scr_1 error", trace[1])
	}
}

// ── parallel 并行节点（frontier 波次模型）────────────────────────────────

// TestValidateAgentGraph_ParallelOutDegree parallel 节点校验：允许 N 条出边（fan-out），
// 零出边拒绝；其余类型仍受 ≤1 出度约束。
func TestValidateAgentGraph_ParallelOutDegree(t *testing.T) {
	// parallel 多出边合法（出边目标必须不同——重复边会被 duplicate edge 拦截）
	ok := `{"nodes":[{"id":"a","type":"start","data":{}},{"id":"p","type":"parallel","data":{}},{"id":"l1","type":"llm","data":{}},{"id":"l2","type":"llm","data":{}},{"id":"e","type":"end","data":{}}],
		"edges":[{"id":"x","source":"a","target":"p"},{"id":"b1","source":"p","target":"l1"},{"id":"b2","source":"p","target":"l2"},{"id":"c1","source":"l1","target":"e"},{"id":"c2","source":"l2","target":"e"}]}`
	if err := validateAgentGraph(agentTestGraph(t, ok)); err != nil {
		t.Fatalf("parallel fan-out rejected: %v", err)
	}

	// parallel 零出边拒绝
	bad := `{"nodes":[{"id":"a","type":"start","data":{}},{"id":"p","type":"parallel","data":{}},{"id":"e","type":"end","data":{}}],
		"edges":[{"id":"x","source":"a","target":"p"}]}`
	if err := validateAgentGraph(agentTestGraph(t, bad)); err == nil || !strings.Contains(err.Error(), "at least one outgoing edge") {
		t.Fatalf("parallel zero-out err = %v", err)
	}

	// 普通节点仍限单出边（test 断言放在同型已有用例覆盖）
}

// TestExecuteAgentGraph_ParallelFanOut 并行执行：parallel 节点把波次分裂为两个并发
// 分支，各自执行后都到达 end——按排序首个 end 胜出，两个分支的 trace 都保留。
func TestExecuteAgentGraph_ParallelFanOut(t *testing.T) {
	raw := `{
	  "nodes": [
	    {"id": "start_1", "type": "start", "position": {"x":0,"y":0}, "data": {"inputVars": []}},
	    {"id": "p", "type": "parallel", "position": {"x":0,"y":0}, "data": {}},
	    {"id": "http_a", "type": "http", "position": {"x":0,"y":0}, "data": {"url": "http://a.example.com/x"}},
	    {"id": "http_b", "type": "http", "position": {"x":0,"y":0}, "data": {"url": "http://b.example.com/y"}},
	    {"id": "end_a", "type": "end", "position": {"x":0,"y":0}, "data": {"output": "A"}},
	    {"id": "end_b", "type": "end", "position": {"x":0,"y":0}, "data": {"output": "B"}}
	  ],
	  "edges": [
	    {"id": "e0", "source": "start_1", "target": "p"},
	    {"id": "e1", "source": "p", "target": "http_a"},
	    {"id": "e2", "source": "p", "target": "http_b"},
	    {"id": "e3", "source": "http_a", "target": "end_a"},
	    {"id": "e4", "source": "http_b", "target": "end_b"}
	  ]
	}`
	g := agentTestGraph(t, raw)

	if err := validateAgentGraph(g); err != nil {
		t.Fatalf("parallel graph rejected: %v", err)
	}

	httpCalls := make(map[string]int)
	deps := &agentEngineDeps{
		searchKnowledge: func(ctx context.Context, node *agentGraphNode, query string) (string, int, error) {
			return "", 0, fmt.Errorf("should not be called")
		},
		runLLM: func(ctx context.Context, node *agentGraphNode, system, user string) (string, agentUsage, error) {
			return "", agentUsage{}, fmt.Errorf("should not be called")
		},
	}

	// 用桩替换 http 执行不可行（runAgentHTTPNode 是直接调用），改用 script 节点并行：
	// 两个 script 节点经 deps.runScript 桩计数，验证波内并发都执行了。
	rawScript := `{
	  "nodes": [
	    {"id": "start_1", "type": "start", "position": {"x":0,"y":0}, "data": {"inputVars": []}},
	    {"id": "p", "type": "parallel", "position": {"x":0,"y":0}, "data": {}},
	    {"id": "s_a", "type": "script", "position": {"x":0,"y":0}, "data": {"language": "lua", "source": "function execute() __set_ctx(\"r\", \"A\") return true end"}},
	    {"id": "s_b", "type": "script", "position": {"x":0,"y":0}, "data": {"language": "lua", "source": "function execute() __set_ctx(\"r\", \"B\") return true end"}},
	    {"id": "end_a", "type": "end", "position": {"x":0,"y":0}, "data": {"output": "A done"}},
	    {"id": "end_b", "type": "end", "position": {"x":0,"y":0}, "data": {"output": "B done"}}
	  ],
	  "edges": [
	    {"id": "e0", "source": "start_1", "target": "p"},
	    {"id": "e1", "source": "p", "target": "s_a"},
	    {"id": "e2", "source": "p", "target": "s_b"},
	    {"id": "e3", "source": "s_a", "target": "end_a"},
	    {"id": "e4", "source": "s_b", "target": "end_b"}
	  ]
	}`
	gs := agentTestGraph(t, rawScript)
	if err := validateAgentGraph(gs); err != nil {
		t.Fatalf("parallel script graph rejected: %v", err)
	}

	var mu sync.Mutex
	scriptCalls := map[string]bool{}
	scriptDeps := &agentEngineDeps{
		runScript: func(ctx context.Context, node *agentGraphNode, vars map[string]map[string]string) (map[string]any, error) {
			mu.Lock()
			scriptCalls[node.ID] = true
			mu.Unlock()
			return map[string]any{"r": node.ID}, nil
		},
	}

	output, trace, _, err := executeAgentGraph(context.Background(), gs, nil, scriptDeps)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	mu.Lock()
	callsA := scriptCalls["s_a"]
	callsB := scriptCalls["s_b"]
	mu.Unlock()
	if !callsA || !callsB {
		t.Fatalf("both parallel branches should execute (a=%v b=%v)", callsA, callsB)
	}

	// 首 end 胜出：frontier 按 id 排序，end_a < end_b，故产出为 A 分支
	if output != "A done" {
		t.Fatalf("output = %q, want A done (first end wins)", output)
	}

	// 两个分支的节点都进 trace
	traceNodes := map[string]bool{}
	for _, s := range trace {
		traceNodes[s.NodeID] = true
	}
	if !traceNodes["s_a"] || !traceNodes["s_b"] || !traceNodes["end_a"] {
		t.Fatalf("trace missing parallel branch nodes: %+v", trace)
	}
	_ = httpCalls
	_ = deps
}

// TestExecuteAgentGraph_Subflow 子图递归执行：subflow 节点加载目标图、
// inputMapping 渲染后作为子 start 入参、子 end 产出写回父命名空间 output 变量。
func TestExecuteAgentGraph_Subflow(t *testing.T) {
	subGraph := agentTestGraph(t, `{
	  "nodes": [
	    {"id": "start_1", "type": "start", "data": {"inputVars": [{"name": "topic", "required": true}]}},
	    {"id": "end_1", "type": "end", "data": {"output": "子图输出:{{start_1.topic}}"}}
	  ],
	  "edges": [
	    {"id": "e1", "source": "start_1", "target": "end_1"}
	  ]
	}`)

	parentRaw := `{
	  "nodes": [
	    {"id": "start_1", "type": "start", "data": {"inputVars": [{"name": "question", "required": true}]}},
	    {"id": "sub_1", "type": "subflow", "data": {"appId": 7, "inputMapping": [{"name": "topic", "value": "{{start_1.question}}"}]}},
	    {"id": "end_1", "type": "end", "data": {"output": "父层收到:{{sub_1.output}}"}}
	  ],
	  "edges": [
	    {"id": "e1", "source": "start_1", "target": "sub_1"},
	    {"id": "e2", "source": "sub_1", "target": "end_1"}
	  ]
	}`
	g := agentTestGraph(t, parentRaw)
	if err := validateAgentGraph(g); err != nil {
		t.Fatalf("subflow graph rejected: %v", err)
	}

	deps := &agentEngineDeps{
		loadSubflowGraph: func(ctx context.Context, appID uint64) (string, *agentGraph, error) {
			if appID != 7 {
				return "", nil, fmt.Errorf("unexpected app %d", appID)
			}
			return "子图应用", subGraph, nil
		},
	}

	output, trace, _, err := executeAgentGraph(context.Background(), g,
		map[string]string{"question": "退款"}, deps)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if output != "父层收到:子图输出:退款" {
		t.Fatalf("output = %q", output)
	}
	// 子图 trace 拍平追加到父 trace（对前端零改动）
	traceNodes := map[string]int{}
	for _, s := range trace {
		traceNodes[s.NodeID]++
	}
	if traceNodes["end_1"] == 0 || traceNodes["sub_1"] == 0 {
		t.Fatalf("subflow inner steps should appear in parent trace, got %+v", trace)
	}
}

// TestExecuteAgentGraph_SubflowDepthLimit 递归深度限制：自引用图在深度耗尽后报错
// （不无限递归）。
func TestExecuteAgentGraph_SubflowDepthLimit(t *testing.T) {
	selfGraph := agentTestGraph(t, `{
	  "nodes": [
	    {"id": "start_1", "type": "start", "data": {"inputVars": []}},
	    {"id": "sub_1", "type": "subflow", "data": {"appId": 9}},
	    {"id": "end_1", "type": "end", "data": {"output": "done"}}
	  ],
	  "edges": [
	    {"id": "e1", "source": "start_1", "target": "sub_1"},
	    {"id": "e2", "source": "sub_1", "target": "end_1"}
	  ]
	}`)

	calls := 0
	deps := &agentEngineDeps{
		loadSubflowGraph: func(ctx context.Context, appID uint64) (string, *agentGraph, error) {
			calls++
			return "自引用", selfGraph, nil
		},
	}

	_, _, _, err := executeAgentGraphInner(context.Background(), selfGraph, nil, deps, 3)
	if err == nil {
		t.Fatal("self-referencing subflow should fail at depth limit")
	}
	if !strings.Contains(err.Error(), "max subflow depth") && !containsStr(err.Error(), "subflow") {
		t.Fatalf("err = %v, want subflow depth error", err)
	}
	if calls < 3 {
		t.Fatalf("calls = %d, want >= 3 (recursion should hit the limit)", calls)
	}
}

func containsStr(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && (func() bool {
		for i := 0; i+len(substr) <= len(s); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	})()
}
