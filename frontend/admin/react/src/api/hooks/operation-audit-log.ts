import { useQuery, type UseQueryOptions } from '@tanstack/react-query';
import {
  type auditservicev1_GetOperationAuditLogRequest,
  type auditservicev1_ListOperationAuditLogResponse,
  type auditservicev1_OperationAuditLog,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, queryClient } from '@/core';
import { listOperationAuditLogs, getOperationAuditLog } from '@/api/service/operation-audit-log';

// ==============================
// 操作审计日志
// ==============================

export function useListOperationAuditLogs(
  query: PaginationQuery,
  options?: UseQueryOptions<auditservicev1_ListOperationAuditLogResponse, Error>,
) {
  return useQuery({
    queryKey: ['listOperationAuditLogs', query],
    queryFn: () => listOperationAuditLogs(query),
    ...options,
  });
}

export async function fetchListOperationAuditLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listOperationAuditLogs', params],
    queryFn: () => listOperationAuditLogs(params),
    retry: 0,
  });
}

export function useGetOperationAuditLog(
  req: auditservicev1_GetOperationAuditLogRequest,
  options?: UseQueryOptions<auditservicev1_OperationAuditLog, Error>,
) {
  return useQuery({
    queryKey: ['getOperationAuditLog', req],
    queryFn: () => getOperationAuditLog(req),
    ...options,
  });
}
