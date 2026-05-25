/**
 * 数据访问审计日志模块枚举映射常量
 */

type TFn = (key: string, options?: Record<string, any>) => string;

// ========== 访问类型 ==========

/** 访问类型颜色映射 */
export const ACCESS_TYPE_COLORS: Record<string, string> = {
  READ: 'processing',
  CREATE: 'success',
  UPDATE: 'warning',
  DELETE: 'error',
};

/** 获取访问类型映射（text + color），供 render 使用 */
export function getAccessTypeMap(t: TFn) {
  return {
    READ: { text: t('accessType.READ'), color: ACCESS_TYPE_COLORS.READ },
    CREATE: { text: t('accessType.CREATE'), color: ACCESS_TYPE_COLORS.CREATE },
    UPDATE: { text: t('accessType.UPDATE'), color: ACCESS_TYPE_COLORS.UPDATE },
    DELETE: { text: t('accessType.DELETE'), color: ACCESS_TYPE_COLORS.DELETE },
  };
}

/** 获取访问类型选项列表，供搜索/Select 使用 */
export function getAccessTypeOptions(t: TFn) {
  return [
    { label: t('accessType.READ'), value: 'READ' },
    { label: t('accessType.CREATE'), value: 'CREATE' },
    { label: t('accessType.UPDATE'), value: 'UPDATE' },
    { label: t('accessType.DELETE'), value: 'DELETE' },
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

/** 根据成功状态获取显示文本 */
export function successToName(success: boolean | undefined): string {
  if (success === true) return '成功';
  if (success === false) return '失败';
  return '-';
}
