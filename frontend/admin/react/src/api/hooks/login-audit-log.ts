import { useMutation, type UseMutationOptions } from '@tanstack/react-query';
import {
  type auditservicev1_GetLoginAuditLogRequest,
  type auditservicev1_ListLoginAuditLogResponse,
  type auditservicev1_LoginAuditLog,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery } from '@/core';
import { listLoginAuditLogs, getLoginAuditLog } from '@/api/service/login-audit-log';

// ==============================
// 登录审计日志
// ==============================

export function useListLoginAuditLogs(
  options?: UseMutationOptions<auditservicev1_ListLoginAuditLogResponse, Error, PaginationQuery>,
) {
  return useMutation({
    mutationFn: (query) => listLoginAuditLogs(query),
    ...options,
  });
}

export function useGetLoginAuditLog(
  options?: UseMutationOptions<
    auditservicev1_LoginAuditLog,
    Error,
    auditservicev1_GetLoginAuditLogRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => getLoginAuditLog(req),
    ...options,
  });
}
