// Package sseevent 是全仓 SSE 事件类型的注册表。
//
// SSE 的事件名是一条跨语言、跨部署单元的契约：Go 侧把它写进帧的 event: 行，三端前端
// 按同名注册回调，中间没有任何编译期检查（前端各自的 SSEEventName 类型带 `| string`，
// 等于不约束）。本包因此成为注册表（docs/sse_architecture.md 第 4 节末、第 8 节），
// docs/notification_domain_design.md §3.6 要求新域 own 一个注册表。
//
// 新增事件：在本包加常量，并同步三端的事件名常量表——
// react `src/core/transport/sse/event.ts`、vue-element `src/core/transport/sse/event.ts`、
// vue-vben `src/transport/sse/event.ts`（单包迁移后不在 apps/ 下）。三端覆盖由
// sseevent_test.go 的 TestPublishedEventsReachAllFrontends 源扫描守护——漏一端 CI 即红。
// 已发布事件的常量值不可改：线上前端按字面值订阅，改名等于静默断流。
package sseevent

// Notification 站内信收件行推送。
//
// data: 为 internal_message.service.v1.InternalMessageRecipient 的 protojson
// （camelCase 键、status 是枚举名字符串），与 REST 收件箱接口的返回形状一致——
// 三端通知面板因此能复用同一套字段读取代码。
//
// 换成 encoding/json 会同时坏掉三处（2026-09-25 同一消息两跑对照实测）：
//
//	protojson     {"id":10,"messageId":11,"status":"RECEIVED","createdAt":"2023-11-14T22:13:20.000000123Z"}
//	encoding/json {"id":10,"message_id":11,"status":1,"created_at":{"seconds":1700000000,"nanos":123}}
//
// 生成的 pb.go 带 `json:"message_id,omitempty"` 一类蛇形 tag，所以键名变蛇形、
// status 变数字、时间戳变成 {seconds,nanos} 对象。前端读 messageId/createdAt 全取到 undefined。
const Notification = "notification"

// AIChatChunk AI 对话流式片段推送。
//
// data: 为 ai.service.v1.ChatChunkEvent 的 protojson（camelCase 键：
// conversationId / seq / delta）。一次对话会产生多个 chunk 帧，前端按
// conversationId 归组、按 seq 顺序累积渲染；POST /admin/v1/ai/chat/completions
// 的同步响应携带完整回复，以响应为准校正累积文本（chunk 尽力而为，缓冲满即丢帧）。
const AIChatChunk = "ai_chat_chunk"

// AIChatTool AI 对话的工具调用可见化推送。
//
// data: 为 ai.service.v1.ChatToolEvent 的 protojson（camelCase 键：
// conversationId / name / arguments / result）。模型每发起一次本地工具调用
// 并执行完成后推一帧；工具轮不进消息落库（落库的 assistant 消息只有最终
// 答案），本事件是前端展示"模型正在调工具"的唯一信息源（尽力而为，丢帧可容忍）。
const AIChatTool = "ai_chat_tool"

// AgentRun AI Agent 运行实时推送（试运行流式观感）。
//
// data: 为 ai.service.v1.AgentRunEvent 的 protojson（camelCase 键：
// runId / appId / seq / phase / nodeId / summary / status）。
// 一次试运行产生多帧：每节点 node_start + node_end 各一帧，收尾 done 或 error 一帧。
// 尽力而为丢帧可容忍：Run 同步响应携带完整 trace 与终态，以响应为准。
// streamId 与 AI 对话同为发起用户 userId（复用既有用户流，免重复连接）。
const AgentRun = "ai_agent_run"
