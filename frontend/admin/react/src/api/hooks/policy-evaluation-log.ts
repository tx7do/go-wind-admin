import { useQuery, type UseQueryOptions } from '@tanstack/react-query';
import {
  type permissionservicev1_PolicyEvaluationLog,
  type permissionservicev1_GetPolicyEvaluationLogRequest,
  type permissionservicev1_ListPolicyEvaluationLogResponse,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, queryClient } from '@go-wind/admin-react';
import { apiClient } from '@/api/client';

// ==============================
// 策略评估日志管理
// ==============================

export function useListPolicyEvaluationLogs(
  query: PaginationQuery,
  options?: UseQueryOptions<permissionservicev1_ListPolicyEvaluationLogResponse, Error>,
) {
  return useQuery({
    queryKey: ['listPolicyEvaluationLogs', query],
    queryFn: () => apiClient.policyEvaluationLogService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListPolicyEvaluationLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listPolicyEvaluationLogs', params],
    queryFn: () => apiClient.policyEvaluationLogService.List(params.toRawParams()),
    retry: 0,
  });
}

export function useGetPolicyEvaluationLog(
  req: permissionservicev1_GetPolicyEvaluationLogRequest,
  options?: UseQueryOptions<permissionservicev1_PolicyEvaluationLog, Error>,
) {
  return useQuery({
    queryKey: ['getPolicyEvaluationLog', req],
    queryFn: () => apiClient.policyEvaluationLogService.Get(req),
    ...options,
  });
}
