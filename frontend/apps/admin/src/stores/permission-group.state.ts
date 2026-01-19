import { $t } from '@vben/locales';
import { useUserStore } from '@vben/stores';

import { defineStore } from 'pinia';

import {
  createPermissionGroupServiceClient,
  type permissionservicev1_PermissionGroup as PermissionGroup,
} from '#/generated/api/admin/service/v1';
import { makeOrderBy, makeQueryString, makeUpdateMask } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const usePermissionGroupStore = defineStore('permission-group', () => {
  const service = createPermissionGroupServiceClient(
    requestClientRequestHandler,
  );

  const userStore = useUserStore();

  /**
   * 查询权限点分组列表
   */
  async function listPermissionGroup(
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
   * 获取权限点分组
   */
  async function getPermissionGroup(id: number) {
    return await service.Get({ id });
  }

  /**
   * 创建权限点分组
   */
  async function createPermissionGroup(values: object) {
    return await service.Create({
      // @ts-ignore proto generated code is error.
      data: {
        ...values,
      },
    });
  }

  /**
   * 更新权限点分组
   */
  async function updatePermissionGroup(id: number, values: object) {
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
   * 删除权限点分组
   */
  async function deletePermissionGroup(id: number) {
    return await service.Delete({ id });
  }

  function $reset() {}

  return {
    $reset,
    listPermissionGroup,
    getPermissionGroup,
    createPermissionGroup,
    updatePermissionGroup,
    deletePermissionGroup,
  };
});

/** 遍历分组子节点
 * @param nodes 节点列表
 * @param parent 父节点
 * @return 是否找到并添加
 */
export function travelPermissionGroupChild(
  nodes: PermissionGroup[] | undefined,
  parent: PermissionGroup,
): boolean {
  if (nodes === undefined) {
    return false;
  }

  if (parent.parentId === 0 || parent.parentId === undefined) {
    if (parent?.name) {
      parent.name = $t(parent?.name ?? '');
    }
    nodes.push(parent);
    return true;
  }

  for (const node of nodes) {
    if (node === undefined) {
      continue;
    }
    if (node.id === parent.parentId) {
      if (parent?.name) {
        parent.name = $t(parent?.name ?? '');
      }
      if (node.children !== undefined) {
        node.children.push(parent);
      }
      return true;
    }

    if (travelPermissionGroupChild(node.children, parent)) {
      return true;
    }
  }

  return false;
}

/**
 * 构建分组树
 * @param groups 分组列表
 * @return 分组树
 */
export function buildPermissionGroupTree(
  groups: PermissionGroup[],
): PermissionGroup[] {
  const tree: PermissionGroup[] = [];

  for (const group of groups) {
    if (!group) {
      continue;
    }

    if (group.parentId !== 0 && group.parentId !== undefined) {
      continue;
    }

    if (group?.name) {
      group.name = $t(group?.name ?? '');
    }
    tree.push(group);
  }

  for (const group of groups) {
    if (!group) {
      continue;
    }

    if (group.parentId === 0 || group.parentId === undefined) {
      continue;
    }

    if (travelPermissionGroupChild(tree, group)) {
      continue;
    }

    if (group?.name) {
      group.name = $t(group?.name ?? '');
    }
    tree.push(group);
  }

  return tree;
}
