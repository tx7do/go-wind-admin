import { useMutation, type UseMutationOptions } from '@tanstack/react-query';
import {
  type dictservicev1_DictType,
  type dictservicev1_ListDictTypeResponse,
  type dictservicev1_ListDictEntryResponse,
  type dictservicev1_UpdateDictTypeRequest,
  type dictservicev1_DeleteDictTypeRequest,
  type dictservicev1_CreateDictEntryRequest,
  type dictservicev1_UpdateDictEntryRequest,
  type dictservicev1_DeleteDictEntryRequest,
  type dictservicev1_CreateDictTypeRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery } from '@/core/transport/rest';
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
  options?: UseMutationOptions<dictservicev1_ListDictTypeResponse, Error, PaginationQuery>,
) {
  return useMutation({
    mutationFn: (query) => listDictTypes(query),
    ...options,
  });
}

export function useGetDictType(
  options?: UseMutationOptions<dictservicev1_DictType, Error, number>,
) {
  return useMutation({
    mutationFn: (id) => getDictType(id),
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
  options?: UseMutationOptions<dictservicev1_DictType, Error, dictservicev1_UpdateDictTypeRequest>,
) {
  return useMutation({
    mutationFn: (data) => updateDictType(data),
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
  options?: UseMutationOptions<dictservicev1_ListDictEntryResponse, Error, PaginationQuery>,
) {
  return useMutation({
    mutationFn: (query) => listDictEntries(query),
    ...options,
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
  options?: UseMutationOptions<{}, Error, dictservicev1_UpdateDictEntryRequest>,
) {
  return useMutation({
    mutationFn: (data) => updateDictEntry(data),
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
