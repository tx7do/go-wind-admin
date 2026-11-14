import { useQuery, type UseQueryOptions } from '@tanstack/react-query';
import {
  type auditservicev1_GetOperationAuditLogRequest,
  type auditservicev1_ListOperationAuditLogResponse,
  type auditservicev1_OperationAuditLog,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, queryClient } from '@go-wind/admin-react';
import { apiClient } from '@/api/client';

// ==============================
// 操作审计日志
// ==============================

export function useListOperationAuditLogs(
  query: PaginationQuery,
  options?: UseQueryOptions<auditservicev1_ListOperationAuditLogResponse, Error>,
) {
  return useQuery({
    queryKey: ['listOperationAuditLogs', query],
    queryFn: () => apiClient.operationAuditLogService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListOperationAuditLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listOperationAuditLogs', params],
    queryFn: () => apiClient.operationAuditLogService.List(params.toRawParams()),
    retry: 0,
  });
}

export function useGetOperationAuditLog(
  req: auditservicev1_GetOperationAuditLogRequest,
  options?: UseQueryOptions<auditservicev1_OperationAuditLog, Error>,
) {
  return useQuery({
    queryKey: ['getOperationAuditLog', req],
    queryFn: () => apiClient.operationAuditLogService.Get(req),
    ...options,
  });
}
