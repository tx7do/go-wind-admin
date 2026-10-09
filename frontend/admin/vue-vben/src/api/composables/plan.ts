import type {
  identityservicev1_DeletePlanQuotaRequest,
  identityservicev1_DeletePlanRequest,
  identityservicev1_DeletePlanModuleRequest,
  identityservicev1_GetPlanRequest,
  identityservicev1_ListPlanQuotaResponse,
  identityservicev1_ListPlanResponse,
  identityservicev1_ListPlanModuleResponse,
  identityservicev1_Plan,
  identityservicev1_PlanModule,
  identityservicev1_PlanQuota,
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
// 套餐管理
// ==============================

export function useListPlans(
  query: PaginationQuery,
  options?: UseQueryOptions<identityservicev1_ListPlanResponse, Error>,
) {
  return useQuery({
    queryKey: ['listPlans', query],
    queryFn: () => apiClient.planService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListPlans(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listPlans', params],
    queryFn: () => apiClient.planService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetPlan(
  req: identityservicev1_GetPlanRequest,
  options?: UseQueryOptions<identityservicev1_Plan, Error>,
) {
  return useQuery({
    queryKey: ['getPlan', req],
    queryFn: () => apiClient.planService.Get(req),
    ...options,
  });
}

export function useCreatePlan(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.planService.Create({ data: { ...values } as identityservicev1_Plan }),
    ...options,
  });
}

export function useUpdatePlan(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
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
  options?: UseMutationOptions<
    object,
    Error,
    identityservicev1_DeletePlanRequest
  >,
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
  options?: UseQueryOptions<identityservicev1_ListPlanQuotaResponse, Error>,
) {
  return useQuery({
    queryKey: ['listPlanQuotas', query],
    queryFn: () => apiClient.planQuotaService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListPlanQuotas(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listPlanQuotas', params],
    queryFn: () => apiClient.planQuotaService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useCreatePlanQuota(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.planQuotaService.Create({ data: { ...values } as identityservicev1_PlanQuota }),
    ...options,
  });
}

export function useUpdatePlanQuota(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
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
  options?: UseMutationOptions<
    object,
    Error,
    identityservicev1_DeletePlanQuotaRequest
  >,
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
  options?: UseQueryOptions<identityservicev1_ListPlanModuleResponse, Error>,
) {
  return useQuery({
    queryKey: ['listPlanModules', query],
    queryFn: () => apiClient.planModuleService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListPlanModules(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listPlanModules', params],
    queryFn: () => apiClient.planModuleService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useCreatePlanModule(
  options?: UseMutationOptions<object, Error, identityservicev1_PlanModule>,
) {
  return useMutation({
    mutationFn: (data) =>
      apiClient.planModuleService.Create({ data } as any),
    ...options,
  });
}

export function useDeletePlanModule(
  options?: UseMutationOptions<
    object,
    Error,
    identityservicev1_DeletePlanModuleRequest
  >,
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
  { value: 'DASHBOARD', label: t('enum.module.DASHBOARD') },
  { value: 'OPM', label: t('enum.module.OPM') },
  { value: 'SYSTEM', label: t('enum.module.SYSTEM') },
  { value: 'DICT', label: t('enum.module.DICT') },
  { value: 'TENANT', label: t('enum.module.TENANT') },
  { value: 'PERMISSION', label: t('enum.module.PERMISSION') },
  { value: 'LOG', label: t('enum.module.LOG') },
  { value: 'INTERNAL_MESSAGE', label: t('enum.module.INTERNAL_MESSAGE') },
  { value: 'FILE', label: t('enum.module.FILE') },
  { value: 'TASK', label: t('enum.module.TASK') },
]);

export const planVersionList = computed(() => [
  { value: 'FREE', label: t('enum.plan.version.FREE') },
  { value: 'STANDARD', label: t('enum.plan.version.STANDARD') },
  { value: 'ENTERPRISE', label: t('enum.plan.version.ENTERPRISE') },
]);

export function planVersionToName(version: string) {
  const values = planVersionList.value;
  const matchedItem = values.find((item) => item.value === version);
  return matchedItem ? matchedItem.label : '';
}

// 版本分类色使用 antd 预设调色板名（由主题 token 驱动，亮/暗自动切换）。
const PLAN_VERSION_COLOR_MAP: Record<
  string,
  'blue' | 'purple' | 'orange' | 'default'
> = {
  FREE: 'blue',
  STANDARD: 'purple',
  ENTERPRISE: 'orange',
  DEFAULT: 'default',
};

export function planVersionToColor(
  version: string,
): 'blue' | 'orange' | 'purple' | 'default' {
  return PLAN_VERSION_COLOR_MAP[version as string] ?? 'default';
}

export const planExpiryPolicyList = computed(() => [
  { value: 'BLOCK_LOGIN', label: t('enum.plan.expiryPolicy.BLOCK_LOGIN') },
  { value: 'FREEZE', label: t('enum.plan.expiryPolicy.FREEZE') },
  { value: 'READONLY', label: t('enum.plan.expiryPolicy.READONLY') },
]);

export function planExpiryPolicyToName(expiryPolicy: string) {
  const values = planExpiryPolicyList.value;
  const matchedItem = values.find((item) => item.value === expiryPolicy);
  return matchedItem ? matchedItem.label : '';
}

// 到期处置策略色使用 antd 预设状态名（由主题 token 驱动）。
// 语义：FREEZE=warning，BLOCK_LOGIN/READONLY/default=default。
const PLAN_EXPIRY_POLICY_COLOR_MAP: Record<
  string,
  'warning' | 'default'
> = {
  FREEZE: 'warning',
  DEFAULT: 'default',
};

export function planExpiryPolicyToColor(
  expiryPolicy: string,
): 'default' | 'warning' {
  return (
    PLAN_EXPIRY_POLICY_COLOR_MAP[expiryPolicy as string] ?? 'default'
  );
}

export const planQuotaTypeList = computed(() => [
  {
    value: 'STORAGE',
    label: t('enum.plan.quotaType.STORAGE'),
  },
  {
    value: 'USER_LIMIT',
    label: t('enum.plan.quotaType.USER_LIMIT'),
  },
  {
    value: 'API_CALL',
    label: t('enum.plan.quotaType.API_CALL'),
  },
]);

export function planQuotaTypeToName(quotaType: string) {
  const values = planQuotaTypeList.value;
  const matchedItem = values.find((item) => item.value === quotaType);
  return matchedItem ? matchedItem.label : '';
}
