import type { AppRouteObject } from '@go-wind/admin-react';

// 自动导入 modules 下的所有路由模块（仅包含业务功能路由）
const modulesRoutes = import.meta.glob<AppRouteObject[][]>('./modules/**/*.tsx', {
  eager: true, // 同步加载，确保路由立即生效
});

// 提取并展平所有模块路由（这些都是相对路径的业务路由）
// 独立成文件而非挂在 router/index.tsx：页面（菜单同步）需要引用它，
// 而 index.tsx 会 eager glob 全部页面模块，页面反向引用会形成循环依赖
export const businessRoutes: AppRouteObject[] = Object.values(modulesRoutes).flatMap((module) => {
  // 模块可能导出 default 或具名导出 (如 dashboardRoutes)
  const routes = (module as any).default || Object.values(module)[0];
  return Array.isArray(routes) ? routes : [];
});
