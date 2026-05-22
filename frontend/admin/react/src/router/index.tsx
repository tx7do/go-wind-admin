import {useState, useEffect, useMemo} from 'react';
import {Navigate, RouterProvider} from 'react-router-dom';
import {createAccessibleRouter} from '@/core/router/factory';
import {staticRoutes} from './config/static';
import {useAuthStore, useUserStore} from '@/stores';

import Forbidden from '@/pages/core/error/401.tsx';

export const AppRouter = () => {
    const [router, setRouter] = useState<any>(null);
    const [loading, setLoading] = useState(true);

    const {accessToken} = useAuthStore();
    const {userInfo} = useUserStore();

    // 计算属性：是否已认证、权限列表（使用 useMemo 稳定化）
    const isAuthenticated = !!accessToken;
    const permissions = useMemo(() => userInfo?.permissions || [], [userInfo?.permissions]);

    useEffect(() => {
        const initRouter = async () => {
            setLoading(true);

            try {
                // 未登录：只允许访问登录页、注册页和 404，并强制重定向到登录页
                if (!isAuthenticated) {
                    const publicRoutes = [
                        {
                            path: '/login',
                            element: staticRoutes.find(r => r.path === '/login')?.element,
                            meta: {ignoreAccess: true, hideInMenu: true},
                        },
                        {
                            path: '/register',
                            element: staticRoutes.find(r => r.path === '/register')?.element,
                            meta: {ignoreAccess: true, hideInMenu: true},
                        },
                        {
                            path: '/404',
                            element: staticRoutes.find(r => r.path === '*')?.element,
                            meta: {ignoreAccess: true, hideInMenu: true},
                        },
                        {
                            path: '*',
                            element: <Navigate to="/login" replace/>,
                            meta: {ignoreAccess: true, hideInMenu: true},
                        },
                    ];
                    const router = await createAccessibleRouter({
                        routes: publicRoutes as any,
                        permissions: [],
                        autoInjectRedirect: false,
                    });
                    setRouter(router);
                    setLoading(false);
                    return;
                }

                // 已登录：生成完整路由
                const appRouter = await createAccessibleRouter({
                    routes: staticRoutes,
                    permissions,
                    forbiddenElement: <Forbidden/>,
                    autoInjectRedirect: true,
                    autoSort: true,
                });
                setRouter(appRouter);
            } catch (err) {
                console.error('Router init failed:', err);
            } finally {
                setLoading(false);
            }
        };

        initRouter();
    }, [isAuthenticated, permissions]);

    if (loading || !router) return <div className="h-screen flex items-center justify-center">初始化中...</div>;

    return <RouterProvider router={router}/>;
};
