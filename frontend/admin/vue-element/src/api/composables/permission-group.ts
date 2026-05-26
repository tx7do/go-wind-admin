import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from "@tanstack/vue-query";
import type {
  permissionservicev1_CreatePermissionGroupRequest,
  permissionservicev1_DeletePermissionGroupRequest,
  permissionservicev1_GetPermissionGroupRequest,
  permissionservicev1_ListPermissionGroupResponse,
  permissionservicev1_PermissionGroup,
} from "@/api/generated/admin/service/v1";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import {
  listPermissionGroups,
  getPermissionGroup,
  createPermissionGroup,
  updatePermissionGroup,
  deletePermissionGroup,
} from "@/api/service/permission-group";
import { queryClient } from "@/plugins/vue-query";

// ==============================
// 权限组管理
// ==============================

export function useListPermissionGroups(
  query: PaginationQuery,
  options?: UseQueryOptions<permissionservicev1_ListPermissionGroupResponse, Error>
) {
  return useQuery({
    queryKey: ["listPermissionGroups", query],
    queryFn: () => listPermissionGroups(query),
    ...options,
  });
}

export async function fetchListPermissionGroups(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listPermissionGroups", params],
    queryFn: () => listPermissionGroups(params),
    retry: 0,
  });
}

export function useGetPermissionGroup(
  req: permissionservicev1_GetPermissionGroupRequest,
  options?: UseQueryOptions<permissionservicev1_PermissionGroup, Error>
) {
  return useQuery({
    queryKey: ["getPermissionGroup", req],
    queryFn: () => getPermissionGroup(req),
    ...options,
  });
}

export function useCreatePermissionGroup(
  options?: UseMutationOptions<{}, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      createPermissionGroup({
        data: { ...values } as permissionservicev1_PermissionGroup,
      }),
    ...options,
  });
}

export function useUpdatePermissionGroup(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      updatePermissionGroup({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeletePermissionGroup(
  options?: UseMutationOptions<{}, Error, permissionservicev1_DeletePermissionGroupRequest>
) {
  return useMutation({
    mutationFn: (req) => deletePermissionGroup(req),
    ...options,
  });
}
