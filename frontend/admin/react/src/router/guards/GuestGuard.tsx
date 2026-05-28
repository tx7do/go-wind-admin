import { Navigate, useSearchParams } from 'react-router-dom';
import React from 'react';
import { useAuthStore, useUserStore } from '@/stores';

interface GuestGuardProps {
  isAuthenticated?: boolean;
  children: React.ReactNode;
  redirectPath?: string;
}

/**
 * 访客守卫：已登录用户不能访问（如登录页、注册页）
 * 如果已登录，重定向到 redirect 参数指定页面或首页
 */
export const GuestGuard = ({
  isAuthenticated: isAuthenticatedProp,
  children,
  redirectPath = '/',
}: GuestGuardProps) => {
  const [searchParams] = useSearchParams();
  // 使用 Hook 订阅，状态变化会触发重渲染
  const accessToken = useAuthStore((s) => s.accessToken);
  const userInfo = useUserStore((s) => s.userInfo);
  const isAuthenticated = isAuthenticatedProp ?? !!accessToken;

  if (isAuthenticated) {
    // 优先使用 URL 中的 redirect 参数，其次使用用户 homePath，最后用默认值
    const redirect = searchParams.get('redirect') || userInfo?.homePath || redirectPath;
    return <Navigate to={decodeURIComponent(redirect)} replace />;
  }

  return <>{children}</>;
};
