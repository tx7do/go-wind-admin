import { useMutation, type UseMutationOptions } from '@tanstack/react-query';
import {
  type permissionservicev1_CreatePermissionGroupRequest,
  type permissionservicev1_DeletePermissionGroupRequest,
  type permissionservicev1_GetPermissionGroupRequest,
  type permissionservicev1_ListPermissionGroupResponse,
  type permissionservicev1_PermissionGroup,
  type permissionservicev1_UpdatePermissionGroupRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery } from '@/core';
import { listPermissionGroups, getPermissionGroup, createPermissionGroup, updatePermissionGroup, deletePermissionGroup } from '@/api/service/permission-group';

// ==============================
// 权限组管理
// ==============================

export function useListPermissionGroups(
  options?: UseMutationOptions<
    permissionservicev1_ListPermissionGroupResponse,
    Error,
    PaginationQuery
  >,
) {
  return useMutation({
    mutationFn: (query) => listPermissionGroups(query),
    ...options,
  });
}

export function useGetPermissionGroup(
  options?: UseMutationOptions<
    permissionservicev1_PermissionGroup,
    Error,
    permissionservicev1_GetPermissionGroupRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => getPermissionGroup(req),
    ...options,
  });
}

export function useCreatePermissionGroup(
  options?: UseMutationOptions<{}, Error, permissionservicev1_CreatePermissionGroupRequest>,
) {
  return useMutation({
    mutationFn: (data) => createPermissionGroup(data),
    ...options,
  });
}

export function useUpdatePermissionGroup(
  options?: UseMutationOptions<{}, Error, permissionservicev1_UpdatePermissionGroupRequest>,
) {
  return useMutation({
    mutationFn: (data) => updatePermissionGroup(data),
    ...options,
  });
}

export function useDeletePermissionGroup(
  options?: UseMutationOptions<{}, Error, permissionservicev1_DeletePermissionGroupRequest>,
) {
  return useMutation({
    mutationFn: (req) => deletePermissionGroup(req),
    ...options,
  });
}
