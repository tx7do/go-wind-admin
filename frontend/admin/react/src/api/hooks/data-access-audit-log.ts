import { useMutation, type UseMutationOptions } from '@tanstack/react-query';
import {
  type auditservicev1_DataAccessAuditLog,
  type auditservicev1_GetDataAccessAuditLogRequest,
  type auditservicev1_ListDataAccessAuditLogResponse,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery } from '@/core';
import { listDataAccessAuditLogs, getDataAccessAuditLog } from '@/api/service/data-access-audit-log';

// ==============================
// 数据访问审计日志
// ==============================

export function useListDataAccessAuditLogs(
  options?: UseMutationOptions<
    auditservicev1_ListDataAccessAuditLogResponse,
    Error,
    PaginationQuery
  >,
) {
  return useMutation({
    mutationFn: (query) => listDataAccessAuditLogs(query),
    ...options,
  });
}

export function useGetDataAccessAuditLog(
  options?: UseMutationOptions<
    auditservicev1_DataAccessAuditLog,
    Error,
    auditservicev1_GetDataAccessAuditLogRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => getDataAccessAuditLog(req),
    ...options,
  });
}
