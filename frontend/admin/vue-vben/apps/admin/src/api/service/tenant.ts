import {
  createTenantServiceClient,
  type identityservicev1_CreateTenantRequest,
  type identityservicev1_CreateTenantWithAdminUserRequest,
  type identityservicev1_DeleteTenantRequest,
  type identityservicev1_GetTenantRequest,
  type identityservicev1_UpdateTenantRequest,
} from '#/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '#/transport/rest';

let _instance: null | ReturnType<typeof createTenantServiceClient> = null;

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

export async function createTenant(
  request: identityservicev1_CreateTenantRequest,
) {
  return getTenantService().Create(request);
}

export async function updateTenant(
  request: identityservicev1_UpdateTenantRequest,
) {
  return getTenantService().Update(request);
}

export async function deleteTenant(
  request: identityservicev1_DeleteTenantRequest,
) {
  return getTenantService().Delete(request);
}

export async function createTenantWithAdminUser(
  request: identityservicev1_CreateTenantWithAdminUserRequest,
) {
  return getTenantService().CreateTenantWithAdminUser(request);
}

export async function tenantExists(request: { code: string; name: string }) {
  return getTenantService().TenantExists(request);
}
