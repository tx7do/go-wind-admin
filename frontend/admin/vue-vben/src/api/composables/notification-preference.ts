import type {
  notificationservicev1_ListMyNotificationCategoriesResponse,
  notificationservicev1_NotificationPreference,
  notificationservicev1_UpdateNotificationPreferenceRequest,
} from '#/api/generated/admin/service/v1';

import {
  useMutation,
  useQuery,
  type UseMutationOptions,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import { apiClient } from '#/api/client';

// ==============================
// 个人中心：通知偏好（自助视图，通知域 P3）
// 静音时段只抑制 SSE 实时推送；分类退订只约束全员广播。
// ==============================

const PREF_KEY = 'myNotificationPreference';
const CATEGORIES_KEY = 'myNotifiableCategories';

export function useMyNotificationPreference(
  options?: UseQueryOptions<notificationservicev1_NotificationPreference, Error>,
) {
  return useQuery({
    queryKey: [PREF_KEY],
    queryFn: () => apiClient.notificationPreferenceService.GetMyNotificationPreference({}),
    ...options,
  });
}

export function useMyNotifiableCategories(
  options?: UseQueryOptions<
    notificationservicev1_ListMyNotificationCategoriesResponse,
    Error
  >,
) {
  return useQuery({
    queryKey: [CATEGORIES_KEY],
    queryFn: () => apiClient.notificationPreferenceService.ListMyNotificationCategories({}),
    ...options,
  });
}

export function useUpdateMyNotificationPreference(
  options?: UseMutationOptions<
    notificationservicev1_NotificationPreference,
    Error,
    notificationservicev1_UpdateNotificationPreferenceRequest
  >,
) {
  return useMutation({
    mutationFn: (req: notificationservicev1_UpdateNotificationPreferenceRequest) =>
      apiClient.notificationPreferenceService.UpdateMyNotificationPreference(req),
    ...options,
  });
}
