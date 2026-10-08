import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import type { AppRouteObject } from '@/core/router/types';
import { transformRoutesToMenu } from '@/core/router/utils/menu';

interface UseMenuDataOptions {
  staticRoutes?: AppRouteObject[]; // 静态路由（前端模式）
  dynamicRoutes?: AppRouteObject[]; // 动态路由（后端模式）
  permissions: string[]; // 权限列表
}

export const useMenuData = ({
  staticRoutes = [],
  dynamicRoutes,
  permissions,
}: UseMenuDataOptions) => {
  // 使用 useTranslation 来监听语言变化
  const { i18n } = useTranslation();
  
  // 优先使用动态路由（后端模式），否则用静态路由
  const routes = useMemo(() => {
    // 菜单源：优先实际挂载的路由树（AccessibleRoutesContext：后端模式=后端下发，
    // 前端模式=权限过滤后），统一从主布局容器（path='/'）提取子路由，保证侧栏
    // 镜像真实挂载路由而非静态全量表；后端根节点不是 '/'（BasicLayout 目录各有
    // 实路径）时退回整树，形态对齐 vue-vben / vue-element 的后端菜单。
    // '/' 容器必须带可见子路由才认：后端模式树尾还有 errorRoutes 的 '/' 容器
    // （404 通配组，子项全 hideInMenu），误匹配会把侧栏清空
    const source = dynamicRoutes?.length ? dynamicRoutes : staticRoutes;
    const layoutRoute = source.find(
      (route) => route.path === '/' && route.children?.some((child) => !child.meta?.hideInMenu),
    );
    if (layoutRoute?.children?.length) {
      return layoutRoute.children;
    }
    // 仅动态路由整树兜底（静态源无 '/' 容器时保持原行为返回空）
    return dynamicRoutes?.length ? source : [];
  }, [dynamicRoutes, staticRoutes]);

  // 转换路由 → 菜单
  // 关键：添加 i18n.language 作为依赖，语言切换时重新生成菜单
  return useMemo(() => {
    return transformRoutesToMenu(routes, permissions);
  }, [routes, permissions, i18n.language]);
};
