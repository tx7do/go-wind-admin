import {
  useMutation,
  type UseMutationOptions,
} from '@tanstack/vue-query';
import type {
  printservicev1_CreatePrintTemplateRequest,
  printservicev1_DeletePrintTemplateRequest,
} from '#/api/generated/admin/service/v1';

import { makeUpdateMask, type PaginationQuery } from '#/transport/rest';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';

// ==============================
// 打印模板（单据打印设计器 TemplateData 载体）
// ==============================

export async function fetchListPrintTemplates(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listPrintTemplates', params],
    queryFn: () => apiClient.printTemplateService.List(params.toRawParams()),
    retry: 0,
  });
}

export function useCreatePrintTemplate(
  options?: UseMutationOptions<{}, Error, printservicev1_CreatePrintTemplateRequest>
) {
  return useMutation({
    mutationFn: (data) => apiClient.printTemplateService.Create(data),
    ...options,
  });
}

export function useUpdatePrintTemplate(
  options?: UseMutationOptions<
    {},
    Error,
    { id: number; values: Record<string, any> }
  >
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.printTemplateService.Update({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeletePrintTemplate(
  options?: UseMutationOptions<{}, Error, printservicev1_DeletePrintTemplateRequest>
) {
  return useMutation({
    mutationFn: (req) => apiClient.printTemplateService.Delete(req),
    ...options,
  });
}
