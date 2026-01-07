import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const tenant: RouteRecordRaw[] = [
  {
    path: '/tenant',
    name: 'TenantManagement',
    component: BasicLayout,
    redirect: '/tenant/members',
    meta: {
      order: 2000,
      icon: 'lucide:building-2',
      title: $t('menu.tenant.moduleName'),
      authority: ['platform_admin'],
    },
    children: [
      {
        path: 'members',
        name: 'TenantMemberManagement',
        meta: {
          icon: 'lucide:building-2',
          title: $t('menu.tenant.member'),
          hideInTab: false,
          authority: ['platform_admin'],
        },
        component: () => import('#/views/app/tenant/tenant/index.vue'),
      },
    ],
  },
];

export default tenant;
