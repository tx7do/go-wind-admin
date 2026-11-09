import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/react-query';
import {
  type identityservicev1_Plan,
  type identityservicev1_ListPlanResponse,
  type identityservicev1_ListPlanQuotaResponse,
  type identityservicev1_DeletePlanRequest,
  type identityservicev1_CreatePlanRequest,
  type identityservicev1_GetPlanRequest,
  type identityservicev1_CreatePlanQuotaRequest,
  type identityservicev1_DeletePlanQuotaRequest,
  type identityservicev1_ListPlanModuleResponse,
  type identityservicev1_CreatePlanModuleRequest,
  type identityservicev1_DeletePlanModuleRequest,
} from '@/api/generated/admin/service/v1';
import { makeUpdateMask, type PaginationQuery } from '@/core/transport/rest';
import { queryClient } from '@/core';
import { apiClient } from '@/api/client';

// ==============================
// 套餐目录管理
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
  options?: UseMutationOptions<identityservicev1_Plan, Error, identityservicev1_CreatePlanRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.planService.Create(data),
    ...options,
  });
}

export function useUpdatePlan(
  options?: UseMutationOptions<
    identityservicev1_Plan,
    Error,
    { id: string; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: Record<string, any> }) =>
      apiClient.planService.Update({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeletePlan(
  options?: UseMutationOptions<{}, Error, identityservicev1_DeletePlanRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.planService.Delete(data),
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
    retry: 0,
  });
}

export function useCreatePlanQuota(
  options?: UseMutationOptions<{}, Error, identityservicev1_CreatePlanQuotaRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.planQuotaService.Create(data),
    ...options,
  });
}

export function useUpdatePlanQuota(
  options?: UseMutationOptions<{}, Error, { id: string; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: Record<string, any> }) =>
      apiClient.planQuotaService.Update({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeletePlanQuota(
  options?: UseMutationOptions<{}, Error, identityservicev1_DeletePlanQuotaRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.planQuotaService.Delete(data),
    ...options,
  });
}

// ==============================
// 套餐模块白名单管理
// ==============================

export function useListPlanModules(
  query: PaginationQuery,
  options?: Omit<UseQueryOptions<identityservicev1_ListPlanModuleResponse, Error, identityservicev1_ListPlanModuleResponse, readonly unknown[]>, 'queryKey' | 'queryFn'>,
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
    retry: 0,
  });
}

export function useCreatePlanModule(
  options?: UseMutationOptions<{}, Error, identityservicev1_CreatePlanModuleRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.planModuleService.Create(data),
    ...options,
  });
}

export function useDeletePlanModule(
  options?: UseMutationOptions<{}, Error, identityservicev1_DeletePlanModuleRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.planModuleService.Delete(data),
    ...options,
  });
}
