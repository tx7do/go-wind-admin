import type { RouteRecordRaw } from "vue-router";

import { mapTree } from "@/utils";

/**
 * 动态生成路由 - 后端方式
 */
async function generateRoutesByBackend(
  options: GenerateMenuAndRoutesOptions
): Promise<RouteRecordRaw[]> {
  const { fetchMenuListAsync, layoutMap = {}, pageMap = {} } = options;

  try {
    const menuRoutes = await fetchMenuListAsync?.();
    if (!menuRoutes) {
      return [];
    }

    const normalizePageMap: ComponentRecordType = {};

    for (const [key, value] of Object.entries(pageMap)) {
      normalizePageMap[normalizeViewPath(key)] = value;
    }

    return convertRoutes(menuRoutes, layoutMap, normalizePageMap);
  } catch (error) {
    console.error(error);
    return [];
  }
}

/**
 * 将后端返回的路由（component 为字符串）转换为 vue-router 路由（component 为实际组件）
 * 后端路由的 component 字段值是组件路径字符串，需要通过 layoutMap/pageMap 映射为实际组件
 */
function convertRoutes(
  routes: RouteRecordStringComponent[],
  layoutMap: ComponentRecordType,
  pageMap: ComponentRecordType
): RouteRecordRaw[] {
  return mapTree(routes, (node) => {
    const route = node as unknown as RouteRecordRaw;
    const { component, name } = node;

    if (!name) {
      console.error("route name is required", route);
    }

    // layout转换
    if (component && layoutMap[component]) {
      route.component = layoutMap[component];
      // 页面组件转换
    } else if (component) {
      const normalizePath = normalizeViewPath(component);

      route.component =
        pageMap[normalizePath.endsWith(".vue") ? normalizePath : `${normalizePath}.vue`];
    }

    return route;
  });
}

function normalizeViewPath(path: string): string {
  // 去除相对路径前缀
  const normalizedPath = path.replace(/^(\.\/|\.\.\/)+/, "");

  // 确保路径以 '/' 开头
  const viewPath = normalizedPath.startsWith("/") ? normalizedPath : `/${normalizedPath}`;

  // 这里耦合了admin的目录结构
  return viewPath.replace(/^\/views/, "");
}
export { generateRoutesByBackend };
