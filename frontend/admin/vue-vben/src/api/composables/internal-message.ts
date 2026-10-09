import type {
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

import { apiClient } from '#/api/client';
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
    queryFn: () => apiClient.internalMessageService.ListMessage(query.toRawParams()),
    ...options,
  });
}

export async function fetchListInternalMessages(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listInternalMessages', params],
    queryFn: () => apiClient.internalMessageService.ListMessage(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetInternalMessage(
  req: internal_messageservicev1_GetInternalMessageRequest,
  options?: UseQueryOptions<internal_messageservicev1_InternalMessage, Error>,
) {
  return useQuery({
    queryKey: ['getInternalMessage', req],
    queryFn: () => apiClient.internalMessageService.GetMessage(req),
    ...options,
  });
}

export async function fetchGetInternalMessage(
  params: internal_messageservicev1_GetInternalMessageRequest,
) {
  return queryClient.fetchQuery({
    queryKey: ['getInternalMessage', params],
    queryFn: () => apiClient.internalMessageService.GetMessage(params),
    staleTime: 0,
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
      apiClient.internalMessageService.UpdateMessage({
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
  options?: UseMutationOptions<
    object,
    Error,
    internal_messageservicev1_RevokeMessageRequest
  >,
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
  options?: UseQueryOptions<
    internal_messageservicev1_ListInternalMessageCategoryResponse,
    Error
  >,
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
    staleTime: 0,
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
    queryFn: () => apiClient.internalMessageCategoryService.Get(req),
    ...options,
  });
}

export function useCreateMessageCategory(
  options?: UseMutationOptions<object, Error, Record<string, any>>
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.internalMessageCategoryService.Create({
        data: {
          ...values,
        } as internal_messageservicev1_InternalMessageCategory,
      }),
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
    object,
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
  options?: UseQueryOptions<
    internal_messageservicev1_ListUserInboxResponse,
    Error
  >,
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
    staleTime: 0,
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
    mutationFn: (data) => apiClient.internalMessageRecipientService.DeleteNotificationFromInbox(data),
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
    mutationFn: (data) => apiClient.internalMessageRecipientService.MarkNotificationAsRead(data),
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

// 状态色使用 antd 预设状态名（由主题 token 驱动）。
// 语义：PUBLISHED=success，SCHEDULED=warning，REVOKED=error，其余/default=default。
const INTERNAL_MESSAGE_STATUS_COLOR_MAP: Record<
  string,
  'success' | 'warning' | 'error' | 'default'
> = {
  ARCHIVED: 'default',
  DELETED: 'default',
  DRAFT: 'default',
  PUBLISHED: 'success',
  REVOKED: 'error',
  SCHEDULED: 'warning',
  DEFAULT: 'default',
};

export function internalMessageStatusColor(
  status: InternalMessage_Status,
): 'success' | 'warning' | 'error' | 'default' {
  return INTERNAL_MESSAGE_STATUS_COLOR_MAP[status as string] ?? 'default';
}

export function internalMessageTypeLabel(value: InternalMessage_Type): string {
  const values = internalMessageTypeList.value;
  const matchedItem = values.find((item) => item.value === value);
  return matchedItem ? matchedItem.label : '';
}

// 类型分类色使用 antd 预设调色板名（由主题 token 驱动，亮/暗自动切换）。
const INTERNAL_MESSAGE_TYPE_COLOR_MAP: Record<
  string,
  'green' | 'blue' | 'purple' | 'default'
> = {
  GROUP: 'green',
  NOTIFICATION: 'blue',
  PRIVATE: 'purple',
  DEFAULT: 'default',
};

export function internalMessageTypeColor(
  type: InternalMessage_Type,
): 'green' | 'blue' | 'purple' | 'default' {
  return INTERNAL_MESSAGE_TYPE_COLOR_MAP[type as string] ?? 'default';
}

export function internalMessageRecipientStatusLabel(
  value: InternalMessageRecipient_Status,
): string {
  const values = internalMessageRecipientStatusList.value;
  const matchedItem = values.find((item) => item.value === value);
  return matchedItem ? matchedItem.label : '';
}

// 状态色使用 antd 预设状态名（由主题 token 驱动），不再需要 light/dark 双套调色板。
// 语义：REVOKED=error，其余/default=default。theme 参数保留以维持调用签名兼容，但不再生效。
const INTERNAL_MESSAGE_RECIPIENT_COLOR_MAP: Record<
  string,
  'error' | 'default'
> = {
  DELETED: 'default',
  READ: 'default',
  RECEIVED: 'default',
  REVOKED: 'error',
  SENT: 'default',
  DEFAULT: 'default',
};

export function internalMessageRecipientStatusColor(
  status: InternalMessageRecipient_Status,
  _theme: 'dark' | 'light' = 'light',
): 'error' | 'default' {
  return (
    INTERNAL_MESSAGE_RECIPIENT_COLOR_MAP[status as string] ?? 'default'
  );
}
