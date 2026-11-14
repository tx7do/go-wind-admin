import { useMutation, useQueryClient, type UseMutationOptions } from '@tanstack/react-query';
import {
  type monitor_alertservicev1_CreateMonitorAlertRuleRequest,
  type monitor_alertservicev1_DeleteMonitorAlertRuleRequest,
  type monitor_alertservicev1_EvaluateMonitorAlertsResponse,
  type monitor_alertservicev1_ListMonitorAlertRuleResponse,
  type monitor_alertservicev1_MonitorAlertRule,
  type monitor_alertservicev1_UpdateMonitorAlertRuleRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, queryClient } from '@go-wind/admin-react';
import { apiClient } from '@/api/client';

// ==============================
// 监控告警规则（指标阈值 → 触发通知）
// ==============================

const LIST_KEY = 'listMonitorAlertRules';

export function fetchListMonitorAlertRules(
  query: PaginationQuery,
): Promise<monitor_alertservicev1_ListMonitorAlertRuleResponse> {
  return queryClient.fetchQuery({
    queryKey: [LIST_KEY, query],
    queryFn: () => apiClient.monitorAlertService.ListMonitorAlertRule(query.toRawParams()),
    retry: 0,
  });
}

export function useCreateMonitorAlertRule(
  options?: UseMutationOptions<
    monitor_alertservicev1_MonitorAlertRule,
    Error,
    monitor_alertservicev1_CreateMonitorAlertRuleRequest
  >,
) {
  const queryClientRef = useQueryClient();
  return useMutation({
    mutationFn: (req) => apiClient.monitorAlertService.CreateMonitorAlertRule(req),
    onSuccess: () => queryClientRef.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

export function useUpdateMonitorAlertRule(
  options?: UseMutationOptions<{}, Error, monitor_alertservicev1_UpdateMonitorAlertRuleRequest>,
) {
  const queryClientRef = useQueryClient();
  return useMutation({
    mutationFn: (req) => apiClient.monitorAlertService.UpdateMonitorAlertRule(req),
    onSuccess: () => queryClientRef.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

export function useDeleteMonitorAlertRule(
  options?: UseMutationOptions<{}, Error, monitor_alertservicev1_DeleteMonitorAlertRuleRequest>,
) {
  const queryClientRef = useQueryClient();
  return useMutation({
    mutationFn: (req) => apiClient.monitorAlertService.DeleteMonitorAlertRule(req),
    onSuccess: () => queryClientRef.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

export function useEvaluateMonitorAlerts(
  options?: UseMutationOptions<monitor_alertservicev1_EvaluateMonitorAlertsResponse, Error, {}>,
) {
  return useMutation({
    mutationFn: () => apiClient.monitorAlertService.EvaluateMonitorAlerts({}),
    ...options,
  });
}
