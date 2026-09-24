// Package sseevent 是全仓 SSE 事件类型的注册表。
//
// SSE 的事件名是一条跨语言、跨部署单元的契约：Go 侧把它写进帧的 event: 行，三端前端
// 按同名注册回调，中间没有任何编译期检查（前端各自的 SSEEventName 类型带 `| string`，
// 等于不约束）。本包因此成为注册表（docs/sse_architecture.md 第 4 节末、第 8 节），
// docs/notification_domain_design.md §3.6 要求新域 own 一个注册表。
//
// 新增事件：在本包加常量，并同步三端的事件名常量表——
// react `src/core/transport/sse/event.ts`、vue-element `src/core/transport/sse/event.ts`、
// vue-vben `apps/admin/src/transport/sse/event.ts`。
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
