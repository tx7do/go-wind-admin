import {
  createApiAuditLogServiceClient,
  type auditservicev1_GetApiAuditLogRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '@/core';

let _instance: ReturnType<typeof createApiAuditLogServiceClient> | null = null;

export function getApiAuditLogService() {
  if (!_instance) {
    _instance = createApiAuditLogServiceClient(requestApi);
  }
  return _instance;
}

export async function listApiAuditLogs(query: PaginationQuery) {
  const params = query.toRawParams();
  return getApiAuditLogService().List(params);
}

export async function getApiAuditLog(request: auditservicev1_GetApiAuditLogRequest) {
  return getApiAuditLogService().Get(request);
}
