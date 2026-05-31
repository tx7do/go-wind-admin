import type {
  permissionservicev1_DeleteRoleRequest,
  permissionservicev1_GetRoleRequest,
  permissionservicev1_ListRoleResponse,
  permissionservicev1_Role,
} from '#/api/generated/admin/service/v1';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import {
  createRole,
  deleteRole,
  getRole,
  listRoles,
  updateRole,
} from '#/api/service/role';
import { queryClient } from '#/plugins/vue-query';
import { makeUpdateMask, type PaginationQuery } from '#/transport/rest';

// ==============================
// 角色管理
// ==============================

export function useListRoles(
  query: PaginationQuery,
  options?: UseQueryOptions<permissionservicev1_ListRoleResponse, Error>,
) {
  return useQuery({
    queryKey: ['listRoles', query],
    queryFn: () => listRoles(query),
    ...options,
  });
}

export async function fetchListRoles(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listRoles', params],
    queryFn: () => listRoles(params),
    retry: 0,
  });
}

export function useGetRole(
  req: permissionservicev1_GetRoleRequest,
  options?: UseQueryOptions<permissionservicev1_Role, Error>,
) {
  return useQuery({
    queryKey: ['getRole', req],
    queryFn: () => getRole(req),
    ...options,
  });
}

export function useCreateRole(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      createRole({ data: { ...values } as permissionservicev1_Role }),
    ...options,
  });
}

export function useUpdateRole(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      updateRole({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteRole(
  options?: UseMutationOptions<
    object,
    Error,
    permissionservicev1_DeleteRoleRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => deleteRole(req),
    ...options,
  });
}
