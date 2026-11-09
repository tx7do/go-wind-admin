import type {
  permissionservicev1_Api,
  permissionservicev1_DeleteApiRequest,
  permissionservicev1_GetApiRequest,
  permissionservicev1_ListApiResponse,
} from '#/api/generated/admin/service/v1';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';
import { makeUpdateMask, type PaginationQuery } from '#/transport/rest';

// ==============================
// API 管理
// ==============================

export function useListApis(
  query: PaginationQuery,
  options?: UseQueryOptions<permissionservicev1_ListApiResponse, Error>,
) {
  return useQuery({
    queryKey: ['listApis', query],
    queryFn: () => apiClient.apiService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListApis(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listApis', params],
    queryFn: () => apiClient.apiService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetApi(
  req: permissionservicev1_GetApiRequest,
  options?: UseQueryOptions<permissionservicev1_Api, Error>,
) {
  return useQuery({
    queryKey: ['getApi', req],
    queryFn: () => apiClient.apiService.Get(req),
    ...options,
  });
}

export function useCreateApi(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.apiService.Create({ data: { ...values } as permissionservicev1_Api }),
    ...options,
  });
}

export function useUpdateApi(
  options?: UseMutationOptions<
    object,
    Error,
    { id: string; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: Record<string, any> }) =>
      apiClient.apiService.Update({
        id,
        data: {
          ...values,
        },
        updateMask: makeUpdateMask(Object.keys(values ?? [])),
      }),
    ...options,
  });
}

export function useDeleteApi(
  options?: UseMutationOptions<
    object,
    Error,
    permissionservicev1_DeleteApiRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => apiClient.apiService.Delete(data),
    ...options,
  });
}

export function useSyncApisApi(options?: UseMutationOptions<object, Error>) {
  return useMutation({
    mutationFn: () => apiClient.apiService.SyncApis({}),
    ...options,
  });
}

// ==============================
// API 枚举与工具函数
// ==============================
