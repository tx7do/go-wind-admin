/**
 * 登录审计日志模块枚举映射常量
 */

type TFn = (key: string, options?: Record<string, any>) => string;

// ========== 状态 ==========

/** 状态颜色映射 */
export const STATUS_COLORS: Record<string, string> = {
  SUCCESS: 'success',
  FAILED: 'error',
};

/** 获取状态映射（text + color） */
export function getStatusMap(t: TFn) {
  return {
    SUCCESS: { text: t('status.SUCCESS'), color: STATUS_COLORS.SUCCESS },
    FAILED: { text: t('status.FAILED'), color: STATUS_COLORS.FAILED },
  };
}

/** 获取状态选项列表 */
export function getStatusOptions(t: TFn) {
  return [
    { label: t('status.SUCCESS'), value: 'SUCCESS' },
    { label: t('status.FAILED'), value: 'FAILED' },
  ];
}

// ========== 操作类型 ==========

/** 操作类型颜色映射 */
export const ACTION_TYPE_COLORS: Record<string, string> = {
  LOGIN: 'success',
  LOGOUT: 'default',
  LOGIN_FAILED: 'error',
};

/** 获取操作类型映射（text + color） */
export function getActionTypeMap(t: TFn) {
  return {
    LOGIN: { text: t('actionType.LOGIN'), color: ACTION_TYPE_COLORS.LOGIN },
    LOGOUT: { text: t('actionType.LOGOUT'), color: ACTION_TYPE_COLORS.LOGOUT },
    LOGIN_FAILED: { text: t('actionType.LOGIN_FAILED'), color: ACTION_TYPE_COLORS.LOGIN_FAILED },
  };
}

/** 获取操作类型选项列表 */
export function getActionTypeOptions(t: TFn) {
  return [
    { label: t('actionType.LOGIN'), value: 'LOGIN' },
    { label: t('actionType.LOGOUT'), value: 'LOGOUT' },
    { label: t('actionType.LOGIN_FAILED'), value: 'LOGIN_FAILED' },
  ];
}

// ========== 风险等级 ==========

/** 风险等级颜色映射 */
export const RISK_LEVEL_COLORS: Record<string, string> = {
  LOW: 'success',
  MEDIUM: 'warning',
  HIGH: 'error',
  CRITICAL: 'error',
};

/** 获取风险等级映射（text + color） */
export function getRiskLevelMap(t: TFn) {
  return {
    LOW: { text: t('riskLevel.LOW'), color: RISK_LEVEL_COLORS.LOW },
    MEDIUM: { text: t('riskLevel.MEDIUM'), color: RISK_LEVEL_COLORS.MEDIUM },
    HIGH: { text: t('riskLevel.HIGH'), color: RISK_LEVEL_COLORS.HIGH },
    CRITICAL: { text: t('riskLevel.CRITICAL'), color: RISK_LEVEL_COLORS.CRITICAL },
  };
}

/** 获取风险等级选项列表 */
export function getRiskLevelOptions(t: TFn) {
  return [
    { label: t('riskLevel.LOW'), value: 'LOW' },
    { label: t('riskLevel.MEDIUM'), value: 'MEDIUM' },
    { label: t('riskLevel.HIGH'), value: 'HIGH' },
    { label: t('riskLevel.CRITICAL'), value: 'CRITICAL' },
  ];
}
