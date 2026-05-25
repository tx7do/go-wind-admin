import { useQuery, type UseQueryOptions } from '@tanstack/react-query';
import {
  type auditservicev1_DataAccessAuditLog,
  type auditservicev1_GetDataAccessAuditLogRequest,
  type auditservicev1_ListDataAccessAuditLogResponse,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, queryClient } from '@/core';
import {
  listDataAccessAuditLogs,
  getDataAccessAuditLog,
} from '@/api/service/data-access-audit-log';

// ==============================
// 数据访问审计日志
// ==============================

export function useListDataAccessAuditLogs(
  query: PaginationQuery,
  options?: UseQueryOptions<auditservicev1_ListDataAccessAuditLogResponse, Error>,
) {
  return useQuery({
    queryKey: ['listDataAccessAuditLogs', query],
    queryFn: () => listDataAccessAuditLogs(query),
    ...options,
  });
}

export async function fetchListDataAccessAuditLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listDataAccessAuditLogs', params],
    queryFn: () => listDataAccessAuditLogs(params),
    retry: 0,
  });
}

export function useGetDataAccessAuditLog(
  req: auditservicev1_GetDataAccessAuditLogRequest,
  options?: UseQueryOptions<auditservicev1_DataAccessAuditLog, Error>,
) {
  return useQuery({
    queryKey: ['getDataAccessAuditLog', req],
    queryFn: () => getDataAccessAuditLog(req),
    ...options,
  });
}
