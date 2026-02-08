import type { RouteRecordRaw } from 'vue-router';

import { $t } from '@vben/locales';

import { BasicLayout } from '#/layouts';

const userRoutes: RouteRecordRaw[] = [
  {
    name: 'Profile',
    path: '/profile',
    component: BasicLayout,
    meta: {
      title: $t('menu.profile.settings'),
      requiresAuth: true,
      hideInTab: false,
      hideInMenu: true,
    },

    children: [
      {
        path: '/profile',
        name: 'ProfilePage',
        component: () => import('#/views/profile/index.vue'),
        meta: {
          title: $t('menu.profile.settings'),
          requiresAuth: true,
          hideInTab: false,
          hideInMenu: true,
        },
      },
    ],
  },
];

export default userRoutes;
