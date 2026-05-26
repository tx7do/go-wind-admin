import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from "@tanstack/vue-query";
import type {
  permissionservicev1_Permission,
  permissionservicev1_ListPermissionResponse,
  permissionservicev1_GetPermissionRequest,
  permissionservicev1_CreatePermissionRequest,
  permissionservicev1_DeletePermissionRequest,
} from "@/api/generated/admin/service/v1";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import {
  listPermissions,
  getPermission,
  createPermission,
  updatePermission,
  deletePermission,
} from "@/api/service/permission";
import { queryClient } from "@/plugins/vue-query";

// ==============================
// 权限点管理
// ==============================

export function useListPermissions(
  query: PaginationQuery,
  options?: UseQueryOptions<permissionservicev1_ListPermissionResponse, Error>
) {
  return useQuery({
    queryKey: ["listPermissions", query],
    queryFn: () => listPermissions(query),
    ...options,
  });
}

export async function fetchListPermissions(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listPermissions", params],
    queryFn: () => listPermissions(params),
    retry: 0,
  });
}

export function useGetPermission(
  req: permissionservicev1_GetPermissionRequest,
  options?: UseQueryOptions<permissionservicev1_Permission, Error>
) {
  return useQuery({
    queryKey: ["getPermission", req],
    queryFn: () => getPermission(req),
    ...options,
  });
}

export function useCreatePermission(options?: UseMutationOptions<{}, Error, Record<string, any>>) {
  return useMutation({
    mutationFn: (values) =>
      createPermission({ data: { ...values } as permissionservicev1_Permission }),
    ...options,
  });
}

export function useUpdatePermission(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>
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
  options?: UseMutationOptions<{}, Error, permissionservicev1_DeletePermissionRequest>
) {
  return useMutation({
    mutationFn: (req) => deletePermission(req),
    ...options,
  });
}
