import {
  type auditservicev1_GetApiAuditLogRequest,
  createApiAuditLogServiceClient,
} from '#/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '#/transport/rest';

let _instance: null | ReturnType<typeof createApiAuditLogServiceClient> = null;

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

export async function getApiAuditLog(
  request: auditservicev1_GetApiAuditLogRequest,
) {
  return getApiAuditLogService().Get(request);
}
