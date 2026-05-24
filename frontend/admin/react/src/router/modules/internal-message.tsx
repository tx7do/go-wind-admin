import type { AppRouteObject } from '@/core/router/types';
import { createLazyRoute } from '@/router/utils/lazy';

/**
 * 内部消息管理路由配置
 * 包括消息列表、消息分类管理等页面
 */
export const internalMessageRoutes: AppRouteObject[] = [
  {
    name: 'internal-message',
    path: 'internal-message', // 相对路径，会自动拼接到父路由 '/'
    meta: {
      title: 'menu:internal-message',
      icon: 'lucide:mail', // Iconify 格式
      order: 2003,
      keepAlive: true, // 保持组件状态
      // permission: 'sys:platform_admin', // 平台管理员或租户管理员权限（取并集）（开发阶段暂时注释）
    },
    children: [
      {
        name: 'internal-message-list',
        path: 'messages', // 相对路径，最终为 /internal-message/messages
        element: createLazyRoute(() => import('@/pages/app/internal-message/message')),
        meta: {
          title: 'menu:internal-message-list',
          icon: 'lucide:message-circle-more', // Iconify 格式
          order: 1,
          // permission: 'sys:platform_admin', // 平台管理员或租户管理员权限（开发阶段暂时注释）
        },
      },
      {
        name: 'internal-message-categories',
        path: 'categories', // 相对路径，最终为 /internal-message/categories
        element: createLazyRoute(() => import('@/pages/app/internal-message/category')),
        meta: {
          title: 'menu:internal-message-categories',
          icon: 'lucide:calendar-check', // Iconify 格式
          order: 2,
          // permission: 'sys:platform_admin', // 仅平台管理员权限（开发阶段暂时注释）
        },
      },
    ],
  },
];

export default internalMessageRoutes;
