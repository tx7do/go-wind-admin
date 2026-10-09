import type {
  configservicev1_Config,
  configservicev1_DeleteConfigRequest,
  configservicev1_GetConfigRequest,
  configservicev1_ListConfigResponse,
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
// 参数管理（系统参数，动态 KV）
// ==============================

export function useListConfigs(
  query: PaginationQuery,
  options?: UseQueryOptions<configservicev1_ListConfigResponse, Error>,
) {
  return useQuery({
    queryKey: ['listConfigs', query],
    queryFn: () => apiClient.configService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListConfigs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listConfigs', params],
    queryFn: () => apiClient.configService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetConfig(
  req: configservicev1_GetConfigRequest,
  options?: UseQueryOptions<configservicev1_Config, Error>,
) {
  return useQuery({
    queryKey: ['getConfig', req],
    queryFn: () => apiClient.configService.Get(req),
    ...options,
  });
}

export function useCreateConfig(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.configService.Create({ data: { ...values } as configservicev1_Config }),
    ...options,
  });
}

export function useUpdateConfig(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.configService.Update({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteConfig(
  options?: UseMutationOptions<object, Error, configservicev1_DeleteConfigRequest>,
) {
  return useMutation({
    mutationFn: (req) => apiClient.configService.Delete(req),
    ...options,
  });
}

// ==============================
// 参数值类型枚举与工具函数
// ==============================

export const configValueTypeList = computed(() => [
  { value: 'STRING', label: t('enum.config.valueType.STRING') },
  { value: 'BOOL', label: t('enum.config.valueType.BOOL') },
  { value: 'INT', label: t('enum.config.valueType.INT') },
]);

export function configValueTypeToName(valueType: any) {
  const values = configValueTypeList.value;
  const matchedItem = values.find((item) => item.value === valueType);
  return matchedItem ? matchedItem.label : '';
}

// 类型分类色使用 antd 预设调色板名（由主题 token 驱动，亮/暗自动切换）。
const CONFIG_VALUE_TYPE_COLOR_MAP: Record<
  string,
  'blue' | 'purple' | 'orange' | 'default'
> = {
  STRING: 'blue',
  BOOL: 'purple',
  INT: 'orange',
  DEFAULT: 'default',
};

export function configValueTypeToColor(
  valueType: any,
): 'blue' | 'purple' | 'orange' | 'default' {
  return CONFIG_VALUE_TYPE_COLOR_MAP[valueType as string] ?? 'default';
}
