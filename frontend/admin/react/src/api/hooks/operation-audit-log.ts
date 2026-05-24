import { useMutation, type UseMutationOptions } from '@tanstack/react-query';
import {
  type auditservicev1_GetOperationAuditLogRequest,
  type auditservicev1_ListOperationAuditLogResponse,
  type auditservicev1_OperationAuditLog,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery } from '@/core';
import { listOperationAuditLogs, getOperationAuditLog } from '@/api/service/operation-audit-log';

// ==============================
// 操作审计日志
// ==============================

export function useListOperationAuditLogs(
  options?: UseMutationOptions<
    auditservicev1_ListOperationAuditLogResponse,
    Error,
    PaginationQuery
  >,
) {
  return useMutation({
    mutationFn: (query) => listOperationAuditLogs(query),
    ...options,
  });
}

export function useGetOperationAuditLog(
  options?: UseMutationOptions<
    auditservicev1_OperationAuditLog,
    Error,
    auditservicev1_GetOperationAuditLogRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => getOperationAuditLog(req),
    ...options,
  });
}
