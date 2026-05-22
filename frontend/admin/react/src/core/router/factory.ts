import { createBrowserRouter, type RouteObject } from 'react-router-dom';
import { injectRedirects } from './utils/inject-redirect';
import { sortRoutes } from './utils/sort-routes';
import { transformRoutesWithHandle } from './utils/transform-meta-to-handle';
import type { GenerateMenuAndRoutesOptions, AppRoute, AppRouteObject } from './types';
import { generateRoutesByBackend, generateRoutesByFrontend } from '@/core/router/generators';
import type { AccessModeType } from '@/core/preferences';
import React from 'react';

export const createAccessibleRouter = async (options: GenerateMenuAndRoutesOptions) => {
  let routes: AppRouteObject[] = [...options.routes];

  // 前端模式：基于权限过滤路由
  routes = await generateRoutesByFrontend(
    routes,
    options.permissions ?? [],
    options.forbiddenElement,
  );

  if (options.autoInjectRedirect !== false)
    routes = injectRedirects(routes as unknown as AppRoute[]) as unknown as AppRouteObject[];
  if (options.autoSort !== false)
    routes = sortRoutes(routes as unknown as AppRoute[]) as unknown as AppRouteObject[];

  // 将 meta 转换为 handle，使 useMatches() 能获取路由元数据
  routes = transformRoutesWithHandle(routes);

  return createBrowserRouter(routes as RouteObject[], {
    future: {
      v7_relativeSplatPath: true,
    },
  });
};

/**
 * 根据模式生成路由
 */
export async function generateRoutes(
  mode: AccessModeType,
  options: {
    routes: AppRouteObject[];
    permissions: string[];
    roles: string[];
    forbiddenElement?: React.ReactNode;
    fetchMenuListAsync?: () => Promise<any[]>;
    layoutMap?: Record<string, React.ComponentType<any>>;
    pageMap?: Record<string, React.ComponentType<any>>;
  },
): Promise<AppRouteObject[]> {
  const { routes, permissions, forbiddenElement, fetchMenuListAsync, layoutMap, pageMap } = options;

  let resultRoutes: AppRouteObject[] = routes;

  switch (mode) {
    case 'backend': {
      // 后端模式：从接口获取菜单树，动态转换组件
      if (!fetchMenuListAsync) {
        throw new Error('Backend mode requires fetchMenuListAsync');
      }
      resultRoutes = await generateRoutesByBackend({
        staticRoutes: routes,
        mode,
        fetchMenuListAsync,
        layoutMap,
        pageMap,
      });
      break;
    }
    case 'frontend': {
      // 前端模式：基于静态路由 + 权限过滤
      resultRoutes = await generateRoutesByFrontend(routes, permissions, forbiddenElement);
      break;
    }
  }

  return resultRoutes;
}
