import type { ProLayoutProps } from '@ant-design/pro-components';
import type { AppRouteObject } from '@/core/router/types';

type MenuRoute = AppRouteObject;

/**
 * 递归排序菜单树（按 order 字段）
 */
const sortMenuTree = (menuList: any[]): any[] => {
  return menuList
    .map((menu) => ({
      ...menu,
      // 递归排序子菜单
      children: menu.children?.length ? sortMenuTree(menu.children) : undefined,
    }))
    .sort((a, b) => {
      // order 越小越靠前，undefined 排最后
      const orderA = a.order ?? Number.MAX_SAFE_INTEGER;
      const orderB = b.order ?? Number.MAX_SAFE_INTEGER;
      return orderA - orderB;
    });
};

export const transformRoutesToMenu = (
  routes: MenuRoute[],
  permissions: string[],
  parentPath: string = '',
): NonNullable<ProLayoutProps['route']>['routes'] => {
  const menus = routes
    .filter((route) => {
      // 过滤掉隐藏菜单或没有权限的路由
      if (route.meta?.hideInMenu) return false;

      const meta = route.meta;
      // 如果有权限要求且用户不在权限列表中，则过滤
      return !(meta?.authority?.length && !meta.authority.some((code: string) => permissions.includes(code)));
    })
    .map((route) => {
      // 处理路径：将相对路径转换为绝对路径
      const fullPath = route.path?.startsWith('/')
        ? route.path
        : `${parentPath}/${route.path}`.replace(/\/+/g, '/');

      const menuItem: any = {
        path: fullPath, // 使用完整路径作为 key
        name: route.label || route.meta?.title,
        icon: route.meta?.icon,
        order: route.meta?.order, // 提取排序字段
      };

      if (route.children) {
        menuItem.children = transformRoutesToMenu(route.children, permissions, fullPath);
      }

      return menuItem;
    })
    .filter(Boolean);

  // 排序菜单树
  return sortMenuTree(menus);
};
