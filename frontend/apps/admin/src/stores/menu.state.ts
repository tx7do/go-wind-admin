import { computed } from 'vue';

import { $t } from '@vben/locales';
import { useUserStore } from '@vben/stores';

import { defineStore } from 'pinia';

import {
  createMenuServiceClient,
  type permissionservicev1_Menu as Menu,
  type permissionservicev1_Menu_Type as Menu_Type,
} from '#/generated/api/admin/service/v1';
import { makeOrderBy, makeQueryString, makeUpdateMask } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

const parseToArray = (str: string | string[] | undefined): string[] => {
  if (str === undefined || str === null) return [];
  if (Array.isArray(str)) return str;

  if (!str.trim()) {
    return []; // 空输入返回空数组
  }
  // 按逗号分割，去除每个元素的前后空格，过滤空字符串
  return str
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean); // 排除空字符串（如连续逗号产生的空项）
};

export const useMenuStore = defineStore('menu', () => {
  const service = createMenuServiceClient(requestClientRequestHandler);
  const userStore = useUserStore();

  /**
   * 查询菜单列表
   */
  async function listMenu(
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
   * 获取菜单
   */
  async function getMenu(id: number) {
    return await service.Get({ id });
  }

  function prepareMenuData(values: object): Menu {
    // eslint-disable-next-line unicorn/prefer-structured-clone
    const copyData: Menu = JSON.parse(JSON.stringify(values));

    // @ts-ignore divider1
    delete copyData.divider1;
    if (
      copyData.meta?.authority !== undefined &&
      copyData.meta?.authority !== null &&
      // @ts-ignore string to array
      copyData.meta?.authority !== ''
    ) {
      copyData.meta.authority = parseToArray(copyData.meta?.authority);
    }

    return copyData;
  }

  /**
   * 创建菜单
   */
  async function createMenu(values: object) {
    const copyData = prepareMenuData(values);

    return await service.Create({
      data: {
        ...copyData,
        children: [],
      },
    });
  }

  /**
   * 更新菜单
   */
  async function updateMenu(id: number, values: object) {
    const copyData = prepareMenuData(values);

    console.log('updateMenu', copyData);

    return await service.Update({
      id,
      data: {
        ...copyData,
        children: [],
      },
      // @ts-ignore proto generated code is error.
      updateMask: makeUpdateMask(Object.keys(copyData ?? [])),
    });
  }

  /**
   * 删除菜单
   */
  async function deleteMenu(id: number) {
    return await service.Delete({ id });
  }

  function $reset() {}

  return {
    $reset,
    listMenu,
    getMenu,
    createMenu,
    updateMenu,
    deleteMenu,
  };
});

export const menuTypeList = computed(() => [
  { value: 'CATALOG', label: $t('enum.menu.type.CATALOG') },
  { value: 'MENU', label: $t('enum.menu.type.MENU') },
  { value: 'BUTTON', label: $t('enum.menu.type.BUTTON') },
  { value: 'EMBEDDED', label: $t('enum.menu.type.EMBEDDED') },
  { value: 'LINK', label: $t('enum.menu.type.LINK') },
]);

/**
 * 目录类型转名称
 * @param menuType 目录类型
 */
export function menuTypeToName(menuType: any): string {
  const values = menuTypeList.value;
  const matchedItem = values.find((item) => item.value === menuType);
  return matchedItem ? matchedItem.label : '';
}

/**
 * 菜单类型转颜色值
 * @param menuType 菜单类型枚举
 * @returns 十六进制颜色值（兼容所有UI框架）
 */
export function menuTypeToColor(menuType: Menu_Type) {
  switch (menuType) {
    case 'BUTTON': {
      // 按钮：操作型元素，醒目柔和
      return '#F56C6C';
    } // 柔和红色
    case 'CATALOG': {
      // 文件夹：归类属性
      return '#27AE60';
    } // 深绿色
    case 'EMBEDDED': {
      // 嵌入式菜单：融合科技感
      return '#4096FF';
    } // 浅蓝色
    case 'LINK': {
      // 链接菜单：跳转属性
      return '#9B59B6';
    } // 紫色
    case 'MENU': {
      // 普通菜单：基础导航
      return '#165DFF';
    } // 深蓝色
    default: {
      // 未知类型：中性色
      return '#86909C';
    } // 浅灰色
  }
}

export const isCatalog = (type: string) => type === 'CATALOG';
export const isMenu = (type: string) => type === 'MENU';
export const isButton = (type: string) => type === 'BUTTON';
export const isEmbedded = (type: string) => type === 'EMBEDDED';
export const isLink = (type: string) => type === 'LINK';

/** 遍历菜单子节点
 * @param nodes 节点列表
 * @param parent 父节点
 * @return 是否找到并添加
 */
export function travelMenuChild(
  nodes: Menu[] | undefined,
  parent: Menu,
): boolean {
  if (nodes === undefined) {
    return false;
  }

  if (parent.parentId === 0 || parent.parentId === undefined) {
    if (parent?.meta?.title) {
      parent.meta.title = $t(parent?.meta?.title ?? '');
    }
    nodes.push(parent);
    return true;
  }

  for (const node of nodes) {
    if (node === undefined) {
      continue;
    }
    if (node.id === parent.parentId) {
      if (parent?.meta?.title) {
        parent.meta.title = $t(parent?.meta?.title ?? '');
      }
      if (node.children !== undefined) {
        node.children.push(parent);
      }
      return true;
    }

    if (travelMenuChild(node.children, parent)) {
      return true;
    }
  }

  return false;
}

/**
 * 构建菜单树
 * @param menus 菜单列表
 * @return 菜单树
 */
export function buildMenuTree(menus: Menu[]): Menu[] {
  const tree: Menu[] = [];

  for (const menu of menus) {
    if (!menu) {
      continue;
    }

    if (menu.parentId !== 0 && menu.parentId !== undefined) {
      continue;
    }

    if (menu?.meta?.title) {
      menu.meta.title = $t(menu?.meta?.title ?? '');
    }
    tree.push(menu);
  }

  for (const menu of menus) {
    if (!menu) {
      continue;
    }

    if (menu.parentId === 0 || menu.parentId === undefined) {
      continue;
    }

    if (travelMenuChild(tree, menu)) {
      continue;
    }

    if (menu?.meta?.title) {
      menu.meta.title = $t(menu?.meta?.title ?? '');
    }
    tree.push(menu);
  }

  return tree;
}
