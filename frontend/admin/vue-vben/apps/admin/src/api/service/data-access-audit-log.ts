import {
  type auditservicev1_GetDataAccessAuditLogRequest,
  createDataAccessAuditLogServiceClient,
} from '#/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '#/transport/rest';

let _instance: null | ReturnType<typeof createDataAccessAuditLogServiceClient> =
  null;

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

export async function getDataAccessAuditLog(
  request: auditservicev1_GetDataAccessAuditLogRequest,
) {
  return getDataAccessAuditLogService().Get(request);
}
