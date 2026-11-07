# AI Agent 编排（Agent Orchestration）设计文档

> **状态：M1 已落地（2026-10-15，见 §5 分期）**：图执行引擎 + 应用 CRUD + 同步试运行 + 运行台账 + react 画布先行。
> **M2 画布移植已落地**：ele/vben 画布编辑器（vue-flow 版）随 c6504f7a 补齐、d765b75e 真机修正——三端编排齐平。
> M2+ 进展（2026-11-08）：**SSE 流式与 ele/vben 画布早已落地**（后者随 c6504f7a）；
> **HTTP 节点已落地**（后端 + 三端画布，见 §2.1）。剩余（脚本节点、版本与发布、
> 对外 API、循环/子图/并行）未排期，需求驱动。
>
> 立项依据：JeecgBoot 对标时"AI 应用平台（编排）"按品类边界归档，触发条件为需求驱动；
> 2026-10-15 用户明确立项，归档项按条款重开。**这是 AI 模块内的编排能力，不是 OA 工作流**——
> 节点全部是 AI 语义（开始/知识检索/LLM/条件/结束），没有审批/会签/抄送，与
> `workflow-engine-direction` 的"底座不做 OA"定论不相交。

## 1. 目标与边界

回答三个问题：

1. 业务人员把"检索知识库 → 让模型按提示词回答 → 按答案分流"这类流程**画出来**（可视化画布），
   存成一个可反复调用的 **Agent 应用**；
2. 应用被**试运行**时，引擎按图逐节点执行，留**运行台账**（状态/逐节点 trace/token 用量）；
3. 与既有 AI 资产**复用而非重建**：模型调用走 ai_providers（客户端构造/密钥解密共用）、
   知识检索走 ai_knowledge 的 RAG 链路、工具循环复用 `aiToolLoop`、配额与用量记进
   同一条 `sys_ai_usage_logs` 流水（受同一套 AI_TOKENS 配额约束）。

**明确不做什么（M1）**：

| 不做 | 原因 |
| --- | --- |
| OA 语义节点（审批/会签/抄送/人工任务） | 品类边界，见文件头；工作流归独立业务模块 |
| 循环/子图/并行分支 | M1 图是"确定性分流 DAG"，循环是编排器复杂度的大头，需求驱动再上 |
| SSE 流式执行 | M1 同步执行返回完整 trace；流式在 M2（复用 ai_chat_chunk 的事件通道） |
| 应用版本与发布流（draft/published） | M1 单 graph 列即存即得；版本化等出现"改坏了想回滚"的真实诉求再上 |
| 对外 API（AK/SK 调用 Agent） | M1 只在管理台内运行；开放 API 等 M2+ |

## 2. 图 DSL（v1）

`sys_ai_agent_apps.graph` 存 JSON 文本：

```json
{
  "nodes": [
    {"id": "start_1", "type": "start", "position": {"x": 80, "y": 200},
     "data": {"inputVars": [{"name": "question", "label": "问题", "required": true}]}},
    {"id": "kb_1", "type": "knowledge", "position": {...},
     "data": {"baseId": 1, "topK": 4, "query": "{{start_1.question}}"}},
    {"id": "llm_1", "type": "llm", "position": {...},
     "data": {"providerId": 1, "model": "deepseek-chat",
              "systemPrompt": "你是客服助手，只依据资料回答",
              "userPrompt": "资料：\n{{kb_1.context}}\n\n问题：{{start_1.question}}",
              "temperature": 0.3}},
    {"id": "cond_1", "type": "condition", "position": {...},
     "data": {"expression": "llm_1.output contains \"退款\""}},
    {"id": "end_1", "type": "end", "position": {...}, "data": {"output": "{{llm_1.output}}"}}
  ],
  "edges": [
    {"id": "e1", "source": "start_1", "target": "kb_1"},
    {"id": "e4", "source": "cond_1", "target": "end_1", "branch": "true"},
    {"id": "e5", "source": "cond_1", "target": "end_2", "branch": "false"}
  ]
}
```

### 2.1 节点集（M1 五种 + M2 http）

