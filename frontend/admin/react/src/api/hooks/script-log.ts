import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/react-query';
import {
  type scriptservicev1_ListScriptLogsResponse,
  type scriptservicev1_PurgeScriptLogsRequest,
  type scriptservicev1_PurgeScriptLogsResponse,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, queryClient } from '@go-wind/admin-react';
import { apiClient } from '@/api/client';

// ==============================
// 脚本执行日志
// ==============================

export function useListScriptLogs(
  query: PaginationQuery,
  options?: UseQueryOptions<scriptservicev1_ListScriptLogsResponse, Error>,
) {
  return useQuery({
    queryKey: ['listScriptLogs', query],
    queryFn: () => apiClient.scriptLogService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListScriptLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listScriptLogs', params],
    queryFn: () => apiClient.scriptLogService.List(params.toRawParams()),
    retry: 0,
  });
}

export function usePurgeScriptLogs(
  options?: UseMutationOptions<
    scriptservicev1_PurgeScriptLogsResponse,
    Error,
    scriptservicev1_PurgeScriptLogsRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.scriptLogService.Purge(req),
    ...options,
  });
}
