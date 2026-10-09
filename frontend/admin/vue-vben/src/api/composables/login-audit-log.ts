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
    staleTime: 0,
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

// 状态色使用 antd 预设状态名（由主题 token 驱动，亮/暗自动切换），替代原硬编码 hex 调色板。
// 语义：SUCCESS/LOW=success，PARTIAL/LOCKED/SESSION_EXPIRED/KICKED_OUT/PASSWORD_RESET/MEDIUM=warning，
// FAILED/HIGH=error，UNSPECIFIED/LOGOUT/default=default。
type LoginAuditStatusColor = 'success' | 'error' | 'warning' | 'default';

const LOGIN_AUDIT_LOG_STATUS_COLOR_MAP: Record<
  string,
  LoginAuditStatusColor
> = {
  STATUS_UNSPECIFIED: 'default',
  SUCCESS: 'success',
  FAILED: 'error',
  PARTIAL: 'warning',
  LOCKED: 'warning',
};

const LOGIN_AUDIT_LOG_ACTION_TYPE_COLOR_MAP: Record<
  string,
  LoginAuditStatusColor
> = {
  ACTION_TYPE_UNSPECIFIED: 'default',
  LOGIN: 'success',
  LOGOUT: 'default',
  SESSION_EXPIRED: 'warning',
  KICKED_OUT: 'warning',
  PASSWORD_RESET: 'warning',
};

const LOGIN_AUDIT_LOG_RISK_LEVEL_COLOR_MAP: Record<
  string,
  LoginAuditStatusColor
> = {
  RISK_LEVEL_UNSPECIFIED: 'default',
  LOW: 'success',
  MEDIUM: 'warning',
  HIGH: 'error',
};

export function getLoginAuditLogStatusColor(
  status: LoginAuditLog_Status,
): LoginAuditStatusColor {
  return LOGIN_AUDIT_LOG_STATUS_COLOR_MAP[status as string] ?? 'default';
}

export function getLoginAuditLogActionTypeColor(
  actionType: LoginAuditLog_ActionType,
): LoginAuditStatusColor {
  return (
    LOGIN_AUDIT_LOG_ACTION_TYPE_COLOR_MAP[actionType as string] ?? 'default'
  );
}

export function getLoginAuditLogRiskLevelColor(
  riskLevel: LoginAuditLog_RiskLevel,
): LoginAuditStatusColor {
  return (
    LOGIN_AUDIT_LOG_RISK_LEVEL_COLOR_MAP[riskLevel as string] ?? 'default'
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
    case 'LOCKED': {
      return t('enum.loginAuditLog.status.LOCKED');
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
  { value: 'LOCKED', label: t('enum.loginAuditLog.status.LOCKED') },
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
