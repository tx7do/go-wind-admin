import type { RouteRecordRaw } from 'vue-router';

import type {
  permissionservicev1_Menu as Menu,
  permissionservicev1_Menu_Type as Menu_Type,
  permissionservicev1_DeleteMenuRequest,
  permissionservicev1_GetMenuRequest,
  permissionservicev1_ListMenuResponse,
  permissionservicev1_Menu,
  permissionservicev1_MenuMeta,
  permissionservicev1_SyncMenusRequest,
} from '#/api/generated/admin/service/v1';

import { computed } from 'vue';

import { i18n } from '@vben/locales';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { BasicLayout, IFrameView } from '#/layouts';
import { queryClient } from '#/plugins/vue-query';
import { makeUpdateMask, type PaginationQuery } from '#/transport/rest';

const t = i18n.global.t;

// ==============================
// 菜单管理
// ==============================

export function useListMenus(
  query: PaginationQuery,
  options?: UseQueryOptions<permissionservicev1_ListMenuResponse, Error>,
) {
  return useQuery({
    queryKey: ['listMenus', query],
    queryFn: () => apiClient.menuService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListMenus(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listMenus', params],
    queryFn: () => apiClient.menuService.List(params.toRawParams()),
    retry: 0,
  });
}

export function useGetMenu(
  req: permissionservicev1_GetMenuRequest,
  options?: UseQueryOptions<permissionservicev1_Menu, Error>,
) {
  return useQuery({
    queryKey: ['getMenu', req],
    queryFn: () => apiClient.menuService.Get(req),
    ...options,
  });
}

export function useCreateMenu(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.menuService.Create({ data: { ...values } as Menu }),
    ...options,
  });
}

export function useUpdateMenu(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.menuService.Update({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteMenu(
  options?: UseMutationOptions<
    object,
    Error,
    permissionservicev1_DeleteMenuRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => apiClient.menuService.Delete(data),
    ...options,
  });
}

// ==============================
// 同步菜单（将前端静态路由推送到后端）
// ==============================

export function useSyncMenus(
  options?: UseMutationOptions<
    object,
    Error,
    permissionservicev1_SyncMenusRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => apiClient.menuService.SyncMenus(data),
    ...options,
  });
}

/**
 * 将前端静态路由定义转换为后端菜单同步请求格式
 *
 * 遍历路由树，提取 path / name / component / redirect / meta 等字段，
 * 映射为 permissionservicev1_Menu 结构。
 *
 * component 路径规则:
 * - 布局组件 → "BasicLayout" / "IFrameView"
 * - 页面组件 → 相对于 views/ 的路径，如 "app/opm/org_unit/index.vue"
 */
function routeToMenu(
  route: RouteRecordRaw,
  depth: number = 0,
): null | permissionservicev1_Menu {
  if (!route.path || !route.name) return null;

  // 解析 component
  let component: string | undefined;
  if (route.component) {
    const comp = route.component as any;
    if (comp === BasicLayout) {
      component = 'BasicLayout';
    } else if (comp === IFrameView) {
      component = 'IFrameView';
    } else if (typeof comp === 'function') {
      // 从 import() 路径中提取相对于 views/ 的路径
      const source = comp.toString();
      const match = source.match(/['"]([^'"]+\.vue)['"]/);
      if (match) {
        const fullPath = match[1];
        const viewsIdx = fullPath.indexOf('/views/');
        component =
          viewsIdx === -1
            ? fullPath.replace(/^(\.\.|\.\/)+/, '')
            : fullPath.slice(viewsIdx + '/views/'.length);
      }
    }
  }

  const menu: permissionservicev1_Menu = {
    name: route.name as string,
    path: route.path,
    type: depth === 0 ? 'CATALOG' : 'MENU',
    component,
    children: undefined,
    redirect: route.redirect as string | undefined,
    meta: route.meta as permissionservicev1_MenuMeta | undefined,
  };

  if (route.children && route.children.length > 0) {
    menu.children = route.children
      .map((child) => routeToMenu(child as RouteRecordRaw, depth + 1))
      .filter((item): item is permissionservicev1_Menu => item !== null);
  }

  return menu;
}

/**
 * 将前端静态路由定义列表转换为后端菜单同步请求
 */
export function buildSyncMenusRequest(
  routes: RouteRecordRaw[],
): permissionservicev1_SyncMenusRequest {
  const items = routes
    .map((route) => routeToMenu(route))
    .filter((item): item is permissionservicev1_Menu => item !== null);
  return { items };
}

// ==============================
// 菜单枚举与工具函数
// ==============================

export const menuTypeList = computed(() => [
  { value: 'CATALOG', label: t('enum.menu.type.CATALOG') },
  { value: 'MENU', label: t('enum.menu.type.MENU') },
  { value: 'BUTTON', label: t('enum.menu.type.BUTTON') },
  { value: 'EMBEDDED', label: t('enum.menu.type.EMBEDDED') },
  { value: 'LINK', label: t('enum.menu.type.LINK') },
]);

export function menuTypeToName(menuType: any): string {
  const values = menuTypeList.value;
  const matchedItem = values.find((item) => item.value === menuType);
  return matchedItem ? matchedItem.label : '';
}

export function menuTypeToColor(menuType: Menu_Type) {
  switch (menuType) {
    case 'BUTTON': {
      return '#F56C6C';
    }
    case 'CATALOG': {
      return '#27AE60';
    }
    case 'EMBEDDED': {
      return '#4096FF';
    }
    case 'LINK': {
      return '#9B59B6';
    }
    case 'MENU': {
      return '#165DFF';
    }
    default: {
      return '#86909C';
    }
  }
}

export const isCatalog = (type: string) => type === 'CATALOG';
export const isMenu = (type: string) => type === 'MENU';
export const isButton = (type: string) => type === 'BUTTON';
export const isEmbedded = (type: string) => type === 'EMBEDDED';
export const isLink = (type: string) => type === 'LINK';

export function travelMenuChild(
  nodes: Menu[] | undefined,
  parent: Menu,
): boolean {
  if (nodes === undefined) return false;
  if (parent.parentId === 0 || parent.parentId === undefined) {
    nodes.push(parent);
    return true;
  }
  for (const node of nodes) {
    if (node === undefined) continue;
    if (node.id === parent.parentId) {
      if (node.children !== undefined) node.children.push(parent);
      return true;
    }
    if (travelMenuChild(node.children, parent)) return true;
  }
  return false;
}

export function buildMenuTree(menus: Menu[]): Menu[] {
  // 深拷贝，避免修改 vue-query 缓存中的原始数据
  const cloned = structuredClone(menus);
  const tree: Menu[] = [];
  for (const menu of cloned) {
    if (!menu) continue;
    if (menu.parentId !== 0 && menu.parentId !== undefined) continue;
    tree.push(menu);
  }
  for (const menu of cloned) {
    if (!menu) continue;
    if (menu.parentId === 0 || menu.parentId === undefined) continue;
    if (travelMenuChild(tree, menu)) continue;
    tree.push(menu);
  }
  return tree;
}
