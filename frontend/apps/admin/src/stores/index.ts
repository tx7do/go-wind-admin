import { computed } from 'vue';

import { $t } from '@vben/locales';

export * from './admin-portal.state';
export * from './api.state';
export * from './api-audit-log';
export * from './authentication.state';
export * from './data-access-audit-log.state';
export * from './dict.state';
export * from './file.state';
export * from './file-transfer.state';
export * from './internal-message.state';
export * from './internal-message-category.state';
export * from './language.state';
export * from './login-audit-log';
export * from './login-policy.state';
export * from './menu.state';
export * from './operation-audit-log.state';
export * from './org-unit.state';
export * from './permission.state';
export * from './permission-audit-log.state';
export * from './permission-group.state';
export * from './policy-evaluation-log.state';
export * from './position.state';
export * from './role.state';
export * from './task.state';
export * from './tenant.state';
export * from './user.state';
export * from './user-profile.state';

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

/**
 * 状态转颜色值
 * @param enable 状态值
 */
export function enableBoolToColor(
  enable: 'false' | 'FALSE' | 'False' | 'true' | 'TRUE' | 'True' | boolean,
) {
  switch (enable) {
    case false:
    case 'false':
    case 'FALSE':
    case 'False': {
      // 关闭/停用：深灰色，明确非激活状态
      return '#8C8C8C';
    } // 中深灰色，与“关闭”语义匹配，区别于浅灰的“未知”
    case true:
    case 'true':
    case 'TRUE':
    case 'True': {
      // 开启/激活：标准成功绿，体现正常运行
      return '#52C41A';
    } // 对应Element Plus的success色，大众认知中的“正常”色
    default: {
      // 异常状态：浅灰色，代表未定义状态
      return '#C9CDD4';
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

export const statusList = computed(() => [
  { value: 'ON', label: $t('enum.status.ON') },
  { value: 'OFF', label: $t('enum.status.OFF') },
]);

/**
 * 状态转名称
 * @param status 状态值
 */
export function statusToName(status: 'OFF' | 'ON' | undefined) {
  const values = statusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : '';
}

/**
 * 状态转颜色值
 * @param status 状态值
 */
export function statusToColor(status: 'OFF' | 'ON' | undefined) {
  switch (status) {
    case 'OFF': {
      // 关闭/停用：深灰色，明确非激活状态
      return '#8C8C8C';
    } // 中深灰色，与“关闭”语义匹配，区别于浅灰的“未知”
    case 'ON': {
      // 开启/激活：标准成功绿，体现正常运行
      return '#52C41A';
    } // 对应Element Plus的success色，大众认知中的“正常”色
    default: {
      // 异常状态：浅灰色，代表未定义状态
      return '#C9CDD4';
    }
  }
}
