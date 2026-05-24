import {
  createLoginAuditLogServiceClient,
  type auditservicev1_GetLoginAuditLogRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '@/core';

let _instance: ReturnType<typeof createLoginAuditLogServiceClient> | null = null;

export function getLoginAuditLogService() {
  if (!_instance) {
    _instance = createLoginAuditLogServiceClient(requestApi);
  }
  return _instance;
}

export async function listLoginAuditLogs(query: PaginationQuery) {
  const params = query.toRawParams();
  return getLoginAuditLogService().List(params);
}

export async function getLoginAuditLog(request: auditservicev1_GetLoginAuditLogRequest) {
  return getLoginAuditLogService().Get(request);
}
