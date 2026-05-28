import { Navigate, useLocation } from 'react-router-dom';
import React from 'react';
import { useAuthStore } from '@/stores';

interface AuthGuardProps {
    isAuthenticated?: boolean;
    children: React.ReactNode;
    loginPath?: string;
}

export const AuthGuard = ({
                              isAuthenticated: isAuthenticatedProp,
                              children,
                              loginPath = '/auth/login'
                          }: AuthGuardProps) => {
    const location = useLocation();
    // 优先使用 props，否则从 store 订阅（Hook 形式，状态变化会触发重渲染）
    const accessToken = useAuthStore((s) => s.accessToken);
    const isAuthenticated = isAuthenticatedProp ?? !!accessToken;
    if (!isAuthenticated) {
        const redirect = encodeURIComponent(location.pathname + location.search);
        return <Navigate to={`${loginPath}?redirect=${redirect}`} replace state={{ from: location }}/>;
    }
    return <>{children}</>;
};
