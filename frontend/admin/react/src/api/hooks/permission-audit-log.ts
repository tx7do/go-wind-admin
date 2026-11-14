import { useQuery, type UseQueryOptions } from '@tanstack/react-query';
import {
  type auditservicev1_GetPermissionAuditLogRequest,
  type auditservicev1_ListPermissionAuditLogResponse,
  type auditservicev1_PermissionAuditLog,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, queryClient } from '@go-wind/admin-react';
import { apiClient } from '@/api/client';

// ==============================
// 权限审计日志
// ==============================

export function useListPermissionAuditLogs(
  query: PaginationQuery,
  options?: UseQueryOptions<auditservicev1_ListPermissionAuditLogResponse, Error>,
) {
  return useQuery({
    queryKey: ['listPermissionAuditLogs', query],
    queryFn: () => apiClient.permissionAuditLogService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListPermissionAuditLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listPermissionAuditLogs', params],
    queryFn: () => apiClient.permissionAuditLogService.List(params.toRawParams()),
    retry: 0,
  });
}

export function useGetPermissionAuditLog(
  req: auditservicev1_GetPermissionAuditLogRequest,
  options?: UseQueryOptions<auditservicev1_PermissionAuditLog, Error>,
) {
  return useQuery({
    queryKey: ['getPermissionAuditLog', req],
    queryFn: () => apiClient.permissionAuditLogService.Get(req),
    ...options,
  });
}
