import { useMutation, type UseMutationOptions } from '@tanstack/react-query';
import {
  type permissionservicev1_CreateRoleRequest,
  type permissionservicev1_DeleteRoleRequest,
  type permissionservicev1_GetRoleRequest,
  type permissionservicev1_ListRoleResponse,
  type permissionservicev1_Role,
  type permissionservicev1_UpdateRoleRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery } from '@/core';
import { listRoles, getRole, createRole, updateRole, deleteRole } from '@/api/service/role';

// ==============================
// 角色管理
// ==============================

export function useListRoles(
  options?: UseMutationOptions<permissionservicev1_ListRoleResponse, Error, PaginationQuery>,
) {
  return useMutation({
    mutationFn: (query) => listRoles(query),
    ...options,
  });
}

export function useGetRole(
  options?: UseMutationOptions<permissionservicev1_Role, Error, permissionservicev1_GetRoleRequest>,
) {
  return useMutation({
    mutationFn: (req) => getRole(req),
    ...options,
  });
}

export function useCreateRole(
  options?: UseMutationOptions<{}, Error, permissionservicev1_CreateRoleRequest>,
) {
  return useMutation({
    mutationFn: (data) => createRole(data),
    ...options,
  });
}

export function useUpdateRole(
  options?: UseMutationOptions<{}, Error, permissionservicev1_UpdateRoleRequest>,
) {
  return useMutation({
    mutationFn: (data) => updateRole(data),
    ...options,
  });
}

export function useDeleteRole(
  options?: UseMutationOptions<{}, Error, permissionservicev1_DeleteRoleRequest>,
) {
  return useMutation({
    mutationFn: (req) => deleteRole(req),
    ...options,
  });
}
