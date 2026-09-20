// Package sseevent 是全仓 SSE 事件类型的注册表。
//
// SSE 的事件名是一条跨语言、跨部署单元的契约：Go 侧把它写进帧的 event: 行，三端前端
// 按同名注册回调，中间没有任何编译期检查（前端各自的 SSEEventName 类型带 `| string`，
// 等于不约束）。docs/sse_architecture.md 记录的"没有事件类型注册表、生产方与消费方
// 手工对齐裸字符串"就是指这里，docs/notification_domain_design.md §3.6 要求新域 own 一个注册表。
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
// 三端通知面板因此能复用同一套字段读取代码。用 encoding/json 序列化该消息会得到
// snake_case 键，前端读 messageId/createdAt 全取到 undefined，所以这里必须是 protojson。
const Notification = "notification"
