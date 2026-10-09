/**
 * SSE 事件名注册表（vue-vben 端副本）。
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
   * 与 REST 收件箱接口返回的形状一致。
   */
  Notification: 'notification',

  /**
   * AI 对话流式片段推送。
   *
   * data 为 `ChatChunkEvent` 的 protojson：camelCase 键（`conversationId` /
   * `seq` / `delta`）。一次对话产生多个 chunk 帧，按 conversationId 归组、
   * seq 顺序累积渲染；POST /admin/v1/ai/chat/completions 的同步响应携带
   * 完整回复，以响应为准校正累积文本。
   */
  AIChatChunk: 'ai_chat_chunk',

  /**
   * AI 对话的工具调用可见化推送。
   *
   * data 为 `ChatToolEvent` 的 protojson：camelCase 键（`conversationId` /
   * `name` / `arguments` / `result`）。模型每发起一次本地工具调用并执行完成
   * 后推一帧；工具轮不进消息落库（落库的 assistant 消息只有最终答案），
   * 本事件是前端展示"模型正在调工具"的唯一信息源（尽力而为，丢帧可容忍）。
   */
  AIChatTool: 'ai_chat_tool',
} as const;
