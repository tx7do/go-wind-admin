import { computed } from 'vue';

import { $t } from '@vben/locales';

import { defineStore } from 'pinia';

import {
  createOrgUnitServiceClient,
  type userservicev1_OrgUnit as OrgUnit,
  type userservicev1_OrgUnit_Status as OrgUnit_Status,
  type userservicev1_OrgUnit_Type as OrgUnit_Type,
} from '#/generated/api/admin/service/v1';
import { makeQueryString, makeUpdateMask } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const useOrgUnitStore = defineStore('org-unit', () => {
  const service = createOrgUnitServiceClient(requestClientRequestHandler);

  /**
   * 查询组织单位列表
   */
  async function listOrgUnit(
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
      orderBy: orderBy ?? [],
      query: makeQueryString(formValues ?? null),
      page: paging?.page,
      pageSize: paging?.pageSize,
      noPaging,
    });
  }

  /**
   * 获取组织单位
   */
  async function getOrgUnit(id: number) {
    return await service.Get({ id });
  }

  /**
   * 创建组织单位
   */
  async function createOrgUnit(values: object) {
    return await service.Create({
      // @ts-ignore proto generated code is error.
      data: {
        ...values,
        children: [],
      },
    });
  }

  /**
   * 更新组织单位
   */
  async function updateOrgUnit(id: number, values: object) {
    return await service.Update({
      id,
      // @ts-ignore proto generated code is error.
      data: {
        ...values,
        children: [],
      },
      // @ts-ignore proto generated code is error.
      updateMask: makeUpdateMask(Object.keys(values ?? [])),
    });
  }

  /**
   * 删除组织单位
   */
  async function deleteOrgUnit(id: number) {
    return await service.Delete({ id });
  }

  function $reset() {}

  return {
    $reset,
    listOrgUnit,
    getOrgUnit,
    createOrgUnit,
    updateOrgUnit,
    deleteOrgUnit,
  };
});

export const orgUnitStatusList = computed(() => [
  {
    value: 'ON',
    label: $t('enum.status.ON'),
  },
  {
    value: 'OFF',
    label: $t('enum.status.OFF'),
  },
]);

/**
 * 状态转名称
 * @param status 状态值
 */
export function orgUnitStatusToName(status: OrgUnit_Status) {
  const values = orgUnitStatusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : '';
}

/**
 * 状态转颜色值
 * @param status 状态值
 */
export function orgUnitStatusToColor(status: OrgUnit_Status) {
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

const ORG_UNIT_TYPE_ENUM = {
  COMPANY: 'COMPANY',
  DIVISION: 'DIVISION',
  DEPARTMENT: 'DEPARTMENT',
  TEAM: 'TEAM',
  PROJECT: 'PROJECT',
  COMMITTEE: 'COMMITTEE',
  REGION: 'REGION',
  SUBSIDIARY: 'SUBSIDIARY',
  BRANCH: 'BRANCH',
  OTHER: 'OTHER',
} as const;

export const orgUnitTypeList = computed(() => {
  // 按业务优先级排序的类型数组（调整顺序仅需改此数组）
  const typeOrder: OrgUnit_Type[] = [
    ORG_UNIT_TYPE_ENUM.COMPANY,
    ORG_UNIT_TYPE_ENUM.DIVISION,
    ORG_UNIT_TYPE_ENUM.DEPARTMENT,
    ORG_UNIT_TYPE_ENUM.TEAM,
    ORG_UNIT_TYPE_ENUM.PROJECT,
    ORG_UNIT_TYPE_ENUM.COMMITTEE,
    ORG_UNIT_TYPE_ENUM.REGION,
    ORG_UNIT_TYPE_ENUM.SUBSIDIARY,
    ORG_UNIT_TYPE_ENUM.BRANCH,
    ORG_UNIT_TYPE_ENUM.OTHER,
  ];

  return typeOrder.map((type) => ({
    value: type,
    label: $t(`enum.orgUnit.type.${type}`),
  }));
});

export const orgUnitTypeListForQuery = computed(() => {
  // 如需筛选，仅需修改此数组（默认包含全部类型，与原逻辑一致）
  const queryAllowTypes: OrgUnit_Type[] = [
    ORG_UNIT_TYPE_ENUM.BRANCH,
    ORG_UNIT_TYPE_ENUM.COMMITTEE,
    ORG_UNIT_TYPE_ENUM.COMPANY,
    ORG_UNIT_TYPE_ENUM.DEPARTMENT,
    ORG_UNIT_TYPE_ENUM.DIVISION,
    ORG_UNIT_TYPE_ENUM.OTHER,
    ORG_UNIT_TYPE_ENUM.PROJECT,
    ORG_UNIT_TYPE_ENUM.REGION,
    ORG_UNIT_TYPE_ENUM.SUBSIDIARY,
    ORG_UNIT_TYPE_ENUM.TEAM,
  ];

  // 转换为Set提升查找性能（大数据量更优）
  const allowTypeSet = new Set(queryAllowTypes);
  return orgUnitTypeList.value.filter((item) => allowTypeSet.has(item.value));
});

/**
 * 组织单位类型转名称
 * @param orgUnitType
 */
export function orgUnitTypeToName(orgUnitType: OrgUnit_Type) {
  const values = orgUnitTypeList.value;
  const matchedItem = values.find((item) => item.value === orgUnitType);
  return matchedItem ? matchedItem.label : '';
}

// 组织类型-颜色映射常量
const ORG_UNIT_COLOR_MAP = {
  BRANCH: '#4096FF', // 分公司
  COMMITTEE: '#00B42A', // 委员会
  COMPANY: '#165DFF', // 集团
  DEPARTMENT: '#722ED1', // 部门
  DIVISION: '#FF7D00', // 事业部
  OTHER: '#86909C', // 其他
  PROJECT: '#F53F3F', // 项目组
  REGION: '#14C9C9', // 区域中心
  SUBSIDIARY: '#6B778C', // 子公司
  TEAM: '#FFC53D', // 团队
  DEFAULT: '#C9CDD4', // 未知类型
} as const;

/**
 * 组织单位类型转颜色值
 * @param orgUnitType
 */
export function orgUnitTypeToColor(orgUnitType: OrgUnit_Type) {
  return (
    ORG_UNIT_COLOR_MAP[orgUnitType as keyof typeof ORG_UNIT_COLOR_MAP] ||
    ORG_UNIT_COLOR_MAP.DEFAULT
  );
}

export const findOrgUnit = (
  list: OrgUnit[],
  id: number,
): null | OrgUnit | undefined => {
  for (const item of list) {
    // eslint-disable-next-line eqeqeq
    if (item.id == id) {
      return item;
    }

    if (item.children && item.children.length > 0) {
      const found = findOrgUnit(item.children, id);
      if (found) return found;
    }
  }

  return null;
};
