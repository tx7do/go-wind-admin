/**
 * 操作审计日志模块枚举映射常量
 */

type TFn = (key: string, options?: Record<string, any>) => string;

// ========== 操作类型 ==========

/** 操作类型颜色映射 */
export const ACTION_COLORS: Record<string, string> = {
  CREATE: 'success',
  UPDATE: 'warning',
  DELETE: 'error',
};

/** 获取操作类型映射（text + color），供 render 使用 */
export function getActionMap(t: TFn) {
  return {
    CREATE: { text: t('action.CREATE'), color: ACTION_COLORS.CREATE },
    UPDATE: { text: t('action.UPDATE'), color: ACTION_COLORS.UPDATE },
    DELETE: { text: t('action.DELETE'), color: ACTION_COLORS.DELETE },
  };
}

/** 获取操作类型选项列表，供搜索/Select 使用 */
export function getActionOptions(t: TFn) {
  return [
    { label: t('action.CREATE'), value: 'CREATE' },
    { label: t('action.UPDATE'), value: 'UPDATE' },
    { label: t('action.DELETE'), value: 'DELETE' },
  ];
}

// ========== 成功状态 ==========

/** 成功状态列表 */
export const SUCCESS_STATUS_LIST = [
  { label: '成功', value: 'true' },
  { label: '失败', value: 'false' },
];

/** 根据成功状态获取颜色 */
export function successToColor(success: boolean | undefined): string {
  if (success === true) return 'success';
  if (success === false) return 'error';
  return 'default';
}

/** 根据成功状态和状态码获取显示文本 */
export function successToName(success: boolean | undefined, statusCode?: number): string {
  if (success === true) return `成功${statusCode ? ` (${statusCode})` : ''}`;
  if (success === false) return `失败${statusCode ? ` (${statusCode})` : ''}`;
  return '-';
}
