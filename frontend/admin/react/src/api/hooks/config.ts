import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/react-query';
import {
  type configservicev1_CreateConfigRequest,
  type configservicev1_DeleteConfigRequest,
  type configservicev1_GetConfigRequest,
  type configservicev1_ListConfigResponse,
  type configservicev1_Config,
} from '@/api/generated/admin/service/v1';
import { makeUpdateMask, type PaginationQuery, queryClient } from '@/core';
import { apiClient } from '@/api/client';

// ==============================
// 参数管理（系统参数，动态 KV）
// ==============================

export function useListConfigs(
  query: PaginationQuery,
  options?: UseQueryOptions<configservicev1_ListConfigResponse, Error>,
) {
  return useQuery({
    queryKey: ['listConfigs', query],
    queryFn: () => apiClient.configService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListConfigs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listConfigs', params],
    queryFn: () => apiClient.configService.List(params.toRawParams()),
    retry: 0,
  });
}

export function useGetConfig(
  req: configservicev1_GetConfigRequest,
  options?: UseQueryOptions<configservicev1_Config, Error>,
) {
  return useQuery({
    queryKey: ['getConfig', req],
    queryFn: () => apiClient.configService.Get(req),
    ...options,
  });
}

export function useCreateConfig(
  options?: UseMutationOptions<{}, Error, configservicev1_CreateConfigRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.configService.Create(data),
    ...options,
  });
}

export function useUpdateConfig(
  options?: UseMutationOptions<{}, Error, { id: string; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: Record<string, any> }) =>
      apiClient.configService.Update({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteConfig(
  options?: UseMutationOptions<{}, Error, configservicev1_DeleteConfigRequest>,
) {
  return useMutation({
    mutationFn: (req) => apiClient.configService.Delete(req),
    ...options,
  });
}
