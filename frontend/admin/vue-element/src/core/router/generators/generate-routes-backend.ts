import type { RouteRecordRaw } from "vue-router";

import { mapTree } from "@/utils";
import type {
  ComponentRecordType,
  GenerateMenuAndRoutesOptions,
  RouteRecordStringComponent,
} from "../types";

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
 * 页面组件查找：线上库的 component 值横线/下划线混存（历史种子两代约定并存，
 * 如 app/log/login-audit-log 与 app/system/login_policy），页面目录两形态都有，
 * 精确未命中时按分隔符互换兜底
 */
function lookupPageComponent(
  pageMap: ComponentRecordType,
  normalizePath: string
): unknown {
  const key = normalizePath.endsWith(".vue") ? normalizePath : `${normalizePath}.vue`;
  if (pageMap[key]) return pageMap[key];
  const flipped = key.includes("_") ? key.replace(/_/g, "-") : key.replace(/-/g, "_");
  return pageMap[flipped];
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

      route.component = lookupPageComponent(pageMap, normalizePath) as RouteRecordRaw["component"];
    }

    return route;
  });
}

function normalizeViewPath(path: string): string {
  // 去除相对路径前缀
  const normalizedPath = path.replace(/^(\.\/|\.\.\/)+/, "");

  // 确保路径以 '/' 开头
  const viewPath = normalizedPath.startsWith("/") ? normalizedPath : `/${normalizedPath}`;

  // 这里耦合了 admin 的目录结构：pageMap 键来自 glob('../pages/app/**')，
  // 剥掉 /pages 前缀后恒以 /app 开头；后端组件路径则有的带 app/ 段、
  // 有的不带（dashboard 组种子），统一剥掉前缀后补 /app 对齐
  return viewPath.replace(/^\/(pages\/)?(app\/)?/, "/app/");
}
export { generateRoutesByBackend };
