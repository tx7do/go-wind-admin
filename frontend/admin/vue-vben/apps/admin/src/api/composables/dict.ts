import type {
  dictservicev1_DeleteDictEntryRequest,
  dictservicev1_DeleteDictTypeRequest,
  dictservicev1_DictType,
  dictservicev1_GetDictTypeRequest,
  dictservicev1_ListDictEntryResponse,
  dictservicev1_ListDictTypeResponse,
} from '#/api/generated/admin/service/v1';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import {
  createDictEntry,
  createDictType,
  deleteDictEntry,
  deleteDictType,
  getDictType,
  listDictEntries,
  listDictTypes,
  updateDictEntry,
  updateDictType,
} from '#/api/service/dict';
import { queryClient } from '#/plugins/vue-query';
import { makeUpdateMask, type PaginationQuery } from '#/transport/rest';

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
  options?: UseMutationOptions<
    dictservicev1_DictType,
    Error,
    Record<string, any>
  >,
) {
  return useMutation({
    mutationFn: (values) => createDictType({ data: { ...values } as any }),
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
  options?: UseMutationOptions<
    object,
    Error,
    dictservicev1_DeleteDictTypeRequest
  >,
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
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) => createDictEntry({ data: { ...values } as any }),
    ...options,
  });
}

export function useUpdateDictEntry(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
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
  options?: UseMutationOptions<
    object,
    Error,
    dictservicev1_DeleteDictEntryRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => deleteDictEntry(data),
    ...options,
  });
}
