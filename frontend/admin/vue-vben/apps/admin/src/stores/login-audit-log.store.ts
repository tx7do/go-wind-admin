import { computed } from 'vue';

import { $t } from '@vben/locales';
import { useUserStore } from '@vben/stores';

import { defineStore } from 'pinia';

import {
  createLoginAuditLogServiceClient,
  type auditservicev1_LoginAuditLog_ActionType as LoginAuditLog_ActionType,
  type auditservicev1_LoginAuditLog_RiskLevel as LoginAuditLog_RiskLevel,
  type auditservicev1_LoginAuditLog_Status as LoginAuditLog_Status,
} from '#/generated/api/admin/service/v1';
import { makeOrderBy, makeQueryString } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const useLoginAuditLogStore = defineStore('login-audit-log', () => {
  const service = createLoginAuditLogServiceClient(requestClientRequestHandler);
  const userStore = useUserStore();

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
      orderBy: makeOrderBy(orderBy),
      query: makeQueryString(formValues, userStore.isTenantUser()),
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
  neutral: '#86909C', // 中性灰（未知/默认）
  success: '#1F7A34', // 成功绿（低风险/登录成功）
  warning: '#FA8C16', // 警告橙（验证中/会话过期/中风险）
  danger: '#D32F2F', // 危险红（失败/高风险）
  info: '#1890FF', // 信息蓝（登出等正常操作）
};

// 登录审计日志状态-颜色映射（语义严格对齐，视觉统一）
const LOGIN_AUDIT_LOG_STATUS_COLOR_MAP: Record<LoginAuditLog_Status, string> = {
  STATUS_UNSPECIFIED: COLORS.neutral, // 未定义状态：中性灰兜底
  SUCCESS: COLORS.success, // 登录成功：成功绿
  FAILED: COLORS.danger, // 登录失败：危险红（高风险明确标识）
  PARTIAL: COLORS.warning, // 部分成功（如权限部分授予）：警告橙
  LOCKED: COLORS.warning, // 账号锁定：警告橙（中风险，与其他警告状态统一）
};

// 登录审计日志操作类型-颜色映射（按操作风险等级分类）
const LOGIN_AUDIT_LOG_ACTION_TYPE_COLOR_MAP: Record<
  LoginAuditLog_ActionType,
  string
> = {
  ACTION_TYPE_UNSPECIFIED: COLORS.neutral, // 未定义操作：中性灰兜底
  LOGIN: COLORS.success, // 登录操作（成功）：成功绿
  LOGOUT: COLORS.info, // 登出操作（正常）：信息蓝
  SESSION_EXPIRED: COLORS.warning, // 会话过期（需注意）：警告橙
  KICKED_OUT: COLORS.warning, // 被踢出（异常操作）：警告橙
  PASSWORD_RESET: COLORS.warning, // 密码重置（敏感操作）：警告橙
};

// 登录审计日志风险等级-颜色映射（按风险等级语义绑定，直观区分风险程度）
const LOGIN_AUDIT_LOG_RISK_LEVEL_COLOR_MAP: Record<
  LoginAuditLog_RiskLevel,
  string
> = {
  RISK_LEVEL_UNSPECIFIED: COLORS.neutral, // 未定义风险：中性灰兜底
  LOW: COLORS.success, // 低风险：成功绿（与低风险操作语义一致）
  MEDIUM: COLORS.warning, // 中风险：警告橙（需关注但非紧急）
  HIGH: COLORS.danger, // 高风险：危险红（需立即处理）
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

export const loginAuditLogStatusList = computed(() => [
  { value: 'FAILED', label: $t('enum.loginAuditLog.status.FAILED') },
  { value: 'PARTIAL', label: $t('enum.loginAuditLog.status.PARTIAL') },
  { value: 'SUCCESS', label: $t('enum.loginAuditLog.status.SUCCESS') },
]);

export function loginAuditLogActionTypeToName(
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

export const loginAuditLogActionTypeList = computed(() => [
  { value: 'LOGIN', label: $t('enum.loginAuditLog.actionType.LOGIN') },
  { value: 'LOGOUT', label: $t('enum.loginAuditLog.actionType.LOGOUT') },
  {
    value: 'SESSION_EXPIRED',
    label: $t('enum.loginAuditLog.actionType.SESSION_EXPIRED'),
  },
]);

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

export const loginAuditLogRiskLevelList = computed(() => [
  { value: 'HIGH', label: $t('enum.loginAuditLog.riskLevel.HIGH') },
  { value: 'LOW', label: $t('enum.loginAuditLog.riskLevel.LOW') },
  {
    value: 'MEDIUM',
    label: $t('enum.loginAuditLog.riskLevel.MEDIUM'),
  },
]);
