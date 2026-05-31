import {
  createPermissionGroupServiceClient,
  type permissionservicev1_CreatePermissionGroupRequest,
  type permissionservicev1_DeletePermissionGroupRequest,
  type permissionservicev1_GetPermissionGroupRequest,
  type permissionservicev1_UpdatePermissionGroupRequest,
} from '#/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '#/transport/rest';

let _instance: null | ReturnType<typeof createPermissionGroupServiceClient> =
  null;

export function getPermissionGroupService() {
  if (!_instance) {
    _instance = createPermissionGroupServiceClient(requestApi);
  }
  return _instance;
}

export async function listPermissionGroups(query: PaginationQuery) {
  const params = query.toRawParams();
  return getPermissionGroupService().List(params);
}

export async function getPermissionGroup(
  request: permissionservicev1_GetPermissionGroupRequest,
) {
  return getPermissionGroupService().Get(request);
}

export async function createPermissionGroup(
  request: permissionservicev1_CreatePermissionGroupRequest,
) {
  return getPermissionGroupService().Create(request);
}

export async function updatePermissionGroup(
  request: permissionservicev1_UpdatePermissionGroupRequest,
) {
  return getPermissionGroupService().Update(request);
}

export async function deletePermissionGroup(
  request: permissionservicev1_DeletePermissionGroupRequest,
) {
  return getPermissionGroupService().Delete(request);
}
