import { useMutation, type UseMutationOptions } from '@tanstack/react-query';
import {
  type auditservicev1_GetPermissionAuditLogRequest,
  type auditservicev1_ListPermissionAuditLogResponse,
  type auditservicev1_PermissionAuditLog,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery } from '@/core';
import { listPermissionAuditLogs, getPermissionAuditLog } from '@/api/service/permission-audit-log';

// ==============================
// 权限审计日志
// ==============================

export function useListPermissionAuditLogs(
  options?: UseMutationOptions<
    auditservicev1_ListPermissionAuditLogResponse,
    Error,
    PaginationQuery
  >,
) {
  return useMutation({
    mutationFn: (query) => listPermissionAuditLogs(query),
    ...options,
  });
}

export function useGetPermissionAuditLog(
  options?: UseMutationOptions<
    auditservicev1_PermissionAuditLog,
    Error,
    auditservicev1_GetPermissionAuditLogRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => getPermissionAuditLog(req),
    ...options,
  });
}
