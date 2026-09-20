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
      authority: ['sys:platform_admin', 'sys:tenant_manager'],
    },
    children: [
      {
        path: 'dict',
        name: 'DictManagement',
        meta: {
          order: 3,
          icon: 'lucide:library-big',
          title: $t('menu.system.dict'),
          authority: ['sys:platform_admin'],
        },
        component: () => import('#/views/app/system/dict/index.vue'),
      },

      {
        path: 'files',
        name: 'FileManagement',
        meta: {
          order: 4,
          icon: 'lucide:file-search',
          title: $t('menu.system.file'),
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
        },
        component: () => import('#/views/app/system/file/index.vue'),
      },

      {
        path: 'tasks',
        name: 'TaskManagement',
        meta: {
          order: 5,
          icon: 'lucide:list-todo',
          title: $t('menu.system.task'),
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
        },
        component: () => import('#/views/app/system/task/index.vue'),
      },

      {
        path: 'login-policies',
        name: 'LoginPolicyManagement',
        meta: {
          order: 6,
          icon: 'lucide:shield-x',
          title: $t('menu.system.loginPolicy'),
          authority: ['sys:platform_admin'],
        },
        component: () => import('#/views/app/system/login_policy/index.vue'),
      },

      {
        path: 'languages',
        name: 'LanguageManagement',
        meta: {
          order: 7,
          icon: 'lucide:globe',
          title: $t('menu.system.language'),
          authority: ['sys:platform_admin'],
        },
        component: () => import('#/views/app/system/language/index.vue'),
      },

      {
        path: 'online-sessions',
        name: 'OnlineSessionManagement',
        meta: {
          order: 8,
          icon: 'lucide:monitor',
          title: $t('menu.system.onlineSessions'),
          authority: ['sys:platform_admin'],
        },
        component: () => import('#/views/app/system/online_session/index.vue'),
      },

      {
        path: 'server-monitor',
        name: 'ServerMonitor',
        meta: {
          order: 9,
          icon: 'lucide:activity',
          title: $t('menu.system.serverMonitor'),
          authority: ['sys:platform_admin'],
        },
        component: () => import('#/views/app/system/server_monitor/index.vue'),
      },

      {
        path: 'notification-channels',
        name: 'NotificationChannelManagement',
        meta: {
          order: 10,
          icon: 'lucide:mail',
          title: $t('menu.system.notificationChannels'),
          authority: ['sys:platform_admin'],
        },
        component: () => import('#/views/app/system/notification_channel/index.vue'),
      },

      {
        // 投递台账：与渠道配置同域，order 取 14 与 react 端菜单位置保持一致
        path: 'notification-deliveries',
        name: 'NotificationDeliveryManagement',
        meta: {
          order: 14,
          icon: 'lucide:send',
          title: $t('menu.system.notificationDeliveries'),
          authority: ['sys:platform_admin'],
        },
        component: () => import('#/views/app/system/notification_delivery/index.vue'),
      },

      {
        path: 'scripts',
        name: 'ScriptManagement',
        meta: {
          order: 11,
          icon: 'lucide:file-code-2',
          title: $t('menu.system.scripts'),
          authority: ['sys:platform_admin'],
        },
        component: () => import('#/views/app/system/script/index.vue'),
      },

      {
        path: 'configs',
        name: 'ConfigManagement',
        meta: {
          order: 12,
          icon: 'lucide:sliders-horizontal',
          title: $t('menu.system.config'),
          authority: ['sys:platform_admin'],
        },
        component: () => import('#/views/app/system/config/index.vue'),
      },

      {
        path: 'access-keys',
        name: 'AccessKeyManagement',
        meta: {
          order: 13,
          icon: 'lucide:key-round',
          title: $t('menu.system.accessKeys'),
          authority: ['sys:platform_admin'],
        },
        component: () => import('#/views/app/system/access_key/index.vue'),
      },
    ],
  },
];

export default system;
