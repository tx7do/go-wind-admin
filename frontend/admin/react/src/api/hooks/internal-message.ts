import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/react-query';
import {
  type internal_messageservicev1_ListInternalMessageResponse,
  type internal_messageservicev1_InternalMessage,
  type internal_messageservicev1_GetInternalMessageRequest,
  type internal_messageservicev1_DeleteInternalMessageRequest,
  type internal_messageservicev1_SendMessageRequest,
  type internal_messageservicev1_SendMessageResponse,
  type internal_messageservicev1_RevokeMessageRequest,
  type internal_messageservicev1_ListInternalMessageCategoryResponse,
  type internal_messageservicev1_InternalMessageCategory,
  type internal_messageservicev1_GetInternalMessageCategoryRequest,
  type internal_messageservicev1_CreateInternalMessageCategoryRequest,
  type internal_messageservicev1_DeleteInternalMessageCategoryRequest,
  type internal_messageservicev1_ListUserInboxResponse,
  type internal_messageservicev1_DeleteNotificationFromInboxRequest,
  type internal_messageservicev1_MarkNotificationAsReadRequest,
} from '@/api/generated/admin/service/v1';
import { makeUpdateMask, type PaginationQuery, queryClient } from '@/core';
import { apiClient } from '@/api/client';

// ==============================
// 内部消息管理
// ==============================
export function useListInternalMessages(
  query: PaginationQuery,
  options?: UseQueryOptions<internal_messageservicev1_ListInternalMessageResponse, Error>,
) {
  return useQuery({
    queryKey: ['listInternalMessages', query],
    queryFn: () => apiClient.internalMessageService.ListMessage(query.toRawParams()),
    ...options,
  });
}

export async function fetchListInternalMessages(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listInternalMessages', params],
    queryFn: () => apiClient.internalMessageService.ListMessage(params.toRawParams()),
    retry: 0,
  });
}

export function useGetInternalMessage(
  req: internal_messageservicev1_GetInternalMessageRequest,
  // queryKey/queryFn 由本 hook 固定构造，调用方只传行为选项（如 enabled）
  options?: Omit<UseQueryOptions<internal_messageservicev1_InternalMessage, Error>, 'queryKey' | 'queryFn'>,
) {
  return useQuery({
    queryKey: ['getInternalMessage', req],
    queryFn: () => apiClient.internalMessageService.GetMessage(req),
    ...options,
  });
}

export function useUpdateInternalMessage(
  options?: UseMutationOptions<{}, Error, { id: string; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: Record<string, any> }) =>
      apiClient.internalMessageService.UpdateMessage({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteInternalMessage(
  options?: UseMutationOptions<{}, Error, internal_messageservicev1_DeleteInternalMessageRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.internalMessageService.DeleteMessage(data),
    ...options,
  });
}

export function useSendMessage(
  options?: UseMutationOptions<
    internal_messageservicev1_SendMessageResponse,
    Error,
    internal_messageservicev1_SendMessageRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => apiClient.internalMessageService.SendMessage(data),
    ...options,
  });
}

export function useRevokeMessage(
  options?: UseMutationOptions<{}, Error, internal_messageservicev1_RevokeMessageRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.internalMessageService.RevokeMessage(data),
    ...options,
  });
}

// ==============================
// 消息分类管理
// ==============================
export function useListMessageCategories(
  query: PaginationQuery,
  options?: UseQueryOptions<internal_messageservicev1_ListInternalMessageCategoryResponse, Error>,
) {
  return useQuery({
    queryKey: ['listMessageCategories', query],
    queryFn: () => apiClient.internalMessageCategoryService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListMessageCategories(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listMessageCategories', params],
    queryFn: () => apiClient.internalMessageCategoryService.List(params.toRawParams()),
    retry: 0,
  });
}

export function useGetMessageCategory(
  req: internal_messageservicev1_GetInternalMessageCategoryRequest,
  options?: UseQueryOptions<internal_messageservicev1_InternalMessageCategory, Error>,
) {
  return useQuery({
    queryKey: ['getMessageCategory', req],
    queryFn: () => apiClient.internalMessageCategoryService.Get(req),
    ...options,
  });
}

export function useCreateMessageCategory(
  options?: UseMutationOptions<
    {},
    Error,
    internal_messageservicev1_CreateInternalMessageCategoryRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => apiClient.internalMessageCategoryService.Create(data),
    ...options,
  });
}

export function useUpdateMessageCategory(
  options?: UseMutationOptions<
    {},
    Error,
    { id: string; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: Record<string, any> }) =>
      apiClient.internalMessageCategoryService.Update({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteMessageCategory(
  options?: UseMutationOptions<
    {},
    Error,
    internal_messageservicev1_DeleteInternalMessageCategoryRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => apiClient.internalMessageCategoryService.Delete(data),
    ...options,
  });
}

// ==============================
// 消息接收者管理（用户收件箱）
// ==============================
export function useListUserInbox(
  query: PaginationQuery,
  options?: UseQueryOptions<internal_messageservicev1_ListUserInboxResponse, Error>,
) {
  return useQuery({
    queryKey: ['listUserInbox', query],
    queryFn: () => apiClient.internalMessageRecipientService.ListUserInbox(query.toRawParams()),
    ...options,
  });
}

export async function fetchListUserInbox(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listUserInbox', params],
    queryFn: () => apiClient.internalMessageRecipientService.ListUserInbox(params.toRawParams()),
    retry: 0,
  });
}

export function useDeleteNotificationFromInbox(
  options?: UseMutationOptions<
    {},
    Error,
    internal_messageservicev1_DeleteNotificationFromInboxRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => apiClient.internalMessageRecipientService.DeleteNotificationFromInbox(data),
    ...options,
  });
}

export function useMarkNotificationAsRead(
  options?: UseMutationOptions<{}, Error, internal_messageservicev1_MarkNotificationAsReadRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.internalMessageRecipientService.MarkNotificationAsRead(data),
    ...options,
  });
}
