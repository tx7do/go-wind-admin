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
  READ: 'processing',
  ASSIGN: 'processing',
  UNASSIGN: 'default',
  EXPORT: 'cyan',
  IMPORT: 'geekblue',
  OTHER: 'default',
};

/** 操作类型枚举值列表 */
export const ACTION_VALUES = [
  'CREATE',
  'UPDATE',
  'DELETE',
  'READ',
  'ASSIGN',
  'UNASSIGN',
  'EXPORT',
  'IMPORT',
  'OTHER',
] as const;

/** 获取操作类型映射（text + color），供 render 使用 */
export function getActionMap(t: TFn) {
  const map: Record<string, { text: string; color: string }> = {};
  for (const value of ACTION_VALUES) {
    map[value] = { text: t(`action.${value}`), color: ACTION_COLORS[value] };
  }
  return map;
}

/** 获取操作类型选项列表，供搜索/Select 使用 */
export function getActionOptions(t: TFn) {
  return ACTION_VALUES.map((value) => ({
    label: t(`action.${value}`),
    value,
  }));
}

// ========== 成功状态 ==========

/** 成功状态列表 */
export function getSuccessStatusList(t: TFn) {
  return [
    { label: t('success.true'), value: 'true' },
    { label: t('success.false'), value: 'false' },
  ];
}

/** 根据成功状态获取颜色 */
export function successToColor(success: boolean | undefined): string {
  if (success === true) return 'success';
  if (success === false) return 'error';
  return 'default';
}

/** 根据成功状态和状态码获取显示文本 */
export function successToName(t: TFn, success: boolean | undefined, statusCode?: number): string {
  if (success === true) return `${t('success.true')}${statusCode ? ` (${statusCode})` : ''}`;
  if (success === false) return `${t('success.false')}${statusCode ? ` (${statusCode})` : ''}`;
  return '-';
}
