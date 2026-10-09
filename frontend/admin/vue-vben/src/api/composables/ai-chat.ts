import { apiClient } from '#/api/client';

import { makeUpdateMask } from '#/transport/rest';

// ==============================
// AI 对话（会话 / 消息 / 对话发起）
// ==============================

/** 分页查询我的会话（后端强制只返回当前用户的会话） */
export async function fetchListAiConversations(query: any) {
  return apiClient.aiConversationService.List(query.toRawParams());
}

/** 更新会话（改标题） */
export async function updateAiConversation(id: number, values: Record<string, any>) {
  return apiClient.aiConversationService.Update({
    id,
    data: values as any,
    updateMask: makeUpdateMask(Object.keys(values ?? {})),
  });
}

/** 删除会话（后端级联删除会话内全部消息） */
export async function deleteAiConversation(id: number) {
  return apiClient.aiConversationService.Delete({ id });
}

/** 分页查询消息（filter 传 conversationId；后端强制 user_id 归属） */
export async function fetchListAiMessages(query: any) {
  return apiClient.aiMessageService.List(query.toRawParams());
}

/**
 * 发起一轮对话。
 *
 * 流式语义：本请求是普通 POST（body 为扁平请求体，不包 data——与站内信 send 同型），
 * 同步返回完整回复；增量 token 通过 SSE 网关以 `ai_chat_chunk` 事件实时推送
 * （订阅见聊天页），chunk 尽力而为，以本响应为最终事实。
 */
export async function sendAiChat(req: {
  conversationId?: number;
  providerId?: number;
  content: string;
  knowledgeBaseId?: number;
}) {
  return apiClient.aiChatService.Chat({
    conversationId: req.conversationId || 0,
    providerId: req.providerId || 0,
    content: req.content,
    knowledgeBaseId: req.knowledgeBaseId || 0,
  } as any);
}
