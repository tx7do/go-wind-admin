import {
  createRoleServiceClient,
  type permissionservicev1_CreateRoleRequest,
  type permissionservicev1_DeleteRoleRequest,
  type permissionservicev1_GetRoleRequest,
  type permissionservicev1_UpdateRoleRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '@/core';

let _instance: ReturnType<typeof createRoleServiceClient> | null = null;

export function getRoleService() {
  if (!_instance) {
    _instance = createRoleServiceClient(requestApi);
  }
  return _instance;
}

export async function listRoles(query: PaginationQuery) {
  const params = query.toRawParams();
  return getRoleService().List(params);
}

export async function getRole(request: permissionservicev1_GetRoleRequest) {
  return getRoleService().Get(request);
}

export async function createRole(request: permissionservicev1_CreateRoleRequest) {
  return getRoleService().Create(request);
}

export async function updateRole(request: permissionservicev1_UpdateRoleRequest) {
  return getRoleService().Update(request);
}

export async function deleteRole(request: permissionservicev1_DeleteRoleRequest) {
  return getRoleService().Delete(request);
}
