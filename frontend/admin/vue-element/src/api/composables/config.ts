import { computed } from "vue";
import type { TagType } from "./shared";
import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from "@tanstack/vue-query";
import type {
  configservicev1_Config,
  configservicev1_DeleteConfigRequest,
  configservicev1_GetConfigRequest,
  configservicev1_ListConfigResponse,
} from "@/api/generated/admin/service/v1";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import { apiClient } from "@/api/client";
import { queryClient } from "@/plugins/vue-query";
import { i18n } from "@/core/i18n";

const t = i18n.global.t;

// ==============================
// 参数管理（系统参数，动态 KV）
// ==============================

export function useListConfigs(
  query: PaginationQuery,
  options?: UseQueryOptions<configservicev1_ListConfigResponse, Error>
) {
  return useQuery({
    queryKey: ["listConfigs", query],
    queryFn: () => apiClient.configService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListConfigs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listConfigs", params],
    queryFn: () => apiClient.configService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetConfig(
  req: configservicev1_GetConfigRequest,
  options?: UseQueryOptions<configservicev1_Config, Error>
) {
  return useQuery({
    queryKey: ["getConfig", req],
    queryFn: () => apiClient.configService.Get(req),
    ...options,
  });
}

export function useCreateConfig(options?: UseMutationOptions<{}, Error, Record<string, any>>) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.configService.Create({ data: { ...values } as configservicev1_Config }),
    ...options,
  });
}

export function useUpdateConfig(
  options?: UseMutationOptions<{}, Error, { id: string; values: Record<string, any> }>
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: Record<string, any> }) =>
      apiClient.configService.Update({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteConfig(
  options?: UseMutationOptions<{}, Error, configservicev1_DeleteConfigRequest>
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
  { value: "STRING", label: t("enum.config.valueType.STRING") },
  { value: "BOOL", label: t("enum.config.valueType.BOOL") },
  { value: "INT", label: t("enum.config.valueType.INT") },
]);

export function configValueTypeToName(valueType: any) {
  const values = configValueTypeList.value;
  const matchedItem = values.find((item) => item.value === valueType);
  return matchedItem ? matchedItem.label : "";
}

const CONFIG_VALUE_TYPE_TAG_TYPE_MAP: Record<string, TagType> = {
  STRING: "primary",
  BOOL: "success",
  INT: "warning",
  DEFAULT: "info",
};

export function configValueTypeToType(valueType: any): TagType {
  return (
    CONFIG_VALUE_TYPE_TAG_TYPE_MAP[valueType as string] ||
    CONFIG_VALUE_TYPE_TAG_TYPE_MAP.DEFAULT
  );
}
