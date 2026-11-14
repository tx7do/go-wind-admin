import type { AppRouteObject } from '@go-wind/admin-react';
import { createLazyRoute } from '@go-wind/admin-react';

/**
 * 组织人员管理路由配置
 * 包括组织架构、职位管理、用户管理等页面
 */
export const opmRoutes: AppRouteObject[] = [
  {
    name: 'opm',
    path: 'opm', // 相对路径，会自动拼接到父路由 '/'
    meta: {
      title: 'routes:opm',
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
          title: 'routes:org-units',
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
          title: 'routes:positions',
          icon: 'lucide:briefcase', // Iconify 格式
          order: 2,
          // permission: 'sys:platform_admin', // 平台管理员或租户管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'user-list',
        path: 'users', // 相对路径，最终为 /opm/users
        element: createLazyRoute(() => import('@/pages/app/opm/user/list')),
        meta: {
          title: 'routes:users',
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
          title: 'routes:user-detail',
          hideInMenu: true, // 隐藏在菜单中，仅通过编程导航访问
          // permission: 'sys:platform_admin', // 平台管理员或租户管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'user-profile',
        path: 'profile', // 相对路径，最终为 /opm/profile
        element: createLazyRoute(() => import('@/pages/app/opm/user/profile')),
        meta: {
          title: 'routes:user-profile',
          hideInMenu: true, // 左侧菜单不可见，由顶部栏下拉菜单触发
          hideInTab: false,
          // permission: 'sys:platform_admin', // 平台管理员或租户管理员权限（开发阶段暂时注释）
        },
      },
    ],
  },
];

export default opmRoutes;
