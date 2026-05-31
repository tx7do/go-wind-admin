import {
  type auditservicev1_GetLoginAuditLogRequest,
  createLoginAuditLogServiceClient,
} from '#/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '#/transport/rest';

let _instance: null | ReturnType<typeof createLoginAuditLogServiceClient> =
  null;

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

export async function getLoginAuditLog(
  request: auditservicev1_GetLoginAuditLogRequest,
) {
  return getLoginAuditLogService().Get(request);
}
