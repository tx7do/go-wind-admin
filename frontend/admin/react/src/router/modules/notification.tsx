import type { AppRouteObject } from '@/core/router/types';
import { createLazyRoute } from '@/core/router';

/**
 * 通知域路由配置：渠道配置、路由规则（事件 → 渠道）、投递台账
 * 站内信只是本域的一个 INTERNAL 渠道，故这三页与 /internal-message 平级而非挂在其下
 */
export const notificationRoutes: AppRouteObject[] = [
  {
    name: 'notification',
    path: 'notification', // 相对路径，会自动拼接到父路由 '/'
    meta: {
      title: 'routes:notification',
      icon: 'lucide:bell-ring', // Iconify 格式
      order: 2006,
      keepAlive: true, // 保持组件状态
      // permission: 'sys:platform_admin', // 仅平台管理员权限（开发阶段暂时注释）
    },
    children: [
      {
        name: 'notification-channels',
        path: 'channels', // 相对路径，最终为 /notification/channels
        element: createLazyRoute(() => import('@/pages/app/notification/channel')),
        meta: {
          title: 'routes:notification-channels',
          icon: 'lucide:mail', // Iconify 格式
          order: 1,
          // permission: 'sys:platform_admin', // 仅平台管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'notification-rules',
        path: 'rules', // 相对路径，最终为 /notification/rules
        element: createLazyRoute(() => import('@/pages/app/notification/rule')),
        meta: {
          title: 'routes:notification-rules',
          icon: 'lucide:git-branch',
          order: 2,
          // permission: 'sys:platform_admin', // 仅平台管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'notification-deliveries',
        path: 'deliveries', // 相对路径，最终为 /notification/deliveries
        element: createLazyRoute(() => import('@/pages/app/notification/delivery')),
        meta: {
          title: 'routes:notification-deliveries',
          icon: 'lucide:send',
          order: 3,
          // permission: 'sys:platform_admin', // 仅平台管理员权限（开发阶段暂时注释）
        },
      },
    ],
  },
];

export default notificationRoutes;
