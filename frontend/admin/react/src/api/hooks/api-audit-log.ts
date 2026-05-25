import { useQuery, type UseQueryOptions } from '@tanstack/react-query';
import {
  type auditservicev1_ApiAuditLog,
  type auditservicev1_GetApiAuditLogRequest,
  type auditservicev1_ListApiAuditLogResponse,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, queryClient } from '@/core';
import { listApiAuditLogs, getApiAuditLog } from '@/api/service/api-audit-log';

// ==============================
// API 审计日志
// ==============================

export function useListApiAuditLogs(
  query: PaginationQuery,
  options?: UseQueryOptions<auditservicev1_ListApiAuditLogResponse, Error>,
) {
  return useQuery({
    queryKey: ['listApiAuditLogs', query],
    queryFn: () => listApiAuditLogs(query),
    ...options,
  });
}

export async function fetchListApiAuditLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listApiAuditLogs', params],
    queryFn: () => listApiAuditLogs(params),
    retry: 0,
  });
}

export function useGetApiAuditLog(
  req: auditservicev1_GetApiAuditLogRequest,
  options?: UseQueryOptions<auditservicev1_ApiAuditLog, Error>,
) {
  return useQuery({
    queryKey: ['getApiAuditLog', req],
    queryFn: () => getApiAuditLog(req),
    ...options,
  });
}
