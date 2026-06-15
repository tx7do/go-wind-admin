import type {
  dictservicev1_BatchCreateLanguagesRequest,
  dictservicev1_DeleteLanguageRequest,
  dictservicev1_GetLanguageRequest,
  dictservicev1_Language,
  dictservicev1_ListLanguageResponse,
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
// 语言管理
// ==============================

export function useListLanguages(
  query: PaginationQuery,
  options?: UseQueryOptions<dictservicev1_ListLanguageResponse, Error>,
) {
  return useQuery({
    queryKey: ['listLanguages', query],
    queryFn: () => apiClient.languageService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListLanguages(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listLanguages', params],
    queryFn: () => apiClient.languageService.List(params.toRawParams()),
    retry: 0,
  });
}

export function useGetLanguage(
  req: dictservicev1_GetLanguageRequest,
  options?: UseQueryOptions<dictservicev1_Language, Error>,
) {
  return useQuery({
    queryKey: ['getLanguage', req],
    queryFn: () => apiClient.languageService.Get(req),
    ...options,
  });
}

export function useCreateLanguage(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.languageService.Create({ data: { ...values } as dictservicev1_Language }),
    ...options,
  });
}

export function useUpdateLanguage(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.languageService.Update({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteLanguage(
  options?: UseMutationOptions<
    object,
    Error,
    dictservicev1_DeleteLanguageRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => apiClient.languageService.Delete(data),
    ...options,
  });
}

export function useBatchCreateLanguages(
  options?: UseMutationOptions<
    object,
    Error,
    dictservicev1_BatchCreateLanguagesRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => apiClient.languageService.BatchCreate(data),
    ...options,
  });
}
