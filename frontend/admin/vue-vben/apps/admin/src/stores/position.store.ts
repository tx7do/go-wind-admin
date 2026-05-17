import { computed } from 'vue';

import { $t } from '@vben/locales';
import { useUserStore } from '@vben/stores';

import { defineStore } from 'pinia';

import {
  createPositionServiceClient,
  type identityservicev1_Position_Status as Position_Status,
  type identityservicev1_Position_Type as Position_Type,
} from '#/generated/api/admin/service/v1';
import { makeOrderBy, makeQueryString, makeUpdateMask } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const usePositionStore = defineStore('position', () => {
  const service = createPositionServiceClient(requestClientRequestHandler);
  const userStore = useUserStore();

  /**
   * 查询职位列表
   */
  async function listPosition(
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
   * 获取职位
   */
  async function getPosition(id: number) {
    return await service.Get({ id });
  }

  /**
   * 创建职位
   */
  async function createPosition(values: Record<string, any> = {}) {
    return await service.Create({
      data: {
        ...values,
      },
    });
  }

  /**
   * 更新职位
   */
  async function updatePosition(id: number, values: Record<string, any> = {}) {
    return await service.Update({
      id,
      data: {
        ...values,
      },
      // @ts-ignore proto generated code is error.
      updateMask: makeUpdateMask(Object.keys(values ?? [])),
    });
  }

  /**
   * 删除职位
   */
  async function deletePosition(id: number) {
    return await service.Delete({ id });
  }

  function $reset() {}

  return {
    $reset,
    listPosition,
    getPosition,
    createPosition,
    updatePosition,
    deletePosition,
  };
});

export const membershipPositionStatusList = computed(() => [
  { value: 'PROBATION', label: $t('enum.membershipPosition.status.PROBATION') },
  {
    value: 'ACTIVE',
    label: $t('enum.membershipPosition.status.ACTIVE'),
  },
  {
    value: 'LEAVE',
    label: $t('enum.membershipPosition.status.LEAVE'),
  },
  {
    value: 'RESIGNED',
    label: $t('enum.membershipPosition.status.RESIGNED'),
  },
  {
    value: 'TERMINATED',
    label: $t('enum.membershipPosition.status.TERMINATED'),
  },
  {
    value: 'EXPIRED',
    label: $t('enum.membershipPosition.status.EXPIRED'),
  },
]);

/**
 * 状态转名称
 * @param status 状态值
 */
export function membershipPositionStatusToName(status: any) {
  const values = membershipPositionStatusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : '';
}

// 职位状态-颜色映射常量
const MEMBERSHIP_POSITION_STATUS_COLOR_MAP = {
  PROBATION: '#4096FF', // 试用期：浅蓝（过渡状态，正向但未完全激活）
  ACTIVE: '#00B42A', // 在职/激活：企业绿（核心正向状态，醒目且无视觉冲击）
  LEAVE: '#FF9A2E', // 休假/离岗（临时）：暖橙黄（临时状态，非激活但非负面）
  RESIGNED: '#F56C6C', // 已辞职（主动离职）：浅红（负面但非严重警示）
  TERMINATED: '#F53F3F', // 解除合同/开除（被动终止）：深红（强警示，严重负面）
  EXPIRED: '#909399', // 合同到期：中灰（中性提醒，无明确正负向）
  DEFAULT: '#C9CDD4', // 未知状态：浅灰（中性兜底，无倾向）
} as const;

/**
 * 职位状态映射对应颜色
 * @param status 职位状态（INACTIVE/ACTIVE/ON_LEAVE）
 * @returns 标准化十六进制颜色值
 */
export function membershipPositionStatusToColor(status: Position_Status) {
  // 优先匹配状态，无匹配则返回默认色
  return (
    MEMBERSHIP_POSITION_STATUS_COLOR_MAP[
      status as keyof typeof MEMBERSHIP_POSITION_STATUS_COLOR_MAP
    ] || MEMBERSHIP_POSITION_STATUS_COLOR_MAP.DEFAULT
  );
}

export const positionTypeList = computed(() => [
  { value: 'REGULAR', label: $t('enum.position.type.REGULAR') },
  {
    value: 'LEADER',
    label: $t('enum.position.type.LEADER'),
  },
  {
    value: 'MANAGER',
    label: $t('enum.position.type.MANAGER'),
  },
  {
    value: 'INTERN',
    label: $t('enum.position.type.INTERN'),
  },
  {
    value: 'CONTRACT',
    label: $t('enum.position.type.CONTRACT'),
  },
  {
    value: 'OTHER',
    label: $t('enum.position.type.OTHER'),
  },
]);

export function positionTypeToName(status: Position_Status) {
  const values = positionTypeList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : '';
}

// 多主题职位类型颜色映射
const POSITION_TYPE_COLOR_THEME = {
  light: {
    REGULAR: '#165DFF',
    LEADER: '#722ED1',
    MANAGER: '#FF7D00',
    INTERN: '#52C41A',
    CONTRACT: '#14C9C9',
    OTHER: '#86909C',
    DEFAULT: '#C9CDD4',
  },
  dark: {
    REGULAR: '#2F77FF', // 深色模式下蓝色更亮
    LEADER: '#8542E7', // 深色模式下紫色更柔和
    MANAGER: '#FF9529', // 深色模式下橙色更暖
    INTERN: '#67E037', // 深色模式下绿色更清新
    CONTRACT: '#20E0E0', // 深色模式下天蓝色更亮
    OTHER: '#9BA3AD', // 深色模式下灰色更浅
    DEFAULT: '#DCE0E6', // 深色模式下默认浅灰更柔和
  },
} as const;

/**
 * 支持主题的职位类型颜色映射
 * @param positionType 职位类型
 * @param theme 主题模式（light/dark），默认浅色
 * @returns 对应主题的十六进制颜色值
 */
export function positionTypeToColor(
  positionType: Position_Type,
  theme: 'dark' | 'light' = 'light',
): string {
  const colorMap = POSITION_TYPE_COLOR_THEME[theme];
  return colorMap[positionType as keyof typeof colorMap] || colorMap.DEFAULT;
}
