package service

import (
	"context"
	"fmt"
	"strings"
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
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAgentGraph(agentTestGraph(t, tc.raw))
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("validate error = %v, want contains %q", err, tc.wantErr)
			}
		})
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
