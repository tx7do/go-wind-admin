import { useMutation, type UseMutationOptions } from '@tanstack/react-query';
import {
  type internal_messageservicev1_ListInternalMessageResponse,
  type internal_messageservicev1_InternalMessage,
  type internal_messageservicev1_GetInternalMessageRequest,
  type internal_messageservicev1_UpdateInternalMessageRequest,
  type internal_messageservicev1_DeleteInternalMessageRequest,
  type internal_messageservicev1_SendMessageRequest,
  type internal_messageservicev1_SendMessageResponse,
  type internal_messageservicev1_RevokeMessageRequest,
  type internal_messageservicev1_ListInternalMessageCategoryResponse,
  type internal_messageservicev1_InternalMessageCategory,
  type internal_messageservicev1_GetInternalMessageCategoryRequest,
  type internal_messageservicev1_CreateInternalMessageCategoryRequest,
  type internal_messageservicev1_UpdateInternalMessageCategoryRequest,
  type internal_messageservicev1_DeleteInternalMessageCategoryRequest,
  type internal_messageservicev1_ListUserInboxResponse,
  type internal_messageservicev1_DeleteNotificationFromInboxRequest,
  type internal_messageservicev1_MarkNotificationAsReadRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery } from '@/core';
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
  options?: UseMutationOptions<
    internal_messageservicev1_ListInternalMessageResponse,
    Error,
    PaginationQuery
  >,
) {
  return useMutation({
    mutationFn: (query) => listInternalMessages(query),
    ...options,
  });
}

export function useGetInternalMessage(
  options?: UseMutationOptions<
    internal_messageservicev1_InternalMessage,
    Error,
    internal_messageservicev1_GetInternalMessageRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => getInternalMessage(data),
    ...options,
  });
}

export function useUpdateInternalMessage(
  options?: UseMutationOptions<{}, Error, internal_messageservicev1_UpdateInternalMessageRequest>,
) {
  return useMutation({
    mutationFn: (data) => updateInternalMessage(data),
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
  options?: UseMutationOptions<
    internal_messageservicev1_ListInternalMessageCategoryResponse,
    Error,
    PaginationQuery
  >,
) {
  return useMutation({
    mutationFn: (query) => listMessageCategories(query),
    ...options,
  });
}

export function useGetMessageCategory(
  options?: UseMutationOptions<
    internal_messageservicev1_InternalMessageCategory,
    Error,
    internal_messageservicev1_GetInternalMessageCategoryRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => getMessageCategory(data),
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
    internal_messageservicev1_UpdateInternalMessageCategoryRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => updateMessageCategory(data),
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
  options?: UseMutationOptions<
    internal_messageservicev1_ListUserInboxResponse,
    Error,
    PaginationQuery
  >,
) {
  return useMutation({
    mutationFn: (query) => listUserInbox(query),
    ...options,
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
