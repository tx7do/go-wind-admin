import type { AppRouteObject } from '@/core/router/types';
import { createLazyRoute } from '@/router/utils/lazy';

/**
 * 权限管理路由配置
 * 包括权限点管理、角色管理等页面
 */
export const permissionRoutes: AppRouteObject[] = [
  {
    name: 'permission',
    path: 'permission', // 相对路径，会自动拼接到父路由 '/'
    meta: {
      title: 'menu:permission',
      icon: 'lucide:shield-check', // Iconify 格式
      order: 2002,
      keepAlive: true, // 保持组件状态
      // permission: 'sys:platform_admin', // 平台管理员或租户管理员权限（开发阶段暂时注释）
    },
    children: [
      {
        name: 'permission-codes',
        path: 'codes', // 相对路径，最终为 /permission/codes
        element: createLazyRoute(() => import('@/pages/app/permission/permission')),
        meta: {
          title: 'menu:permission-codes',
          icon: 'lucide:shield-ellipsis', // Iconify 格式
          order: 1,
          // permission: 'sys:platform_admin', // 仅平台管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'roles',
        path: 'roles', // 相对路径，最终为 /permission/roles
        element: createLazyRoute(() => import('@/pages/app/permission/role')),
        meta: {
          title: 'menu:roles',
          icon: 'lucide:shield-user', // Iconify 格式
          order: 2,
          // permission: 'sys:platform_admin', // 平台管理员或租户管理员权限（开发阶段暂时注释）
        },
      },
    ],
  },
];

export default permissionRoutes;
