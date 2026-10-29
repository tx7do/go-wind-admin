import type { AppRouteObject } from '@/core/router/types';
import { createLazyRoute } from '@/core/router';

/**
 * BI 分析：仪表盘（渲染态）。order=2 紧跟概览之后，与 ele/vben 两端同序。
 * BI 页面不在后端种子菜单里，两端都靠前端路由自带 meta 出菜单项。
 */
export const biRoutes: AppRouteObject[] = [
  {
    name: 'bi',
    path: 'bi',
    meta: {
      title: 'routes:bi',
      icon: 'lucide:bar-chart-3',
      order: 2,
    },
    children: [
      {
        name: 'bi-dashboard',
        path: 'dashboard', // /bi/dashboard
        element: createLazyRoute(() => import('@/pages/app/bi-dashboard')),
        meta: {
          title: 'routes:bi-dashboard',
          icon: 'lucide:bar-chart-big',
          order: 1,
        },
      },
      {
        name: 'bi-designer',
        path: 'designer', // /bi/designer
        element: createLazyRoute(() => import('@/pages/app/bi-designer')),
        meta: {
          title: 'routes:bi-designer',
          icon: 'lucide:pencil-ruler',
          order: 2,
        },
      },
    ],
  },
];

export default biRoutes;
