import { computed } from "vue";
import type { TagType } from "./shared";
import { useQuery, type UseQueryOptions } from "@tanstack/vue-query";
import type {
  auditservicev1_DataAccessAuditLog,
  auditservicev1_GetDataAccessAuditLogRequest,
  auditservicev1_ListDataAccessAuditLogResponse,
  auditservicev1_DataAccessAuditLog_AccessType as AccessType,
} from "@/api/generated/admin/service/v1";
import type { PaginationQuery } from "@go-wind/admin-core";
import { apiClient } from "@/api/client";
import { queryClient } from "@/plugins/vue-query";
import { i18n } from "@go-wind/admin-element";

const t = i18n.global.t;

// ==============================
// 数据访问审计日志
// ==============================

export function useListDataAccessAuditLogs(
  query: PaginationQuery,
  options?: UseQueryOptions<auditservicev1_ListDataAccessAuditLogResponse, Error>
) {
  return useQuery({
    queryKey: ["listDataAccessAuditLogs", query],
    queryFn: () => apiClient.dataAccessAuditLogService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListDataAccessAuditLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listDataAccessAuditLogs", params],
    queryFn: () => apiClient.dataAccessAuditLogService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetDataAccessAuditLog(
  req: auditservicev1_GetDataAccessAuditLogRequest,
  options?: UseQueryOptions<auditservicev1_DataAccessAuditLog, Error>
) {
  return useQuery({
    queryKey: ["getDataAccessAuditLog", req],
    queryFn: () => apiClient.dataAccessAuditLogService.Get(req),
    ...options,
  });
}

// ==============================
// 数据访问审计日志枚举与工具函数
// ==============================

export const dataAccessAuditLogAccessTypeList = computed(() => [
  { value: "SELECT", label: t("enum.dataAccessAuditLog.accessType.SELECT") },
  { value: "INSERT", label: t("enum.dataAccessAuditLog.accessType.INSERT") },
  { value: "UPDATE", label: t("enum.dataAccessAuditLog.accessType.UPDATE") },
  { value: "DELETE", label: t("enum.dataAccessAuditLog.accessType.DELETE") },
  { value: "VIEW", label: t("enum.dataAccessAuditLog.accessType.VIEW") },
  { value: "BULK_READ", label: t("enum.dataAccessAuditLog.accessType.BULK_READ") },
  { value: "EXPORT", label: t("enum.dataAccessAuditLog.accessType.EXPORT") },
  { value: "IMPORT", label: t("enum.dataAccessAuditLog.accessType.IMPORT") },
  { value: "DDL_CREATE", label: t("enum.dataAccessAuditLog.accessType.DDL_CREATE") },
  { value: "DDL_ALTER", label: t("enum.dataAccessAuditLog.accessType.DDL_ALTER") },
  { value: "DDL_DROP", label: t("enum.dataAccessAuditLog.accessType.DDL_DROP") },
  { value: "METADATA_READ", label: t("enum.dataAccessAuditLog.accessType.METADATA_READ") },
  { value: "SCAN", label: t("enum.dataAccessAuditLog.accessType.SCAN") },
  { value: "ADMIN_OPERATION", label: t("enum.dataAccessAuditLog.accessType.ADMIN_OPERATION") },
  { value: "OTHER", label: t("enum.dataAccessAuditLog.accessType.OTHER") },
]);

const DATA_ACCESS_AUDIT_LOG_ACCESS_TYPE_TAG_TYPE_MAP: Record<string, TagType> = {
  SELECT: "primary",
  INSERT: "success",
  UPDATE: "warning",
  DELETE: "danger",
  VIEW: "info",
  BULK_READ: "info",
  EXPORT: "success",
  IMPORT: "success",
  DDL_CREATE: "primary",
  DDL_ALTER: "warning",
  DDL_DROP: "danger",
  METADATA_READ: "info",
  SCAN: "info",
  ADMIN_OPERATION: "danger",
  OTHER: "info",
  DEFAULT: "info",
};

export function dataAccessAuditLogAccessTypeToType(accessType: AccessType): TagType {
  return (
    DATA_ACCESS_AUDIT_LOG_ACCESS_TYPE_TAG_TYPE_MAP[accessType as string] ||
    DATA_ACCESS_AUDIT_LOG_ACCESS_TYPE_TAG_TYPE_MAP.DEFAULT
  );
}

export function dataAccessAuditLogAccessTypeToName(accessType: AccessType) {
  const values = dataAccessAuditLogAccessTypeList.value;
  const matchedItem = values.find((item) => item.value === accessType);
  return matchedItem ? matchedItem.label : "";
}

// 数据分类码与后端 pkg/audit.ClassifyTable 一一对应
export const dataAccessAuditLogCategoryList = computed(() => [
  { value: "USER_DATA", label: t("enum.dataAccessAuditLog.dataCategory.USER_DATA") },
  { value: "ORG_DATA", label: t("enum.dataAccessAuditLog.dataCategory.ORG_DATA") },
  { value: "ACCESS_CONTROL", label: t("enum.dataAccessAuditLog.dataCategory.ACCESS_CONTROL") },
  { value: "TENANT_DATA", label: t("enum.dataAccessAuditLog.dataCategory.TENANT_DATA") },
  { value: "MESSAGE_DATA", label: t("enum.dataAccessAuditLog.dataCategory.MESSAGE_DATA") },
  { value: "AUDIT_LOG", label: t("enum.dataAccessAuditLog.dataCategory.AUDIT_LOG") },
  { value: "SYSTEM_CONFIG", label: t("enum.dataAccessAuditLog.dataCategory.SYSTEM_CONFIG") },
  { value: "UNKNOWN", label: t("enum.dataAccessAuditLog.dataCategory.UNKNOWN") },
]);

const DATA_ACCESS_AUDIT_LOG_CATEGORY_TAG_TYPE_MAP: Record<string, TagType> = {
  USER_DATA: "primary",
  ORG_DATA: "success",
  ACCESS_CONTROL: "danger",
  TENANT_DATA: "warning",
  MESSAGE_DATA: "success",
  AUDIT_LOG: "warning",
  SYSTEM_CONFIG: "primary",
  UNKNOWN: "info",
  DEFAULT: "info",
};

export function dataAccessAuditLogCategoryToType(category: string): TagType {
  return (
    DATA_ACCESS_AUDIT_LOG_CATEGORY_TAG_TYPE_MAP[category] ||
    DATA_ACCESS_AUDIT_LOG_CATEGORY_TAG_TYPE_MAP.DEFAULT
  );
}

export function dataAccessAuditLogCategoryToName(category: string) {
  const values = dataAccessAuditLogCategoryList.value;
  const matchedItem = values.find((item) => item.value === category);
  return matchedItem ? matchedItem.label : "";
}
