import { computed } from "vue";
import type { TagType } from "./shared";
import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from "@tanstack/vue-query";
import type {
  internal_messageservicev1_ListInternalMessageResponse,
  internal_messageservicev1_InternalMessage,
  internal_messageservicev1_GetInternalMessageRequest,
  internal_messageservicev1_DeleteInternalMessageRequest,
  internal_messageservicev1_SendMessageRequest,
  internal_messageservicev1_SendMessageResponse,
  internal_messageservicev1_RevokeMessageRequest,
  internal_messageservicev1_ListInternalMessageCategoryResponse,
  internal_messageservicev1_InternalMessageCategory,
  internal_messageservicev1_GetInternalMessageCategoryRequest,
  internal_messageservicev1_CreateInternalMessageCategoryRequest,
  internal_messageservicev1_DeleteInternalMessageCategoryRequest,
  internal_messageservicev1_ListUserInboxResponse,
  internal_messageservicev1_DeleteNotificationFromInboxRequest,
  internal_messageservicev1_MarkNotificationAsReadRequest,
  internal_messageservicev1_InternalMessage_Status as InternalMessage_Status,
  internal_messageservicev1_InternalMessage_Type as InternalMessage_Type,
  internal_messageservicev1_InternalMessageRecipient_Status as InternalMessageRecipient_Status,
} from "@/api/generated/admin/service/v1";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import { apiClient } from "@/api/client";

import { queryClient } from "@/plugins/vue-query";
import { i18n } from "@/core/i18n";

const t = i18n.global.t;

// ==============================
// 内部消息管理
// ==============================
export function useListInternalMessages(
  query: PaginationQuery,
  options?: UseQueryOptions<internal_messageservicev1_ListInternalMessageResponse, Error>
) {
  return useQuery({
    queryKey: ["listInternalMessages", query],
    queryFn: () => apiClient.internalMessageService.ListMessage(query.toRawParams()),
    ...options,
  });
}

