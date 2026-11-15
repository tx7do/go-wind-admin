import { computed } from "vue";
import type { TagType } from "./shared";
import { useQuery, type UseQueryOptions } from "@tanstack/vue-query";
import type {
  auditservicev1_GetLoginAuditLogRequest,
  auditservicev1_ListOperationAuditLogResponse,
  auditservicev1_LoginAuditLog,
  auditservicev1_LoginAuditLog_ActionType as LoginAuditLog_ActionType,
  auditservicev1_LoginAuditLog_RiskLevel as LoginAuditLog_RiskLevel,
  auditservicev1_LoginAuditLog_Status as LoginAuditLog_Status,
} from "@/api/generated/admin/service/v1";
import type { PaginationQuery } from "@go-wind/admin-core";
import { apiClient } from "@/api/client";
import { queryClient } from "@/plugins/vue-query";
import { i18n } from "@go-wind/admin-element";

const t = i18n.global.t;

// ==============================
// 登录审计日志
// ==============================

export function useListLoginAuditLogs(
  query: PaginationQuery,
  options?: UseQueryOptions<auditservicev1_ListOperationAuditLogResponse, Error>
) {
  return useQuery({
    queryKey: ["listLoginAuditLogs", query],
    queryFn: () => apiClient.loginAuditLogService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListLoginAuditLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listLoginAuditLogs", params],
    queryFn: () => apiClient.loginAuditLogService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetLoginAuditLog(
  req: auditservicev1_GetLoginAuditLogRequest,
  options?: UseQueryOptions<auditservicev1_LoginAuditLog, Error>
) {
  return useQuery({
    queryKey: ["getLoginAuditLog", req],
    queryFn: () => apiClient.loginAuditLogService.Get(req),
    ...options,
  });
}

// ==============================
// 登录审计日志枚举与工具函数
// ==============================

const LOGIN_AUDIT_LOG_STATUS_TAG_TYPE_MAP: Record<string, TagType> = {
  STATUS_UNSPECIFIED: "info",
  SUCCESS: "success",
  FAILED: "danger",
  PARTIAL: "warning",
  LOCKED: "warning",
};

const LOGIN_AUDIT_LOG_ACTION_TYPE_TAG_TYPE_MAP: Record<string, TagType> = {
  ACTION_TYPE_UNSPECIFIED: "info",
  LOGIN: "success",
  LOGOUT: "info",
  SESSION_EXPIRED: "warning",
  KICKED_OUT: "danger",
  PASSWORD_RESET: "warning",
};

const LOGIN_AUDIT_LOG_RISK_LEVEL_TAG_TYPE_MAP: Record<string, TagType> = {
  RISK_LEVEL_UNSPECIFIED: "info",
  LOW: "success",
  MEDIUM: "warning",
  HIGH: "danger",
};

export function loginAuditLogStatusToType(status: LoginAuditLog_Status): TagType {
  return LOGIN_AUDIT_LOG_STATUS_TAG_TYPE_MAP[status as string] || "info";
}

export function loginAuditLogActionTypeToType(actionType: LoginAuditLog_ActionType): TagType {
  return LOGIN_AUDIT_LOG_ACTION_TYPE_TAG_TYPE_MAP[actionType as string] || "info";
}

export function loginAuditLogRiskLevelToType(riskLevel: LoginAuditLog_RiskLevel): TagType {
  return LOGIN_AUDIT_LOG_RISK_LEVEL_TAG_TYPE_MAP[riskLevel as string] || "info";
}

export function loginAuditLogStatusToName(status: LoginAuditLog_Status) {
  switch (status) {
    case "FAILED":
      return t("enum.loginAuditLog.status.FAILED");
    case "PARTIAL":
      return t("enum.loginAuditLog.status.PARTIAL");
    case "SUCCESS":
      return t("enum.loginAuditLog.status.SUCCESS");
    case "LOCKED":
      return t("enum.loginAuditLog.status.LOCKED");
    default:
      return "";
  }
}

export const loginAuditLogStatusList = computed(() => [
  { value: "FAILED", label: t("enum.loginAuditLog.status.FAILED") },
  { value: "PARTIAL", label: t("enum.loginAuditLog.status.PARTIAL") },
  { value: "SUCCESS", label: t("enum.loginAuditLog.status.SUCCESS") },
  { value: "LOCKED", label: t("enum.loginAuditLog.status.LOCKED") },
]);

export function loginAuditLogActionTypeToName(status: LoginAuditLog_ActionType) {
  switch (status) {
    case "LOGIN":
      return t("enum.loginAuditLog.actionType.LOGIN");
    case "LOGOUT":
      return t("enum.loginAuditLog.actionType.LOGOUT");
    case "SESSION_EXPIRED":
      return t("enum.loginAuditLog.actionType.SESSION_EXPIRED");
    default:
      return "";
  }
}

export const loginAuditLogActionTypeList = computed(() => [
  { value: "LOGIN", label: t("enum.loginAuditLog.actionType.LOGIN") },
  { value: "LOGOUT", label: t("enum.loginAuditLog.actionType.LOGOUT") },
  { value: "SESSION_EXPIRED", label: t("enum.loginAuditLog.actionType.SESSION_EXPIRED") },
]);

export function loginAuditLogRiskLevelToName(status: LoginAuditLog_RiskLevel) {
  switch (status) {
    case "HIGH":
      return t("enum.loginAuditLog.riskLevel.HIGH");
    case "LOW":
      return t("enum.loginAuditLog.riskLevel.LOW");
    case "MEDIUM":
      return t("enum.loginAuditLog.riskLevel.MEDIUM");
    default:
      return "";
  }
}

export const loginAuditLogRiskLevelList = computed(() => [
  { value: "HIGH", label: t("enum.loginAuditLog.riskLevel.HIGH") },
  { value: "LOW", label: t("enum.loginAuditLog.riskLevel.LOW") },
  { value: "MEDIUM", label: t("enum.loginAuditLog.riskLevel.MEDIUM") },
]);
