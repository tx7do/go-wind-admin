import {
  createTenantServiceClient,
  type identityservicev1_CreateTenantRequest,
  type identityservicev1_DeleteTenantRequest,
  type identityservicev1_GetTenantRequest,
  type identityservicev1_UpdateTenantRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '@/core';

let _instance: ReturnType<typeof createTenantServiceClient> | null = null;

export function getTenantService() {
  if (!_instance) {
    _instance = createTenantServiceClient(requestApi);
  }
  return _instance;
}

export async function listTenants(query: PaginationQuery) {
  const params = query.toRawParams();
  return getTenantService().List(params);
}

export async function getTenant(request: identityservicev1_GetTenantRequest) {
  return getTenantService().Get(request);
}

export async function createTenant(request: identityservicev1_CreateTenantRequest) {
  return getTenantService().Create(request);
}

export async function updateTenant(request: identityservicev1_UpdateTenantRequest) {
  return getTenantService().Update(request);
}

export async function deleteTenant(request: identityservicev1_DeleteTenantRequest) {
  return getTenantService().Delete(request);
}
