import { useState, useEffect, useMemo } from 'react';
import { RouterProvider } from 'react-router-dom';

import { createAccessibleRouter } from '@/core/router/factory';
import { useAuthStore, useUserStore } from '@/stores';
import { useAuth } from '@/hooks/useAuth';
import { fetchAllDictEntries } from '@/hooks/useDictCache';

import { Forbidden } from '@/pages/core/error';
import type { AppRouteObject } from '@/core/router';

import { errorRoutes } from './config/error-routes';
import { authRoutes } from './config/auth';
import { staticRoutes } from './config/static';

// 自动导入 modules 下的所有路由模块（仅包含业务功能路由）
const modulesRoutes = import.meta.glob<AppRouteObject[][]>('./modules/**/*.tsx', {
  eager: true, // 同步加载，确保路由立即生效
});

// 提取并展平所有模块路由（这些都是相对路径的业务路由）
const businessRoutes: AppRouteObject[] = Object.values(modulesRoutes).flatMap((module) => {
  // 模块可能导出 default 或具名导出 (如 dashboardRoutes)
  const routes = (module as any).default || Object.values(module)[0];
  return Array.isArray(routes) ? routes : [];
});

// 合并路由：将业务模块路由合并到主布局容器的 children 中
export const allRoutes: AppRouteObject[] = [
  ...staticRoutes.map((route) => {
    // 找到主布局容器路由（path 为 '/' 且包含 children）
    if (route.path === '/' && route.children) {
      return {
        ...route,
        children: [...route.children, ...businessRoutes],
      };
    }
    return route;
  }),
  ...authRoutes, // 认证路由（独立，不受 AuthGuard 保护）
  ...errorRoutes, // 错误路由放在最后
];

export const AppRouter = () => {
  const [router, setRouter] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  const { accessToken } = useAuthStore();
  const { userInfo } = useUserStore();

  // 计算属性：是否已认证、权限列表（使用 useMemo 稳定化）
  const isAuthenticated = !!accessToken;
  const permissions = useMemo(() => userInfo?.permissions || [], [userInfo?.permissions]);

  useEffect(() => {
    let stale = false;

    const initRouter = async () => {
      setLoading(true);

      try {
        // ========== 已认证时的初始化流程 ==========
        // 对齐 Vue 版 setupAccessGuard：权限码获取 + 字典预加载
        if (isAuthenticated) {
          try {
            const auth = useAuth();

            // 1. 获取用户权限码（角色 + 权限码，首次会调 API）
            await auth.getUserPermissionCodes();

            // 2. 预加载字典数据（部分页面依赖字典，未预加载会导致闪烁）
            await fetchAllDictEntries();
          } catch (authErr) {
            // 认证失败（token 过期/无效）：forceLogout 已在拦截器中被调用
            // 清除 userStore 防止脏数据
            console.warn('Auth initialization failed, will redirect to login:', authErr);
            useUserStore.getState().$reset();

            if (stale) return; // 已过期，不继续创建路由
          }
        }

        // 无论认证是否成功，都生成路由（未认证时 permissions 为空，AuthGuard 会拦截）
        const appRouter = await createAccessibleRouter({
          routes: allRoutes,
          permissions,
          forbiddenElement: <Forbidden />,
          autoInjectRedirect: true,
          autoSort: true,
        });

        if (!stale) {
          setRouter(appRouter);
        }
      } catch (err) {
        console.error('Router init failed:', err);
      } finally {
        if (!stale) {
          setLoading(false);
        }
      }
    };

    initRouter();

    // cleanup：当 effect 重新触发时（isAuthenticated 变化），取消上一次 initRouter
    return () => {
      stale = true;
    };
  }, [isAuthenticated, permissions]);

  if (loading || !router)
    return <ThemeLoading fullScreen text="初始化中" subText="正在加载路由配置..." />;

  return <RouterProvider router={router} />;
};
