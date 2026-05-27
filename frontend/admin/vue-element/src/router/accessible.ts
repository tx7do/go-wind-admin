import { cloneDeep, mapTree } from "@/utils";
import { generateMenus } from "@/router/generate-menus";
import { generateRoutesByBackend } from "@/router/generate-routes-backend";
import { generateRoutesByFrontend } from "@/router/generate-routes-frontend";
import type { RouteRecordRaw } from "vue-router";

async function generateAccessible(mode: AccessModeType, options: GenerateMenuAndRoutesOptions) {
  const { router } = options;

  options.routes = cloneDeep(options.routes);
  // 生成路由
  const accessibleRoutes = await generateRoutes(mode, options);

  // 动态添加到router实例内
  accessibleRoutes.forEach((route) => {
    router.addRoute(route);
  });

  // 递归对路由树进行排序（按 meta.order 升序）
  function sortRoutes(routes: RouteRecordRaw[]): RouteRecordRaw[] {
    return routes
      .map((route) => ({
        ...route,
        children: route.children?.length ? sortRoutes(route.children) : [],
      }))
      .sort((a, b) => ((a.meta?.order as number) ?? 999) - ((b.meta?.order as number) ?? 999));
  }

  const sortedRoutes = sortRoutes(accessibleRoutes);

  // 生成菜单
  const accessibleMenus = await generateMenus(sortedRoutes, options.router);

  // 将排序后的路由存入 Store（侧边栏渲染使用的是 accessRoutes）
  return { accessibleMenus, accessibleRoutes: sortedRoutes };
}

/**
 * Generate routes
 * @param mode
 * @param options
 */
async function generateRoutes(mode: AccessModeType, options: GenerateMenuAndRoutesOptions) {
  const { forbiddenComponent, roles, routes } = options;

  let resultRoutes: RouteRecordRaw[] = routes;
  switch (mode) {
    case "backend": {
      resultRoutes = await generateRoutesByBackend(options);
      // 将静态路由中 hideInMenu 的子路由合并到后端生成的路由树中
      resultRoutes = mergeHiddenRoutes(resultRoutes, routes);
      break;
    }
    case "frontend": {
      resultRoutes = await generateRoutesByFrontend(routes, roles || [], forbiddenComponent, options.accessCodes);
      break;
    }
  }

  /**
   * 调整路由树，做以下处理：
   * 1. 对未添加redirect的路由添加redirect
   */
  resultRoutes = mapTree(resultRoutes, (route) => {
    // 如果有redirect或者没有子路由，则直接返回
    if (route.redirect || !route.children || route.children.length === 0) {
      return route;
    }
    const firstChild = route.children[0];

    // 如果子路由不是以/开头，则直接返回,这种情况需要计算全部父级的path才能得出正确的path，这里不做处理
    if (!firstChild?.path || !firstChild.path.startsWith("/")) {
      return route;
    }

    route.redirect = firstChild.path;
    return route;
  });

  return resultRoutes;
}

export { generateAccessible };

/**
 * 将静态路由中 hideInMenu 的子路由合并到后端生成的路由树中。
 * 后端模式只返回菜单可见的路由，但某些隐藏页面（如详情页）
 * 定义在静态路由中需要被保留。
 */
function mergeHiddenRoutes(
  backendRoutes: RouteRecordRaw[],
  staticRoutes: RouteRecordRaw[]
): RouteRecordRaw[] {
  // 收集静态路由中所有 hideInMenu 的子路由，按父路由 name 分组
  const hiddenChildren = new Map<string, RouteRecordRaw[]>();
  for (const staticRoute of staticRoutes) {
    if (!staticRoute.children) continue;
    const hiddens = staticRoute.children.filter(
      (child) => child.meta?.hideInMenu
    );
    if (hiddens.length > 0 && staticRoute.name) {
      hiddenChildren.set(String(staticRoute.name), hiddens);
    }
  }

  if (hiddenChildren.size === 0) return backendRoutes;

  // 遍历后端路由树，将隐藏子路由合并到匹配的父路由下
  return mapTree(backendRoutes, (route) => {
    const routeName = String(route.name ?? "");
    const hiddens = hiddenChildren.get(routeName);
    if (hiddens) {
      route.children = [...(route.children || []), ...cloneDeep(hiddens)];
    }
    return route;
  });
}
