import {
  useMutation,
  useQueryClient,
  type UseMutationOptions,
} from '@tanstack/react-query';
import {
  type notificationservicev1_CreateNotificationRuleRequest,
  type notificationservicev1_DeleteNotificationRuleRequest,
  type notificationservicev1_ListNotificationRuleResponse,
  type notificationservicev1_NotificationRule,
  type notificationservicev1_TestDispatchNotificationRequest,
  type notificationservicev1_TestDispatchNotificationResponse,
  type notificationservicev1_UpdateNotificationRuleRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, queryClient } from '@/core';
import { apiClient } from '@/api/client';

// ==============================
// 通知路由规则（事件类型 → 渠道）
// ==============================

const LIST_KEY = 'listNotificationRules';

export function fetchListNotificationRules(
  query: PaginationQuery,
): Promise<notificationservicev1_ListNotificationRuleResponse> {
  return queryClient.fetchQuery({
    queryKey: [LIST_KEY, query],
    queryFn: () => apiClient.notificationRuleService.ListNotificationRule(query.toRawParams()),
    retry: 0,
  });
}

export function useCreateNotificationRule(
  options?: UseMutationOptions<
    notificationservicev1_NotificationRule,
    Error,
    notificationservicev1_CreateNotificationRuleRequest
  >,
) {
  const queryClientRef = useQueryClient();
  return useMutation({
    mutationFn: (req) => apiClient.notificationRuleService.CreateNotificationRule(req),
    onSuccess: () => queryClientRef.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

export function useUpdateNotificationRule(
  options?: UseMutationOptions<{}, Error, notificationservicev1_UpdateNotificationRuleRequest>,
) {
  const queryClientRef = useQueryClient();
  return useMutation({
    mutationFn: (req) => apiClient.notificationRuleService.UpdateNotificationRule(req),
    onSuccess: () => queryClientRef.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

export function useDeleteNotificationRule(
  options?: UseMutationOptions<{}, Error, notificationservicev1_DeleteNotificationRuleRequest>,
) {
  const queryClientRef = useQueryClient();
  return useMutation({
    mutationFn: (req) => apiClient.notificationRuleService.DeleteNotificationRule(req),
    onSuccess: () => queryClientRef.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

/**
 * 测试投递：按这条规则当场走一遍完整投递链。
 *
 * 异步规则回的是 SENDING 而不是 SENT —— 结论落在队列那一趟，页面上要按台账状态如实显示，
 * 不能把它翻译成都市"发送成功"。
 */
export function useTestDispatchNotification(
  options?: UseMutationOptions<
    notificationservicev1_TestDispatchNotificationResponse,
    Error,
    notificationservicev1_TestDispatchNotificationRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.notificationRuleService.TestDispatchNotification(req),
    ...options,
  });
}
