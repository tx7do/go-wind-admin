import type {
  internal_messageservicev1_CreateInternalMessageCategoryRequest,
  internal_messageservicev1_DeleteInternalMessageCategoryRequest,
  internal_messageservicev1_DeleteInternalMessageRequest,
  internal_messageservicev1_DeleteNotificationFromInboxRequest,
  internal_messageservicev1_GetInternalMessageCategoryRequest,
  internal_messageservicev1_GetInternalMessageRequest,
  internal_messageservicev1_InternalMessage,
  internal_messageservicev1_InternalMessageCategory,
  internal_messageservicev1_ListInternalMessageCategoryResponse,
  internal_messageservicev1_ListInternalMessageResponse,
  internal_messageservicev1_ListUserInboxResponse,
  internal_messageservicev1_MarkNotificationAsReadRequest,
  internal_messageservicev1_RevokeMessageRequest,
  internal_messageservicev1_SendMessageRequest,
  internal_messageservicev1_SendMessageResponse,
  internal_messageservicev1_InternalMessage_Status as InternalMessage_Status,
  internal_messageservicev1_InternalMessage_Type as InternalMessage_Type,
  internal_messageservicev1_InternalMessageRecipient_Status as InternalMessageRecipient_Status,
} from '#/api/generated/admin/service/v1';

import { computed } from 'vue';

import { i18n } from '@vben/locales';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import {
  createMessageCategory,
  deleteInternalMessage,
  deleteMessageCategory,
  deleteNotificationFromInbox,
  getInternalMessage,
  getMessageCategory,
  listInternalMessages,
  listMessageCategories,
  listUserInbox,
  markNotificationAsRead,
  revokeMessage,
  sendMessage,
  updateInternalMessage,
  updateMessageCategory,
} from '#/api/service/internal-message';
import { queryClient } from '#/plugins/vue-query';
import { makeUpdateMask, type PaginationQuery } from '#/transport/rest';

const t = i18n.global.t;

// ==============================
// 内部消息管理
// ==============================
export function useListInternalMessages(
  query: PaginationQuery,
  options?: UseQueryOptions<
    internal_messageservicev1_ListInternalMessageResponse,
    Error
  >,
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

export async function fetchGetInternalMessage(
  params: internal_messageservicev1_GetInternalMessageRequest,
) {
  return queryClient.fetchQuery({
    queryKey: ['getInternalMessage', params],
    queryFn: () => getInternalMessage(params),
    retry: 0,
  });
}

export function useUpdateInternalMessage(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
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
  options?: UseMutationOptions<
    object,
    Error,
    internal_messageservicev1_DeleteInternalMessageRequest
  >,
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
  options?: UseMutationOptions<
    object,
    Error,
    internal_messageservicev1_RevokeMessageRequest
  >,
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
  options?: UseQueryOptions<
    internal_messageservicev1_ListInternalMessageCategoryResponse,
    Error
  >,
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
  options?: UseQueryOptions<
    internal_messageservicev1_InternalMessageCategory,
    Error
  >,
) {
  return useQuery({
    queryKey: ['getMessageCategory', req],
    queryFn: () => getMessageCategory(req),
    ...options,
  });
}

export function useCreateMessageCategory(
  options?: UseMutationOptions<
    object,
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
    object,
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
    object,
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
  options?: UseQueryOptions<
    internal_messageservicev1_ListUserInboxResponse,
    Error
  >,
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
    object,
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
  options?: UseMutationOptions<
    object,
    Error,
    internal_messageservicev1_MarkNotificationAsReadRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => markNotificationAsRead(data),
    ...options,
  });
}

// ==============================
// 内部消息枚举与工具函数
// ==============================

export const internalMessageStatusList = computed(() => [
  { value: 'DRAFT', label: t('enum.internalMessage.status.DRAFT') },
  { value: 'PUBLISHED', label: t('enum.internalMessage.status.PUBLISHED') },
  { value: 'SCHEDULED', label: t('enum.internalMessage.status.SCHEDULED') },
  { value: 'REVOKED', label: t('enum.internalMessage.status.REVOKED') },
  { value: 'ARCHIVED', label: t('enum.internalMessage.status.ARCHIVED') },
  { value: 'DELETED', label: t('enum.internalMessage.status.DELETED') },
]);

