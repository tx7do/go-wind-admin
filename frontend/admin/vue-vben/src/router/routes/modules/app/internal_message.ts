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
      authority: ['sys:platform_admin', 'sys:tenant_manager'],
    },
    children: [
      {
        path: 'messages',
        name: 'InternalMessageList',
        meta: {
          order: 1,
          icon: 'lucide:message-circle-more',
          title: $t('menu.internalMessage.internalMessage'),
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
        },
        component: () =>
          import('#/views/app/internal_message/message/index.vue'),
      },

      {
        path: 'categories',
        name: 'InternalMessageCategoryManagement',
        meta: {
          order: 2,
          icon: 'lucide:calendar-check',
          title: $t('menu.internalMessage.internalMessageCategory'),
          authority: ['sys:platform_admin'],
        },
        component: () =>
          import('#/views/app/internal_message/category/index.vue'),
      },

      {
        // 收件箱不在侧边菜单展示（hideInMenu），入口为顶栏通知"查看所有消息"
        path: 'inbox',
        name: 'InternalMessageInbox',
        meta: {
          order: 3,
          icon: 'lucide:inbox',
          title: $t('menu.internalMessage.inbox'),
          hideInMenu: true,
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
        },
        component: () =>
          import('#/views/app/internal_message/inbox/index.vue'),
      },
    ],
  },
];

export default internal_message;
