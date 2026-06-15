import type {
  auditservicev1_GetLoginAuditLogRequest,
  auditservicev1_ListOperationAuditLogResponse,
  auditservicev1_LoginAuditLog,
  auditservicev1_LoginAuditLog_ActionType as LoginAuditLog_ActionType,
  auditservicev1_LoginAuditLog_RiskLevel as LoginAuditLog_RiskLevel,
  auditservicev1_LoginAuditLog_Status as LoginAuditLog_Status,
} from '#/api/generated/admin/service/v1';
import type { PaginationQuery } from '#/transport/rest';

import { computed } from 'vue';

import { i18n } from '@vben/locales';

import { useQuery, type UseQueryOptions } from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';

const t = i18n.global.t;

// ==============================
// 登录审计日志
// ==============================

export function useListLoginAuditLogs(
  query: PaginationQuery,
  options?: UseQueryOptions<
    auditservicev1_ListOperationAuditLogResponse,
    Error
  >,
) {
  return useQuery({
    queryKey: ['listLoginAuditLogs', query],
    queryFn: () => apiClient.loginAuditLogService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListLoginAuditLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listLoginAuditLogs', params],
    queryFn: () => apiClient.loginAuditLogService.List(params.toRawParams()),
    retry: 0,
  });
}

export function useGetLoginAuditLog(
  req: auditservicev1_GetLoginAuditLogRequest,
  options?: UseQueryOptions<auditservicev1_LoginAuditLog, Error>,
) {
  return useQuery({
    queryKey: ['getLoginAuditLog', req],
    queryFn: () => apiClient.loginAuditLogService.Get(req),
    ...options,
  });
}

// ==============================
// 登录审计日志枚举与工具函数
// ==============================

const COLORS = {
  neutral: '#86909C',
  success: '#1F7A34',
  warning: '#FA8C16',
  danger: '#D32F2F',
  info: '#1890FF',
};

const LOGIN_AUDIT_LOG_STATUS_COLOR_MAP: Record<string, string> = {
  STATUS_UNSPECIFIED: COLORS.neutral,
  SUCCESS: COLORS.success,
  FAILED: COLORS.danger,
  PARTIAL: COLORS.warning,
  LOCKED: COLORS.warning,
};

const LOGIN_AUDIT_LOG_ACTION_TYPE_COLOR_MAP: Record<string, string> = {
  ACTION_TYPE_UNSPECIFIED: COLORS.neutral,
  LOGIN: COLORS.success,
  LOGOUT: COLORS.info,
  SESSION_EXPIRED: COLORS.warning,
  KICKED_OUT: COLORS.warning,
  PASSWORD_RESET: COLORS.warning,
};

const LOGIN_AUDIT_LOG_RISK_LEVEL_COLOR_MAP: Record<string, string> = {
  RISK_LEVEL_UNSPECIFIED: COLORS.neutral,
  LOW: COLORS.success,
  MEDIUM: COLORS.warning,
  HIGH: COLORS.danger,
};

export function getLoginAuditLogStatusColor(
  status: LoginAuditLog_Status,
): string {
  return LOGIN_AUDIT_LOG_STATUS_COLOR_MAP[status as string] || COLORS.neutral;
}

export function getLoginAuditLogActionTypeColor(
  actionType: LoginAuditLog_ActionType,
): string {
  return (
    LOGIN_AUDIT_LOG_ACTION_TYPE_COLOR_MAP[actionType as string] ||
    COLORS.neutral
  );
}

export function getLoginAuditLogRiskLevelColor(
  riskLevel: LoginAuditLog_RiskLevel,
): string {
  return (
    LOGIN_AUDIT_LOG_RISK_LEVEL_COLOR_MAP[riskLevel as string] || COLORS.neutral
  );
}

export function loginAuditLogStatusToName(status: LoginAuditLog_Status) {
  switch (status) {
    case 'FAILED': {
      return t('enum.loginAuditLog.status.FAILED');
    }
    case 'PARTIAL': {
      return t('enum.loginAuditLog.status.PARTIAL');
    }
    case 'SUCCESS': {
      return t('enum.loginAuditLog.status.SUCCESS');
    }
    default: {
      return '';
    }
  }
}

export const loginAuditLogStatusList = computed(() => [
  { value: 'FAILED', label: t('enum.loginAuditLog.status.FAILED') },
  { value: 'PARTIAL', label: t('enum.loginAuditLog.status.PARTIAL') },
  { value: 'SUCCESS', label: t('enum.loginAuditLog.status.SUCCESS') },
]);

export function loginAuditLogActionTypeToName(
  status: LoginAuditLog_ActionType,
) {
  switch (status) {
    case 'LOGIN': {
      return t('enum.loginAuditLog.actionType.LOGIN');
    }
    case 'LOGOUT': {
      return t('enum.loginAuditLog.actionType.LOGOUT');
    }
    case 'SESSION_EXPIRED': {
      return t('enum.loginAuditLog.actionType.SESSION_EXPIRED');
    }
    default: {
      return '';
    }
  }
}

export const loginAuditLogActionTypeList = computed(() => [
  { value: 'LOGIN', label: t('enum.loginAuditLog.actionType.LOGIN') },
  { value: 'LOGOUT', label: t('enum.loginAuditLog.actionType.LOGOUT') },
  {
    value: 'SESSION_EXPIRED',
    label: t('enum.loginAuditLog.actionType.SESSION_EXPIRED'),
  },
]);

export function loginAuditLogRiskLevelToName(status: LoginAuditLog_RiskLevel) {
  switch (status) {
    case 'HIGH': {
      return t('enum.loginAuditLog.riskLevel.HIGH');
    }
    case 'LOW': {
      return t('enum.loginAuditLog.riskLevel.LOW');
    }
    case 'MEDIUM': {
      return t('enum.loginAuditLog.riskLevel.MEDIUM');
    }
    default: {
      return '';
    }
  }
}

export const loginAuditLogRiskLevelList = computed(() => [
  { value: 'HIGH', label: t('enum.loginAuditLog.riskLevel.HIGH') },
  { value: 'LOW', label: t('enum.loginAuditLog.riskLevel.LOW') },
  { value: 'MEDIUM', label: t('enum.loginAuditLog.riskLevel.MEDIUM') },
]);
