import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/react-query';
import {
  type dictservicev1_DictType,
  type dictservicev1_ListDictTypeResponse,
  type dictservicev1_ListDictEntryResponse,
  type dictservicev1_DeleteDictTypeRequest,
  type dictservicev1_CreateDictEntryRequest,
  type dictservicev1_DeleteDictEntryRequest,
  type dictservicev1_CreateDictTypeRequest,
  type dictservicev1_GetDictTypeRequest,
} from '@/api/generated/admin/service/v1';
import { makeUpdateMask, type PaginationQuery } from '@/core/transport/rest';
import { queryClient } from '@/core';
import {
  listDictTypes,
  getDictType,
  createDictType,
  updateDictType,
  deleteDictType,
  listDictEntries,
  createDictEntry,
  updateDictEntry,
  deleteDictEntry,
} from '@/api/service/dict';

// ==============================
// 字典类型管理
// ==============================

export function useListDictTypes(
  query: PaginationQuery,
  options?: UseQueryOptions<dictservicev1_ListDictTypeResponse, Error>,
) {
  return useQuery({
    queryKey: ['listDictTypes', query],
    queryFn: () => listDictTypes(query),
    ...options,
  });
}

export async function fetchListDictTypes(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listDictTypes', params],
    queryFn: () => listDictTypes(params),
    retry: 0,
  });
}

export function useGetDictType(
  req: dictservicev1_GetDictTypeRequest,
  options?: UseQueryOptions<dictservicev1_DictType, Error>,
) {
  return useQuery({
    queryKey: ['getDictType', req],
    queryFn: () => getDictType(req),
    ...options,
  });
}

export function useCreateDictType(
  options?: UseMutationOptions<dictservicev1_DictType, Error, dictservicev1_CreateDictTypeRequest>,
) {
  return useMutation({
    mutationFn: (data) => createDictType(data),
    ...options,
  });
}

export function useUpdateDictType(
  options?: UseMutationOptions<
    dictservicev1_DictType,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      updateDictType({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteDictType(
  options?: UseMutationOptions<{}, Error, dictservicev1_DeleteDictTypeRequest>,
) {
  return useMutation({
    mutationFn: (data) => deleteDictType(data),
    ...options,
  });
}

// ==============================
// 字典条目管理
// ==============================

export function useListDictEntries(
  query: PaginationQuery,
  options?: UseQueryOptions<dictservicev1_ListDictEntryResponse, Error>,
) {
  return useQuery({
    queryKey: ['listDictEntries', query],
    queryFn: () => listDictEntries(query),
    ...options,
  });
}

export async function fetchListDictEntries(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listDictEntries', params],
    queryFn: () => listDictEntries(params),
    retry: 0,
  });
}

export function useCreateDictEntry(
  options?: UseMutationOptions<{}, Error, dictservicev1_CreateDictEntryRequest>,
) {
  return useMutation({
    mutationFn: (data) => createDictEntry(data),
    ...options,
  });
}

export function useUpdateDictEntry(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      updateDictEntry({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteDictEntry(
  options?: UseMutationOptions<{}, Error, dictservicev1_DeleteDictEntryRequest>,
) {
  return useMutation({
    mutationFn: (data) => deleteDictEntry(data),
    ...options,
  });
}
