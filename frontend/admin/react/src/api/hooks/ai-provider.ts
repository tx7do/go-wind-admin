import { useMutation, useQuery, type UseMutationOptions, type UseQueryOptions } from '@tanstack/react-query';
import { apiClient } from '@/api/client';
import { type PaginationQuery, queryClient } from '@/core';
import { makeUpdateMask } from '@/core/transport/rest/utils';
import type {
  aiservicev1_AiProvider,
  aiservicev1_CreateAiProviderRequest,
  aiservicev1_DeleteAiProviderRequest,
  aiservicev1_ListAiProviderResponse,
} from '@/api/generated/admin/service/v1';

// 1. useListAiProviders — 需要响应式数据的组件用
export function useListAiProviders(
  query: PaginationQuery,
  options?: Omit<UseQueryOptions<aiservicev1_ListAiProviderResponse, Error>, 'queryKey' | 'queryFn'>,
) {
  return useQuery({
    queryKey: ['listAiProviders', query],
    queryFn: () => apiClient.aiProviderService.List(query.toRawParams()),
    ...options,
  });
}

// 2. fetchListAiProviders — 非组件上下文（store / guard / effect）
export async function fetchListAiProviders(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listAiProviders', params],
    queryFn: () => apiClient.aiProviderService.List(params.toRawParams()),
    retry: 0,
  });
}

// 3. useGetAiProvider
export function useGetAiProvider(
  req: { id: string },
  options?: Omit<UseQueryOptions<aiservicev1_AiProvider, Error>, 'queryKey' | 'queryFn'>,
) {
  return useQuery({
    queryKey: ['getAiProvider', req],
    queryFn: () => apiClient.aiProviderService.Get({ id: req.id }),
    ...options,
  });
}

// 4. useCreateAiProvider
export function useCreateAiProvider(
  options?: UseMutationOptions<{}, Error, aiservicev1_CreateAiProviderRequest>,
) {
  return useMutation({ mutationFn: (data) => apiClient.aiProviderService.Create(data), ...options });
}

// 5. useUpdateAiProvider — 固定签名 { id, values }，内部自动构建 updateMask
export function useUpdateAiProvider(
  options?: UseMutationOptions<{}, Error, { id: string; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: Record<string, any> }) =>
      apiClient.aiProviderService.Update({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

// 6. useDeleteAiProvider
export function useDeleteAiProvider(
  options?: UseMutationOptions<{}, Error, aiservicev1_DeleteAiProviderRequest>,
) {
  return useMutation({ mutationFn: (req) => apiClient.aiProviderService.Delete(req), ...options });
}
