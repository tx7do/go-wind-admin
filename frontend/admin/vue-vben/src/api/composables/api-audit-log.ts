import type {
  auditservicev1_ApiAuditLog,
  auditservicev1_GetApiAuditLogRequest,
  auditservicev1_ListApiAuditLogResponse,
} from '#/api/generated/admin/service/v1';
import type { PaginationQuery } from '#/transport/rest';

import { useQuery, type UseQueryOptions } from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';

// ==============================
// API 审计日志
// ==============================

export function useListApiAuditLogs(
  query: PaginationQuery,
  options?: UseQueryOptions<auditservicev1_ListApiAuditLogResponse, Error>,
) {
  return useQuery({
    queryKey: ['listApiAuditLogs', query],
    queryFn: () => apiClient.apiAuditLogService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListApiAuditLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listApiAuditLogs', params],
    queryFn: () => apiClient.apiAuditLogService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetApiAuditLog(
  req: auditservicev1_GetApiAuditLogRequest,
  options?: UseQueryOptions<auditservicev1_ApiAuditLog, Error>,
) {
  return useQuery({
    queryKey: ['getApiAuditLog', req],
    queryFn: () => apiClient.apiAuditLogService.Get(req),
    ...options,
  });
}
