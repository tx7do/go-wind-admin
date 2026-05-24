import { useMutation, type UseMutationOptions } from '@tanstack/react-query';
import {
  type permissionservicev1_Permission,
  type permissionservicev1_ListPermissionResponse,
  type permissionservicev1_GetPermissionRequest,
  type permissionservicev1_CreatePermissionRequest,
  type permissionservicev1_UpdatePermissionRequest,
  type permissionservicev1_DeletePermissionRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery } from '@/core/transport/rest';
import { listPermissions, getPermission, createPermission, updatePermission, deletePermission } from '@/api/service/permission';

// ==============================
// 权限点管理
// ==============================

export function useListPermissions(
  options?: UseMutationOptions<permissionservicev1_ListPermissionResponse, Error, PaginationQuery>,
) {
  return useMutation({
    mutationFn: (query) => listPermissions(query),
    ...options,
  });
}

export function useGetPermission(
  options?: UseMutationOptions<
    permissionservicev1_Permission,
    Error,
    permissionservicev1_GetPermissionRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => getPermission(req),
    ...options,
  });
}

export function useCreatePermission(
  options?: UseMutationOptions<{}, Error, permissionservicev1_CreatePermissionRequest>,
) {
  return useMutation({
    mutationFn: (data) => createPermission(data),
    ...options,
  });
}

export function useUpdatePermission(
  options?: UseMutationOptions<{}, Error, permissionservicev1_UpdatePermissionRequest>,
) {
  return useMutation({
    mutationFn: (data) => updatePermission(data),
    ...options,
  });
}

export function useDeletePermission(
  options?: UseMutationOptions<{}, Error, permissionservicev1_DeletePermissionRequest>,
) {
  return useMutation({
    mutationFn: (req) => deletePermission(req),
    ...options,
  });
}
