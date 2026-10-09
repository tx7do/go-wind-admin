import type { RouteRecordRaw } from 'vue-router';

import { DEFAULT_HOME_PATH, LOGIN_PATH } from '@vben/constants';

import { AuthPageLayout } from '#/layouts';
import { $t } from '#/locales';
import Login from '#/views/_core/authentication/login.vue';

/** 全局404页面 */
const fallbackNotFoundRoute: RouteRecordRaw = {
  component: () => import('#/views/_core/fallback/not-found.vue'),
  meta: {
    hideInBreadcrumb: true,
    hideInMenu: true,
    hideInTab: true,
    title: '404',
  },
  name: 'FallbackNotFound',
  path: '/:path(.*)*',
};

/** 基本路由，这些路由是必须存在的 */
const coreRoutes: RouteRecordRaw[] = [
  {
    meta: {
      title: 'Root',
    },
    name: 'Root',
    path: '/',
    redirect: DEFAULT_HOME_PATH,
  },
  {
    component: AuthPageLayout,
    meta: {
      hideInTab: true,
      title: 'Authentication',
    },
    name: 'Authentication',
    path: '/auth',
    redirect: LOGIN_PATH,
    children: [
      {
        name: 'Login',
        path: 'login',
        component: Login,
        meta: {
          title: $t('page.auth.login'),
        },
      },
      {
        name: 'SsoCallback',
        path: 'sso/callback',
        component: () =>
          import('#/views/_core/authentication/sso-callback.vue'),
        meta: {
          title: $t('page.sso.title'),
          hideInMenu: true,
          hideInTab: true,
        },
      },
      {
        name: 'MfaChallenge',
        path: 'mfa-challenge',
        component: () =>
          import('#/views/_core/authentication/mfa-challenge.vue'),
        meta: {
          title: $t('page.auth.mfaChallenge'),
          hideInMenu: true,
          hideInTab: true,
        },
      },
    ],
  },
];

export { coreRoutes, fallbackNotFoundRoute };
