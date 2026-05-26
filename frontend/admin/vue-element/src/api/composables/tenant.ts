import {
  useMutation,
  useQuery,
  type UseMutationOptions,
  type UseQueryOptions,
} from "@tanstack/vue-query";
import type {
  identityservicev1_CreateTenantRequest,
  identityservicev1_CreateTenantWithAdminUserRequest,
  identityservicev1_DeleteTenantRequest,
  identityservicev1_GetTenantRequest,
  identityservicev1_ListTenantResponse,
  identityservicev1_Tenant,
} from "@/api/generated/admin/service/v1";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import {
  listTenants,
  getTenant,
  createTenant,
  updateTenant,
  deleteTenant,
  createTenantWithAdminUser,
} from "@/api/service/tenant";
import { queryClient } from "@/plugins/vue-query";

// ==============================
// 获取租户列表
// ==============================
export function useListTenants(
  query: PaginationQuery,
  options?: UseQueryOptions<identityservicev1_ListTenantResponse, Error>
) {
  return useQuery({
    queryKey: ["listTenants", query],
    queryFn: () => listTenants(query),
    ...options,
  });
}

export async function fetchListTenants(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listTenants", params],
    queryFn: () => listTenants(params),
    retry: 0,
  });
}

// ==============================
// 获取单个租户
// ==============================
export function useGetTenant(
  req: identityservicev1_GetTenantRequest,
  options?: UseQueryOptions<identityservicev1_Tenant, Error>
) {
  return useQuery({
    queryKey: ["getTenant", req],
    queryFn: () => getTenant(req),
    ...options,
  });
}

// ==============================
// 创建租户
// ==============================
export function useCreateTenant(
  options?: UseMutationOptions<{}, Error, identityservicev1_CreateTenantRequest>
) {
  return useMutation({
    mutationFn: (data) => createTenant(data),
    ...options,
  });
}

// ==============================
// 更新租户
// ==============================
export function useUpdateTenant(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      updateTenant({
        id,
        data: {
          ...values,
        },
        updateMask: makeUpdateMask(Object.keys(values ?? [])),
      }),
    ...options,
  });
}

// ==============================
// 删除租户
// ==============================
export function useDeleteTenant(
  options?: UseMutationOptions<{}, Error, identityservicev1_DeleteTenantRequest>
) {
  return useMutation({
    mutationFn: (req) => deleteTenant(req),
    ...options,
  });
}

export function useCreateTenantWithAdminUser(
  options?: UseMutationOptions<{}, Error, identityservicev1_CreateTenantWithAdminUserRequest>
) {
  return useMutation({
    mutationFn: (req) => createTenantWithAdminUser(req),
    ...options,
  });
}
