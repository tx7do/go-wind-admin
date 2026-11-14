import { useQuery, type UseQueryOptions } from '@tanstack/react-query';
import {
  type auditservicev1_GetLoginAuditLogRequest,
  type auditservicev1_ListLoginAuditLogResponse,
  type auditservicev1_LoginAuditLog,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, queryClient } from '@go-wind/admin-react';
import { apiClient } from '@/api/client';

// ==============================
// 登录审计日志
// ==============================

export function useListLoginAuditLogs(
  query: PaginationQuery,
  options?: UseQueryOptions<auditservicev1_ListLoginAuditLogResponse, Error>,
) {
  return useQuery({
    queryKey: ['listLoginAuditLogs', query],
    queryFn: () => apiClient.loginAuditLogService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListLoginAuditLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listLoginAuditLogs', params],
    queryFn: () => apiClient.loginAuditLogService.List(params.toRawParams()),
    retry: 0,
  });
}

export function useGetLoginAuditLog(
  req: auditservicev1_GetLoginAuditLogRequest,
  options?: UseQueryOptions<auditservicev1_LoginAuditLog, Error>,
) {
  return useQuery({
    queryKey: ['getLoginAuditLog', req],
    queryFn: () => apiClient.loginAuditLogService.Get(req),
    ...options,
  });
}
