import { computed } from "vue";
import type { TagType } from "./shared";
import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from "@tanstack/vue-query";
import type {
  identityservicev1_DeletePositionRequest,
  identityservicev1_GetPositionRequest,
  identityservicev1_ListPositionResponse,
  identityservicev1_Position,
  identityservicev1_Position_Status as Position_Status,
  identityservicev1_Position_Type as Position_Type,
} from "@/api/generated/admin/service/v1";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import { apiClient } from "@/api/client";
import { queryClient } from "@/plugins/vue-query";
import { i18n } from "@/core/i18n";

const t = i18n.global.t;

// ==============================
// 职位管理
// ==============================

export function useListPositions(
  query: PaginationQuery,
  options?: UseQueryOptions<identityservicev1_ListPositionResponse, Error>
) {
  return useQuery({
    queryKey: ["listPositions", query],
    queryFn: () => apiClient.positionService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListPositions(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listPositions", params],
    queryFn: () => apiClient.positionService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetPosition(
  req: identityservicev1_GetPositionRequest,
  options?: UseQueryOptions<identityservicev1_Position, Error>
) {
  return useQuery({
    queryKey: ["getPosition", req],
    queryFn: () => apiClient.positionService.Get(req),
    ...options,
  });
}

export function useCreatePosition(options?: UseMutationOptions<{}, Error, Record<string, any>>) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.positionService.Create({ data: { ...values } as identityservicev1_Position }),
    ...options,
  });
}

export function useUpdatePosition(
  options?: UseMutationOptions<{}, Error, { id: string; values: Record<string, any> }>
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: Record<string, any> }) =>
      apiClient.positionService.Update({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeletePosition(
  options?: UseMutationOptions<{}, Error, identityservicev1_DeletePositionRequest>
) {
  return useMutation({
    mutationFn: (req) => apiClient.positionService.Delete(req),
    ...options,
  });
}

// ==============================
// 职位枚举与工具函数
// ==============================

export const membershipPositionStatusList = computed(() => [
  { value: "PROBATION", label: t("enum.membershipPosition.status.PROBATION") },
  { value: "ACTIVE", label: t("enum.membershipPosition.status.ACTIVE") },
  { value: "LEAVE", label: t("enum.membershipPosition.status.LEAVE") },
  { value: "RESIGNED", label: t("enum.membershipPosition.status.RESIGNED") },
  { value: "TERMINATED", label: t("enum.membershipPosition.status.TERMINATED") },
  { value: "EXPIRED", label: t("enum.membershipPosition.status.EXPIRED") },
]);

export function membershipPositionStatusToName(status: any) {
  const values = membershipPositionStatusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : "";
}

export const positionTypeList = computed(() => [
  { value: "REGULAR", label: t("enum.position.type.REGULAR") },
  { value: "LEADER", label: t("enum.position.type.LEADER") },
  { value: "MANAGER", label: t("enum.position.type.MANAGER") },
  { value: "INTERN", label: t("enum.position.type.INTERN") },
  { value: "CONTRACT", label: t("enum.position.type.CONTRACT") },
  { value: "OTHER", label: t("enum.position.type.OTHER") },
]);

export function positionTypeToName(status: Position_Status) {
  const values = positionTypeList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : "";
}

// 职位类型 → tag 语义 type（主题分化交由组件库 token：亮色 dark-2 文字、暗色 _dark-mode 柔化）
const POSITION_TYPE_TAG_TYPE_MAP: Record<string, TagType> = {
  REGULAR: "primary",
  LEADER: "warning",
  MANAGER: "success",
  INTERN: "info",
  CONTRACT: "info",
  OTHER: "info",
  DEFAULT: "info",
};

export function positionTypeToType(positionType: Position_Type): TagType {
  return POSITION_TYPE_TAG_TYPE_MAP[positionType as string] || POSITION_TYPE_TAG_TYPE_MAP.DEFAULT;
}
