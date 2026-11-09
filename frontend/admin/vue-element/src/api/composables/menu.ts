import { computed } from "vue";
import type { TagType } from "./shared";
import type { RouteRecordRaw } from "vue-router";
import type {
  permissionservicev1_DeleteMenuRequest,
  permissionservicev1_GetMenuRequest,
  permissionservicev1_ListMenuResponse,
  permissionservicev1_Menu,
  permissionservicev1_Menu as Menu,
  permissionservicev1_Menu_Type as Menu_Type,
  permissionservicev1_SyncMenusRequest,
} from "@/api/generated/admin/service/v1";
import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from "@tanstack/vue-query";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import { apiClient } from "@/api/client";
import { queryClient } from "@/plugins/vue-query";

import { Layout } from "@/layouts";
import { i18n } from "@/core/i18n";

const t = i18n.global.t;

// ==============================
// 菜单管理
// ==============================

export function useListMenus(
  query: PaginationQuery,
  options?: UseQueryOptions<permissionservicev1_ListMenuResponse, Error>
) {
  return useQuery({
    queryKey: ["listMenus", query],
    queryFn: () => apiClient.menuService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListMenus(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listMenus", params],
    queryFn: () => apiClient.menuService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetMenu(
  req: permissionservicev1_GetMenuRequest,
  options?: UseQueryOptions<permissionservicev1_Menu, Error>
) {
  return useQuery({
    queryKey: ["getMenu", req],
    queryFn: () => apiClient.menuService.Get(req),
    ...options,
  });
}

export function useCreateMenu(options?: UseMutationOptions<{}, Error, Record<string, any>>) {
  return useMutation({
    mutationFn: (values) => apiClient.menuService.Create({ data: { ...values } as Menu }),
    ...options,
  });
}

export function useUpdateMenu(
  options?: UseMutationOptions<{}, Error, { id: string; values: Record<string, any> }>
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
  options?: UseMutationOptions<{}, Error, permissionservicev1_DeleteMenuRequest>
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
  options?: UseMutationOptions<{}, Error, permissionservicev1_SyncMenusRequest>
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
 * - 布局组件 → "BasicLayout"
 * - 页面组件 → 相对于 pages/ 的路径，如 "app/permission/menu/index.vue"
 */
function routeToMenu(route: RouteRecordRaw, depth: number = 0): Menu | null {
  if (!route.path || !route.name) return null;

  // 解析 component
  let component: string | undefined;
  if (route.component) {
    const comp = route.component as any;
    if (comp === Layout) {
      component = "BasicLayout";
    } else if (typeof comp === "function") {
      // 从 import() 路径中提取相对于 pages/ 的路径
      const source = comp.toString();
      const match = source.match(/['"]([^'"]+\.vue)['"]/);
      if (match) {
        const fullPath = match[1];
        const pagesIdx = fullPath.indexOf("/pages/");
        component =
          pagesIdx === -1
            ? fullPath.replace(/^(\.\.|\.\/|@\/)+/, "")
            : fullPath.slice(pagesIdx + "/pages/".length);
      }
    }
  }

  const meta = route.meta as Record<string, any> | undefined;
  const menu: Menu = {
    name: route.name as string,
    path: route.path,
    type: depth === 0 ? "CATALOG" : "MENU",
    component,
    children: undefined,
    redirect: route.redirect as string | undefined,
    meta: meta
      ? ({
          ...meta,
          // meta.title 是 "routes.xxx" i18n key，DB 里存当前语言的标题字符串（vben 同款约定）
          title: meta.title ? t(String(meta.title)) : undefined,
        } as Menu["meta"])
      : undefined,
  };

  if (route.children && route.children.length > 0) {
    menu.children = route.children
      .map((child) => routeToMenu(child as RouteRecordRaw, depth + 1))
      .filter((item): item is Menu => item !== null);
  }

  return menu;
}

/**
 * 将前端静态路由定义列表转换为菜单同步请求
 */
export function buildSyncMenusRequest(
  routes: RouteRecordRaw[]
): permissionservicev1_SyncMenusRequest {
  const items = routes
    .map((route) => routeToMenu(route))
    .filter((item): item is Menu => item !== null);
  // UI 固定走 MERGE（保 ID、不废角色授权）；REPLACE 仅保留在 API 层供全量重建使用
  return { items, mode: "MERGE" };
}

// ==============================
// 菜单枚举与工具函数
// ==============================

export const menuTypeList = computed(() => [
  { value: "CATALOG", label: t("enum.menu.type.CATALOG") },
  { value: "MENU", label: t("enum.menu.type.MENU") },
  { value: "BUTTON", label: t("enum.menu.type.BUTTON") },
  { value: "EMBEDDED", label: t("enum.menu.type.EMBEDDED") },
  { value: "LINK", label: t("enum.menu.type.LINK") },
]);

export function menuTypeToName(menuType: any): string {
  const values = menuTypeList.value;
  const matchedItem = values.find((item) => item.value === menuType);
  return matchedItem ? matchedItem.label : "";
}

export function menuTypeToType(menuType: Menu_Type): TagType {
  switch (menuType) {
    case "BUTTON":
      return "danger";
    case "CATALOG":
      return "success";
    case "EMBEDDED":
    case "MENU":
      return "primary";
    case "LINK":
      return "warning";
    default:
      return "info";
  }
}

export const isCatalog = (type: string) => type === "CATALOG";
export const isMenu = (type: string) => type === "MENU";
export const isButton = (type: string) => type === "BUTTON";
export const isEmbedded = (type: string) => type === "EMBEDDED";
export const isLink = (type: string) => type === "LINK";

/** 根菜单的 parentId 形态：undefined / '' / "0"（ID 已是字符串，uint64 零值序列化为 "0"） */
export const isRootParentId = (parentId?: string) => !parentId || parentId === "0";

export function travelMenuChild(nodes: Menu[] | undefined, parent: Menu): boolean {
  if (nodes === undefined) return false;
  if (isRootParentId(parent.parentId)) {
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

export function buildMenuTree(menus: Menu[], excludeId?: string): Menu[] {
  // 深拷贝，避免修改 vue-query 缓存中的原始数据
  const cloned = structuredClone(menus);

  // 防自环：不仅要排除 excludeId 自身，还要排除它的整个子树。
  // 否则 excludeId 的子节点（parentId === excludeId）在下方构建时会
  // 因找不到父（excludeId 没进 tree）而被提升为顶层根节点，用户仍可能
  // 把该菜单的父级设成自己的后代，形成环。
  const excludedIds = new Set<string>();
  if (excludeId !== undefined) {
    excludedIds.add(excludeId);
    // 迭代收集所有后代：只要某节点的 parentId 已在集合中，它也是后代
    let changed = true;
    while (changed) {
      changed = false;
      for (const m of cloned) {
        if (
          m?.id != null &&
          m.parentId != null &&
          excludedIds.has(m.parentId) &&
          !excludedIds.has(m.id)
        ) {
          excludedIds.add(m.id);
          changed = true;
        }
      }
    }
  }

  const tree: Menu[] = [];
  for (const menu of cloned) {
    if (!menu) continue;
    if (menu.id != null && excludedIds.has(menu.id)) continue;
    if (!isRootParentId(menu.parentId)) continue;
    if (menu?.name) menu.name = t(menu?.name ?? "");
    tree.push(menu);
  }
  for (const menu of cloned) {
    if (!menu) continue;
    if (menu.id != null && excludedIds.has(menu.id)) continue;
    if (isRootParentId(menu.parentId)) continue;
    if (travelMenuChild(tree, menu)) continue;
    if (menu?.name) menu.name = t(menu?.name ?? "");
    tree.push(menu);
  }
  return tree;
}
