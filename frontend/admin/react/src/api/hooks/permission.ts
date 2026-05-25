import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/react-query';
import {
  type permissionservicev1_Permission,
  type permissionservicev1_ListPermissionResponse,
  type permissionservicev1_GetPermissionRequest,
  type permissionservicev1_CreatePermissionRequest,
  type permissionservicev1_DeletePermissionRequest,
} from '@/api/generated/admin/service/v1';
import { makeUpdateMask, type PaginationQuery } from '@/core/transport/rest';
import { queryClient } from '@/core';
import {
  listPermissions,
  getPermission,
  createPermission,
  updatePermission,
  deletePermission,
} from '@/api/service/permission';

// ==============================
// 权限点管理
// ==============================

export function useListPermissions(
  query: PaginationQuery,
  options?: UseQueryOptions<permissionservicev1_ListPermissionResponse, Error>,
) {
  return useQuery({
    queryKey: ['listPermissions', query],
    queryFn: () => listPermissions(query),
    ...options,
  });
}

export async function fetchListPermissions(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listPermissions', params],
    queryFn: () => listPermissions(params),
    retry: 0,
  });
}

export function useGetPermission(
  req: permissionservicev1_GetPermissionRequest,
  options?: UseQueryOptions<permissionservicev1_Permission, Error>,
) {
  return useQuery({
    queryKey: ['getPermission', req],
    queryFn: () => getPermission(req),
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
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      updatePermission({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
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
