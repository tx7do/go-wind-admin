import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const routes: RouteRecordRaw[] = [
  {
    name: 'BI',
    path: '/bi',
    component: BasicLayout,
    meta: {
      icon: 'lucide:bar-chart-3',
      order: 2,
      title: $t('page.bi.title'),
      authority: ['sys:platform_admin', 'sys:tenant_manager'],
    },
    children: [
      {
        name: 'BiDashboard',
        path: '/bi/dashboard',
        component: () => import('#/views/app/bi/dashboard/index.vue'),
        meta: {
          order: 1,
          icon: 'lucide:bar-chart-big',
          title: $t('page.bi.dashboard'),
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
        },
      },
      {
        name: 'BiDesigner',
        path: '/bi/designer',
        component: () => import('#/views/app/bi/designer/index.vue'),
        meta: {
          order: 2,
          icon: 'lucide:pencil-ruler',
          title: $t('page.bi.designer'),
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
        },
      },
      {
        name: 'BiManage',
        path: '/bi/manage',
        component: () => import('#/views/app/bi/manage/index.vue'),
        meta: {
          order: 3,
          icon: 'lucide:layout-dashboard',
          title: $t('page.bi.manage'),
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
        },
      },
    ],
  },
];

export default routes;
