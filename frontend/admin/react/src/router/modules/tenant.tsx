import type { AppRouteObject } from '@go-wind/admin-react';
import { createLazyRoute } from '@go-wind/admin-react';

/**
 * 租户管理路由配置
 * 包括租户成员管理等页面
 */
export const tenantRoutes: AppRouteObject[] = [
  {
    name: 'tenant',
    path: 'tenant', // 相对路径，会自动拼接到父路由 '/'
    meta: {
      title: 'routes:tenant',
      icon: 'lucide:building-2', // Iconify 格式
      order: 2000,
      // permission: 'sys:platform_admin', // 平台管理员权限（开发阶段暂时注释）
    },
    children: [
      {
        name: 'tenant-members',
        path: 'members', // 相对路径，最终为 /tenant/members
        element: createLazyRoute(() => import('@/pages/app/tenant/tenant')),
        meta: {
          title: 'routes:tenant-members',
          icon: 'lucide:users', // Iconify 格式
          order: 1,
          // permission: 'sys:platform_admin', // 平台管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'tenant-plans',
        path: 'plans', // 相对路径，最终为 /tenant/plans
        element: createLazyRoute(() => import('@/pages/app/tenant/plan')),
        meta: {
          title: 'routes:tenant-plans',
          icon: 'lucide:package', // Iconify 格式
          order: 2,
          // permission: 'sys:platform_admin', // 平台管理员权限（开发阶段暂时注释）
        },
      },
    ],
  },
];

export default tenantRoutes;
