import { computed } from "vue";
import type { TagType } from "./shared";
import { useQuery, type UseQueryOptions } from "@tanstack/vue-query";
import type {
  auditservicev1_GetOperationAuditLogRequest,
  auditservicev1_ListOperationAuditLogResponse,
  auditservicev1_OperationAuditLog,
  auditservicev1_OperationAuditLog_ActionType as OperationActionType,
} from "@/api/generated/admin/service/v1";
import type { PaginationQuery } from "@go-wind/admin-core";
import { apiClient } from "@/api/client";
import { queryClient } from "@/plugins/vue-query";
import { i18n } from "@go-wind/admin-element";

const t = i18n.global.t;

// ==============================
// 操作审计日志
// ==============================

export function useListOperationAuditLogs(
  query: PaginationQuery,
  options?: UseQueryOptions<auditservicev1_ListOperationAuditLogResponse, Error>
) {
  return useQuery({
    queryKey: ["listOperationAuditLogs", query],
    queryFn: () => apiClient.operationAuditLogService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListOperationAuditLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listOperationAuditLogs", params],
    queryFn: () => apiClient.operationAuditLogService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetOperationAuditLog(
  req: auditservicev1_GetOperationAuditLogRequest,
  options?: UseQueryOptions<auditservicev1_OperationAuditLog, Error>
) {
  return useQuery({
    queryKey: ["getOperationAuditLog", req],
    queryFn: () => apiClient.operationAuditLogService.Get(req),
    ...options,
  });
}

// ==============================
// 操作审计日志枚举与工具函数
// ==============================

export const operationAuditLogActionList = computed(() => [
  { value: "CREATE", label: t("enum.operationAuditLog.action.CREATE") },
  { value: "UPDATE", label: t("enum.operationAuditLog.action.UPDATE") },
  { value: "DELETE", label: t("enum.operationAuditLog.action.DELETE") },
  { value: "READ", label: t("enum.operationAuditLog.action.READ") },
  { value: "ASSIGN", label: t("enum.operationAuditLog.action.ASSIGN") },
  { value: "UNASSIGN", label: t("enum.operationAuditLog.action.UNASSIGN") },
  { value: "EXPORT", label: t("enum.operationAuditLog.action.EXPORT") },
  { value: "IMPORT", label: t("enum.operationAuditLog.action.IMPORT") },
  { value: "OTHER", label: t("enum.operationAuditLog.action.OTHER") },
]);

const OPERATION_AUDIT_LOG_ACTION_TAG_TYPE_MAP: Record<string, TagType> = {
  CREATE: "success",
  UPDATE: "primary",
  DELETE: "danger",
  READ: "info",
  ASSIGN: "warning",
  UNASSIGN: "warning",
  EXPORT: "success",
  IMPORT: "primary",
  OTHER: "info",
  DEFAULT: "info",
};

export function operationAuditLogActionToType(action: OperationActionType): TagType {
  return (
    OPERATION_AUDIT_LOG_ACTION_TAG_TYPE_MAP[action as string] ||
    OPERATION_AUDIT_LOG_ACTION_TAG_TYPE_MAP.DEFAULT
  );
}

export function operationAuditLogActionToName(action: OperationActionType) {
  const values = operationAuditLogActionList.value;
  const matchedItem = values.find((item) => item.value === action);
  return matchedItem ? matchedItem.label : "";
}
