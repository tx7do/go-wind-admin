import type { RouteRecordRaw } from 'vue-router';

import { $t } from '@vben/locales';

import { BasicLayout } from '#/layouts';

const messageRoutes: RouteRecordRaw[] = [
  {
    name: 'Inbox',
    path: '/inbox',
    component: BasicLayout,
    meta: {
      title: $t('menu.profile.internalMessage'),
      requiresAuth: true,
      hideInTab: false,
      hideInMenu: true,
    },
    children: [
      {
        path: '/inbox',
        name: 'InboxPage',
        component: () => import('#/views/message/index.vue'),
        meta: {
          title: $t('menu.profile.internalMessage'),
          requiresAuth: true,
          hideInTab: false,
          hideInMenu: true,
        },
      },
    ],
  },
];

export default messageRoutes;
