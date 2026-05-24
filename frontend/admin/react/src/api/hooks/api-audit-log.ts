import { useMutation, type UseMutationOptions } from '@tanstack/react-query';
import {
  type auditservicev1_ApiAuditLog,
  type auditservicev1_GetApiAuditLogRequest,
  type auditservicev1_ListApiAuditLogResponse,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery } from '@/core';
import { listApiAuditLogs, getApiAuditLog } from '@/api/service/api-audit-log';

// ==============================
// API 审计日志
// ==============================

export function useListApiAuditLogs(
  options?: UseMutationOptions<auditservicev1_ListApiAuditLogResponse, Error, PaginationQuery>,
) {
  return useMutation({
    mutationFn: (query) => listApiAuditLogs(query),
    ...options,
  });
}

export function useGetApiAuditLog(
  options?: UseMutationOptions<
    auditservicev1_ApiAuditLog,
    Error,
    auditservicev1_GetApiAuditLogRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => getApiAuditLog(req),
    ...options,
  });
}
