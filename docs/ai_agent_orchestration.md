# AI Agent 编排（Agent Orchestration）设计文档

> **状态：M1 已落地（2026-10-15，见 §5 分期）**：图执行引擎 + 应用 CRUD + 同步试运行 + 运行台账 + react 画布先行。
> **M2 画布移植已落地**：ele/vben 画布编辑器（vue-flow 版）随 c6504f7a 补齐、d765b75e 真机修正——三端编排齐平。
> M2+ 进展（2026-11-08）：**SSE 流式与 ele/vben 画布早已落地**（后者随 c6504f7a）；
> **HTTP 节点已落地**（后端 + 三端画布，见 §2.1）。
> **（2026-11-09）脚本节点已落地**（后端引擎 + 三端画布，见 §2.1 script 行）；
> **版本化已落地**：revision 留底/恢复（BI 同款口径，非 draft/published，见 §4 版本化）。
> M2 至此收官。**M3 首批同日落地**：定时触发（agent_run 任务桥）与对外 API（AK/SK，见 §5）。
> 剩余（工具市场、循环/子图/并行）未排期，需求驱动。
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
| ~~SSE 流式执行~~ | **M2 已落地**（§3.1） |
| ~~应用版本与发布流（draft/published）~~ | **版本化已落地（2026-11-09，§4）**：revision 留底/恢复；draft/published 双态仍不做 |
| 对外 API（AK/SK 调用 Agent） | M1 只在管理台内运行；开放 API 等 M3 |

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

### 2.1 节点集（M1 五种 + M2 http/script）

