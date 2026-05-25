/**
 * 权限审计日志模块枚举映射常量
 */

type TFn = (key: string, options?: Record<string, any>) => string;

// ========== 操作类型 ==========

/** 操作类型颜色映射 */
export const ACTION_COLORS: Record<string, string> = {
  GRANT: 'success',
  REVOKE: 'error',
};

/** 获取操作类型映射（text + color），供 render 使用 */
export function getActionMap(t: TFn) {
  return {
    GRANT: { text: t('action.GRANT'), color: ACTION_COLORS.GRANT },
    REVOKE: { text: t('action.REVOKE'), color: ACTION_COLORS.REVOKE },
  };
}

/** 获取操作类型选项列表，供搜索/Select 使用 */
export function getActionOptions(t: TFn) {
  return [
    { label: t('action.GRANT'), value: 'GRANT' },
    { label: t('action.REVOKE'), value: 'REVOKE' },
  ];
}
