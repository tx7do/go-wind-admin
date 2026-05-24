import { type AppRouteObject, GuestGuard } from '@/core/router';

import UserLayout from '@/layouts/UserLayout';
import RouteErrorFallback from '@/layouts/components/ErrorFallback/RouteErrorFallback.tsx';

import Login from '@/pages/core/auth/login';
import Register from '@/pages/core/auth/register';

/**
 * 认证相关路由配置
 * 包括登录、注册等页面
 * 这些路由不受 AuthGuard 保护，使用 GuestGuard 防止已登录用户访问
 */
export const authRoutes: AppRouteObject[] = [
  {
    name: 'auth',
    path: '/auth',
    element: <UserLayout requireAuth={false} />,
    errorElement: <RouteErrorFallback />,
    meta: { title: 'menu:auth', ignoreAccess: true, hideInMenu: true, hideInTab: true },
    children: [
      {
        name: 'login',
        path: 'login',
        element: (
          <GuestGuard>
            <Login />
          </GuestGuard>
        ),
        meta: { title: 'menu:login', ignoreAccess: true },
      },
      {
        name: 'register',
        path: 'register',
        element: (
          <GuestGuard>
            <Register />
          </GuestGuard>
        ),
        meta: { title: 'menu:register', ignoreAccess: true },
      },
    ],
  },
];

export default authRoutes;
