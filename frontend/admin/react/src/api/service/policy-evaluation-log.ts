import {
  createPolicyEvaluationLogServiceClient,
  type permissionservicev1_GetPolicyEvaluationLogRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '@/core';

let _instance: ReturnType<typeof createPolicyEvaluationLogServiceClient> | null = null;

export function getPolicyEvaluationLogService() {
  if (!_instance) {
    _instance = createPolicyEvaluationLogServiceClient(requestApi);
  }
  return _instance;
}

export async function listPolicyEvaluationLogs(query: PaginationQuery) {
  const params = query.toRawParams();
  return getPolicyEvaluationLogService().List(params);
}

export async function getPolicyEvaluationLog(request: permissionservicev1_GetPolicyEvaluationLogRequest) {
  return getPolicyEvaluationLogService().Get(request);
}
