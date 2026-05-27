import type { AppRouteObject, RouteMeta } from '../types';

/**
 * 将 meta 转换为 React Router 的 handle
 * 这样 useMatches() 就能获取到路由元数据
 */
export function transformMetaToHandle(meta?: RouteMeta): RouteHandle | undefined {
  if (!meta) return undefined;

  return {
    title: meta.title,
    icon: typeof meta.icon === 'string' ? meta.icon : undefined,
    authority: meta.authority,
    // 展开其他自定义字段（排除已明确指定的字段）
    ...((({ title, icon, authority, ...rest }) => rest)(meta)),
  };
}

/**
 * 递归转换路由配置，将 meta 映射到 handle
 */
export function transformRoutesWithHandle(routes: AppRouteObject[]): AppRouteObject[] {
  return routes.map((route) => {
    const transformed: AppRouteObject = {
      ...route,
      // 将 meta 同时设置为 handle（React Router v6 需要）
      handle: transformMetaToHandle(route.meta),
    };

    // 递归处理子路由
    if (route.children && route.children.length > 0) {
      transformed.children = transformRoutesWithHandle(route.children);
    }

    return transformed;
  });
}
