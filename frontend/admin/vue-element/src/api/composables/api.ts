import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from "@tanstack/vue-query";
import type {
  resourceservicev1_Api,
  resourceservicev1_CreateApiRequest,
  resourceservicev1_DeleteApiRequest,
  resourceservicev1_GetApiRequest,
  resourceservicev1_ListApiResponse,
} from "@/api/generated/admin/service/v1";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import { listApis, getApi, createApi, updateApi, deleteApi, syncApis } from "@/api/service/api";
import { queryClient } from "@/plugins/vue-query";

// ==============================
// API 管理
// ==============================

export function useListApis(
  query: PaginationQuery,
  options?: UseQueryOptions<resourceservicev1_ListApiResponse, Error>
) {
  return useQuery({
    queryKey: ["listApis", query],
    queryFn: () => listApis(query),
    ...options,
  });
}

export async function fetchListApis(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listApis", params],
    queryFn: () => listApis(params),
    retry: 0,
  });
}

export function useGetApi(
  req: resourceservicev1_GetApiRequest,
  options?: UseQueryOptions<resourceservicev1_Api, Error>
) {
  return useQuery({
    queryKey: ["getApi", req],
    queryFn: () => getApi(req),
    ...options,
  });
}

export function useCreateApi(
  options?: UseMutationOptions<{}, Error, resourceservicev1_CreateApiRequest>
) {
  return useMutation({
    mutationFn: (data) => createApi(data),
    ...options,
  });
}

export function useUpdateApi(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      updateApi({
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
  options?: UseMutationOptions<{}, Error, resourceservicev1_DeleteApiRequest>
) {
  return useMutation({
    mutationFn: (data) => deleteApi(data),
    ...options,
  });
}

export function useSyncApisApi(options?: UseMutationOptions<{}, Error>) {
  return useMutation({
    mutationFn: () => syncApis(),
    ...options,
  });
}
