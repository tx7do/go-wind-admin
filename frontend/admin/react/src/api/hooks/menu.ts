import {
  type permissionservicev1_CreateMenuRequest,
  type permissionservicev1_DeleteMenuRequest,
  type permissionservicev1_GetMenuRequest,
  type permissionservicev1_ListMenuResponse,
  type permissionservicev1_Menu,
  type permissionservicev1_MenuMeta,
  type permissionservicev1_SyncMenusRequest,
} from '@/api/generated/admin/service/v1';
import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/react-query';
import { makeUpdateMask, type PaginationQuery, queryClient } from '@/core';
import type { AppRouteObject } from '@/core/router';
import { apiClient } from '@/api/client';

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
  options?: UseMutationOptions<{}, Error, permissionservicev1_CreateMenuRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.menuService.Create(data),
    ...options,
  });
}

export function useUpdateMenu(
  options?: UseMutationOptions<{}, Error, { id: string; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: Record<string, any> }) =>
      apiClient.menuService.Update({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteMenu(
  options?: UseMutationOptions<{}, Error, permissionservicev1_DeleteMenuRequest>,
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
  options?: UseMutationOptions<{}, Error, permissionservicev1_SyncMenusRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.menuService.SyncMenus(data),
    ...options,
  });
}

// 页面模块 → 数据库 component 路径（"app/system/config/index.vue" 约定）。
// eager glob 与 router/index.tsx 的 pageMap 同款；用于按模块同一性反查页面路径。
const eagerPageModules = import.meta.glob<{ default: unknown }>('/src/pages/app/**/*.tsx', {
  eager: true,
});
const pageModuleToPath = new Map<unknown, string>();
for (const [key, mod] of Object.entries(eagerPageModules)) {
  // 保留 "app/" 前缀：DB component 约定是 "app/system/config/index.vue"
  const matched = key.match(/\/pages\/(.+)\.tsx$/);
  if (matched) pageModuleToPath.set(mod, `${matched[1]}.vue`);
}

/**
 * 从 createLazyRoute 产出的 element 还原页面 component 路径。
 * element = <Suspense><LazyComponent/></Suspense>：
 * - 懒加载未初始化时 _payload._result 是 loader 函数，调用后与 eager glob 的模块做同一性匹配；
 * - 已渲染过时 _payload._result 就是模块本身，直接匹配。
 * 不能只解析 loader 的源码字符串：prod 下 import 路径会被 vite 改写成 chunk 路径。
 */
async function resolveComponentPath(element: unknown): Promise<string | undefined> {
  const lazyType = (element as any)?.props?.children?.type;
  const payload = lazyType?._payload;
  if (!payload || payload._result == null) return undefined;

  let mod: unknown = payload._result;
  if (typeof mod === 'function') {
    try {
      mod = await (mod as () => Promise<unknown>)();
    } catch (error) {
      console.warn('menu sync: resolve page module failed:', error);
      return undefined;
    }
  }
  for (const [knownMod, path] of pageModuleToPath.entries()) {
    if (knownMod === mod || (knownMod as any)?.default === (mod as any)?.default) {
      return path;
    }
  }
  console.warn('menu sync: page module not found in glob map');
  return undefined;
}

async function routeToMenu(
  route: AppRouteObject,
  depth: number,
  t: (key: string) => string,
): Promise<permissionservicev1_Menu | null> {
  if (!route.path || !route.name) return null;

  // 顶层目录节点补绝对路径，与 DefaultMenus 的 "/system" + 相对子路径约定一致
  const path = depth === 0 && !route.path.startsWith('/') ? `/${route.path}` : route.path;
  const component =
    depth === 0 ? 'BasicLayout' : await resolveComponentPath(route.element);

  const meta = route.meta;
  const menu: permissionservicev1_Menu = {
    name: route.name,
    path,
    type: depth === 0 ? 'CATALOG' : 'MENU',
    component,
    redirect: route.redirect,
    // DB 里存当前语言的标题字符串（vben 同款约定），meta.title 是 "routes:xxx" i18n key
    meta: meta
      ? ({
          title: meta.title ? t(meta.title) : undefined,
          icon: meta.icon,
          order: meta.order,
          hideInMenu: meta.hideInMenu,
          hideChildrenInMenu: meta.hideChildrenInMenu,
          hideInTab: meta.hideInTab,
          keepAlive: meta.keepAlive,
          affixTab: meta.affixTab,
          authority: meta.authority,
        } as permissionservicev1_MenuMeta)
      : undefined,
    children: undefined,
  };

  if (route.children && route.children.length > 0) {
    const children = (
      await Promise.all(route.children.map((child) => routeToMenu(child, depth + 1, t)))
    ).filter((item): item is permissionservicev1_Menu => item !== null);
    if (children.length > 0) {
      menu.children = children;
    }
  }

  return menu;
}

/**
 * 将前端业务路由树转换为菜单同步请求。
 * @param t react-i18next 的翻译函数（用于把 "routes:xxx" 标题解析为当前语言字符串）
 */
export async function buildSyncMenusRequest(
  routes: AppRouteObject[],
  t: (key: string) => string,
): Promise<permissionservicev1_SyncMenusRequest> {
  const items = (
    await Promise.all(routes.map((route) => routeToMenu(route, 0, t)))
  ).filter((item): item is permissionservicev1_Menu => item !== null);
  // UI 固定走 MERGE（保 ID、不废角色授权）；REPLACE 仅保留在 API 层供全量重建使用
  return { items, mode: 'MERGE' };
}
