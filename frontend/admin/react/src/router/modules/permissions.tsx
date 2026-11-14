import type { AppRouteObject } from '@go-wind/admin-react';
import { createLazyRoute } from '@go-wind/admin-react';

/**
 * 权限管理路由配置
 * 包括权限点管理、角色管理等页面
 */
export const permissionRoutes: AppRouteObject[] = [
  {
    name: 'permission',
    path: 'permission', // 相对路径，会自动拼接到父路由 '/'
    meta: {
      title: 'routes:permission',
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
          title: 'routes:permission-codes',
          icon: 'lucide:shield-ellipsis', // Iconify 格式
          order: 1,
          // permission: 'sys:platform_admin', // 仅平台管理员权限（开发阶段暂时注释）
        },
      },

      {
        name: 'menus',
        path: 'menus', // 相对路径，最终为 /system/menus
        element: createLazyRoute(() => import('@/pages/app/permission/menu')),
        meta: {
          title: 'routes:menus',
          icon: 'lucide:square-menu', // Iconify 格式
          order: 2,
          // permission: 'sys:platform_admin', // 仅平台管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'apis',
        path: 'apis', // 相对路径，最终为 /system/apis
        element: createLazyRoute(() => import('@/pages/app/permission/api')),
        meta: {
          title: 'routes:apis',
          icon: 'lucide:route', // Iconify 格式
          order: 3,
          // permission: 'sys:platform_admin', // 仅平台管理员权限（开发阶段暂时注释）
        },
      },

      {
        name: 'roles',
        path: 'roles', // 相对路径，最终为 /permission/roles
        element: createLazyRoute(() => import('@/pages/app/permission/role')),
        meta: {
          title: 'routes:roles',
          icon: 'lucide:shield-user', // Iconify 格式
          order: 4,
          // permission: 'sys:platform_admin', // 平台管理员或租户管理员权限（开发阶段暂时注释）
        },
      },
    ],
  },
];

export default permissionRoutes;
