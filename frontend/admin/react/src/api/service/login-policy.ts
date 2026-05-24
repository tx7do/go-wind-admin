import {
  createLoginPolicyServiceClient,
  type authenticationservicev1_DeleteLoginPolicyRequest,
  type authenticationservicev1_GetLoginPolicyRequest,
  type authenticationservicev1_UpdateLoginPolicyRequest,
  type authenticationservicev1_LoginPolicy,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '@/core';

let _instance: ReturnType<typeof createLoginPolicyServiceClient> | null = null;

export function getLoginPolicyService() {
  if (!_instance) {
    _instance = createLoginPolicyServiceClient(requestApi);
  }
  return _instance;
}

export async function listLoginPolicies(query: PaginationQuery) {
  const params = query.toRawParams();
  return getLoginPolicyService().List(params);
}

export async function getLoginPolicy(request: authenticationservicev1_GetLoginPolicyRequest) {
  return getLoginPolicyService().Get(request);
}

export async function createLoginPolicy(data: authenticationservicev1_LoginPolicy) {
  return getLoginPolicyService().Create({ data });
}

export async function updateLoginPolicy(request: authenticationservicev1_UpdateLoginPolicyRequest) {
  return getLoginPolicyService().Update(request);
}

export async function deleteLoginPolicy(request: authenticationservicev1_DeleteLoginPolicyRequest) {
  return getLoginPolicyService().Delete(request);
}
