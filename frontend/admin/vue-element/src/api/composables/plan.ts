import { computed } from "vue";
import {
  useMutation,
  useQuery,
  type UseMutationOptions,
  type UseQueryOptions,
} from "@tanstack/vue-query";
import type {
  identityservicev1_DeletePlanQuotaRequest,
  identityservicev1_DeletePlanRequest,
  identityservicev1_DeletePlanModuleRequest,
  identityservicev1_GetPlanRequest,
  identityservicev1_ListPlanQuotaResponse,
  identityservicev1_ListPlanResponse,
  identityservicev1_ListPlanModuleResponse,
  identityservicev1_Plan,
  identityservicev1_PlanQuota,
  identityservicev1_PlanModule,
  identityservicev1_Plan_ExpiryPolicy as Plan_ExpiryPolicy,
  identityservicev1_Plan_Version as Plan_Version,
  identityservicev1_PlanQuota_QuotaType as PlanQuota_QuotaType,
} from "@/api/generated/admin/service/v1";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import { apiClient } from "@/api/client";
import { queryClient } from "@/plugins/vue-query";
import { i18n } from "@/core/i18n";

const t = i18n.global.t;

// ==============================
// 套餐管理
// ==============================

export function useListPlans(
  query: PaginationQuery,
  options?: UseQueryOptions<identityservicev1_ListPlanResponse, Error>
) {
  return useQuery({
    queryKey: ["listPlans", query],
    queryFn: () => apiClient.planService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListPlans(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listPlans", params],
    queryFn: () => apiClient.planService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetPlan(
  req: identityservicev1_GetPlanRequest,
  options?: UseQueryOptions<identityservicev1_Plan, Error>
) {
  return useQuery({
    queryKey: ["getPlan", req],
    queryFn: () => apiClient.planService.Get(req),
    ...options,
  });
}

export function useCreatePlan(options?: UseMutationOptions<{}, Error, Record<string, any>>) {
  return useMutation({
    mutationFn: (values) => apiClient.planService.Create({ data: { ...values } as any }),
    ...options,
  });
}

export function useUpdatePlan(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.planService.Update({
        id,
        data: {
          ...values,
        },
        updateMask: makeUpdateMask(Object.keys(values ?? [])),
      }),
    ...options,
  });
}

export function useDeletePlan(
  options?: UseMutationOptions<{}, Error, identityservicev1_DeletePlanRequest>
) {
  return useMutation({
    mutationFn: (req) => apiClient.planService.Delete(req),
    ...options,
  });
}

// ==============================
// 套餐配额管理
// ==============================

export function useListPlanQuotas(
  query: PaginationQuery,
  options?: UseQueryOptions<identityservicev1_ListPlanQuotaResponse, Error>
) {
  return useQuery({
    queryKey: ["listPlanQuotas", query],
    queryFn: () => apiClient.planQuotaService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListPlanQuotas(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listPlanQuotas", params],
    queryFn: () => apiClient.planQuotaService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useCreatePlanQuota(options?: UseMutationOptions<{}, Error, Record<string, any>>) {
  return useMutation({
    mutationFn: (values) => apiClient.planQuotaService.Create({ data: { ...values } as any }),
    ...options,
  });
}

export function useUpdatePlanQuota(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.planQuotaService.Update({
        id,
        data: {
          ...values,
        },
        updateMask: makeUpdateMask(Object.keys(values ?? [])),
      }),
    ...options,
  });
}

export function useDeletePlanQuota(
  options?: UseMutationOptions<{}, Error, identityservicev1_DeletePlanQuotaRequest>
) {
  return useMutation({
    mutationFn: (req) => apiClient.planQuotaService.Delete(req),
    ...options,
  });
}

// ==============================
// 套餐模块白名单管理
// ==============================

export function useListPlanModules(
  query: PaginationQuery,
  options?: UseQueryOptions<identityservicev1_ListPlanModuleResponse, Error>
) {
  return useQuery({
    queryKey: ["listPlanModules", query],
    queryFn: () => apiClient.planModuleService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListPlanModules(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listPlanModules", params],
    queryFn: () => apiClient.planModuleService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useCreatePlanModule(
  options?: UseMutationOptions<{}, Error, identityservicev1_PlanModule>
) {
  return useMutation({
    mutationFn: (data) => apiClient.planModuleService.Create({ data } as any),
    ...options,
  });
}

export function useDeletePlanModule(
  options?: UseMutationOptions<{}, Error, identityservicev1_DeletePlanModuleRequest>
) {
  return useMutation({
    mutationFn: (req) => apiClient.planModuleService.Delete(req),
    ...options,
  });
}

// ==============================
// 套餐枚举与工具函数
// ==============================

// 模块白名单多选选项列表（对齐 identityservicev1_Module 枚举字符串值）
export const planModuleList = computed(() => [
  { value: "DASHBOARD", label: t("enum.module.DASHBOARD") },
  { value: "OPM", label: t("enum.module.OPM") },
  { value: "SYSTEM", label: t("enum.module.SYSTEM") },
  { value: "DICT", label: t("enum.module.DICT") },
  { value: "TENANT", label: t("enum.module.TENANT") },
  { value: "PERMISSION", label: t("enum.module.PERMISSION") },
  { value: "LOG", label: t("enum.module.LOG") },
  { value: "INTERNAL_MESSAGE", label: t("enum.module.INTERNAL_MESSAGE") },
  { value: "FILE", label: t("enum.module.FILE") },
  { value: "TASK", label: t("enum.module.TASK") },
]);

export const planVersionList = computed(() => [
  { value: "FREE", label: t("enum.plan.version.FREE") },
  { value: "STANDARD", label: t("enum.plan.version.STANDARD") },
  { value: "ENTERPRISE", label: t("enum.plan.version.ENTERPRISE") },
  { value: "PLAN_VERSION_UNSPECIFIED", label: t("enum.plan.version.PLAN_VERSION_UNSPECIFIED") },
]);

export function planVersionToName(version: Plan_Version) {
  const values = planVersionList.value;
  const matchedItem = values.find((item) => item.value === version);
  return matchedItem ? matchedItem.label : "";
}

export function planVersionToColor(version: Plan_Version) {
  switch (version) {
    case "ENTERPRISE":
      return "#722ED1";
    case "STANDARD":
      return "#1890FF";
    case "FREE":
      return "#8C8C8C";
    default:
      return "#8C8C8C";
  }
}

export const planExpiryPolicyList = computed(() => [
  { value: "BLOCK_LOGIN", label: t("enum.plan.expiryPolicy.BLOCK_LOGIN") },
  { value: "FREEZE", label: t("enum.plan.expiryPolicy.FREEZE") },
  { value: "READONLY", label: t("enum.plan.expiryPolicy.READONLY") },
  {
    value: "PLAN_EXPIRY_POLICY_UNSPECIFIED",
    label: t("enum.plan.expiryPolicy.PLAN_EXPIRY_POLICY_UNSPECIFIED"),
  },
]);

export function planExpiryPolicyToName(expiryPolicy: Plan_ExpiryPolicy) {
  const values = planExpiryPolicyList.value;
  const matchedItem = values.find((item) => item.value === expiryPolicy);
  return matchedItem ? matchedItem.label : "";
}

export function planExpiryPolicyToColor(expiryPolicy: Plan_ExpiryPolicy) {
  switch (expiryPolicy) {
    case "BLOCK_LOGIN":
      return "#F5222D";
    case "FREEZE":
      return "#FAAD14";
    case "READONLY":
      return "#1890FF";
    default:
      return "#8C8C8C";
  }
}

export const planQuotaTypeList = computed(() => [
  { value: "API_CALL", label: t("enum.plan.quotaType.API_CALL") },
  { value: "STORAGE", label: t("enum.plan.quotaType.STORAGE") },
  { value: "USER_LIMIT", label: t("enum.plan.quotaType.USER_LIMIT") },
  {
    value: "PLAN_QUOTA_TYPE_UNSPECIFIED",
    label: t("enum.plan.quotaType.PLAN_QUOTA_TYPE_UNSPECIFIED"),
  },
]);

export function planQuotaTypeToName(quotaType: PlanQuota_QuotaType) {
  const values = planQuotaTypeList.value;
  const matchedItem = values.find((item) => item.value === quotaType);
  return matchedItem ? matchedItem.label : "";
}

export function planQuotaTypeToColor(quotaType: PlanQuota_QuotaType) {
  switch (quotaType) {
    case "API_CALL":
      return "#FAAD14";
    case "STORAGE":
      return "#1890FF";
    case "USER_LIMIT":
      return "#722ED1";
    default:
      return "#8C8C8C";
  }
}
