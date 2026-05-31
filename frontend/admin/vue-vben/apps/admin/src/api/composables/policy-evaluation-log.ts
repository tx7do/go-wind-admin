import type {
  permissionservicev1_GetPolicyEvaluationLogRequest,
  permissionservicev1_ListPolicyEvaluationLogResponse,
  permissionservicev1_PolicyEvaluationLog,
} from '#/api/generated/admin/service/v1';
import type { PaginationQuery } from '#/transport/rest';

import { useQuery, type UseQueryOptions } from '@tanstack/vue-query';

import {
  getPolicyEvaluationLog,
  listPolicyEvaluationLogs,
} from '#/api/service/policy-evaluation-log';
import { queryClient } from '#/plugins/vue-query';

// ==============================
// 策略评估日志管理
// ==============================

export function useListPolicyEvaluationLogs(
  query: PaginationQuery,
  options?: UseQueryOptions<
    permissionservicev1_ListPolicyEvaluationLogResponse,
    Error
  >,
) {
  return useQuery({
    queryKey: ['listPolicyEvaluationLogs', query],
    queryFn: () => listPolicyEvaluationLogs(query),
    ...options,
  });
}

export async function fetchListPolicyEvaluationLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listPolicyEvaluationLogs', params],
    queryFn: () => listPolicyEvaluationLogs(params),
    retry: 0,
  });
}

export function useGetPolicyEvaluationLog(
  req: permissionservicev1_GetPolicyEvaluationLogRequest,
  options?: UseQueryOptions<permissionservicev1_PolicyEvaluationLog, Error>,
) {
  return useQuery({
    queryKey: ['getPolicyEvaluationLog', req],
    queryFn: () => getPolicyEvaluationLog(req),
    ...options,
  });
}
