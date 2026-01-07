import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const log: RouteRecordRaw[] = [
  {
    path: '/logs',
    name: 'LogAuditManagement',
    component: BasicLayout,
    redirect: '/logs/login',
    meta: {
      order: 2004,
      icon: 'lucide:activity',
      title: $t('menu.log.moduleName'),
      keepAlive: true,
      authority: ['platform_admin'],
    },
    children: [
      {
        path: 'login',
        name: 'AdminLoginLog',
        meta: {
          icon: 'lucide:user-lock',
          title: $t('menu.log.adminLoginLog'),
          authority: ['platform_admin'],
        },
        component: () => import('#/views/app/log/admin_login_log/index.vue'),
      },

      {
        path: 'operation',
        name: 'AdminOperationLog',
        meta: {
          icon: 'lucide:file-clock',
          title: $t('menu.log.adminOperationLog'),
          authority: ['platform_admin'],
        },
        component: () =>
          import('#/views/app/log/admin_operation_log/index.vue'),
      },
    ],
  },
];

export default log;
