import { useMutation, useQueryClient, type UseMutationOptions } from '@tanstack/react-query';
import {
  type notificationservicev1_CreateNotificationTemplateRequest,
  type notificationservicev1_DeleteNotificationTemplateRequest,
  type notificationservicev1_ListNotificationTemplateResponse,
  type notificationservicev1_NotificationTemplate,
  type notificationservicev1_RenderNotificationTemplateRequest,
  type notificationservicev1_RenderNotificationTemplateResponse,
  type notificationservicev1_UpdateNotificationTemplateRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, queryClient } from '@go-wind/admin-react';
import { apiClient } from '@/api/client';

// ==============================
// 通知模板（可复用的标题/正文占位模板）
// ==============================

const LIST_KEY = 'listNotificationTemplates';

export function fetchListNotificationTemplates(
  query: PaginationQuery,
): Promise<notificationservicev1_ListNotificationTemplateResponse> {
  return queryClient.fetchQuery({
    queryKey: [LIST_KEY, query],
    queryFn: () => apiClient.notificationTemplateService.ListNotificationTemplate(query.toRawParams()),
    retry: 0,
  });
}

export function useCreateNotificationTemplate(
  options?: UseMutationOptions<
    notificationservicev1_NotificationTemplate,
    Error,
    notificationservicev1_CreateNotificationTemplateRequest
  >,
) {
  const queryClientRef = useQueryClient();
  return useMutation({
    mutationFn: (req) => apiClient.notificationTemplateService.CreateNotificationTemplate(req),
    onSuccess: () => queryClientRef.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

export function useUpdateNotificationTemplate(
  options?: UseMutationOptions<{}, Error, notificationservicev1_UpdateNotificationTemplateRequest>,
) {
  const queryClientRef = useQueryClient();
  return useMutation({
    mutationFn: (req) => apiClient.notificationTemplateService.UpdateNotificationTemplate(req),
    onSuccess: () => queryClientRef.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

export function useDeleteNotificationTemplate(
  options?: UseMutationOptions<{}, Error, notificationservicev1_DeleteNotificationTemplateRequest>,
) {
  const queryClientRef = useQueryClient();
  return useMutation({
    mutationFn: (req) => apiClient.notificationTemplateService.DeleteNotificationTemplate(req),
    onSuccess: () => queryClientRef.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

export function useRenderNotificationTemplate(
  options?: UseMutationOptions<
    notificationservicev1_RenderNotificationTemplateResponse,
    Error,
    notificationservicev1_RenderNotificationTemplateRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.notificationTemplateService.RenderNotificationTemplate(req),
    ...options,
  });
}
