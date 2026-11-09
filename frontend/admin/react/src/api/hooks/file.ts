import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/react-query';
import {
  type storageservicev1_CreateFileRequest,
  type storageservicev1_DeleteFileRequest,
  type storageservicev1_File,
  type storageservicev1_GetFileRequest,
  type storageservicev1_ListFileResponse,
} from '@/api/generated/admin/service/v1';
import { makeUpdateMask, type PaginationQuery } from '@/core/transport/rest';
import { queryClient } from '@/core';
import { apiClient } from '@/api/client';

// ==============================
// 文件管理
// ==============================

export function useListFiles(
  query: PaginationQuery,
  options?: UseQueryOptions<storageservicev1_ListFileResponse, Error>,
) {
  return useQuery({
    queryKey: ['listFiles', query],
    queryFn: () => apiClient.fileService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListFiles(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listFiles', params],
    queryFn: () => apiClient.fileService.List(params.toRawParams()),
    retry: 0,
  });
}

export function useGetFile(
  req: storageservicev1_GetFileRequest,
  options?: UseQueryOptions<storageservicev1_File, Error>,
) {
  return useQuery({
    queryKey: ['getFile', req],
    queryFn: () => apiClient.fileService.Get(req),
    ...options,
  });
}

export function useCreateFile(
  options?: UseMutationOptions<{}, Error, storageservicev1_CreateFileRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.fileService.Create(data),
    ...options,
  });
}

export function useUpdateFile(
  options?: UseMutationOptions<{}, Error, { id: string; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: Record<string, any> }) =>
      apiClient.fileService.Update({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteFile(
  options?: UseMutationOptions<{}, Error, storageservicev1_DeleteFileRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.fileService.Delete(data),
    ...options,
  });
}
