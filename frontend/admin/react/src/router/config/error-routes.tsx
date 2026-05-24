import type { AppRouteObject } from '@/core/router/types';

import BlankLayout from '@/layouts/BlankLayout';
import RouteErrorFallback from '@/layouts/components/ErrorFallback/RouteErrorFallback';
import {
  Unauthorized,
  Forbidden,
  InternalError,
  Offline,
  ComingSoon,
  NotFound,
} from '@/pages/core/error';

export const errorRoutes: AppRouteObject[] = [
  {
    name: 'error-pages',
    path: '/',
    element: <BlankLayout />,
    errorElement: <RouteErrorFallback />,
    meta: { title: 'menu:error-pages', hideInMenu: true, hideInTab: true },
    children: [
      {
        name: 'unauthorized',
        path: '401',
        element: <Unauthorized />,
        meta: { title: 'menu:unauthorized', ignoreAccess: true, hideInMenu: true, hideInTab: true },
      },
      {
        name: 'forbidden',
        path: '403',
        element: <Forbidden />,
        meta: { title: 'menu:forbidden', ignoreAccess: true, hideInMenu: true, hideInTab: true },
      },
      {
        name: 'server-error',
        path: '500',
        element: <InternalError />,
        meta: { title: 'menu:server-error', ignoreAccess: true, hideInMenu: true, hideInTab: true },
      },
      {
        name: 'offline',
        path: 'offline',
        element: <Offline />,
        meta: { title: 'menu:offline', ignoreAccess: true, hideInMenu: true, hideInTab: true },
      },
      {
        name: 'coming-soon',
        path: 'coming-soon',
        element: <ComingSoon />,
        meta: { title: 'menu:coming-soon', ignoreAccess: true, hideInMenu: true, hideInTab: true },
      },

      // ========== 404 通配符路由（必须放在最后！） ==========
      {
        name: 'not-found',
        path: '*',
        element: <NotFound />,
        meta: { title: 'menu:not-found', ignoreAccess: true, hideInMenu: true, hideInTab: true },
      },
    ],
  },
];

export default errorRoutes;
