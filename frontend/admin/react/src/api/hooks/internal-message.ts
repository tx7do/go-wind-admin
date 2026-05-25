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
import {
  listInternalMessages,
  getInternalMessage,
  updateInternalMessage,
  deleteInternalMessage,
  sendMessage,
  revokeMessage,
  listMessageCategories,
  getMessageCategory,
  createMessageCategory,
  updateMessageCategory,
  deleteMessageCategory,
  listUserInbox,
  deleteNotificationFromInbox,
  markNotificationAsRead,
} from '@/api/service/internal-message';

// ==============================
// 内部消息管理
// ==============================
export function useListInternalMessages(
  query: PaginationQuery,
  options?: UseQueryOptions<internal_messageservicev1_ListInternalMessageResponse, Error>,
) {
  return useQuery({
    queryKey: ['listInternalMessages', query],
    queryFn: () => listInternalMessages(query),
    ...options,
  });
}

export async function fetchListInternalMessages(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listInternalMessages', params],
    queryFn: () => listInternalMessages(params),
    retry: 0,
  });
}

export function useGetInternalMessage(
  req: internal_messageservicev1_GetInternalMessageRequest,
  options?: UseQueryOptions<internal_messageservicev1_InternalMessage, Error>,
) {
  return useQuery({
    queryKey: ['getInternalMessage', req],
    queryFn: () => getInternalMessage(req),
    ...options,
  });
}

export function useUpdateInternalMessage(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      updateInternalMessage({
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
    mutationFn: (data) => deleteInternalMessage(data),
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
    mutationFn: (data) => sendMessage(data),
    ...options,
  });
}

export function useRevokeMessage(
  options?: UseMutationOptions<{}, Error, internal_messageservicev1_RevokeMessageRequest>,
) {
  return useMutation({
    mutationFn: (data) => revokeMessage(data),
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
    queryFn: () => listMessageCategories(query),
    ...options,
  });
}

export async function fetchListMessageCategories(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listMessageCategories', params],
    queryFn: () => listMessageCategories(params),
    retry: 0,
  });
}

export function useGetMessageCategory(
  req: internal_messageservicev1_GetInternalMessageCategoryRequest,
  options?: UseQueryOptions<internal_messageservicev1_InternalMessageCategory, Error>,
) {
  return useQuery({
    queryKey: ['getMessageCategory', req],
    queryFn: () => getMessageCategory(req),
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
    mutationFn: (data) => createMessageCategory(data),
    ...options,
  });
}

export function useUpdateMessageCategory(
  options?: UseMutationOptions<
    {},
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      updateMessageCategory({
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
    mutationFn: (data) => deleteMessageCategory(data),
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
    queryFn: () => listUserInbox(query),
    ...options,
  });
}

export async function fetchListUserInbox(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listUserInbox', params],
    queryFn: () => listUserInbox(params),
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
    mutationFn: (data) => deleteNotificationFromInbox(data),
    ...options,
  });
}

export function useMarkNotificationAsRead(
  options?: UseMutationOptions<{}, Error, internal_messageservicev1_MarkNotificationAsReadRequest>,
) {
  return useMutation({
    mutationFn: (data) => markNotificationAsRead(data),
    ...options,
  });
}
