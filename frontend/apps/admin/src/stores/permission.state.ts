import { computed } from 'vue';

import { $t } from '@vben/locales';

import { defineStore } from 'pinia';

import {
  createPermissionServiceClient,
  type permissionservicev1_Permission as Permission,
} from '#/generated/api/admin/service/v1';
import { makeQueryString, makeUpdateMask } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const usePermissionStore = defineStore('permission', () => {
  const service = createPermissionServiceClient(requestClientRequestHandler);

  /**
   * 查询权限列表
   */
  async function listPermission(
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
   * 获取权限
   */
  async function getPermission(id: number) {
    return await service.Get({ id });
  }

  /**
   * 创建权限
   */
  async function createPermission(values: object) {
    return await service.Create({
      // @ts-ignore proto generated code is error.
      data: {
        ...values,
      },
    });
  }

  /**
   * 更新权限
   */
  async function updatePermission(id: number, values: object) {
    return await service.Update({
      id,
      // @ts-ignore proto generated code is error.
      data: {
        ...values,
      },
      // @ts-ignore proto generated code is error.
      updateMask: makeUpdateMask(Object.keys(values ?? [])),
    });
  }

  /**
   * 删除权限
   */
  async function deletePermission(id: number) {
    return await service.Delete({ id });
  }

  async function syncApis() {
    return await service.SyncApis({});
  }

  async function syncMenus() {
    return await service.SyncMenus({});
  }

  function $reset() {}

  return {
    $reset,
    listPermission,
    getPermission,
    createPermission,
    updatePermission,
    deletePermission,
    syncApis,
    syncMenus,
  };
});

export const roleDataScopeList = computed(() => [
  { label: $t('enum.role.dataScope.ALL'), value: 'ALL' },
  { label: $t('enum.role.dataScope.UNIT_AND_CHILD'), value: 'UNIT_AND_CHILD' },
  { label: $t('enum.role.dataScope.UNIT_ONLY'), value: 'UNIT_ONLY' },
  { label: $t('enum.role.dataScope.SELECTED_UNITS'), value: 'SELECTED_UNITS' },
  { label: $t('enum.role.dataScope.SELF'), value: 'SELF' },
]);

// 数据范围-颜色映射常量（按权限范围从大到小匹配差异化色值）
const DATA_SCOPE_COLOR_MAP = {
  ALL: '#F53F3F', // 全部数据：红色（最大权限、高危、核心管控）
  UNIT_AND_CHILD: '#165DFF', // 本单位及子单位：深蓝色（大范围、层级化权限）
  UNIT_ONLY: '#FF7D00', // 仅本单位：橙色（中等范围、核心业务权限）
  SELECTED_UNITS: '#722ED1', // 指定单位：紫色（灵活范围、自定义权限）
  SELF: '#86909C', // 仅自己：中灰色（最小范围、基础权限）
  DEFAULT: '#C9CDD4', // 未知数据范围：浅灰色（中性、无倾向）
} as const;

/**
 * 数据范围映射对应颜色
 * @param dataScope 数据范围（ALL/SELF/UNIT_ONLY/UNIT_AND_CHILD/SELECTED_UNITS）
 * @returns 标准化十六进制颜色值
 */
export function dataScopeToColor(dataScope: any): string {
  return (
    DATA_SCOPE_COLOR_MAP[dataScope as keyof typeof DATA_SCOPE_COLOR_MAP] ||
    DATA_SCOPE_COLOR_MAP.DEFAULT
  );
}

/**
 * 角色数据范围转名称
 * @param dataScope
 */
export function roleDataScopeToName(dataScope: any) {
  const values = roleDataScopeList.value;
  const matchedItem = values.find((item) => item.value === dataScope);
  return matchedItem ? matchedItem.label : '';
}

interface PermissionTreeDataNode {
  key: number | string; // 节点唯一标识（父节点用groupId，子节点用id）
  title: string; // 节点显示文本（父节点用groupName，子节点用name）
  children?: PermissionTreeDataNode[]; // 子节点（仅父节点有）
  disabled?: boolean;
  permission?: Permission;
}

export function convertPermissionToTree(
  rawApiList: Permission[],
): PermissionTreeDataNode[] {
  const permMap = new Map<string, Permission[]>();
  rawApiList.forEach((perm) => {
    const groupName = typeof perm.groupName === 'string' ? perm.groupName : '';
    if (!permMap.has(groupName)) {
      permMap.set(groupName, []);
    }
    permMap.get(groupName)?.push(perm);
  });

  return [...permMap.entries()].map(([groupName, permissionList]) => ({
    key: `group-${groupName}`,
    title: groupName,
    children: permissionList.map((perm, index) => ({
      key: perm.id ?? `perm-default-${index}`,
      title: perm.name,
      permission: perm,
    })),
  }));
}
