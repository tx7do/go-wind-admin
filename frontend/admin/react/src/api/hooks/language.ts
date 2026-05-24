import { useMutation, type UseMutationOptions } from '@tanstack/react-query';
import {
  type dictservicev1_CreateLanguageRequest,
  type dictservicev1_DeleteLanguageRequest,
  type dictservicev1_GetLanguageRequest,
  type dictservicev1_Language,
  type dictservicev1_ListLanguageResponse,
  type dictservicev1_UpdateLanguageRequest,
  type dictservicev1_BatchCreateLanguagesRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery } from '@/core';
import { listLanguages, getLanguage, createLanguage, updateLanguage, deleteLanguage, batchCreateLanguages } from '@/api/service/language';

// ==============================
// 语言管理
// ==============================

export function useListLanguages(
  options?: UseMutationOptions<dictservicev1_ListLanguageResponse, Error, PaginationQuery>,
) {
  return useMutation({
    mutationFn: (query) => listLanguages(query),
    ...options,
  });
}

export function useGetLanguage(
  options?: UseMutationOptions<dictservicev1_Language, Error, dictservicev1_GetLanguageRequest>,
) {
  return useMutation({
    mutationFn: (data) => getLanguage(data),
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
  options?: UseMutationOptions<{}, Error, dictservicev1_UpdateLanguageRequest>,
) {
  return useMutation({
    mutationFn: (data) => updateLanguage(data),
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
