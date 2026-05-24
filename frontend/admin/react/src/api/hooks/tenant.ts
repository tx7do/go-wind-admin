import { useMutation, type UseMutationOptions } from '@tanstack/react-query';
import {
  type identityservicev1_CreateTenantRequest,
  type identityservicev1_DeleteTenantRequest,
  type identityservicev1_GetTenantRequest,
  type identityservicev1_ListTenantResponse,
  type identityservicev1_Tenant,
  type identityservicev1_UpdateTenantRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery } from '@/core/transport/rest';
import {
  listTenants,
  getTenant,
  createTenant,
  updateTenant,
  deleteTenant,
} from '@/api/service/tenant';

// ==============================
// 获取租户列表
// ==============================
export function useListTenants(
  options?: UseMutationOptions<identityservicev1_ListTenantResponse, Error, PaginationQuery>,
) {
  return useMutation({
    mutationFn: (query) => listTenants(query),
    ...options,
  });
}

// ==============================
// 获取单个租户
// ==============================
export function useGetTenant(
  options?: UseMutationOptions<identityservicev1_Tenant, Error, identityservicev1_GetTenantRequest>,
) {
  return useMutation({
    mutationFn: (req) => getTenant(req),
    ...options,
  });
}

// ==============================
// 创建租户
// ==============================
export function useCreateTenant(
  options?: UseMutationOptions<{}, Error, identityservicev1_CreateTenantRequest>,
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
  options?: UseMutationOptions<{}, Error, identityservicev1_UpdateTenantRequest>,
) {
  return useMutation({
    mutationFn: (data) => updateTenant(data),
    ...options,
  });
}

// ==============================
// 删除租户
// ==============================
export function useDeleteTenant(
  options?: UseMutationOptions<{}, Error, identityservicev1_DeleteTenantRequest>,
) {
  return useMutation({
    mutationFn: (req) => deleteTenant(req),
    ...options,
  });
}
