import type { AppRouteObject } from '@go-wind/admin-react';

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
    meta: { title: 'routes:error-pages', hideInMenu: true, hideInTab: true },
    children: [
      {
        name: 'unauthorized',
        path: '401',
        element: <Unauthorized />,
        meta: { title: 'routes:unauthorized', ignoreAccess: true, hideInMenu: true, hideInTab: true },
      },
      {
        name: 'forbidden',
        path: '403',
        element: <Forbidden />,
        meta: { title: 'routes:forbidden', ignoreAccess: true, hideInMenu: true, hideInTab: true },
      },
      {
        name: 'server-error',
        path: '500',
        element: <InternalError />,
        meta: { title: 'routes:server-error', ignoreAccess: true, hideInMenu: true, hideInTab: true },
      },
      {
        name: 'offline',
        path: 'offline',
        element: <Offline />,
        meta: { title: 'routes:offline', ignoreAccess: true, hideInMenu: true, hideInTab: true },
      },
      {
        name: 'coming-soon',
        path: 'coming-soon',
        element: <ComingSoon />,
        meta: { title: 'routes:coming-soon', ignoreAccess: true, hideInMenu: true, hideInTab: true },
      },

      // ========== 404 通配符路由（必须放在最后！） ==========
      {
        name: 'not-found',
        path: '*',
        element: <NotFound />,
        meta: { title: 'routes:not-found', ignoreAccess: true, hideInMenu: true, hideInTab: true },
      },
    ],
  },
];

export default errorRoutes;
