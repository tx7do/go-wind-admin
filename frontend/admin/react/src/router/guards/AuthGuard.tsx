import { Navigate, useLocation } from 'react-router-dom';
import React from 'react';
import { useAuthStore } from '@go-wind/admin-react';

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
    const mfaOperationId = useAuthStore((s) => s.mfaOperationId);
    const isAuthenticated = isAuthenticatedProp ?? !!accessToken;

    // MFA 待验证状态：mfaOperationId 非空表示密码已通过但需二次验证。
    // 此时用户无有效 token，且不应访问登录页以外的受保护页面 —— 强制跳 MFA 挑战页，
    // 除非已经在挑战页。该守卫与挑战页配合确保半登录态被收敛到唯一出口。
    if (mfaOperationId && location.pathname !== '/auth/mfa-challenge') {
        return <Navigate to="/auth/mfa-challenge" replace state={{ from: location }}/>;
    }

    if (!isAuthenticated) {
        const redirect = encodeURIComponent(location.pathname + location.search);
        return <Navigate to={`${loginPath}?redirect=${redirect}`} replace state={{ from: location }}/>;
    }
    return <>{children}</>;
};
