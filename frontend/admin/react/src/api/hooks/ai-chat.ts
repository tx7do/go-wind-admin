import { useMutation, useQuery, type UseMutationOptions, type UseQueryOptions } from '@tanstack/react-query';
import { apiClient } from '@/api/client';
import { type PaginationQuery, queryClient } from '@/core';
import { makeUpdateMask } from '@/core/transport/rest/utils';
import type {
  aiservicev1_AiConversation,
  aiservicev1_ChatRequest,
  aiservicev1_ChatResponse,
  aiservicev1_ListAiConversationResponse,
  aiservicev1_ListAiMessageResponse,
  aiservicev1_UpdateAiConversationRequest,
} from '@/api/generated/admin/service/v1';

// ── 会话 ────────────────────────────────────────────────────────────

// useListAiConversations — 聊天页左栏（后端强制只返回当前用户的会话）
export function useListAiConversations(
  query: PaginationQuery,
  options?: Omit<UseQueryOptions<aiservicev1_ListAiConversationResponse, Error>, 'queryKey' | 'queryFn'>,
) {
  return useQuery({
    queryKey: ['listAiConversations', query],
    queryFn: () => apiClient.aiConversationService.List(query.toRawParams()),
    ...options,
  });
}

// fetchListAiConversations — 非组件上下文
export async function fetchListAiConversations(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listAiConversations', params],
    queryFn: () => apiClient.aiConversationService.List(params.toRawParams()),
    retry: 0,
  });
}

// useUpdateAiConversation — 改标题
export function useUpdateAiConversation(
  options?: UseMutationOptions<{}, Error, { id: string; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: Record<string, any> }) =>
      apiClient.aiConversationService.Update({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      } as aiservicev1_UpdateAiConversationRequest),
    ...options,
  });
}

// useDeleteAiConversation — 后端级联删除会话内全部消息
export function useDeleteAiConversation(
  options?: UseMutationOptions<{}, Error, { id: string }>,
) {
  return useMutation({
    mutationFn: ({ id }: { id: string }) => apiClient.aiConversationService.Delete({ id }),
    ...options,
  });
}

// ── 消息 ────────────────────────────────────────────────────────────

// useListAiMessages — 按 conversation_id 过滤（后端强制 user_id 归属）
export function useListAiMessages(
  query: PaginationQuery,
  options?: Omit<UseQueryOptions<aiservicev1_ListAiMessageResponse, Error>, 'queryKey' | 'queryFn'>,
) {
  return useQuery({
    queryKey: ['listAiMessages', query],
    queryFn: () => apiClient.aiMessageService.List(query.toRawParams()),
    ...options,
  });
}

// fetchListAiMessages — 发送后补拉/刷新用
export async function fetchListAiMessages(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listAiMessages', params],
    queryFn: () => apiClient.aiMessageService.List(params.toRawParams()),
    retry: 0,
  });
}

// ── 对话 ────────────────────────────────────────────────────────────

// useSendChat — 发起一轮对话。
// 增量 token 经 SSE `ai_chat_chunk` 事件实时推送（订阅见聊天页），
// 本 mutation 的响应携带完整回复（含会话与 tokens 用量），是最终事实。
export function useSendChat(
  options?: UseMutationOptions<aiservicev1_ChatResponse, Error, aiservicev1_ChatRequest>,
) {
  return useMutation({ mutationFn: (req) => apiClient.aiChatService.Chat(req), ...options });
}

// 导出会话类型别名（页面直接用）
export type AiConversation = aiservicev1_AiConversation;
