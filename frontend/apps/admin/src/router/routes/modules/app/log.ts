import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const log: RouteRecordRaw[] = [
  {
    path: '/log',
    name: 'LogAuditManagement',
    component: BasicLayout,
    redirect: '/log/login-audit-logs',
    meta: {
      order: 2004,
      icon: 'lucide:logs',
      title: $t('menu.log.moduleName'),
      keepAlive: true,
      authority: ['sys:platform_admin'],
    },
    children: [
      {
        path: 'login-audit-logs',
        name: 'LoginAuditLog',
        meta: {
          icon: 'lucide:user-lock',
          title: $t('menu.log.loginAuditLog'),
          authority: ['sys:platform_admin'],
        },
        component: () => import('#/views/app/log/login_audit_log/index.vue'),
      },

      {
        path: 'api-audit-logs',
        name: 'ApiAuditLog',
        meta: {
          icon: 'lucide:file-clock',
          title: $t('menu.log.apiAuditLog'),
          authority: ['sys:platform_admin'],
        },
        component: () => import('#/views/app/log/api_audit_log/index.vue'),
      },

      {
        path: 'operation-audit-logs',
        name: 'OperationAuditLog',
        meta: {
          icon: 'lucide:shield-ellipsis',
          title: $t('menu.log.operationAuditLog'),
          authority: ['sys:platform_admin'],
        },
        component: () =>
          import('#/views/app/log/operation_audit_log/index.vue'),
      },

      {
        path: 'data-access-audit-logs',
        name: 'DataAccessAuditLog',
        meta: {
          icon: 'lucide:shield-check',
          title: $t('menu.log.dataAccessAuditLog'),
          authority: ['sys:platform_admin'],
        },
        component: () =>
          import('#/views/app/log/data_access_audit_log/index.vue'),
      },

      {
        path: 'permission-audit-logs',
        name: 'PermissionAuditLog',
        meta: {
          icon: 'lucide:shield-alert',
          title: $t('menu.log.permissionAuditLog'),
          authority: ['sys:platform_admin'],
        },
        component: () =>
          import('#/views/app/log/permission_audit_log/index.vue'),
      },
    ],
  },
];

export default log;
