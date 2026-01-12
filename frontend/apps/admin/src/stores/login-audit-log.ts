import { $t } from '@vben/locales';

import { defineStore } from 'pinia';

import {
  createLoginAuditLogServiceClient,
  type auditservicev1_LoginAuditLog_ActionType as LoginAuditLog_ActionType,
  type auditservicev1_LoginAuditLog_RiskLevel as LoginAuditLog_RiskLevel,
  type auditservicev1_LoginAuditLog_Status as LoginAuditLog_Status,
} from '#/generated/api/admin/service/v1';
import { makeQueryString } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const useLoginAuditLogStore = defineStore('login-audit-log', () => {
  const service = createLoginAuditLogServiceClient(requestClientRequestHandler);

  /**
   * 查询登录审计日志列表
   */
  async function listLoginAuditLog(
    paging?: Paging,
    formValues?: null | object,
    fieldMask?: null | string,
    orderBy?: null | string[],
  ) {
    const noPaging =
      paging?.page === undefined && paging?.pageSize === undefined;
    return await service.List({
      // @ts-ignore proto generated code is error.
      fieldMask,
      orderBy: orderBy ?? [],
      query: makeQueryString(formValues ?? null),
      page: paging?.page,
      pageSize: paging?.pageSize,
      noPaging,
    });
  }

  /**
   * 查询登录审计日志
   */
  async function getLoginAuditLog(id: number) {
    return await service.Get({ id });
  }

  function $reset() {}

  return {
    $reset,
    listLoginAuditLog,
    getLoginAuditLog,
  };
});

/**
 * 成功失败的颜色
 * @param success
 */
export function successToColor(success: boolean) {
  // 成功用柔和的绿色，失败用柔和的红色，兼顾视觉舒适度与直观性
  return success ? 'limegreen' : 'crimson';
}

export function successToName(success: boolean) {
  return success
    ? $t('enum.successStatus.success')
    : $t('enum.successStatus.failed');
}

export function successToNameWithStatusCode(
  success: boolean,
  statusCode: number,
) {
  return success
    ? $t('enum.successStatus.success')
    : ` ${$t('enum.successStatus.failed')} (${statusCode})`;
}

// 通用色值常量
const COLORS = {
  neutral: '#6c757d', // 中性灰（未知/默认）
  success: '#28a745', // 成功绿（低风险/登录成功）
  warning: '#fd7e14', // 警告橙（验证中/会话过期/中风险）
  danger: '#dc3545', // 危险红（失败/高风险）
  info: '#17a2b8', // 信息蓝（登出等正常操作）
};

const LOGIN_AUDIT_LOG_STATUS_COLOR_MAP: Record<LoginAuditLog_Status, string> = {
  STATUS_UNSPECIFIED: COLORS.neutral,
  SUCCESS: COLORS.success,
  FAILED: COLORS.danger,
  PARTIAL: COLORS.warning,
};

const LOGIN_AUDIT_LOG_ACTION_TYPE_COLOR_MAP: Record<
  LoginAuditLog_ActionType,
  string
> = {
  ACTION_TYPE_UNSPECIFIED: COLORS.neutral,
  LOGIN: COLORS.success,
  LOGOUT: COLORS.info,
  SESSION_EXPIRED: COLORS.warning,
};

const LOGIN_AUDIT_LOG_RISK_LEVEL_COLOR_MAP: Record<
  LoginAuditLog_RiskLevel,
  string
> = {
  RISK_LEVEL_UNSPECIFIED: COLORS.neutral,
  LOW: COLORS.success,
  MEDIUM: COLORS.warning,
  HIGH: COLORS.danger,
};

/**
 * 获取登录状态对应的颜色（兜底处理未知枚举值）
 */
export function getLoginAuditLogStatusColor(
  status: LoginAuditLog_Status,
): string {
  return LOGIN_AUDIT_LOG_STATUS_COLOR_MAP[status] || COLORS.neutral;
}

/**
 * 获取登录动作类型对应的颜色
 */
export function getLoginAuditLogActionTypeColor(
  actionType: LoginAuditLog_ActionType,
): string {
  return LOGIN_AUDIT_LOG_ACTION_TYPE_COLOR_MAP[actionType] || COLORS.neutral;
}

/**
 * 获取登录风险等级对应的颜色
 */
export function getLoginAuditLogRiskLevelColor(
  riskLevel: LoginAuditLog_RiskLevel,
): string {
  return LOGIN_AUDIT_LOG_RISK_LEVEL_COLOR_MAP[riskLevel] || COLORS.neutral;
}

export function loginAuditLogStatusToName(status: LoginAuditLog_Status) {
  switch (status) {
    case 'FAILED': {
      return $t('enum.loginAuditLog.status.FAILED');
    }
    case 'PARTIAL': {
      return $t('enum.loginAuditLog.status.PARTIAL');
    }
    case 'SUCCESS': {
      return $t('enum.loginAuditLog.status.SUCCESS');
    }
  }
}

export function loginAuditLogActiveTypeToName(
  status: LoginAuditLog_ActionType,
) {
  switch (status) {
    case 'LOGIN': {
      return $t('enum.loginAuditLog.actionType.LOGIN');
    }
    case 'LOGOUT': {
      return $t('enum.loginAuditLog.actionType.LOGOUT');
    }
    case 'SESSION_EXPIRED': {
      return $t('enum.loginAuditLog.actionType.SESSION_EXPIRED');
    }
  }
}

export function loginAuditLogRiskLevelToName(status: LoginAuditLog_RiskLevel) {
  switch (status) {
    case 'HIGH': {
      return $t('enum.loginAuditLog.riskLevel.HIGH');
    }
    case 'LOW': {
      return $t('enum.loginAuditLog.riskLevel.LOW');
    }
    case 'MEDIUM': {
      return $t('enum.loginAuditLog.riskLevel.MEDIUM');
    }
  }
}
