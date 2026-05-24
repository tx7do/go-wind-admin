import {
  createOperationAuditLogServiceClient,
  type auditservicev1_GetOperationAuditLogRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '@/core';

let _instance: ReturnType<typeof createOperationAuditLogServiceClient> | null = null;

export function getOperationAuditLogService() {
  if (!_instance) {
    _instance = createOperationAuditLogServiceClient(requestApi);
  }
  return _instance;
}

export async function listOperationAuditLogs(query: PaginationQuery) {
  const params = query.toRawParams();
  return getOperationAuditLogService().List(params);
}

export async function getOperationAuditLog(request: auditservicev1_GetOperationAuditLogRequest) {
  return getOperationAuditLogService().Get(request);
}
