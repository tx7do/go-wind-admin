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
import { makeUpdateMask, PaginationQuery, type PaginationQuery as PaginationQueryType } from "@/core/transport/rest";
import { listRoles, getRole, createRole, updateRole, deleteRole } from "@/api/service/role";
import { queryClient } from "@/plugins/vue-query";
import { useAppUserStore } from "@/stores";

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

export function useCreateRole(
  options?: UseMutationOptions<{}, Error, permissionservicev1_CreateRoleRequest>
) {
  return useMutation({
    mutationFn: (data) => createRole(data),
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

// ==============================
// Store 兼容层
// ==============================

export function useRoleStore() {
  async function listRoleList(
    paging?: { page?: number; pageSize?: number },
    formValues?: null | object,
    fieldMask?: null | string,
    orderBy?: null | string[]
  ) {
    const userStore = useAppUserStore();
    const query = new PaginationQuery({
      paging,
      formValues,
      fieldMask,
      orderBy,
      isTenantUser: userStore.isTenantUser(),
    });
    return await fetchListRoles(query);
  }

  async function getRoleById(id: number) {
    return await getRole({ id });
  }

  async function createRoleData(values: Record<string, any> = {}) {
    return await createRole({ data: { ...values } });
  }

  async function updateRoleData(id: number, values: Record<string, any> = {}) {
    if ("id" in values) delete values.id;
    return await updateRole({
      id,
      data: { ...values } as any,
      updateMask: makeUpdateMask(Object.keys(values ?? {})),
    });
  }

  async function deleteRoleById(id: number) {
    return await deleteRole({ id });
  }

  return {
    listRole: listRoleList,
    getRole: getRoleById,
    createRole: createRoleData,
    updateRole: updateRoleData,
    deleteRole: deleteRoleById,
  };
}
