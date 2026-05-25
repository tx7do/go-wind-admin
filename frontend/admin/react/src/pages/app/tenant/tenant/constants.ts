/**
 * 租户模块枚举映射常量
 * 页面和 Drawer 共用
 */

type TFn = (key: string, options?: Record<string, any>) => string;

// ========== 租户类型 ==========

/** 租户类型颜色映射 */
export const TENANT_TYPE_COLORS: Record<string, string> = {
  TRIAL: 'default',
  PAID: 'success',
  INTERNAL: 'processing',
  PARTNER: 'warning',
  CUSTOM: 'default',
};

/** 租户类型 ProTable valueEnum 用的 status 映射 */
export const TENANT_TYPE_STATUS: Record<string, string> = {
  TRIAL: 'Default',
  PAID: 'Success',
  INTERNAL: 'Processing',
  PARTNER: 'Warning',
  CUSTOM: 'Default',
};

/** 获取租户类型映射（text + color），供 render 使用 */
export function getTenantTypeMap(t: TFn) {
  return {
    TRIAL: { text: t('type.TRIAL'), color: TENANT_TYPE_COLORS.TRIAL },
    PAID: { text: t('type.PAID'), color: TENANT_TYPE_COLORS.PAID },
    INTERNAL: { text: t('type.INTERNAL'), color: TENANT_TYPE_COLORS.INTERNAL },
    PARTNER: { text: t('type.PARTNER'), color: TENANT_TYPE_COLORS.PARTNER },
    CUSTOM: { text: t('type.CUSTOM'), color: TENANT_TYPE_COLORS.CUSTOM },
  };
}

/** 获取租户类型选项列表，供 ProFormSelect 使用 */
export function getTenantTypeOptions(t: TFn) {
  return [
    { label: t('type.TRIAL'), value: 'TRIAL' },
    { label: t('type.PAID'), value: 'PAID' },
    { label: t('type.INTERNAL'), value: 'INTERNAL' },
    { label: t('type.PARTNER'), value: 'PARTNER' },
    { label: t('type.CUSTOM'), value: 'CUSTOM' },
  ];
}

// ========== 审核状态 ==========

/** 审核状态颜色映射 */
export const AUDIT_STATUS_COLORS: Record<string, string> = {
  PENDING: 'warning',
  APPROVED: 'success',
  REJECTED: 'error',
};

/** 审核状态 ProTable valueEnum 用的 status 映射 */
export const AUDIT_STATUS_STATUS: Record<string, string> = {
  PENDING: 'Warning',
  APPROVED: 'Success',
  REJECTED: 'Error',
};

/** 获取审核状态映射（text + color），供 render 使用 */
export function getAuditStatusMap(t: TFn) {
  return {
    PENDING: { text: t('audit.PENDING'), color: AUDIT_STATUS_COLORS.PENDING },
    APPROVED: { text: t('audit.APPROVED'), color: AUDIT_STATUS_COLORS.APPROVED },
    REJECTED: { text: t('audit.REJECTED'), color: AUDIT_STATUS_COLORS.REJECTED },
  };
}

/** 获取审核状态选项列表，供 ProFormSelect 使用 */
export function getAuditStatusOptions(t: TFn) {
  return [
    { label: t('audit.PENDING'), value: 'PENDING' },
    { label: t('audit.APPROVED'), value: 'APPROVED' },
    { label: t('audit.REJECTED'), value: 'REJECTED' },
  ];
}

// ========== 租户状态 ==========

/** 租户状态颜色映射 */
export const TENANT_STATUS_COLORS: Record<string, string> = {
  ON: 'success',
  OFF: 'error',
  EXPIRED: 'warning',
  FREEZE: 'default',
};

/** 租户状态 ProTable valueEnum 用的 status 映射 */
export const TENANT_STATUS_STATUS: Record<string, string> = {
  ON: 'Success',
  OFF: 'Error',
  EXPIRED: 'Warning',
  FREEZE: 'Default',
};

/** 获取租户状态映射（text + color），供 render 使用 */
export function getTenantStatusMap(t: TFn) {
  return {
    ON: { text: t('tenantStatus.ON'), color: TENANT_STATUS_COLORS.ON },
    OFF: { text: t('tenantStatus.OFF'), color: TENANT_STATUS_COLORS.OFF },
    EXPIRED: { text: t('tenantStatus.EXPIRED'), color: TENANT_STATUS_COLORS.EXPIRED },
    FREEZE: { text: t('tenantStatus.FREEZE'), color: TENANT_STATUS_COLORS.FREEZE },
  };
}

/** 获取租户状态选项列表，供 ProFormSelect 使用 */
export function getTenantStatusOptions(t: TFn) {
  return [
    { label: t('tenantStatus.ON'), value: 'ON' },
    { label: t('tenantStatus.OFF'), value: 'OFF' },
    { label: t('tenantStatus.EXPIRED'), value: 'EXPIRED' },
    { label: t('tenantStatus.FREEZE'), value: 'FREEZE' },
  ];
}

// ========== 通用 Select fieldProps ==========

/** 通用 Select 筛选配置 */
export const SELECT_FILTER_PROPS = {
  showSearch: true,
  allowClear: true,
  filterOption: (input: string, option: any) =>
    (option?.label ?? '').toLowerCase().includes(input.toLowerCase()),
};
