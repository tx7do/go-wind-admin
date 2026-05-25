import { useQuery, type UseQueryOptions } from '@tanstack/react-query';
import {
  type auditservicev1_GetLoginAuditLogRequest,
  type auditservicev1_ListOperationAuditLogResponse,
  type auditservicev1_LoginAuditLog,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, queryClient } from '@/core';
import { listLoginAuditLogs, getLoginAuditLog } from '@/api/service/login-audit-log';

// ==============================
// 登录审计日志
// ==============================

export function useListLoginAuditLogs(
  query: PaginationQuery,
  options?: UseQueryOptions<auditservicev1_ListOperationAuditLogResponse, Error>,
) {
  return useQuery({
    queryKey: ['listLoginAuditLogs', query],
    queryFn: () => listLoginAuditLogs(query),
    ...options,
  });
}

export async function fetchListLoginAuditLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listLoginAuditLogs', params],
    queryFn: () => listLoginAuditLogs(params),
    retry: 0,
  });
}

export function useGetLoginAuditLog(
  req: auditservicev1_GetLoginAuditLogRequest,
  options?: UseQueryOptions<auditservicev1_LoginAuditLog, Error>,
) {
  return useQuery({
    queryKey: ['getLoginAuditLog', req],
    queryFn: () => getLoginAuditLog(req),
    ...options,
  });
}
