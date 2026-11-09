import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/react-query';
import {
  type permissionservicev1_CreatePermissionGroupRequest,
  type permissionservicev1_DeletePermissionGroupRequest,
  type permissionservicev1_GetPermissionGroupRequest,
  type permissionservicev1_ListPermissionGroupResponse,
  type permissionservicev1_PermissionGroup,
} from '@/api/generated/admin/service/v1';
import { makeUpdateMask, type PaginationQuery, queryClient } from '@/core';
import { apiClient } from '@/api/client';

// ==============================
// 权限组管理
// ==============================

export function useListPermissionGroups(
  query: PaginationQuery,
  options?: UseQueryOptions<permissionservicev1_ListPermissionGroupResponse, Error>,
) {
  return useQuery({
    queryKey: ['listPermissionGroups', query],
    queryFn: () => apiClient.permissionGroupService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListPermissionGroups(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listPermissionGroups', params],
    queryFn: () => apiClient.permissionGroupService.List(params.toRawParams()),
    retry: 0,
  });
}

export function useGetPermissionGroup(
  req: permissionservicev1_GetPermissionGroupRequest,
  options?: UseQueryOptions<permissionservicev1_PermissionGroup, Error>,
) {
  return useQuery({
    queryKey: ['getPermissionGroup', req],
    queryFn: () => apiClient.permissionGroupService.Get(req),
    ...options,
  });
}

export function useCreatePermissionGroup(
  options?: UseMutationOptions<{}, Error, permissionservicev1_CreatePermissionGroupRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.permissionGroupService.Create(data),
    ...options,
  });
}

export function useUpdatePermissionGroup(
  options?: UseMutationOptions<{}, Error, { id: string; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: Record<string, any> }) =>
      apiClient.permissionGroupService.Update({
        id,
        data: { ...values, children: [] } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeletePermissionGroup(
  options?: UseMutationOptions<{}, Error, permissionservicev1_DeletePermissionGroupRequest>,
) {
  return useMutation({
    mutationFn: (req) => apiClient.permissionGroupService.Delete(req),
    ...options,
  });
}
