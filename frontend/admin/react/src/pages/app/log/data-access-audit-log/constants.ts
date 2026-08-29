/**
 * 数据访问审计日志模块枚举映射常量
 */

type TFn = (key: string, options?: Record<string, any>) => string;

// ========== 访问类型 ==========

/** 访问类型枚举值列表（与后端 proto DataAccessAuditLog.AccessType 一致） */
export const ACCESS_TYPE_VALUES = [
  'SELECT',
  'INSERT',
  'UPDATE',
  'DELETE',
  'VIEW',
  'BULK_READ',
  'EXPORT',
  'IMPORT',
  'DDL_CREATE',
  'DDL_ALTER',
  'DDL_DROP',
  'METADATA_READ',
  'SCAN',
  'ADMIN_OPERATION',
  'OTHER',
] as const;

/** 访问类型颜色映射 */
export const ACCESS_TYPE_COLORS: Record<string, string> = {
  SELECT: 'processing',
  VIEW: 'processing',
  METADATA_READ: 'processing',
  SCAN: 'cyan',
  BULK_READ: 'cyan',
  INSERT: 'success',
  IMPORT: 'success',
  EXPORT: 'geekblue',
  UPDATE: 'warning',
  DDL_ALTER: 'warning',
  DELETE: 'error',
  DDL_DROP: 'error',
  DDL_CREATE: 'purple',
  ADMIN_OPERATION: 'purple',
  OTHER: 'default',
};

/** 获取访问类型映射（text + color），供 render 使用 */
export function getAccessTypeMap(t: TFn) {
  const map: Record<string, { text: string; color: string }> = {};
  for (const value of ACCESS_TYPE_VALUES) {
    map[value] = { text: t(`accessType.${value}`), color: ACCESS_TYPE_COLORS[value] };
  }
  return map;
}

/** 获取访问类型选项列表，供搜索/Select 使用 */
export function getAccessTypeOptions(t: TFn) {
  return ACCESS_TYPE_VALUES.map((value) => ({
    label: t(`accessType.${value}`),
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

// ========== 数据分类（码值与后端 pkg/audit.ClassifyTable 一致） ==========

/** 数据分类颜色映射 */
export const DATA_CATEGORY_COLORS: Record<string, string> = {
  USER_DATA: 'processing',
  ORG_DATA: 'cyan',
  ACCESS_CONTROL: 'purple',
  TENANT_DATA: 'geekblue',
  MESSAGE_DATA: 'success',
  AUDIT_LOG: 'warning',
  SYSTEM_CONFIG: 'default',
  UNKNOWN: 'default',
};

/** 获取数据分类映射（text + color），未收录码值回退显示原始码值 */
export function getDataCategoryMap(t: TFn) {
  return {
    USER_DATA: { text: t('dataCategory.USER_DATA'), color: DATA_CATEGORY_COLORS.USER_DATA },
    ORG_DATA: { text: t('dataCategory.ORG_DATA'), color: DATA_CATEGORY_COLORS.ORG_DATA },
    ACCESS_CONTROL: {
      text: t('dataCategory.ACCESS_CONTROL'),
      color: DATA_CATEGORY_COLORS.ACCESS_CONTROL,
    },
    TENANT_DATA: { text: t('dataCategory.TENANT_DATA'), color: DATA_CATEGORY_COLORS.TENANT_DATA },
    MESSAGE_DATA: {
      text: t('dataCategory.MESSAGE_DATA'),
      color: DATA_CATEGORY_COLORS.MESSAGE_DATA,
    },
    AUDIT_LOG: { text: t('dataCategory.AUDIT_LOG'), color: DATA_CATEGORY_COLORS.AUDIT_LOG },
    SYSTEM_CONFIG: {
      text: t('dataCategory.SYSTEM_CONFIG'),
      color: DATA_CATEGORY_COLORS.SYSTEM_CONFIG,
    },
    UNKNOWN: { text: t('dataCategory.UNKNOWN'), color: DATA_CATEGORY_COLORS.UNKNOWN },
  };
}

/** 根据成功状态获取颜色 */
export function successToColor(success: boolean | undefined): string {
  if (success === true) return 'success';
  if (success === false) return 'error';
  return 'default';
}

/** 根据成功状态获取显示文本 */
export function successToName(t: TFn, success: boolean | undefined): string {
  if (success === true) return t('success.true');
  if (success === false) return t('success.false');
  return '-';
}
