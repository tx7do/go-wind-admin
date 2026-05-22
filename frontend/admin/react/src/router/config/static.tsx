import type {AppRouteObject} from '@/core/router';

import Login from '@/pages/core/auth/login';
import Register from '@/pages/core/auth/register';

import NotFound from '@/pages/core/error/404.tsx';
import MainLayout from '@/layouts/MainLayout';
import React from "react";

// 懒加载页面
const Dashboard = React.lazy(() => import('@/pages/app/dashboard'));

export const staticRoutes: AppRouteObject[] = [
    {
        name: 'login',
        path: '/login',
        label: '登录',
        element: <Login/>,
        meta: {title: '登录', ignoreAccess: true, hideInMenu: true},
    },
    {
        name: 'register',
        path: '/register',
        label: '注册',
        element: <Register/>,
        meta: {title: '注册', ignoreAccess: true, hideInMenu: true},
    },
    {
        name: 'root',
        path: '/',
        label: '首页',
        element: <MainLayout/>,
        meta: {title: '首页', requiresAuth: true, hideInMenu: true},
        children: [
            {
                name: 'dashboard',
                path: 'dashboard',
                label: '仪表盘',
                element: <Dashboard/>,
                meta: {title: '仪表盘', permission: 'dashboard:view', order: 1},
            },
        ],
    },
    {
        name: '404',
        path: '*',
        label: '404',
        element: <NotFound/>,
        meta: {title: '404', ignoreAccess: true, hideInMenu: true},
    },
];
