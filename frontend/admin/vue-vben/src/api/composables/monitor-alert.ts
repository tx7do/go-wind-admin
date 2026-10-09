import type {
  monitor_alertservicev1_CreateMonitorAlertRuleRequest,
  monitor_alertservicev1_DeleteMonitorAlertRuleRequest,
  monitor_alertservicev1_EvaluateMonitorAlertsRequest,
  monitor_alertservicev1_EvaluateMonitorAlertsResponse,
  monitor_alertservicev1_ListMonitorAlertRuleResponse,
  monitor_alertservicev1_MonitorAlertRule,
  monitor_alertservicev1_UpdateMonitorAlertRuleRequest,
} from '#/api/generated/admin/service/v1';

import { useMutation } from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { PaginationQuery } from '#/api';
import { queryClient } from '#/plugins/vue-query';

// ==============================
// 监控告警规则（指标阈值 → 触发通知，联动通知域）
// ==============================
//
// 渠道/目标显式写在规则上（不经路由表）；评估由系统级常驻任务每 5 分钟跑一轮，
// 「立即评估」手动触发同一内核。告警→恢复成对通知，持续越限按冷却间隔重发。

const LIST_KEY = 'listMonitorAlertRules';

/** 分页查询告警规则（非 Hook，列表页 query / 手动调用） */
export async function fetchListMonitorAlertRules(
  params: PaginationQuery,
): Promise<monitor_alertservicev1_ListMonitorAlertRuleResponse> {
  return queryClient.fetchQuery({
    queryKey: [LIST_KEY, params],
    queryFn: () => apiClient.monitorAlertService.ListMonitorAlertRule(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

/** 创建规则：CRUD 请求体必须包 { data: {...} } */
export function useCreateMonitorAlertRule(
  options?: Parameters<
    typeof useMutation<
      monitor_alertservicev1_MonitorAlertRule,
      Error,
      monitor_alertservicev1_CreateMonitorAlertRuleRequest
    >
  >[0],
) {
  return useMutation({
    mutationFn: (req: monitor_alertservicev1_CreateMonitorAlertRuleRequest) =>
      apiClient.monitorAlertService.CreateMonitorAlertRule(req),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

/** 更新规则 */
export function useUpdateMonitorAlertRule(
  options?: Parameters<
    typeof useMutation<any, Error, monitor_alertservicev1_UpdateMonitorAlertRuleRequest>
  >[0],
) {
  return useMutation({
    mutationFn: (req: monitor_alertservicev1_UpdateMonitorAlertRuleRequest) =>
      apiClient.monitorAlertService.UpdateMonitorAlertRule(req),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

/** 删除规则 */
export function useDeleteMonitorAlertRule(
  options?: Parameters<
    typeof useMutation<any, Error, monitor_alertservicev1_DeleteMonitorAlertRuleRequest>
  >[0],
) {
  return useMutation({
    mutationFn: (req: monitor_alertservicev1_DeleteMonitorAlertRuleRequest) =>
      apiClient.monitorAlertService.DeleteMonitorAlertRule(req),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

/** 立即评估：与周期扫描同一内核，返回各启用规则的求值结论 */
export function useEvaluateMonitorAlerts(
  options?: Parameters<
    typeof useMutation<
      monitor_alertservicev1_EvaluateMonitorAlertsResponse,
      Error,
      monitor_alertservicev1_EvaluateMonitorAlertsRequest
    >
  >[0],
) {
  return useMutation({
    mutationFn: (req: monitor_alertservicev1_EvaluateMonitorAlertsRequest) =>
      apiClient.monitorAlertService.EvaluateMonitorAlerts(req),
    ...options,
  });
}
