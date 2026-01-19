import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const system: RouteRecordRaw[] = [
  {
    path: '/system',
    name: 'System',
    component: BasicLayout,
    redirect: '/system/dict',
    meta: {
      order: 2005,
      icon: 'lucide:settings',
      title: $t('menu.system.moduleName'),
      keepAlive: true,
      authority: ['platform:admin', 'tenant:manager'],
    },
    children: [
      {
        path: 'dict',
        name: 'DictManagement',
        meta: {
          icon: 'lucide:library-big',
          title: $t('menu.system.dict'),
          authority: ['platform:admin', 'tenant:manager'],
        },
        component: () => import('#/views/app/system/dict/index.vue'),
      },

      {
        path: 'files',
        name: 'FileManagement',
        meta: {
          icon: 'lucide:file-search',
          title: $t('menu.system.file'),
          authority: ['platform:admin', 'tenant:manager'],
        },
        component: () => import('#/views/app/system/file/index.vue'),
      },

      {
        path: 'tasks',
        name: 'TaskManagement',
        meta: {
          icon: 'lucide:list-todo',
          title: $t('menu.system.task'),
          authority: ['platform:admin', 'tenant:manager'],
        },
        component: () => import('#/views/app/system/task/index.vue'),
      },

      {
        path: 'login-policies',
        name: 'LoginPolicyManagement',
        meta: {
          icon: 'lucide:shield-x',
          title: $t('menu.system.loginPolicy'),
          authority: ['platform:admin'],
        },
        component: () => import('#/views/app/system/login_policy/index.vue'),
      },
    ],
  },
];

export default system;
