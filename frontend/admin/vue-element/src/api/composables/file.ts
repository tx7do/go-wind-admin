import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from "@tanstack/vue-query";
import type {
  storageservicev1_CreateFileRequest,
  storageservicev1_DeleteFileRequest,
  storageservicev1_File,
  storageservicev1_GetFileRequest,
  storageservicev1_ListFileResponse,
} from "@/api/generated/admin/service/v1";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import { queryClient } from "@/plugins/vue-query";
import { listFiles, getFile, createFile, updateFile, deleteFile } from "@/api/service/file";

// ==============================
// 文件管理
// ==============================

export function useListFiles(
  query: PaginationQuery,
  options?: UseQueryOptions<storageservicev1_ListFileResponse, Error>
) {
  return useQuery({
    queryKey: ["listFiles", query],
    queryFn: () => listFiles(query),
    ...options,
  });
}

export async function fetchListFiles(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listFiles", params],
    queryFn: () => listFiles(params),
    retry: 0,
  });
}

export function useGetFile(
  req: storageservicev1_GetFileRequest,
  options?: UseQueryOptions<storageservicev1_File, Error>
) {
  return useQuery({
    queryKey: ["getFile", req],
    queryFn: () => getFile(req),
    ...options,
  });
}

export function useCreateFile(options?: UseMutationOptions<{}, Error, Record<string, any>>) {
  return useMutation({
    mutationFn: (values) => createFile({ data: { ...values } as storageservicev1_File }),
    ...options,
  });
}

export function useUpdateFile(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      updateFile({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteFile(
  options?: UseMutationOptions<{}, Error, storageservicev1_DeleteFileRequest>
) {
  return useMutation({
    mutationFn: (data) => deleteFile(data),
    ...options,
  });
}
