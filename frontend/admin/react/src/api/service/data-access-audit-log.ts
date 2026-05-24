import {
  createDataAccessAuditLogServiceClient,
  type auditservicev1_GetDataAccessAuditLogRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '@/core';

let _instance: ReturnType<typeof createDataAccessAuditLogServiceClient> | null = null;

export function getDataAccessAuditLogService() {
  if (!_instance) {
    _instance = createDataAccessAuditLogServiceClient(requestApi);
  }
  return _instance;
}

export async function listDataAccessAuditLogs(query: PaginationQuery) {
  const params = query.toRawParams();
  return getDataAccessAuditLogService().List(params);
}

export async function getDataAccessAuditLog(request: auditservicev1_GetDataAccessAuditLogRequest) {
  return getDataAccessAuditLogService().Get(request);
}
