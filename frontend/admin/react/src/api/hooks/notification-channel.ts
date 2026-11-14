import { useMutation, useQuery, useQueryClient, type UseMutationOptions, type UseQueryOptions } from '@tanstack/react-query';
import {
  type notification_channelservicev1_CreateNotificationChannelRequest,
  type notification_channelservicev1_DeleteNotificationChannelRequest,
  type notification_channelservicev1_ListNotificationChannelResponse,
  type notification_channelservicev1_NotificationChannel,
  type notification_channelservicev1_SendTestEmailRequest,
  type notification_channelservicev1_SendTestRequest,
  type notification_channelservicev1_UpdateNotificationChannelRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, queryClient } from '@go-wind/admin-react';
import { apiClient } from '@/api/client';

// ==============================
// 通知渠道（平台级配置）
// ==============================

const LIST_KEY = 'listNotificationChannels';

export function useListNotificationChannels(
  query: PaginationQuery,
  options?: UseQueryOptions<notification_channelservicev1_ListNotificationChannelResponse, Error>,
) {
  return useQuery({
    queryKey: ['listNotificationChannels', query],
    queryFn: () => apiClient.notificationChannelService.ListNotificationChannel(query.toRawParams()),
    ...options,
  });
}

export async function fetchListNotificationChannels(query: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: [LIST_KEY, query],
    queryFn: () => apiClient.notificationChannelService.ListNotificationChannel(query.toRawParams()),
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
  const queryClientRef = useQueryClient();
  return useMutation({
    mutationFn: (req) => apiClient.notificationChannelService.CreateNotificationChannel(req),
    onSuccess: () => queryClientRef.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

export function useUpdateNotificationChannel(
  options?: UseMutationOptions<{}, Error, notification_channelservicev1_UpdateNotificationChannelRequest>,
) {
  const queryClientRef = useQueryClient();
  return useMutation({
    mutationFn: (req) => apiClient.notificationChannelService.UpdateNotificationChannel(req),
    onSuccess: () => queryClientRef.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

export function useDeleteNotificationChannel(
  options?: UseMutationOptions<{}, Error, notification_channelservicev1_DeleteNotificationChannelRequest>,
) {
  const queryClientRef = useQueryClient();
  return useMutation({
    mutationFn: (req) => apiClient.notificationChannelService.DeleteNotificationChannel(req),
    onSuccess: () => queryClientRef.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

export function useSendTestEmail(
  options?: UseMutationOptions<{}, Error, notification_channelservicev1_SendTestEmailRequest>,
) {
  return useMutation({
    mutationFn: (req) => apiClient.notificationChannelService.SendTestEmail(req),
    ...options,
  });
}

export function useSendTest(
  options?: UseMutationOptions<{}, Error, notification_channelservicev1_SendTestRequest>,
) {
  return useMutation({
    mutationFn: (req) => apiClient.notificationChannelService.SendTest(req),
    ...options,
  });
}
