import type { AppRouteObject } from '@go-wind/admin-react';
import { createLazyRoute } from '@go-wind/admin-react';

/**
 * 日志审计管理路由配置
 * 包括登录审计、API审计、操作审计、数据访问审计、权限审计等页面
 */
export const logRoutes: AppRouteObject[] = [
  {
    name: 'log',
    path: 'log', // 相对路径，会自动拼接到父路由 '/'
    meta: {
      title: 'routes:log',
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
          title: 'routes:login-audit-log',
          icon: 'lucide:user-lock',
          order: 1,
          // permission: 'sys:platform_admin', // 平台管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'api-audit-log',
        path: 'api-audit-logs', // 相对路径，最终为 /log/api-audit-logs
        element: createLazyRoute(() => import('@/pages/app/log/api-audit-log')),
        meta: {
          title: 'routes:api-audit-log',
          icon: 'lucide:file-clock',
          order: 2,
          // permission: 'sys:platform_admin', // 平台管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'operation-audit-log',
        path: 'operation-audit-logs', // 相对路径，最终为 /log/operation-audit-logs
        element: createLazyRoute(() => import('@/pages/app/log/operation-audit-log')),
        meta: {
          title: 'routes:operation-audit-log',
          icon: 'lucide:shield-ellipsis', // Iconify 格式
          order: 3,
          // permission: 'sys:platform_admin', // 平台管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'data-access-audit-log',
        path: 'data-access-audit-logs', // 相对路径，最终为 /log/data-access-audit-logs
        element: createLazyRoute(() => import('@/pages/app/log/data-access-audit-log')),
        meta: {
          title: 'routes:data-access-audit-log',
          icon: 'lucide:shield-check',
          order: 4,
          // permission: 'sys:platform_admin', // 平台管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'permission-audit-log',
        path: 'permission-audit-logs', // 相对路径，最终为 /log/permission-audit-logs
        element: createLazyRoute(() => import('@/pages/app/log/permission-audit-log')),
        meta: {
          title: 'routes:permission-audit-log',
          icon: 'lucide:shield-alert',
          order: 5,
          // permission: 'sys:platform_admin', // 平台管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'policy-evaluation-log',
        path: 'policy-evaluation-logs', // 相对路径，最终为 /log/policy-evaluation-logs
        element: createLazyRoute(() => import('@/pages/app/log/policy-evaluation-log')),
        meta: {
          title: 'routes:policy-evaluation-log',
          icon: 'lucide:gavel',
          order: 6,
          // permission: 'sys:platform_admin', // 平台管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'redis-cache-monitor',
        path: 'redis-cache-monitor', // 相对路径，最终为 /log/redis-cache-monitor
        element: createLazyRoute(() => import('@/pages/app/log/redis-cache-monitor')),
        meta: {
          title: 'routes:redis-cache-monitor',
          icon: 'lucide:database',
          order: 7,
          // permission: 'sys:platform_admin', // 平台管理员权限（开发阶段暂时注释）
        },
      },
    ],
  },
];

export default logRoutes;
