import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from "@tanstack/vue-query";
import type {
  permissionservicev1_CreateRoleRequest,
  permissionservicev1_DeleteRoleRequest,
  permissionservicev1_GetRoleRequest,
  permissionservicev1_ListRoleResponse,
  permissionservicev1_Role,
} from "@/api/generated/admin/service/v1";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import { listRoles, getRole, createRole, updateRole, deleteRole } from "@/api/service/role";
import { queryClient } from "@/plugins/vue-query";

// ==============================
// 角色管理
// ==============================

export function useListRoles(
  query: PaginationQuery,
  options?: UseQueryOptions<permissionservicev1_ListRoleResponse, Error>
) {
  return useQuery({
    queryKey: ["listRoles", query],
    queryFn: () => listRoles(query),
    ...options,
  });
}

export async function fetchListRoles(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listRoles", params],
    queryFn: () => listRoles(params),
    retry: 0,
  });
}

export function useGetRole(
  req: permissionservicev1_GetRoleRequest,
  options?: UseQueryOptions<permissionservicev1_Role, Error>
) {
  return useQuery({
    queryKey: ["getRole", req],
    queryFn: () => getRole(req),
    ...options,
  });
}

export function useCreateRole(options?: UseMutationOptions<{}, Error, Record<string, any>>) {
  return useMutation({
    mutationFn: (values) => createRole({ data: { ...values } as permissionservicev1_Role }),
    ...options,
  });
}

export function useUpdateRole(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>
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
  options?: UseMutationOptions<{}, Error, permissionservicev1_DeleteRoleRequest>
) {
  return useMutation({
    mutationFn: (req) => deleteRole(req),
    ...options,
  });
}
