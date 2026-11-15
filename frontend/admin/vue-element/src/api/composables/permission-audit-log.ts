import { computed } from "vue";
import type { TagType } from "./shared";
import { useQuery, type UseQueryOptions } from "@tanstack/vue-query";
import type {
  auditservicev1_GetPermissionAuditLogRequest,
  auditservicev1_ListPermissionAuditLogResponse,
  auditservicev1_PermissionAuditLog,
  auditservicev1_PermissionAuditLog_ActionType as PermissionAuditActionType,
} from "@/api/generated/admin/service/v1";
import type { PaginationQuery } from "@go-wind/admin-core";
import { apiClient } from "@/api/client";
import { queryClient } from "@/plugins/vue-query";
import { i18n } from "@go-wind/admin-element";

const t = i18n.global.t;

// ==============================
// 权限审计日志
// ==============================

export function useListPermissionAuditLogs(
  query: PaginationQuery,
  options?: UseQueryOptions<auditservicev1_ListPermissionAuditLogResponse, Error>
) {
  return useQuery({
    queryKey: ["listPermissionAuditLogs", query],
    queryFn: () => apiClient.permissionAuditLogService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListPermissionAuditLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listPermissionAuditLogs", params],
    queryFn: () => apiClient.permissionAuditLogService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetPermissionAuditLog(
  req: auditservicev1_GetPermissionAuditLogRequest,
  options?: UseQueryOptions<auditservicev1_PermissionAuditLog, Error>
) {
  return useQuery({
    queryKey: ["getPermissionAuditLog", req],
    queryFn: () => apiClient.permissionAuditLogService.Get(req),
    ...options,
  });
}

// ==============================
// 权限审计日志枚举与工具函数
// ==============================

export const permissionAuditLogActionList = computed(() => [
  { value: "GRANT", label: t("enum.permissionAuditLog.action.GRANT") },
  { value: "REVOKE", label: t("enum.permissionAuditLog.action.REVOKE") },
  { value: "UPDATE", label: t("enum.permissionAuditLog.action.UPDATE") },
  { value: "RESET", label: t("enum.permissionAuditLog.action.RESET") },
  { value: "CREATE", label: t("enum.permissionAuditLog.action.CREATE") },
  { value: "DELETE", label: t("enum.permissionAuditLog.action.DELETE") },
  { value: "ASSIGN", label: t("enum.permissionAuditLog.action.ASSIGN") },
  { value: "UNASSIGN", label: t("enum.permissionAuditLog.action.UNASSIGN") },
  { value: "BULK_GRANT", label: t("enum.permissionAuditLog.action.BULK_GRANT") },
  { value: "BULK_REVOKE", label: t("enum.permissionAuditLog.action.BULK_REVOKE") },
  { value: "EXPIRE", label: t("enum.permissionAuditLog.action.EXPIRE") },
  { value: "SUSPEND", label: t("enum.permissionAuditLog.action.SUSPEND") },
  { value: "RESUME", label: t("enum.permissionAuditLog.action.RESUME") },
  { value: "ROLLBACK", label: t("enum.permissionAuditLog.action.ROLLBACK") },
  { value: "OTHER", label: t("enum.permissionAuditLog.action.OTHER") },
]);

const PERMISSION_AUDIT_LOG_ACTION_TAG_TYPE_MAP: Record<string, TagType> = {
  GRANT: "success",
  REVOKE: "danger",
  UPDATE: "primary",
  RESET: "warning",
  CREATE: "primary",
  DELETE: "danger",
  ASSIGN: "success",
  UNASSIGN: "danger",
  BULK_GRANT: "success",
  BULK_REVOKE: "danger",
  EXPIRE: "danger",
  SUSPEND: "danger",
  RESUME: "success",
  ROLLBACK: "warning",
  OTHER: "info",
  DEFAULT: "info",
};

export function permissionAuditLogActionToType(action: PermissionAuditActionType): TagType {
  return (
    PERMISSION_AUDIT_LOG_ACTION_TAG_TYPE_MAP[action as string] ||
    PERMISSION_AUDIT_LOG_ACTION_TAG_TYPE_MAP.DEFAULT
  );
}

export function permissionAuditLogActionToName(action: PermissionAuditActionType) {
  const values = permissionAuditLogActionList.value;
  const matchedItem = values.find((item) => item.value === action);
  return matchedItem ? matchedItem.label : "";
}
