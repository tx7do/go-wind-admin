import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const system: RouteRecordRaw[] = [
  {
    path: '/system',
    name: 'System',
    component: BasicLayout,
    redirect: '/system/dictionaries',
    meta: {
      order: 2005,
      icon: 'lucide:settings',
      title: $t('menu.system.moduleName'),
      keepAlive: true,
      authority: ['platform_admin'],
    },
    children: [
      {
        path: 'dictionaries',
        name: 'DictManagement',
        meta: {
          icon: 'lucide:library-big',
          title: $t('menu.system.dict'),
          authority: ['platform_admin'],
        },
        component: () => import('#/views/app/system/dict/index.vue'),
      },

      {
        path: 'files',
        name: 'FileManagement',
        meta: {
          icon: 'lucide:file-search',
          title: $t('menu.system.file'),
          authority: ['platform_admin'],
        },
        component: () => import('#/views/app/system/file/index.vue'),
      },

      {
        path: 'tasks',
        name: 'TaskManagement',
        meta: {
          icon: 'lucide:list-todo',
          title: $t('menu.system.task'),
          authority: ['platform_admin'],
        },
        component: () => import('#/views/app/system/task/index.vue'),
      },

      {
        path: 'admin-login-restrictions',
        name: 'AdminLoginRestrictionManagement',
        meta: {
          icon: 'lucide:shield-x',
          title: $t('menu.system.adminLoginRestriction'),
          authority: ['platform_admin'],
        },
        component: () =>
          import('#/views/app/system/admin_login_restriction/index.vue'),
      },
    ],
  },
];

export default system;
