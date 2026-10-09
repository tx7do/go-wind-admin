/**
 * 通用枚举与工具函数
 * 从 stores/modules/api 迁移而来
 */
import { computed } from 'vue';

import { $t } from '@vben/locales';

export const enableList = computed(() => [
  { value: 'true', label: $t('enum.enable.true') },
  { value: 'false', label: $t('enum.enable.false') },
]);

export const enableBoolList = computed(() => [
  { value: true, label: $t('enum.enable.true') },
  { value: false, label: $t('enum.enable.false') },
]);

export const successStatusList = computed(() => [
  { value: true, label: $t('enum.successStatus.success') },
  { value: false, label: $t('enum.successStatus.failed') },
]);

// 颜色统一使用 antd 预设状态名，由主题 token 驱动，亮/暗模式自动切换，
// 避免硬编码 hex 在暗黑模式下产生刺眼浅色块标签。
// 语义：启用=success，禁用=default
export function enableBoolToColor(
  enable: 'false' | 'FALSE' | 'False' | 'true' | 'TRUE' | 'True' | boolean,
): 'default' | 'success' {
  switch (enable) {
    case true:
    case 'true':
    case 'TRUE':
    case 'True': {
      return 'success';
    }
    default: {
      return 'default';
    }
  }
}

export function enableBoolToName(
  enable: 'false' | 'FALSE' | 'False' | 'true' | 'TRUE' | 'True' | boolean,
) {
  switch (enable) {
    case true:
    case 'true':
    case 'TRUE':
    case 'True': {
      return $t('enum.enable.true');
    }
    default: {
      return $t('enum.enable.false');
    }
  }
}

export const methodList = [
  { value: 'GET', label: 'GET' },
  { value: 'POST', label: 'POST' },
  { value: 'PUT', label: 'PUT' },
  { value: 'PATCH', label: 'PATCH' },
  { value: 'DELETE', label: 'DELETE' },
];

// HTTP 方法对应的 Tag 颜色类型
export const httpMethodTagTypeMap: Record<
  string,
  'danger' | 'info' | 'primary' | 'success' | 'warning'
> = {
  GET: 'success',
  POST: 'primary',
  PUT: 'warning',
  DELETE: 'danger',
  PATCH: 'info',
  HEAD: 'info',
  OPTIONS: 'info',
};

export const statusList = computed(() => [
  { value: 'ON', label: $t('enum.status.ON') },
  { value: 'OFF', label: $t('enum.status.OFF') },
]);

export function statusToName(status: 'OFF' | 'ON' | undefined) {
  const values = statusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : '';
}

// 语义：ON=success，OFF/default=default
export function statusToColor(
  status: 'OFF' | 'ON' | undefined,
): 'default' | 'success' {
  switch (status) {
    case 'ON': {
      return 'success';
    }
    default: {
      return 'default';
    }
  }
}

// 成功/失败状态
// ==============================

// 语义：成功=success，失败=error（均由 antd 主题 token 驱动，暗黑自动切换）
export function successToColor(success: boolean): 'success' | 'error' {
  return success ? 'success' : 'error';
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