export const internalMessageTypeList = computed(() => [
  { value: 'NOTIFICATION', label: t('enum.internalMessage.type.NOTIFICATION') },
  { value: 'PRIVATE', label: t('enum.internalMessage.type.PRIVATE') },
  { value: 'GROUP', label: t('enum.internalMessage.type.GROUP') },
]);

export const internalMessageRecipientStatusList = computed(() => [
  { value: 'SENT', label: t('enum.internalMessageRecipient.status.SENT') },
  {
    value: 'RECEIVED',
    label: t('enum.internalMessageRecipient.status.RECEIVED'),
  },
  { value: 'READ', label: t('enum.internalMessageRecipient.status.READ') },
  {
    value: 'REVOKED',
    label: t('enum.internalMessageRecipient.status.REVOKED'),
  },
  {
    value: 'DELETED',
    label: t('enum.internalMessageRecipient.status.DELETED'),
  },
]);

export function internalMessageStatusLabel(
  value: InternalMessage_Status,
): string {
  const values = internalMessageStatusList.value;
  const matchedItem = values.find((item) => item.value === value);
  return matchedItem ? matchedItem.label : '';
}

const INTERNAL_MESSAGE_STATUS_COLOR_MAP: Record<string, string> = {
  ARCHIVED: '#86909C',
  DELETED: '#C9CDD4',
  DRAFT: '#9CA3AF',
  PUBLISHED: '#00B42A',
  REVOKED: '#F53F3F',
  SCHEDULED: '#165DFF',
  DEFAULT: '#E5E7EB',
};

export function internalMessageStatusColor(
  status: InternalMessage_Status,
): string {
  return (
    INTERNAL_MESSAGE_STATUS_COLOR_MAP[status as string] ??
    INTERNAL_MESSAGE_STATUS_COLOR_MAP.DEFAULT ??
    '#E5E7EB'
  );
}

export function internalMessageTypeLabel(value: InternalMessage_Type): string {
  const values = internalMessageTypeList.value;
  const matchedItem = values.find((item) => item.value === value);
  return matchedItem ? matchedItem.label : '';
}

const INTERNAL_MESSAGE_TYPE_COLOR_MAP: Record<string, string> = {
  GROUP: '#00B42A',
  NOTIFICATION: '#165DFF',
  PRIVATE: '#722ED1',
  DEFAULT: '#C9CDD4',
};

export function internalMessageTypeColor(type: InternalMessage_Type): string {
  return (
    INTERNAL_MESSAGE_TYPE_COLOR_MAP[type as string] ??
    INTERNAL_MESSAGE_TYPE_COLOR_MAP.DEFAULT ??
    '#C9CDD4'
  );
}

export function internalMessageRecipientStatusLabel(
  value: InternalMessageRecipient_Status,
): string {
  const values = internalMessageRecipientStatusList.value;
  const matchedItem = values.find((item) => item.value === value);
  return matchedItem ? matchedItem.label : '';
}

const INTERNAL_MESSAGE_RECIPIENT_COLOR_THEME: Record<
  string,
  Record<string, string>
> = {
  light: {
    DELETED: '#C9CDD4',
    READ: '#86909C',
    RECEIVED: '#165DFF',
    REVOKED: '#F53F3F',
    SENT: '#4096FF',
    DEFAULT: '#E5E7EB',
  },
  dark: {
    DELETED: '#6E7681',
    READ: '#4E5969',
    RECEIVED: '#2F77FF',
    REVOKED: '#F87171',
    SENT: '#69B1FF',
    DEFAULT: '#4B5563',
  },
};

export function internalMessageRecipientStatusColor(
  status: InternalMessageRecipient_Status,
  theme: 'dark' | 'light' = 'light',
): string {
  const colorMap = INTERNAL_MESSAGE_RECIPIENT_COLOR_THEME[theme] ?? {
    DEFAULT: '#E5E7EB',
  };
  return colorMap[status as string] ?? colorMap.DEFAULT ?? '#E5E7EB';
}