| type | data 键 | 写入上下文 |
| --- | --- | --- |
| `start` | `inputVars: [{name,label,required}]`（运行入参声明） | 每个输入变量按名直挂自己（`start_1.question`） |
| `knowledge` | `baseId, topK(默认4), query(模板)` | `context`（命中片段拼接）、`hits`（命中条数） |
| `llm` | `providerId, model, systemPrompt?(模板), userPrompt(模板), temperature?` | `output`（最终答案文本） |
| `http` **（M2，2026-11-08）** | `url`(必填,模板), `method`(白名单 GET/POST/PUT/DELETE/PATCH/HEAD, 默认 GET), `headers`(`[{key,value}]` 或 map，键值均可模板), `body`(模板，非 GET/HEAD), `timeoutSeconds`(1-60, 默认 15) | `status`（状态码）、`body`（响应体，读上限 64KB、入上下文截断 8KB）、`contentType` |
| `condition` | `expression`（[expr-lang](https://expr-lang.org) 语法，变量即节点命名空间） | 无（分流由带 `branch` 的出边承接） |
| `end` | `output`(模板，可空) | 运行产出 = 到达的 end 节点渲染值 |

**http 节点的 SSRF 防线（2026-11-08）**：与 RenderPdf（`print_template_sanitize.go`）同源——
scheme 白名单（http/https）、主机字面量/后缀判定（localhost/.local/.internal）、解析后逐 IP
复核（私网/回环/链路本地/CGNAT 段）；`ai_agent_http_node.go` 的 `validateAgentOutboundURL`。
与那条线同样存在解析与请求两次解析之间的 TOCTOU 窗口（DNS 重绑定），强度一致、接受同一权衡。
静态校验（Run 前）：url 必填、method 白名单、timeoutSeconds 钳制；校验不过 400、不建运行行。

### 2.2 模板与表达式

- **模板**：`{{节点id.变量}}` 单趟替换（与 webhook 载荷模板同一套防递归思路），
  值按字符串插值；引用未知的 `节点id.变量` 在该节点执行时报错进 trace，不静默变空串。
- **条件表达式**：走 [expr-lang/expr v1.17.8](https://github.com/expr-lang/expr)——与 go-scripts
  家族 expr 引擎同语言同版本。节点命名空间作为 env（`llm_1.output`），表达式必须产出布尔。
  不经 go-scripts 的 engine 包装：那套 source.Reader/Load 管线面向"存储的脚本"，
  与"每次求值带运行时 env 的一次性表达式"不匹配；语言与版本仍与家族对齐。

### 2.3 校验（Run 前静态校验，不过则 400、不建运行行）

唯一 start；≥1 个 end；节点 id 唯一且匹配 `[a-zA-Z_][a-zA-Z0-9_]{0,63}`；边引用存在的节点；
condition 恰有两条出边且 `branch` 恰为 true/false 各一；其余节点出度 ≤1、end 出度 0；
从 start 可达所有节点；无环（可达性即隐式覆盖——出度受限的图里环必然可达）。

## 3. 执行语义（M1 同步）

1. `Run`：应用未启用 → 400；图解析/校验失败 → 400（**不建运行行**——配置错误，同 webhook 预检
   SKIPPED 的"没尝试过"语义）；
2. 通过后建运行行（RUNNING）→ 引擎从 start 走图：每节点渲染模板→执行→写 trace 步骤
   （`{nodeId,type,status,ms,summary,usage}`）→ 沿出边前进；condition 求值选分支；
   到达 end 终止；步骤上限 50（防表达式写出意外环路时的兜底，正常图到不了）；
3. LLM 节点：`newOpenAIClientForProvider` 构造客户端（密文解密就地、不出 service 层），
   `aiToolLoop` 跑多轮工具调用（内置工具同 chat），model 空时回落 provider 行的 ModelName；
4. 用量：跨节点累计进运行行三列，**逐 LLM 节点**另写一条 `sys_ai_usage_logs`
   （conversation_id=0，`source` 语义上属非对话调用）——配额（FetchTenantTokenQuotaLimit /
   SumTokensByTenantSince）在 Run 入口检查一次，与 chat 同口径；
5. 失败：节点执行错误 → 运行行 FAILED + 已产生的 trace 与用量照记（部分执行可见）；
   引擎错误（图走不通/超出步数）同 FAILED，error 列写原因。

### 3.1 SSE 流式执行（M2 已落地）

Run 保持同步 RPC 语义不变（M1 trace/台账路径原样），流式是**实时观感增强**：

- 新事件 `ai_agent_run`（`sseevent.AgentRun`）：data 为 `AgentRunEvent` protojson——
  `{runId, appId, seq, phase, nodeId, summary, status}`；phase ∈
  `node_start | node_end | error | done`；
- 发布点在引擎 `deps.onStep` 回调（每节点 node_start/node_end 各一帧）与 service
  收尾（done/error 帧）；streamId 与 AI 对话同为发起用户 userId（复用既有用户流）；
- 语义与 chat 同哲学：SSE 帧尽力而为丢帧可容忍（`TryPublish` 满即丢），
  Run 同步响应携带完整 trace 与终态，前端以响应为准校正实时帧；
- 未接线 publisher（旧装配/单测）noop 降级——流式观感静默缺失，不影响同步语义；
- 三端消费：事件注册表加 `AgentRun` 项（真相源 sseevent.go 两边同改）；
  react 试运行面板 `useAgentRunStream` 订阅（appId 过滤），响应未到达时先呈现
  实时帧、到达后全量 trace 覆盖。

## 4. 数据模型

| 表 | 关键列 | 说明 |
| --- | --- | --- |
| `sys_ai_agent_apps` | `name`(租户内唯一) `description` `graph`(Text) `enabled`(bool) | 租户隔离 mixin（TenantID），登录即可用（同知识库权限口径） |
| `sys_ai_agent_runs` | `app_id` `status`(RUNNING/SUCCEEDED/FAILED) `input` `output` `error` `trace` `prompt/completion/total_tokens` `duration_ms` | 租户隔离；索引 (tenant_id, app_id) |

## 5. 分期

- **M1（本期）**：引擎 + CRUD + 同步 Run + 运行台账 + react 画布先行
  （列表页 + 全屏画布编辑器 + 试运行面板，@xyflow/react）。门禁全绿。
- **M2（部分落地）**：SSE 流式已落地（见 §3.1，事件名定为 `ai_agent_run`，非早期
  设想的 `ai_agent_chunk`——推的是节点阶段帧而非 LLM token 增量）；ele/vben 画布移植
  （vue-flow）与 script/http 节点、应用版本/发布仍未排期。
- **M3（设想）**：对外 OpenAPI（AK/SK 调用 Agent）、定时触发（任务桥）、工具市场
  （把脚本引擎的钩子暴露为 LLM 工具）。

## 6. 相关文档

- `docs/ai_module.md`：提供商/知识库/配额（本能力全部复用其链路）
- `docs/list_query_rule.md`：列表协议；`docs/crud_module_guide/`：CRUD 管线
- 已部署实例升级：`make openapi` + 重启 + 管理页「接口同步」（新端点 `/admin/v1/ai/agents*`）+
  「菜单同步」（新菜单行 AgentApps）+ 权限勾选
