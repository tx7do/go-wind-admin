import { useMutation, type UseMutationOptions } from '@tanstack/react-query';
import {
  type resourceservicev1_Api,
  type resourceservicev1_CreateApiRequest,
  type resourceservicev1_DeleteApiRequest,
  type resourceservicev1_GetApiRequest,
  type resourceservicev1_ListApiResponse,
  type resourceservicev1_UpdateApiRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery } from '@/core';
import { listApis, getApi, createApi, updateApi, deleteApi } from '@/api/service/api';

// ==============================
// API 管理
// ==============================

export function useListApis(
  options?: UseMutationOptions<resourceservicev1_ListApiResponse, Error, PaginationQuery>,
) {
  return useMutation({
    mutationFn: (query) => listApis(query),
    ...options,
  });
}

export function useGetApi(
  options?: UseMutationOptions<resourceservicev1_Api, Error, resourceservicev1_GetApiRequest>,
) {
  return useMutation({
    mutationFn: (data) => getApi(data),
    ...options,
  });
}

export function useCreateApi(
  options?: UseMutationOptions<{}, Error, resourceservicev1_CreateApiRequest>,
) {
  return useMutation({
    mutationFn: (data) => createApi(data),
    ...options,
  });
}

export function useUpdateApi(
  options?: UseMutationOptions<{}, Error, resourceservicev1_UpdateApiRequest>,
) {
  return useMutation({
    mutationFn: (data) => updateApi(data),
    ...options,
  });
}

export function useDeleteApi(
  options?: UseMutationOptions<{}, Error, resourceservicev1_DeleteApiRequest>,
) {
  return useMutation({
    mutationFn: (data) => deleteApi(data),
    ...options,
  });
}
