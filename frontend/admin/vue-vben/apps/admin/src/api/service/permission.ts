import {
  createPermissionServiceClient,
  type permissionservicev1_CreatePermissionRequest,
  type permissionservicev1_DeletePermissionRequest,
  type permissionservicev1_GetPermissionRequest,
  type permissionservicev1_UpdatePermissionRequest,
} from '#/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '#/transport/rest';

let _instance: null | ReturnType<typeof createPermissionServiceClient> = null;

export function getPermissionService() {
  if (!_instance) {
    _instance = createPermissionServiceClient(requestApi);
  }
  return _instance;
}

export async function listPermissions(query: PaginationQuery) {
  const params = query.toRawParams();
  return getPermissionService().List(params);
}

export async function getPermission(
  request: permissionservicev1_GetPermissionRequest,
) {
  return getPermissionService().Get(request);
}

export async function createPermission(
  request: permissionservicev1_CreatePermissionRequest,
) {
  return getPermissionService().Create(request);
}

export async function updatePermission(
  request: permissionservicev1_UpdatePermissionRequest,
) {
  return getPermissionService().Update(request);
}

export async function deletePermission(
  request: permissionservicev1_DeletePermissionRequest,
) {
  return getPermissionService().Delete(request);
}

export async function syncPermissions() {
  return getPermissionService().SyncPermissions({});
}
