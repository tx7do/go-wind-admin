import type {
  storageservicev1_OSSProvider as OSSProvider,
  storageservicev1_DeleteFileRequest,
  storageservicev1_File,
  storageservicev1_GetFileRequest,
  storageservicev1_ListFileResponse,
} from '#/api/generated/admin/service/v1';

import { computed } from 'vue';

import { i18n } from '@vben/locales';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';
import { makeUpdateMask, type PaginationQuery } from '#/transport/rest';

const t = i18n.global.t;

// ==============================
// 文件管理
// ==============================

export function useListFiles(
  query: PaginationQuery,
  options?: UseQueryOptions<storageservicev1_ListFileResponse, Error>,
) {
  return useQuery({
    queryKey: ['listFiles', query],
    queryFn: () => apiClient.fileService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListFiles(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listFiles', params],
    queryFn: () => apiClient.fileService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetFile(
  req: storageservicev1_GetFileRequest,
  options?: UseQueryOptions<storageservicev1_File, Error>,
) {
  return useQuery({
    queryKey: ['getFile', req],
    queryFn: () => apiClient.fileService.Get(req),
    ...options,
  });
}

export function useCreateFile(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.fileService.Create({ data: { ...values } as storageservicev1_File }),
    ...options,
  });
}

export function useUpdateFile(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.fileService.Update({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteFile(
  options?: UseMutationOptions<
    object,
    Error,
    storageservicev1_DeleteFileRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => apiClient.fileService.Delete(data),
    ...options,
  });
}

// ==============================
// 文件存储枚举与工具函数
// ==============================

export const ossProviderList = computed(() => [
  { value: 'LOCAL', label: t('enum.ossProvider.LOCAL') },
  { value: 'MINIO', label: t('enum.ossProvider.MINIO') },
  { value: 'ALIYUN', label: t('enum.ossProvider.ALIYUN') },
  { value: 'QINIU', label: t('enum.ossProvider.QINIU') },
  { value: 'TENCENT', label: t('enum.ossProvider.TENCENT') },
  { value: 'BAIDU', label: t('enum.ossProvider.BAIDU') },
  { value: 'HUAWEI', label: t('enum.ossProvider.HUAWEI') },
  { value: 'AWS', label: t('enum.ossProvider.AWS') },
  { value: 'AZURE', label: t('enum.ossProvider.AZURE') },
  { value: 'GOOGLE', label: t('enum.ossProvider.GOOGLE') },
]);

export function ossProviderLabel(value: OSSProvider): string {
  const values = ossProviderList.value;
  const matchedItem = values.find((item) => item.value === value);
  return matchedItem ? matchedItem.label : '';
}

const OSS_PROVIDER_COLOR_MAP: Record<string, string> = {
  LOCAL: '#36D399',
  MINIO: '#2563EB',
  QINIU: '#722ED1',
  ALIYUN: '#FF6A00',
  TENCENT: '#12B7F5',
  BAIDU: '#4080FF',
  HUAWEI: '#E64340',
  AWS: '#FF9900',
  AZURE: '#0078D4',
  GOOGLE: '#4285F4',
  DEFAULT: '#C9CDD4',
};

export function ossProviderColor(type: OSSProvider): string {
  return (
    OSS_PROVIDER_COLOR_MAP[type as string] ??
    OSS_PROVIDER_COLOR_MAP.DEFAULT ??
    '#C9CDD4'
  );
}
