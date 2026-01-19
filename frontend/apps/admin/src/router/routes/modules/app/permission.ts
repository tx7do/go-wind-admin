import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const permission: RouteRecordRaw[] = [
  {
    path: '/permission',
    name: 'PermissionManagement',
    component: BasicLayout,
    redirect: '/permission/permissions',
    meta: {
      order: 2002,
      icon: 'lucide:shield-check',
      title: $t('menu.permission.moduleName'),
      keepAlive: true,
      authority: ['platform:admin', 'tenant:manager'],
    },
    children: [
      {
        path: 'permissions',
        name: 'PermissionPointsManagement',
        meta: {
          order: 1,
          icon: 'lucide:shield-ellipsis',
          title: $t('menu.permission.permission'),
          hideInTab: false,
          authority: ['platform:admin'],
        },
        component: () => import('#/views/app/permission/permission/index.vue'),
      },

      {
        path: 'roles',
        name: 'RoleManagement',
        meta: {
          order: 2,
          icon: 'lucide:shield-user',
          title: $t('menu.permission.role'),
          hideInTab: false,
          authority: ['platform:admin', 'tenant:manager'],
        },
        component: () => import('#/views/app/permission/role/index.vue'),
      },

      {
        path: 'menus',
        name: 'MenuManagement',
        meta: {
          order: 3,
          icon: 'lucide:square-menu',
          title: $t('menu.permission.menu'),
          authority: ['platform:admin'],
        },
        component: () => import('#/views/app/permission/menu/index.vue'),
      },

      {
        path: 'apis',
        name: 'APIManagement',
        meta: {
          order: 4,
          icon: 'lucide:route',
          title: $t('menu.permission.api'),
          authority: ['platform:admin'],
        },
        component: () => import('#/views/app/permission/api/index.vue'),
      },
    ],
  },
];

export default permission;
