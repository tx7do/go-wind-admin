import { computed } from 'vue';

import { $t } from '@vben/locales';
import { useUserStore } from '@vben/stores';

import { defineStore } from 'pinia';

import {
  createPermissionServiceClient,
  type permissionservicev1_Permission as Permission,
  type permissionservicev1_PermissionGroup as PermissionGroup,
} from '#/generated/api/admin/service/v1';
import { makeOrderBy, makeQueryString, makeUpdateMask } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const usePermissionStore = defineStore('permission', () => {
  const service = createPermissionServiceClient(requestClientRequestHandler);
  const userStore = useUserStore();

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
      orderBy: makeOrderBy(orderBy),
      query: makeQueryString(formValues, userStore.isTenantUser()),
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

  async function syncPermissions() {
    return await service.SyncPermissions({});
  }

  function $reset() {}

  return {
    $reset,
    listPermission,
    getPermission,
    createPermission,
    updatePermission,
    deletePermission,
    syncPermissions,
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
  permission?: Permission;
}

export function buildPermissionTree(
  permissionGroups: PermissionGroup[],
  permissions: Permission[],
): PermissionTreeDataNode[] {
  const groupNodes: Map<string, PermissionTreeDataNode> = new Map();
  const idToKey: Map<number | string, string> = new Map();
  const nameToKey: Map<string, string> = new Map();

  const defaultKey = 'group-default';
  groupNodes.set(defaultKey, { key: defaultKey, title: '', children: [] });

  function processGroup(
    g: PermissionGroup,
    idxPath: string[],
  ): PermissionTreeDataNode {
    const idPart = g.id ?? `idx-${idxPath.join('-')}`;
    const key = `group-${idPart}`;
    const title =
      typeof (g as any).name === 'string'
        ? (g as any).name
        : String(g.id ?? '');
    const node: PermissionTreeDataNode = { key, title, children: [] };

    groupNodes.set(key, node);
    if (g.id !== undefined && g.id !== null) {
      idToKey.set(g.id as any, key);
    }
    if (typeof (g as any).name === 'string' && (g as any).name) {
      nameToKey.set((g as any).name, key);
    }

    const children = (g as any).children;
    if (Array.isArray(children) && children.length > 0) {
      children.forEach((child: PermissionGroup, idx) => {
        const childNode = processGroup(child, [...idxPath, String(idx)]);
        node.children = node.children ?? [];
        node.children.push(childNode);
      });
    }

    return node;
  }

  // 递归建立所有分组节点（保留原始传入顺序的根节点 key）
  permissionGroups.forEach((g, idx) => {
    processGroup(g, [String(idx)]);
  });

  // 将 permissions 作为叶子挂到对应分组（优先 groupId，其次 groupName，最后默认组）
  permissions.forEach((perm, idx) => {
    let targetKey: string | undefined;

    if (
      perm &&
      (perm as any).groupId !== undefined &&
      (perm as any).groupId !== null
    ) {
      targetKey =
        idToKey.get((perm as any).groupId) ?? `group-${(perm as any).groupId}`;
    }

    if (!targetKey && typeof (perm as any).groupName === 'string') {
      targetKey =
        nameToKey.get((perm as any).groupName) ??
        `group-${(perm as any).groupName}`;
    }

    if (!targetKey || !groupNodes.has(targetKey)) {
      targetKey = defaultKey;
    }

    const childNode: PermissionTreeDataNode = {
      key: perm.id ?? `perm-${idx}`,
      title:
        typeof (perm as any).name === 'string'
          ? `${(perm as any).name} (${(perm as any).code})`
          : '',
      permission: perm,
    };

    const parent = groupNodes.get(targetKey);
    if (parent) {
      parent.children = parent.children ?? [];
      parent.children.push(childNode);
    }
  });

  // 按传入分组顺序输出根节点，保留其递归 children；默认组放最后（有子节点时）
  const result: PermissionTreeDataNode[] = permissionGroups.map((g, idx) => {
    const idPart = g.id ?? `idx-${idx}`;
    const key = `group-${idPart}`;
    return (
      groupNodes.get(key) ?? {
        key,
        title: typeof (g as any).name === 'string' ? (g as any).name : '',
        children: [],
      }
    );
  });

  const defaultNode = groupNodes.get(defaultKey);
  if (defaultNode && (defaultNode.children?.length ?? 0) > 0) {
    result.push(defaultNode);
  }

  return result;
}
