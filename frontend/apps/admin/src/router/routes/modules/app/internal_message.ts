import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const internal_message: RouteRecordRaw[] = [
  {
    path: '/internal-message',
    name: 'InternalMessageManagement',
    redirect: '/internal-message/messages',
    component: BasicLayout,
    meta: {
      order: 2003,
      icon: 'lucide:mail',
      title: $t('menu.internalMessage.moduleName'),
      keepAlive: true,
      authority: ['platform:admin', 'tenant:manager'],
    },
    children: [
      {
        path: 'messages',
        name: 'InternalMessageList',
        meta: {
          icon: 'lucide:message-circle-more',
          title: $t('menu.internalMessage.internalMessage'),
          authority: ['platform:admin', 'tenant:manager'],
        },
        component: () =>
          import('#/views/app/internal_message/message/index.vue'),
      },

      {
        path: 'categories',
        name: 'InternalMessageCategoryManagement',
        meta: {
          icon: 'lucide:calendar-check',
          title: $t('menu.internalMessage.internalMessageCategory'),
          authority: ['platform:admin'],
        },
        component: () =>
          import('#/views/app/internal_message/category/index.vue'),
      },
    ],
  },
];

export default internal_message;
