import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/react-query';
import {
  type dictservicev1_CreateLanguageRequest,
  type dictservicev1_DeleteLanguageRequest,
  type dictservicev1_GetLanguageRequest,
  type dictservicev1_Language,
  type dictservicev1_ListLanguageResponse,
  type dictservicev1_BatchCreateLanguagesRequest,
} from '@/api/generated/admin/service/v1';
import { makeUpdateMask, type PaginationQuery, queryClient } from '@/core';
import {
  listLanguages,
  getLanguage,
  createLanguage,
  updateLanguage,
  deleteLanguage,
  batchCreateLanguages,
} from '@/api/service/language';

// ==============================
// 语言管理
// ==============================

export function useListLanguages(
  query: PaginationQuery,
  options?: UseQueryOptions<dictservicev1_ListLanguageResponse, Error>,
) {
  return useQuery({
    queryKey: ['listLanguages', query],
    queryFn: () => listLanguages(query),
    ...options,
  });
}

export async function fetchListLanguages(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listLanguages', params],
    queryFn: () => listLanguages(params),
    retry: 0,
  });
}

export function useGetLanguage(
  req: dictservicev1_GetLanguageRequest,
  options?: UseQueryOptions<dictservicev1_Language, Error>,
) {
  return useQuery({
    queryKey: ['getLanguage', req],
    queryFn: () => getLanguage(req),
    ...options,
  });
}

export function useCreateLanguage(
  options?: UseMutationOptions<{}, Error, dictservicev1_CreateLanguageRequest>,
) {
  return useMutation({
    mutationFn: (data) => createLanguage(data),
    ...options,
  });
}

export function useUpdateLanguage(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      updateLanguage({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteLanguage(
  options?: UseMutationOptions<{}, Error, dictservicev1_DeleteLanguageRequest>,
) {
  return useMutation({
    mutationFn: (data) => deleteLanguage(data),
    ...options,
  });
}

export function useBatchCreateLanguages(
  options?: UseMutationOptions<{}, Error, dictservicev1_BatchCreateLanguagesRequest>,
) {
  return useMutation({
    mutationFn: (data) => batchCreateLanguages(data),
    ...options,
  });
}
