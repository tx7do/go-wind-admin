import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const log: RouteRecordRaw[] = [
  {
    path: '/log',
    name: 'LogAuditManagement',
    component: BasicLayout,
    redirect: '/log/login',
    meta: {
      order: 2004,
      icon: 'lucide:activity',
      title: $t('menu.log.moduleName'),
      keepAlive: true,
      authority: ['platform:admin'],
    },
    children: [
      {
        path: 'login-audit-logs',
        name: 'LoginAuditLog',
        meta: {
          icon: 'lucide:user-lock',
          title: $t('menu.log.loginAuditLog'),
          authority: ['platform:admin'],
        },
        component: () => import('#/views/app/log/login_audit_log/index.vue'),
      },

      {
        path: 'api-audit-logs',
        name: 'ApiAuditLog',
        meta: {
          icon: 'lucide:file-clock',
          title: $t('menu.log.apiAuditLog'),
          authority: ['platform:admin'],
        },
        component: () => import('#/views/app/log/api_audit_log/index.vue'),
      },
    ],
  },
];

export default log;
