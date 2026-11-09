import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/react-query';
import {
  type permissionservicev1_Api,
  type permissionservicev1_CreateApiRequest,
  type permissionservicev1_DeleteApiRequest,
  type permissionservicev1_GetApiRequest,
  type permissionservicev1_ListApiResponse,
} from '@/api/generated/admin/service/v1';
import { makeUpdateMask, type PaginationQuery, queryClient } from '@/core';
import { apiClient } from '@/api/client';

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
  options?: UseMutationOptions<{}, Error, permissionservicev1_CreateApiRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.apiService.Create(data),
    ...options,
  });
}

export function useUpdateApi(
  options?: UseMutationOptions<{}, Error, { id: string; values: Record<string, any> }>,
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
  options?: UseMutationOptions<{}, Error, permissionservicev1_DeleteApiRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.apiService.Delete(data),
    ...options,
  });
}

export function useSyncApisApi(options?: UseMutationOptions<{}, Error>) {
  return useMutation({
    mutationFn: () => apiClient.apiService.SyncApis({}),
    ...options,
  });
}
