import type { AppRouteObject } from '@/core/router/types';
import { createLazyRoute } from '@/core/router';

/**
 * 系统管理路由配置
 * 包括菜单管理、API管理、字典管理、文件管理、任务管理、登录策略、语言管理等页面
 */
export const systemRoutes: AppRouteObject[] = [
  {
    name: 'system',
    path: 'system', // 相对路径，会自动拼接到父路由 '/'
    meta: {
      title: 'routes:system',
      icon: 'lucide:settings', // Iconify 格式
      order: 2005,
      keepAlive: true, // 保持组件状态
      // permission: 'sys:platform_admin', // 平台管理员或租户管理员权限（开发阶段暂时注释）
    },
    children: [
      {
        name: 'dict',
        path: 'dict', // 相对路径，最终为 /system/dict
        element: createLazyRoute(() => import('@/pages/app/system/dict')),
        meta: {
          title: 'routes:dict',
          icon: 'lucide:library-big', // Iconify 格式
          order: 3,
          // permission: 'sys:platform_admin', // 仅平台管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'files',
        path: 'files', // 相对路径，最终为 /system/files
        element: createLazyRoute(() => import('@/pages/app/system/file')),
        meta: {
          title: 'routes:files',
          icon: 'lucide:file-search', // Iconify 格式
          order: 4,
          // permission: 'sys:platform_admin', // 平台管理员或租户管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'tasks',
        path: 'tasks', // 相对路径，最终为 /system/tasks
        element: createLazyRoute(() => import('@/pages/app/system/task')),
        meta: {
          title: 'routes:tasks',
          icon: 'lucide:list-todo', // Iconify 格式
          order: 5,
          // permission: 'sys:platform_admin', // 平台管理员或租户管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'login-policies',
        path: 'login-policies', // 相对路径，最终为 /system/login-policies
        element: createLazyRoute(() => import('@/pages/app/system/login-policy')),
        meta: {
          title: 'routes:login-policies',
          icon: 'lucide:shield-x', // Iconify 格式
          order: 6,
          // permission: 'sys:platform_admin', // 仅平台管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'languages',
        path: 'languages', // 相对路径，最终为 /system/languages
        element: createLazyRoute(() => import('@/pages/app/system/language')),
        meta: {
          title: 'routes:languages',
          icon: 'lucide:globe', // Iconify 格式
          order: 7,
          // permission: 'sys:platform_admin', // 仅平台管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'online-sessions',
        path: 'online-sessions', // 相对路径，最终为 /system/online-sessions
        element: createLazyRoute(() => import('@/pages/app/system/online-session')),
        meta: {
          title: 'routes:online-sessions',
          icon: 'lucide:monitor', // Iconify 格式
          order: 8,
          // permission: 'sys:platform_admin', // 仅平台管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'server-monitor',
        path: 'server-monitor', // 相对路径，最终为 /system/server-monitor
        element: createLazyRoute(() => import('@/pages/app/system/server-monitor')),
        meta: {
          title: 'routes:server-monitor',
          icon: 'lucide:activity', // Iconify 格式
          order: 9,
          // permission: 'sys:platform_admin', // 仅平台管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'notification-channels',
        path: 'notification-channels', // 相对路径，最终为 /system/notification-channels
        element: createLazyRoute(() => import('@/pages/app/system/notification-channel')),
        meta: {
          title: 'routes:notification-channels',
          icon: 'lucide:mail', // Iconify 格式
          order: 10,
          // permission: 'sys:platform_admin', // 仅平台管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'scripts',
        path: 'scripts', // 相对路径，最终为 /system/scripts
        element: createLazyRoute(() => import('@/pages/app/system/script')),
        meta: {
          title: 'routes:scripts',
          icon: 'lucide:file-code-2', // Iconify 格式
          order: 11,
          // permission: 'sys:platform_admin', // 仅平台管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'configs',
        path: 'configs', // 相对路径，最终为 /system/configs
        element: createLazyRoute(() => import('@/pages/app/system/config')),
        meta: {
          title: 'routes:configs',
          icon: 'lucide:sliders-horizontal', // Iconify 格式
          order: 12,
          // permission: 'sys:platform_admin', // 仅平台管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'access-keys',
        path: 'access-keys', // 相对路径，最终为 /system/access-keys
        element: createLazyRoute(() => import('@/pages/app/system/access-key')),
        meta: {
          title: 'routes:accessKeys',
          icon: 'lucide:key-round',
          order: 13,
        },
      },
      {
        name: 'notification-deliveries',
        path: 'notification-deliveries', // 相对路径，最终为 /system/notification-deliveries
        element: createLazyRoute(() => import('@/pages/app/system/notification-delivery')),
        meta: {
          title: 'routes:notification-deliveries',
          icon: 'lucide:send',
          order: 14,
        },
      },
    ],
  },
];

export default systemRoutes;
