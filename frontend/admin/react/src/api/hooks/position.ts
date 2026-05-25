import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/react-query';
import {
  type identityservicev1_CreatePositionRequest,
  type identityservicev1_DeletePositionRequest,
  type identityservicev1_GetPositionRequest,
  type identityservicev1_ListPositionResponse,
  type identityservicev1_Position,
} from '@/api/generated/admin/service/v1';
import { makeUpdateMask, type PaginationQuery, queryClient } from '@/core';
import {
  listPositions,
  getPosition,
  createPosition,
  updatePosition,
  deletePosition,
} from '@/api/service/position';

// ==============================
// 职位管理
// ==============================

export function useListPositions(
  query: PaginationQuery,
  options?: UseQueryOptions<identityservicev1_ListPositionResponse, Error>,
) {
  return useQuery({
    queryKey: ['listPositions', query],
    queryFn: () => listPositions(query),
    ...options,
  });
}

export async function fetchListPositions(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listPositions', params],
    queryFn: () => listPositions(params),
    retry: 0,
  });
}

export function useGetPosition(
  req: identityservicev1_GetPositionRequest,
  options?: UseQueryOptions<identityservicev1_Position, Error>,
) {
  return useQuery({
    queryKey: ['getPosition', req],
    queryFn: () => getPosition(req),
    ...options,
  });
}

export function useCreatePosition(
  options?: UseMutationOptions<{}, Error, identityservicev1_CreatePositionRequest>,
) {
  return useMutation({
    mutationFn: (data) => createPosition(data),
    ...options,
  });
}

export function useUpdatePosition(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>,
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
  options?: UseMutationOptions<{}, Error, identityservicev1_DeletePositionRequest>,
) {
  return useMutation({
    mutationFn: (req) => deletePosition(req),
    ...options,
  });
}
