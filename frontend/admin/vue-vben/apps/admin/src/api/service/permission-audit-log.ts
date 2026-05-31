import {
  type auditservicev1_GetPermissionAuditLogRequest,
  createPermissionAuditLogServiceClient,
} from '#/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '#/transport/rest';

let _instance: null | ReturnType<typeof createPermissionAuditLogServiceClient> =
  null;

export function getPermissionAuditLogService() {
  if (!_instance) {
    _instance = createPermissionAuditLogServiceClient(requestApi);
  }
  return _instance;
}

export async function listPermissionAuditLogs(query: PaginationQuery) {
  const params = query.toRawParams();
  return getPermissionAuditLogService().List(params);
}

export async function getPermissionAuditLog(
  request: auditservicev1_GetPermissionAuditLogRequest,
) {
  return getPermissionAuditLogService().Get(request);
}
