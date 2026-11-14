import { type AppRouteObject } from '@go-wind/admin-react';
import { GuestGuard } from '@/router/guards';

import UserLayout from '@/layouts/UserLayout';
import RouteErrorFallback from '@/layouts/components/ErrorFallback/RouteErrorFallback.tsx';

import Login from '@/pages/core/auth/login';
import MfaChallenge from '@/pages/core/auth/mfa-challenge';
import SsoCallback from '@/pages/core/auth/sso-callback';
import ForgotPassword from '@/pages/core/auth/forgot-password';

/**
 * 认证相关路由配置
 * 登录等页面
 * 这些路由不受 AuthGuard 保护，使用 GuestGuard 防止已登录用户访问
 */
export const authRoutes: AppRouteObject[] = [
  {
    name: 'auth',
    path: '/auth',
    element: <UserLayout requireAuth={false} />,
    errorElement: <RouteErrorFallback />,
    meta: { title: 'routes:auth', ignoreAccess: true, hideInMenu: true, hideInTab: true },
    children: [
      {
        name: 'login',
        path: 'login',
        element: (
          <GuestGuard>
            <Login />
          </GuestGuard>
        ),
        meta: { title: 'routes:login', ignoreAccess: true },
      },
      {
        name: 'mfa-challenge',
        path: 'mfa-challenge',
        element: <MfaChallenge />,
        meta: { title: 'routes:mfaChallenge', ignoreAccess: true, hideInMenu: true },
      },
      {
        // OIDC SSO 回调：IdP 授权后重定向回这里（注意：本页必须不受 GuestGuard
        // 限制吗？——受，也未尝不可；已登录用户重复 SSO 登录是边缘场景，直接放行）。
        name: 'sso-callback',
        path: 'sso/callback',
        element: <SsoCallback />,
        meta: { title: 'routes:ssoCallback', ignoreAccess: true, hideInMenu: true },
      },
      {
        name: 'forgot-password',
        path: 'forgot-password',
        element: (
          <GuestGuard>
            <ForgotPassword />
          </GuestGuard>
        ),
        meta: { title: 'routes:forgot-password', ignoreAccess: true, hideInMenu: true },
      },
    ],
  },
];

export default authRoutes;