| type | data 键 | 写入上下文 |
| --- | --- | --- |
| `start` | `inputVars: [{name,label,required}]`（运行入参声明） | 每个输入变量按名直挂自己（`start_1.question`） |
| `knowledge` | `baseId, topK(默认4), query(模板)` | `context`（命中片段拼接）、`hits`（命中条数） |
| `llm` | `providerId, model, systemPrompt?(模板), userPrompt(模板), temperature?` | `output`（最终答案文本） |
| `http` **（M2，2026-11-08）** | `url`(必填,模板), `method`(白名单 GET/POST/PUT/DELETE/PATCH/HEAD, 默认 GET), `headers`(`[{key,value}]` 或 map，键值均可模板), `body`(模板，非 GET/HEAD), `timeoutSeconds`(1-60, 默认 15) | `status`（状态码）、`body`（响应体，读上限 64KB、入上下文截断 8KB）、`contentType` |
| `script` **（M2，2026-11-09）** | `language`(白名单 lua/javascript), `source`(脚本源码，**非模板**——代码里的 `{{...}}` 是脚本自身语法)。脚本**必须定义 `execute()`**（引擎哨兵探测该入口）：上下文经 `__get_ctx()` 取——返回带 `get/set/stop` 方法的表，`c.get("节点id")` 读上游命名空间、`__set_ctx(k, v)` 写产出、`__stop(reason)` 中止；`return false` = 中止 | 脚本新写入的键收口为字符串命名空间（字符串直通、标量转写、嵌套值 JSON 编码；**输入键回声剔除**——注入的输入命名空间不出现在产出里） |
| `condition` | `expression`（[expr-lang](https://expr-lang.org) 语法，变量即节点命名空间） | 无（分流由带 `branch` 的出边承接） |
| `end` | `output`(模板，可空) | 运行产出 = 到达的 end 节点渲染值 |

**http 节点的 SSRF 防线（2026-11-08）**：与 RenderPdf（`print_template_sanitize.go`）同源——
scheme 白名单（http/https）、主机字面量/后缀判定（localhost/.local/.internal）、解析后逐 IP
复核（私网/回环/链路本地/CGNAT 段）；`ai_agent_http_node.go` 的 `validateAgentOutboundURL`。
与那条线同样存在解析与请求两次解析之间的 TOCTOU 窗口（DNS 重绑定），强度一致、接受同一权衡。
静态校验（Run 前）：url 必填、method 白名单、timeoutSeconds 钳制；校验不过 400、不建运行行。

**script 节点的执行与信任模型（2026-11-09）**：执行走 `ScriptRuntime.TestRun`——一次性隔离
沙箱引擎（与常驻脚本引擎/钩子互不污染），依赖面与脚本系统试运行同款（redis/oss/ai 模块可用）；
超时沿用引擎 VMTimeout 缺省 5s，节点不带 timeout 字段。入参是**全图上下文**（按节点 id 命名
的表，脚本经 `c.get("节点id")` 读任意上游输出）。源码上限 32KB（script 节点语义是"小段数据加工"，
不是承载完整应用的载体；超限请到脚本系统建正式脚本）。**信任口径**：能编辑 Agent 应用的用户
（登录即可用，同知识库权限口径）即可在图里运行沙箱内脚本——这是对"脚本执行此前仅平台管理员可达"
的有意扩展，依据是 go-scripts 沙箱本身就是为不可信代码设计的（VM 超时/受限模块）；执行留
script_log 审计行（name 带 `agent:<节点id>` 定位）。`scriptRuntime` 未装配（单测/极简部署）时
报错不静默跳过。静态校验（Run 前）：language 白名单、source 非空且 ≤32KB。
e2e 实测（本机 gwa 库，2026-11-09）：lua `execute()` 读 start 入参 → `__set_ctx` 产出 →
end 模板渲染 `{{script_1.answer}}` 全链 SUCCEEDED；Save/Restore 快照链路同批实测通过。

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
按节点类型的字面校验：http（url 必填/method 白名单/timeout 钳制）、script（language 白名单/
source 非空 ≤32KB）。

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
| `sys_ai_agent_revisions`（2026-11-09） | `app_id` `graph`(Text) `source`(SAVE/RESTORE) | 版本快照：只增不改不删，随主表级联删除；索引 (app_id, created_at) |

**版本化（revision 留底/恢复，2026-11-09，BI 同款口径）**：
每次更新且 **graph 实际变化**时，把更新前的 graph 以 `SAVE` 来源留一行快照（快照失败不阻断更新，
Errorf 留痕——主功能不被辅功能拖死，但丢快照必须可观测）；恢复（`RestoreRevision`）先把**当前**
graph 以 `RESTORE` 来源留底再写回，恢复可逆；恢复前快照 graph 过静态校验，非法 400 且不落任何
痕迹（历史快照可能早于校验规则演进）。快照行不设版本号列——agent 主表无乐观锁版本号，按
created_at 排序即可（与 BI 的 version 列形态刻意不同）。端点：`POST /admin/v1/ai/agents/{app_id}/revisions`
（历史列表，新快照在前）、`POST /admin/v1/ai/agents/revisions/{id}:restore`。不做 draft/published
双态发布流——留底/恢复已覆盖"改坏了想回滚"的真实诉求，发布流等真需求出现再议。

## 5. 分期

- **M1（本期）**：引擎 + CRUD + 同步 Run + 运行台账 + react 画布先行
  （列表页 + 全屏画布编辑器 + 试运行面板，@xyflow/react）。门禁全绿。
- **M2（✅ 2026-11-09 收官）**：SSE 流式（§3.1）；ele/vben 画布移植（vue-flow，c6504f7a）；
  http 节点（§2.1，2026-11-08）；script 节点（§2.1，2026-11-09）；版本化留底/恢复（§4，2026-11-09）。
- **M3（2026-11-09 首批落地）**：
  - **定时触发（agent_run 任务桥）已落地**：「任务管理」建 PERIODIC 行
    （`type_name="agent_run"`，`task_payload={"appId":1,"input":{"question":"…"},"notifyUserId":1}`），
    cron 到点执行应用图。执行身份 UserId=0（机器，台账 created_by=0、SSE 流 ID="0"），
    租户取应用归属租户——LLM/RAG 用量记该租户流水、配额按该租户闸；graph 执行失败已落
    FAILED 台账（确定性结论不重试），仅基础设施错误交 asynq 重试；`notifyUserId>0` 时
    结果站内信回传（发送失败仅记日志）。enabled=false 的应用本轮跳过（不落台账不重试）。
    handler `AiAgentService.AsyncAgentRun`（ai_agent_task.go），sqlite 回归测试在位。
  - **对外 API（AK/SK 调用 Agent）已打通**：`AiAgentService` 补登记 Module_AI
    （module_mapping.go，原缺失使租户/AK 令牌过租户闸门即 403 "module not allowed"）。
    用法：管理页建 AK（ak-*/sk-*）→ `POST /admin/v1/access-keys/token` 换机器 JWT
    （uid=0、tid=AK 归属租户、roles=["machine"]）→ `POST /admin/v1/ai/agents/{id}/run`
    带 `Authorization: Bearer`。平台侧 AK（tid=0）开箱可用（租户闸门跳过、配额不限）；
    租户 AK 需套餐白名单含 AI 模块且 API_CALL 未超限。authz 切 casbin 时需给 machine
    角色显式配策略（docs/authentication.md §6 既有口径）。顺带补登记 AiContentService。
  - **未排期（需求驱动）**：工具市场（把脚本引擎的钩子暴露为 LLM 工具）、循环/子图/并行。

## 6. 相关文档

- `docs/ai_module.md`：提供商/知识库/配额（本能力全部复用其链路）
- `docs/list_query_rule.md`：列表协议；`docs/crud_module_guide/`：CRUD 管线
- 已部署实例升级：`make openapi` + 重启 + 管理页「接口同步」（新端点 `/admin/v1/ai/agents*`）+
  「菜单同步」（新菜单行 AgentApps）+ 权限勾选
