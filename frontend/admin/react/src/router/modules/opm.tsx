import type { AppRouteObject } from '@/core/router/types';
import { createLazyRoute } from '@/router/utils/lazy';

/**
 * 组织人员管理路由配置
 * 包括组织架构、职位管理、用户管理等页面
 */
export const opmRoutes: AppRouteObject[] = [
  {
    name: 'opm',
    path: 'opm', // 相对路径，会自动拼接到父路由 '/'
    meta: {
      title: 'menu:opm',
      icon: 'lucide:users', // Iconify 格式
      order: 2001,
      keepAlive: true, // 保持组件状态
      // permission: 'sys:platform_admin', // 平台管理员或租户管理员权限（开发阶段暂时注释）
    },
    children: [
      {
        name: 'org-units',
        path: 'org-units', // 相对路径，最终为 /opm/org-units
        element: createLazyRoute(() => import('@/pages/app/opm/org-unit')),
        meta: {
          title: 'menu:org-units',
          icon: 'lucide:layers', // Iconify 格式
          order: 1,
          // permission: 'sys:platform_admin', // 平台管理员或租户管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'positions',
        path: 'positions', // 相对路径，最终为 /opm/positions
        element: createLazyRoute(() => import('@/pages/app/opm/position')),
        meta: {
          title: 'menu:positions',
          icon: 'lucide:briefcase', // Iconify 格式
          order: 2,
          // permission: 'sys:platform_admin', // 平台管理员或租户管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'users',
        path: 'users', // 相对路径，最终为 /opm/users
        element: createLazyRoute(() => import('@/pages/app/opm/user')),
        meta: {
          title: 'menu:users',
          icon: 'lucide:user', // Iconify 格式
          order: 3,
          // permission: 'sys:platform_admin', // 平台管理员或租户管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'user-detail',
        path: 'users/detail/:id', // 动态路由，最终为 /opm/users/detail/:id
        element: createLazyRoute(() => import('@/pages/app/opm/user/detail')),
        meta: {
          title: 'menu:user-detail',
          hideInMenu: true, // 隐藏在菜单中，仅通过编程导航访问
          // permission: 'sys:platform_admin', // 平台管理员或租户管理员权限（开发阶段暂时注释）
        },
      },
    ],
  },
];

export default opmRoutes;
