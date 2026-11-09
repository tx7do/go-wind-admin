import { computed } from "vue";
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
  identityservicev1_GetTenantUsageRequest,
  identityservicev1_CleanupTenantDataRequest,
  identityservicev1_ListTenantResponse,
  identityservicev1_Tenant,
  identityservicev1_TenantUsage,
  identityservicev1_Tenant_AuditStatus as Tenant_AuditStatus,
  identityservicev1_Tenant_Status as Tenant_Status,
  identityservicev1_Tenant_Type as Tenant_Type,
} from "@/api/generated/admin/service/v1";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import { apiClient } from "@/api/client";
import { queryClient } from "@/plugins/vue-query";
import { i18n } from "@/core/i18n";
import type { TagType } from "./shared";

const t = i18n.global.t;

// ==============================
// 获取租户列表
// ==============================
export function useListTenants(
  query: PaginationQuery,
  options?: UseQueryOptions<identityservicev1_ListTenantResponse, Error>
) {
  return useQuery({
    queryKey: ["listTenants", query],
    queryFn: () => apiClient.tenantService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListTenants(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listTenants", params],
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
  options?: UseQueryOptions<identityservicev1_Tenant, Error>
) {
  return useQuery({
    queryKey: ["getTenant", req],
    queryFn: () => apiClient.tenantService.Get(req),
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
    mutationFn: (data) => apiClient.tenantService.Create(data),
    ...options,
  });
}

// ==============================
// 更新租户
// ==============================
export function useUpdateTenant(
  options?: UseMutationOptions<{}, Error, { id: string; values: Record<string, any> }>
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: Record<string, any> }) =>
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
  options?: UseMutationOptions<{}, Error, identityservicev1_DeleteTenantRequest>
) {
  return useMutation({
    mutationFn: (req) => apiClient.tenantService.Delete(req),
    ...options,
  });
}

export function useCreateTenantWithAdminUser(
  options?: UseMutationOptions<{}, Error, identityservicev1_CreateTenantWithAdminUserRequest>
) {
  return useMutation({
    mutationFn: (req) => apiClient.tenantService.CreateTenantWithAdminUser(req),
    ...options,
  });
}

// ==============================
// 租户用量查询
// ==============================
export function useGetTenantUsage(
  req: identityservicev1_GetTenantUsageRequest,
  options?: UseQueryOptions<identityservicev1_TenantUsage, Error>
) {
  return useQuery({
    queryKey: ["getTenantUsage", req],
    queryFn: () => apiClient.tenantService.GetUsage(req),
    ...options,
  });
}

export async function fetchTenantUsage(req: identityservicev1_GetTenantUsageRequest) {
  return queryClient.fetchQuery({
    queryKey: ["getTenantUsage", req],
    queryFn: () => apiClient.tenantService.GetUsage(req),
    staleTime: 0,
    retry: 0,
  });
}

// ==============================
// 清理租户数据
// ==============================
export function useCleanupTenantData(
  options?: UseMutationOptions<{}, Error, identityservicev1_CleanupTenantDataRequest>
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
  { value: "TRIAL", label: t("enum.tenant.type.TRIAL") },
  { value: "PAID", label: t("enum.tenant.type.PAID") },
  { value: "INTERNAL", label: t("enum.tenant.type.INTERNAL") },
  { value: "PARTNER", label: t("enum.tenant.type.PARTNER") },
  { value: "CUSTOM", label: t("enum.tenant.type.CUSTOM") },
]);

export function tenantTypeToName(tenantType: Tenant_Type) {
  const values = tenantTypeList.value;
  const matchedItem = values.find((item) => item.value === tenantType);
  return matchedItem ? matchedItem.label : "";
}

// 租户类型 → tag 语义 type（与 react 端 TENANT_TYPE_COLORS 同构）
export function tenantTypeToType(tenantType: Tenant_Type): TagType {
  switch (tenantType) {
    case "INTERNAL":
      return "primary";
    case "PAID":
      return "success";
    case "PARTNER":
      return "warning";
    case "TRIAL":
    case "CUSTOM":
      return "info";
    default:
      return "info";
  }
}

export const tenantStatusList = computed(() => [
  { value: "ON", label: t("enum.tenant.status.ON") },
  { value: "OFF", label: t("enum.tenant.status.OFF") },
  { value: "EXPIRED", label: t("enum.tenant.status.EXPIRED") },
  { value: "FREEZE", label: t("enum.tenant.status.FREEZE") },
]);

export function tenantStatusToName(tenantStatus: Tenant_Status) {
  const values = tenantStatusList.value;
  const matchedItem = values.find((item) => item.value === tenantStatus);
  return matchedItem ? matchedItem.label : "";
}

// 启用状态 → tag 语义 type（与 react 端 TENANT_STATUS_COLORS 同构：ON 启用、OFF 停用、
// EXPIRED 到期告警、FREEZE 冻结中性）
export function tenantStatusToType(tenantStatus: Tenant_Status): TagType {
  switch (tenantStatus) {
    case "ON":
      return "success";
    case "OFF":
      return "danger";
    case "EXPIRED":
      return "warning";
    case "FREEZE":
      return "info";
    default:
      return "info";
  }
}

export const tenantAuditStatusList = computed(() => [
  { value: "PENDING", label: t("enum.tenant.auditStatus.PENDING") },
  { value: "APPROVED", label: t("enum.tenant.auditStatus.APPROVED") },
  { value: "REJECTED", label: t("enum.tenant.auditStatus.REJECTED") },
]);

export function tenantAuditStatusToName(tenantAuditStatus: Tenant_AuditStatus) {
  const values = tenantAuditStatusList.value;
  const matchedItem = values.find((item) => item.value === tenantAuditStatus);
  return matchedItem ? matchedItem.label : "";
}

// 审核状态 → tag 语义 type（与 react 端 AUDIT_STATUS_COLORS 同构）
export function tenantAuditStatusToType(tenantAuditStatus: Tenant_AuditStatus): TagType {
  switch (tenantAuditStatus) {
    case "APPROVED":
      return "success";
    case "PENDING":
      return "warning";
    case "REJECTED":
      return "danger";
    default:
      return "info";
  }
}
