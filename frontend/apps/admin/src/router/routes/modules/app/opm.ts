import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const opm: RouteRecordRaw[] = [
  {
    path: '/opm',
    name: 'OrganizationalPersonnelManagement',
    component: BasicLayout,
    redirect: '/opm/users',
    meta: {
      order: 2001,
      icon: 'lucide:users',
      title: $t('menu.opm.moduleName'),
      keepAlive: true,
      authority: ['sys:platform_admin', 'sys:tenant_manager'],
    },
    children: [
      {
        path: 'org-units',
        name: 'OrgUnitManagement',
        meta: {
          order: 1,
          icon: 'lucide:layers',
          title: $t('menu.opm.orgUnit'),
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
        },
        component: () => import('#/views/app/opm/org_unit/index.vue'),
      },

      {
        path: 'positions',
        name: 'PositionManagement',
        meta: {
          order: 2,
          icon: 'lucide:briefcase',
          title: $t('menu.opm.position'),
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
        },
        component: () => import('#/views/app/opm/position/index.vue'),
      },

      {
        path: 'users',
        name: 'UserManagement',
        meta: {
          order: 3,
          icon: 'lucide:user',
          title: $t('menu.opm.user'),
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
        },
        component: () => import('#/views/app/opm/user/index.vue'),
      },
      {
        path: 'users/detail/:id',
        name: 'UserDetail',
        meta: {
          hideInMenu: true,
          title: $t('menu.opm.userDetail'),
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
        },
        component: () => import('#/views/app/opm/user/detail/index.vue'),
      },
    ],
  },
];

export default opm;
