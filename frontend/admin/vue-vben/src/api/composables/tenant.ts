import type {
  identityservicev1_CreateTenantRequest,
  identityservicev1_CreateTenantWithAdminUserRequest,
  identityservicev1_DeleteTenantRequest,
  identityservicev1_CleanupTenantDataRequest,
  identityservicev1_GetTenantRequest,
  identityservicev1_GetTenantUsageRequest,
  identityservicev1_ListTenantResponse,
  identityservicev1_Tenant,
  identityservicev1_TenantUsage,
  identityservicev1_Tenant_AuditStatus as Tenant_AuditStatus,
  identityservicev1_Tenant_Status as Tenant_Status,
  identityservicev1_Tenant_Type as Tenant_Type,
} from '#/api/generated/admin/service/v1';

import { computed } from 'vue';

import { i18n } from '@vben/locales';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';
import { makeUpdateMask, type PaginationQuery } from '#/transport/rest';

const t = i18n.global.t;

// ==============================
// 获取租户列表
// ==============================
export function useListTenants(
  query: PaginationQuery,
  options?: UseQueryOptions<identityservicev1_ListTenantResponse, Error>,
) {
  return useQuery({
    queryKey: ['listTenants', query],
    queryFn: () => apiClient.tenantService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListTenants(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listTenants', params],
    queryFn: () => apiClient.tenantService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

// ==============================
// 获取单个租户
// ==============================
export function useGetTenant(
  req: identityservicev1_GetTenantRequest,
  options?: UseQueryOptions<identityservicev1_Tenant, Error>,
) {
  return useQuery({
    queryKey: ['getTenant', req],
    queryFn: () => apiClient.tenantService.Get(req),
    ...options,
  });
}

// ==============================
// 创建租户
// ==============================
export function useCreateTenant(
  options?: UseMutationOptions<
    object,
    Error,
    identityservicev1_CreateTenantRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => apiClient.tenantService.Create(data),
    ...options,
  });
}

// ==============================
// 更新租户
// ==============================
export function useUpdateTenant(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.tenantService.Update({
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
  options?: UseMutationOptions<
    object,
    Error,
    identityservicev1_DeleteTenantRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.tenantService.Delete(req),
    ...options,
  });
}

export function useCreateTenantWithAdminUser(
  options?: UseMutationOptions<
    object,
    Error,
    identityservicev1_CreateTenantWithAdminUserRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.tenantService.CreateTenantWithAdminUser(req),
    ...options,
  });
}

export function useTenantExists(
  options?: UseMutationOptions<object, Error, { code: string; name: string }>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.tenantService.TenantExists(data),
    ...options,
  });
}

// ==============================
// 租户用量查询
// ==============================
export function useGetTenantUsage(
  req: identityservicev1_GetTenantUsageRequest,
  options?: UseQueryOptions<identityservicev1_TenantUsage, Error>,
) {
  return useQuery({
    queryKey: ['getTenantUsage', req],
    queryFn: () => apiClient.tenantService.GetUsage(req),
    ...options,
  });
}

export async function fetchTenantUsage(req: identityservicev1_GetTenantUsageRequest) {
  return queryClient.fetchQuery({
    queryKey: ['getTenantUsage', req],
    queryFn: () => apiClient.tenantService.GetUsage(req),
    staleTime: 0,
    retry: 0,
  });
}

// ==============================
// 清理租户数据
// ==============================
export function useCleanupTenantData(
  options?: UseMutationOptions<
    object,
    Error,
    identityservicev1_CleanupTenantDataRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.tenantService.CleanupData(req),
    ...options,
  });
}

// ==============================
// 租户枚举与工具函数
// ==============================

export const tenantTypeList = computed(() => [
  { value: 'TRIAL', label: t('enum.tenant.type.TRIAL') },
  { value: 'PAID', label: t('enum.tenant.type.PAID') },
  { value: 'INTERNAL', label: t('enum.tenant.type.INTERNAL') },
  { value: 'PARTNER', label: t('enum.tenant.type.PARTNER') },
  { value: 'CUSTOM', label: t('enum.tenant.type.CUSTOM') },
]);

export function tenantTypeToName(tenantType: Tenant_Type) {
  const values = tenantTypeList.value;
  const matchedItem = values.find((item) => item.value === tenantType);
  return matchedItem ? matchedItem.label : '';
}

// 类型分类色使用 antd 预设调色板名（由主题 token 驱动，亮/暗自动切换）。
export function tenantTypeToColor(
  tenantType: Tenant_Type,
): 'blue' | 'green' | 'purple' | 'orange' | 'default' {
  switch (tenantType) {
    case 'CUSTOM': {
      return 'blue';
    }
    case 'INTERNAL': {
      return 'blue';
    }
    case 'PAID': {
      return 'green';
    }
    case 'PARTNER': {
      return 'purple';
    }
    case 'TRIAL': {
      return 'orange';
    }
    default: {
      return 'default';
    }
  }
}

export const tenantStatusList = computed(() => [
  { value: 'ON', label: t('enum.tenant.status.ON') },
  { value: 'OFF', label: t('enum.tenant.status.OFF') },
  { value: 'EXPIRED', label: t('enum.tenant.status.EXPIRED') },
  { value: 'FREEZE', label: t('enum.tenant.status.FREEZE') },
]);

export function tenantStatusToName(tenantStatus: Tenant_Status) {
  const values = tenantStatusList.value;
  const matchedItem = values.find((item) => item.value === tenantStatus);
  return matchedItem ? matchedItem.label : '';
}

// 状态色使用 antd 预设状态名（由主题 token 驱动）。
// 语义：ON=success，FREEZE=warning，EXPIRED/OFF/default=default。
export function tenantStatusToColor(
  tenantStatus: Tenant_Status,
): 'success' | 'warning' | 'default' {
  switch (tenantStatus) {
    case 'FREEZE': {
      return 'warning';
    }
    case 'ON': {
      return 'success';
    }
    default: {
      return 'default';
    }
  }
}

export const tenantAuditStatusList = computed(() => [
  { value: 'PENDING', label: t('enum.tenant.auditStatus.PENDING') },
  { value: 'APPROVED', label: t('enum.tenant.auditStatus.APPROVED') },
  { value: 'REJECTED', label: t('enum.tenant.auditStatus.REJECTED') },
]);

export function tenantAuditStatusToName(tenantAuditStatus: Tenant_AuditStatus) {
  const values = tenantAuditStatusList.value;
  const matchedItem = values.find((item) => item.value === tenantAuditStatus);
  return matchedItem ? matchedItem.label : '';
}

// 审核状态色使用 antd 预设状态名（由主题 token 驱动）。
// 语义：APPROVED=success，PENDING=warning，REJECTED=error，default=default。
export function tenantAuditStatusToColor(
  tenantAuditStatus: Tenant_AuditStatus,
): 'success' | 'warning' | 'error' | 'default' {
  switch (tenantAuditStatus) {
    case 'APPROVED': {
      return 'success';
    }
    case 'PENDING': {
      return 'warning';
    }
    case 'REJECTED': {
      return 'error';
    }
    default: {
      return 'default';
    }
  }
}
