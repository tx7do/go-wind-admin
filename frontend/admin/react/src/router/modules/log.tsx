import type { AppRouteObject } from '@/core/router/types';
import { createLazyRoute } from '@/router/utils/lazy';

/**
 * 日志审计管理路由配置
 * 包括登录审计、API审计、操作审计、数据访问审计、权限审计等页面
 */
export const logRoutes: AppRouteObject[] = [
  {
    name: 'log',
    path: 'log', // 相对路径，会自动拼接到父路由 '/'
    meta: {
      title: 'menu:log',
      icon: 'lucide:logs', // Iconify 格式
      order: 2004,
      keepAlive: true, // 保持组件状态
      // permission: 'sys:platform_admin', // 平台管理员权限（开发阶段暂时注释）
    },
    children: [
      {
        name: 'login-audit-log',
        path: 'login-audit-logs', // 相对路径，最终为 /log/login-audit-logs
        element: createLazyRoute(() => import('@/pages/app/log/login-audit-log')),
        meta: {
          title: 'menu:login-audit-log',
          icon: 'lucide:user-lock', // Iconify 格式
          // permission: 'sys:platform_admin', // 平台管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'api-audit-log',
        path: 'api-audit-logs', // 相对路径，最终为 /log/api-audit-logs
        element: createLazyRoute(() => import('@/pages/app/log/api-audit-log')),
        meta: {
          title: 'menu:api-audit-log',
          icon: 'lucide:file-clock', // Iconify 格式
          // permission: 'sys:platform_admin', // 平台管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'operation-audit-log',
        path: 'operation-audit-logs', // 相对路径，最终为 /log/operation-audit-logs
        element: createLazyRoute(() => import('@/pages/app/log/operation-audit-log')),
        meta: {
          title: 'menu:operation-audit-log',
          icon: 'lucide:shield-ellipsis', // Iconify 格式
          // permission: 'sys:platform_admin', // 平台管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'data-access-audit-log',
        path: 'data-access-audit-logs', // 相对路径，最终为 /log/data-access-audit-logs
        element: createLazyRoute(() => import('@/pages/app/log/data-access-audit-log')),
        meta: {
          title: 'menu:data-access-audit-log',
          icon: 'lucide:shield-check', // Iconify 格式
          // permission: 'sys:platform_admin', // 平台管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'permission-audit-log',
        path: 'permission-audit-logs', // 相对路径，最终为 /log/permission-audit-logs
        element: createLazyRoute(() => import('@/pages/app/log/permission-audit-log')),
        meta: {
          title: 'menu:permission-audit-log',
          icon: 'lucide:shield-alert', // Iconify 格式
          // permission: 'sys:platform_admin', // 平台管理员权限（开发阶段暂时注释）
        },
      },
    ],
  },
];

export default logRoutes;