export async function fetchListInternalMessages(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listInternalMessages", params],
    queryFn: () => apiClient.internalMessageService.ListMessage(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetInternalMessage(
  req: internal_messageservicev1_GetInternalMessageRequest,
  options?: UseQueryOptions<internal_messageservicev1_InternalMessage, Error>
) {
  return useQuery({
    queryKey: ["getInternalMessage", req],
    queryFn: () => apiClient.internalMessageService.GetMessage(req),
    ...options,
  });
}

export async function fetchGetInternalMessage(
  params: internal_messageservicev1_GetInternalMessageRequest
) {
  return queryClient.fetchQuery({
    queryKey: ["getInternalMessage", params],
    queryFn: () => apiClient.internalMessageService.GetMessage(params),
    staleTime: 0,
    retry: 0,
  });
}

export function useUpdateInternalMessage(
  options?: UseMutationOptions<{}, Error, { id: string; values: Record<string, any> }>
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
  options?: UseMutationOptions<{}, Error, internal_messageservicev1_DeleteInternalMessageRequest>
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
  >
) {
  return useMutation({
    mutationFn: (data) => apiClient.internalMessageService.SendMessage(data),
    ...options,
  });
}

export function useRevokeMessage(
  options?: UseMutationOptions<{}, Error, internal_messageservicev1_RevokeMessageRequest>
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
  options?: UseQueryOptions<internal_messageservicev1_ListInternalMessageCategoryResponse, Error>
) {
  return useQuery({
    queryKey: ["listMessageCategories", query],
    queryFn: () => apiClient.internalMessageCategoryService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListMessageCategories(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listMessageCategories", params],
    queryFn: () => apiClient.internalMessageCategoryService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetMessageCategory(
  req: internal_messageservicev1_GetInternalMessageCategoryRequest,
  options?: UseQueryOptions<internal_messageservicev1_InternalMessageCategory, Error>
) {
  return useQuery({
    queryKey: ["getMessageCategory", req],
    queryFn: () => apiClient.internalMessageCategoryService.Get(req),
    ...options,
  });
}

export function useCreateMessageCategory(
  options?: UseMutationOptions<
    {},
    Error,
    internal_messageservicev1_CreateInternalMessageCategoryRequest
  >
) {
  return useMutation({
    mutationFn: (data) => apiClient.internalMessageCategoryService.Create(data),
    ...options,
  });
}

export function useUpdateMessageCategory(
  options?: UseMutationOptions<{}, Error, { id: string; values: Record<string, any> }>
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
  >
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
  // queryKey/queryFn 由本 hook 固定构造，调用方只传行为选项（如 enabled）
  options?: Omit<
    UseQueryOptions<internal_messageservicev1_ListUserInboxResponse, Error>,
    "queryKey" | "queryFn"
  >
) {
  return useQuery({
    queryKey: ["listUserInbox", query],
    queryFn: () => apiClient.internalMessageRecipientService.ListUserInbox(query.toRawParams()),
    ...options,
  });
}

export async function fetchListUserInbox(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listUserInbox", params],
    queryFn: () => apiClient.internalMessageRecipientService.ListUserInbox(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useDeleteNotificationFromInbox(
  options?: UseMutationOptions<
    {},
    Error,
    internal_messageservicev1_DeleteNotificationFromInboxRequest
  >
) {
  return useMutation({
    mutationFn: (data) =>
      apiClient.internalMessageRecipientService.DeleteNotificationFromInbox(data),
    ...options,
  });
}

export function useMarkNotificationAsRead(
  options?: UseMutationOptions<{}, Error, internal_messageservicev1_MarkNotificationAsReadRequest>
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
  { value: "DRAFT", label: t("enum.internalMessage.status.DRAFT") },
  { value: "PUBLISHED", label: t("enum.internalMessage.status.PUBLISHED") },
  { value: "SCHEDULED", label: t("enum.internalMessage.status.SCHEDULED") },
  { value: "REVOKED", label: t("enum.internalMessage.status.REVOKED") },
  { value: "ARCHIVED", label: t("enum.internalMessage.status.ARCHIVED") },
  { value: "DELETED", label: t("enum.internalMessage.status.DELETED") },
]);

export const internalMessageTypeList = computed(() => [
  { value: "NOTIFICATION", label: t("enum.internalMessage.type.NOTIFICATION") },
  { value: "PRIVATE", label: t("enum.internalMessage.type.PRIVATE") },
  { value: "GROUP", label: t("enum.internalMessage.type.GROUP") },
]);

export const internalMessageRecipientStatusList = computed(() => [
  { value: "SENT", label: t("enum.internalMessageRecipient.status.SENT") },
  { value: "RECEIVED", label: t("enum.internalMessageRecipient.status.RECEIVED") },
  { value: "READ", label: t("enum.internalMessageRecipient.status.READ") },
  { value: "REVOKED", label: t("enum.internalMessageRecipient.status.REVOKED") },
  { value: "DELETED", label: t("enum.internalMessageRecipient.status.DELETED") },
]);

export function internalMessageStatusLabel(value: InternalMessage_Status): string {
  const values = internalMessageStatusList.value;
  const matchedItem = values.find((item) => item.value === value);
  return matchedItem ? matchedItem.label : "";
}

const INTERNAL_MESSAGE_STATUS_TAG_TYPE_MAP: Record<string, TagType> = {
  ARCHIVED: "info",
  DELETED: "info",
  DRAFT: "info",
  PUBLISHED: "success",
  REVOKED: "danger",
  SCHEDULED: "primary",
  DEFAULT: "info",
};

export function internalMessageStatusToType(status: InternalMessage_Status): TagType {
  return (
    INTERNAL_MESSAGE_STATUS_TAG_TYPE_MAP[status as string] ||
    INTERNAL_MESSAGE_STATUS_TAG_TYPE_MAP.DEFAULT
  );
}

export function internalMessageTypeLabel(value: InternalMessage_Type): string {
  const values = internalMessageTypeList.value;
  const matchedItem = values.find((item) => item.value === value);
  return matchedItem ? matchedItem.label : "";
}

const INTERNAL_MESSAGE_TYPE_TAG_TYPE_MAP: Record<string, TagType> = {
  GROUP: "success",
  NOTIFICATION: "primary",
  PRIVATE: "warning",
  DEFAULT: "info",
};

export function internalMessageTypeToType(type: InternalMessage_Type): TagType {
  return (
    INTERNAL_MESSAGE_TYPE_TAG_TYPE_MAP[type as string] ||
    INTERNAL_MESSAGE_TYPE_TAG_TYPE_MAP.DEFAULT
  );
}

export function internalMessageRecipientStatusLabel(
  value: InternalMessageRecipient_Status
): string {
  const values = internalMessageRecipientStatusList.value;
  const matchedItem = values.find((item) => item.value === value);
  return matchedItem ? matchedItem.label : "";
}

const INTERNAL_MESSAGE_RECIPIENT_TAG_TYPE_MAP: Record<string, TagType> = {
  DELETED: "info",
  READ: "success",
  RECEIVED: "primary",
  REVOKED: "danger",
  SENT: "warning",
  DEFAULT: "info",
};

export function internalMessageRecipientStatusToType(
  status: InternalMessageRecipient_Status
): TagType {
  return (
    INTERNAL_MESSAGE_RECIPIENT_TAG_TYPE_MAP[status as string] ||
    INTERNAL_MESSAGE_RECIPIENT_TAG_TYPE_MAP.DEFAULT
  );
}
