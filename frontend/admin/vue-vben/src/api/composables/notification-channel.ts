import type {
  notification_channelservicev1_CreateNotificationChannelRequest,
  notification_channelservicev1_DeleteNotificationChannelRequest,
  notification_channelservicev1_NotificationChannel,
  notification_channelservicev1_SendTestEmailRequest,
  notification_channelservicev1_UpdateNotificationChannelRequest,
} from '#/api/generated/admin/service/v1';

import { useMutation, type UseMutationOptions } from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';

// ==============================
// 通知渠道（平台级配置）
// ==============================

const LIST_KEY = 'listNotificationChannels';

export async function fetchListNotificationChannels(params: {
  page?: number;
  pageSize?: number;
}) {
  return queryClient.fetchQuery({
    queryKey: [LIST_KEY, params],
    queryFn: () =>
      apiClient.notificationChannelService.ListNotificationChannel({
        page: params.page,
        pageSize: params.pageSize,
        sorting: undefined,
      }),
    staleTime: 0,
    retry: 0,
  });
}

export function useCreateNotificationChannel(
  options?: UseMutationOptions<
    notification_channelservicev1_NotificationChannel,
    Error,
    notification_channelservicev1_CreateNotificationChannelRequest
  >,
) {
  return useMutation({
    mutationFn: (req: notification_channelservicev1_CreateNotificationChannelRequest) =>
      apiClient.notificationChannelService.CreateNotificationChannel(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [LIST_KEY] });
    },
    ...options,
  });
}

export function useUpdateNotificationChannel(
  options?: UseMutationOptions<
    {},
    Error,
    notification_channelservicev1_UpdateNotificationChannelRequest
  >,
) {
  return useMutation({
    mutationFn: (req: notification_channelservicev1_UpdateNotificationChannelRequest) =>
      apiClient.notificationChannelService.UpdateNotificationChannel(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [LIST_KEY] });
    },
    ...options,
  });
}

export function useDeleteNotificationChannel(
  options?: UseMutationOptions<
    {},
    Error,
    notification_channelservicev1_DeleteNotificationChannelRequest
  >,
) {
  return useMutation({
    mutationFn: (req: notification_channelservicev1_DeleteNotificationChannelRequest) =>
      apiClient.notificationChannelService.DeleteNotificationChannel(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [LIST_KEY] });
    },
    ...options,
  });
}

export function useSendTestEmail(
  options?: UseMutationOptions<
    {},
    Error,
    notification_channelservicev1_SendTestEmailRequest
  >,
) {
  return useMutation({
    mutationFn: (req: notification_channelservicev1_SendTestEmailRequest) =>
      apiClient.notificationChannelService.SendTestEmail(req),
    ...options,
  });
}
