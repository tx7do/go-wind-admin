import type { Router, RouteRecordRaw } from "vue-router";

import { filterTree, mapTree } from "@/utils";

/**
 * 根据 routes 生成菜单列表
 * @param routes
 * @param router
 */
async function generateMenus(routes: RouteRecordRaw[], router: Router): Promise<MenuRecordRaw[]> {
  // 将路由列表转换为一个以 name 为键的对象映射
  // 获取所有router最终的path及name
  const finalRoutesMap: { [key: string]: string } = Object.fromEntries(
    router.getRoutes().map(({ name, path }) => [name, path])
  );

  let menus = mapTree<ExRouteRecordRaw, MenuRecordRaw>(routes, (route): MenuRecordRaw => {
    // 路由表的路径写法有多种,这里从router获取到最终的path并赋值
    const path = finalRoutesMap[route.name as string] ?? route.path;

    // 转换为菜单结构
    const { meta, name: routeName, redirect, children } = route;
    const {
      activeIcon,
      badge,
      badgeType,
      badgeVariants,
      hideChildrenInMenu = false,
      icon,
      link,
      order,
      title = "",
    } = (meta || {}) as Record<string, any>;

    const name = (title || routeName || "") as string;

    // 隐藏子菜单
    const resultChildren = hideChildrenInMenu ? [] : (children as MenuRecordRaw[]);

    // 将菜单的所有父级和父级菜单记录到菜单项内
    if (resultChildren && resultChildren.length > 0) {
      resultChildren.forEach((child) => {
        child.parents = [...(route.parents || []), path];
        child.parent = path;
      });
    }
    // 隐藏子菜单
    const resultPath = hideChildrenInMenu ? redirect || path : link || path;
    return {
      activeIcon,
      badge,
      badgeType,
      badgeVariants,
      icon,
      name,
      order,
      parent: route.parent,
      parents: route.parents,
      path: resultPath as string,
      show: !route?.meta?.hideInMenu,
      children: resultChildren || [],
    };
  });

  /**
   * 递归对菜单树进行排序
   */
  function sortMenus(menuList: MenuRecordRaw[]): MenuRecordRaw[] {
    return menuList
      .map((menu) => ({
        ...menu,
        // 递归排序子菜单
        children: menu.children?.length ? sortMenus(menu.children) : [],
      }))
      .sort((a, b) => (a.order ?? 999) - (b.order ?? 999));
  }

  // 对菜单树进行递归排序
  menus = sortMenus(menus);

  return filterTree<MenuRecordRaw>(menus, (menu) => {
    return !!menu.show;
  });
}

export { generateMenus };
