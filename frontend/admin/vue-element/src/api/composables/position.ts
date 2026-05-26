import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from "@tanstack/vue-query";
import type {
  identityservicev1_CreatePositionRequest,
  identityservicev1_DeletePositionRequest,
  identityservicev1_GetPositionRequest,
  identityservicev1_ListPositionResponse,
  identityservicev1_Position,
} from "@/api/generated/admin/service/v1";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import {
  listPositions,
  getPosition,
  createPosition,
  updatePosition,
  deletePosition,
} from "@/api/service/position";
import { queryClient } from "@/plugins/vue-query";

// ==============================
// 职位管理
// ==============================

export function useListPositions(
  query: PaginationQuery,
  options?: UseQueryOptions<identityservicev1_ListPositionResponse, Error>
) {
  return useQuery({
    queryKey: ["listPositions", query],
    queryFn: () => listPositions(query),
    ...options,
  });
}

export async function fetchListPositions(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listPositions", params],
    queryFn: () => listPositions(params),
    retry: 0,
  });
}

export function useGetPosition(
  req: identityservicev1_GetPositionRequest,
  options?: UseQueryOptions<identityservicev1_Position, Error>
) {
  return useQuery({
    queryKey: ["getPosition", req],
    queryFn: () => getPosition(req),
    ...options,
  });
}

export function useCreatePosition(options?: UseMutationOptions<{}, Error, Record<string, any>>) {
  return useMutation({
    mutationFn: (values) => createPosition({ data: { ...values } as identityservicev1_Position }),
    ...options,
  });
}

export function useUpdatePosition(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      updatePosition({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeletePosition(
  options?: UseMutationOptions<{}, Error, identityservicev1_DeletePositionRequest>
) {
  return useMutation({
    mutationFn: (req) => deletePosition(req),
    ...options,
  });
}
