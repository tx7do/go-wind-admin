/**
 * SSE 事件名注册表（vue-element 端副本）。
 *
 * 真相源是后端 `backend/pkg/sseevent/sseevent.go`：它把常量写进帧的 `event:` 行，
 * 这里按同名注册回调，中间没有编译期检查（`SSEEventName` 带 `| string`，等于不约束）。
 * 新增事件要两边同时改；已发布事件的取值不可改——改名等于对线上前端静默断流。
 */
export const SSE_EVENT = {
  /**
   * 站内信收件行推送。
   *
   * data 为 `InternalMessageRecipient` 的 protojson：camelCase 键（`messageId` /
   * `recipientUserId` / `createdAt`），`status` 是枚举名字符串（`RECEIVED` / `READ`），
   * 与 REST 收件箱接口返回的形状一致。取值不匹配时 handleSseNotification 会在
   * `if (!data.messageId) return` 处静默退出。
   */
  Notification: "notification",
} as const;
