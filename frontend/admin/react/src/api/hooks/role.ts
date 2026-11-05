import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/react-query';
import {
  type permissionservicev1_CreateRoleRequest,
  type permissionservicev1_DeleteRoleRequest,
  type permissionservicev1_GetRoleRequest,
  type permissionservicev1_ListRoleResponse,
  type permissionservicev1_Role,
} from '@/api/generated/admin/service/v1';
import { makeUpdateMask, type PaginationQuery, queryClient } from '@/core';
import { apiClient } from '@/api/client';

// ==============================
// 角色管理
// ==============================

export function useListRoles(
  query: PaginationQuery,
  options?: UseQueryOptions<permissionservicev1_ListRoleResponse, Error>,
) {
  return useQuery({
    queryKey: ['listRoles', query],
    queryFn: () => apiClient.roleService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListRoles(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listRoles', params],
    queryFn: () => apiClient.roleService.List(params.toRawParams()),
    retry: 0,
  });
}

export function useGetRole(
  req: permissionservicev1_GetRoleRequest,
  options?: UseQueryOptions<permissionservicev1_Role, Error>,
) {
  return useQuery({
    queryKey: ['getRole', req],
    queryFn: () => apiClient.roleService.Get(req),
    ...options,
  });
}

export async function fetchGetRole(req: permissionservicev1_GetRoleRequest) {
  return queryClient.fetchQuery({
    queryKey: ['getRole', req],
    queryFn: () => apiClient.roleService.Get(req),
    retry: 0,
  });
}

export function useCreateRole(
  options?: UseMutationOptions<{}, Error, permissionservicev1_CreateRoleRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.roleService.Create(data),
    ...options,
  });
}

export function useUpdateRole(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.roleService.Update({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteRole(
  options?: UseMutationOptions<{}, Error, permissionservicev1_DeleteRoleRequest>,
) {
  return useMutation({
    mutationFn: (req) => apiClient.roleService.Delete(req),
    ...options,
  });
}
