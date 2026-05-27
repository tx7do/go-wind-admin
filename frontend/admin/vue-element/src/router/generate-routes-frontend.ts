import type { RouteRecordRaw } from "vue-router";

import { filterTree, mapTree } from "@/utils";

/**
 * 动态生成路由 - 前端方式
 */
async function generateRoutesByFrontend(
  routes: RouteRecordRaw[],
  roles: string[],
  forbiddenComponent?: RouteRecordRaw["component"],
  accessCodes?: string[]
): Promise<RouteRecordRaw[]> {
  // 根据权限标识过滤路由表，判断当前用户是否拥有指定权限
  const finalRoutes = filterTree(routes, (route) => {
    return hasAuthority(route, roles, accessCodes);
  });

  if (!forbiddenComponent) {
    return finalRoutes;
  }

  // 如果有禁止访问的页面，将禁止访问的页面替换为403页面
  return mapTree(finalRoutes, (route) => {
    if (menuHasVisibleWithForbidden(route)) {
      route.component = forbiddenComponent;
    }
    return route;
  });
}

/**
 * 判断路由是否有权限访问
 * authority 中存储的是权限标识（角色码或权限码均可），同时检查角色码和权限码（取并集）
 * @param route - 路由记录
 * @param roles - 用户拥有的角色码列表
 * @param accessCodes - 用户拥有的权限码列表
 */
function hasAuthority(route: RouteRecordRaw, roles: string[], accessCodes?: string[]) {
  // 路由声明忽略权限检查，直接放行
  if (route.meta?.ignoreAccess) {
    return true;
  }

  const authority = route.meta?.authority as string[] | undefined;
  if (!authority) {
    return true;
  }

  // 检查角色码（精确匹配）
  const roleSet = new Set(roles);
  const hasRole = authority.some((auth) => roleSet.has(auth));
  if (hasRole) return true;

  // 检查权限码（前缀匹配）
  if (accessCodes && accessCodes.length > 0) {
    const codeSet = new Set(accessCodes);
    // 精确匹配
    const hasCode = authority.some((auth) => codeSet.has(auth));
    if (hasCode) return true;
    // 前缀匹配
    const hasPrefix = authority.some((auth) =>
      Array.from(codeSet).some((code) => auth.startsWith(code + ":"))
    );
    if (hasPrefix) return true;
  }

  // 无匹配但可能标记为 menuVisibleWithForbidden
  return menuHasVisibleWithForbidden(route);
}

/**
 * 判断路由是否在菜单中显示，但是访问会被重定向到403
 * @param route
 */
function menuHasVisibleWithForbidden(route: RouteRecordRaw) {
  return (
    !!route.meta?.authority &&
    Reflect.has(route.meta || {}, "menuVisibleWithForbidden") &&
    !!route.meta?.menuVisibleWithForbidden
  );
}

export { generateRoutesByFrontend, hasAuthority };
