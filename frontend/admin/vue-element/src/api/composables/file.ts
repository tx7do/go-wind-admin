import { computed } from "vue";
import type { TagType } from "./shared";
import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from "@tanstack/vue-query";
import type {
  storageservicev1_DeleteFileRequest,
  storageservicev1_File,
  storageservicev1_GetFileRequest,
  storageservicev1_ListFileResponse,
  storageservicev1_OSSProvider as OSSProvider,
} from "@/api/generated/admin/service/v1";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import { queryClient } from "@/plugins/vue-query";
import { apiClient } from "@/api/client";
import { i18n } from "@/core/i18n";

const t = i18n.global.t;

// ==============================
// 文件管理
// ==============================

export function useListFiles(
  query: PaginationQuery,
  options?: UseQueryOptions<storageservicev1_ListFileResponse, Error>
) {
  return useQuery({
    queryKey: ["listFiles", query],
    queryFn: () => apiClient.fileService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListFiles(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listFiles", params],
    queryFn: () => apiClient.fileService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetFile(
  req: storageservicev1_GetFileRequest,
  options?: UseQueryOptions<storageservicev1_File, Error>
) {
  return useQuery({
    queryKey: ["getFile", req],
    queryFn: () => apiClient.fileService.Get(req),
    ...options,
  });
}

export function useCreateFile(options?: UseMutationOptions<{}, Error, Record<string, any>>) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.fileService.Create({ data: { ...values } as storageservicev1_File }),
    ...options,
  });
}

export function useUpdateFile(
  options?: UseMutationOptions<{}, Error, { id: string; values: Record<string, any> }>
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: Record<string, any> }) =>
      apiClient.fileService.Update({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteFile(
  options?: UseMutationOptions<{}, Error, storageservicev1_DeleteFileRequest>
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
  { value: "LOCAL", label: t("enum.ossProvider.LOCAL") },
  { value: "MINIO", label: t("enum.ossProvider.MINIO") },
  { value: "ALIYUN", label: t("enum.ossProvider.ALIYUN") },
  { value: "QINIU", label: t("enum.ossProvider.QINIU") },
  { value: "TENCENT", label: t("enum.ossProvider.TENCENT") },
  { value: "BAIDU", label: t("enum.ossProvider.BAIDU") },
  { value: "HUAWEI", label: t("enum.ossProvider.HUAWEI") },
  { value: "AWS", label: t("enum.ossProvider.AWS") },
  { value: "AZURE", label: t("enum.ossProvider.AZURE") },
  { value: "GOOGLE", label: t("enum.ossProvider.GOOGLE") },
]);

export function ossProviderLabel(value: OSSProvider): string {
  const values = ossProviderList.value;
  const matchedItem = values.find((item) => item.value === value);
  return matchedItem ? matchedItem.label : "";
}

// OSS 供应商 → tag 语义 type：自建=success/primary，云厂商中性 info（品牌彩虹在语义色板下无法承载）
const OSS_PROVIDER_TAG_TYPE_MAP: Record<string, TagType> = {
  LOCAL: "success",
  MINIO: "primary",
  QINIU: "info",
  ALIYUN: "warning",
  TENCENT: "primary",
  BAIDU: "info",
  HUAWEI: "danger",
  AWS: "warning",
  AZURE: "primary",
  GOOGLE: "info",
  DEFAULT: "info",
};

export function ossProviderToType(type: OSSProvider): TagType {
  return OSS_PROVIDER_TAG_TYPE_MAP[type as string] || OSS_PROVIDER_TAG_TYPE_MAP.DEFAULT;
}
