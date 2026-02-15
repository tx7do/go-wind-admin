import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const permission: RouteRecordRaw[] = [
  {
    path: '/permission',
    name: 'PermissionManagement',
    component: BasicLayout,
    redirect: '/permission/codes',
    meta: {
      order: 2002,
      icon: 'lucide:shield-check',
      title: $t('menu.permission.moduleName'),
      keepAlive: true,
      authority: ['sys:platform_admin', 'sys:tenant_manager'],
    },
    children: [
      {
        path: 'codes',
        name: 'PermissionPointManagement',
        meta: {
          order: 1,
          icon: 'lucide:shield-ellipsis',
          title: $t('menu.permission.permission'),
          authority: ['sys:platform_admin'],
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
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
        },
        component: () => import('#/views/app/permission/role/index.vue'),
      },
    ],
  },
];

export default permission;
