import React from 'react';
import { Outlet, Navigate, useLocation } from 'react-router-dom';
import { ConfigProvider, Button, Tooltip } from 'antd';
import { GlobalOutlined, MoonOutlined, SunOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

import { useAuthStore } from '@/stores';
import { usePreferencesStore } from '@/core/preferences/store';
import { useThemeConfig } from '@/core/preferences/hooks/useThemeConfig';
import { useLocale } from '@/core/preferences/hooks/useLocale';

import SloganIcon from '@/components/bussiness/AuthLayout/icons/SloganIcon';

import './UserLayout.style.less';

interface UserLayoutProps {
  requireAuth?: boolean; // 是否需要登录（用于保护路由）
}

export const UserLayout = ({ requireAuth = false }: UserLayoutProps) => {
  const { accessToken } = useAuthStore();
  const mfaOperationId = useAuthStore((s) => s.mfaOperationId);
  const location = useLocation();
  const preferences = usePreferencesStore((state) => state.preferences);
  const themeConfig = useThemeConfig();
  const { locale, toggleLocale } = useLocale();
  const { t } = useTranslation('auth');

  // 根据主题模式判断当前是否为亮色模式
  const isLightMode = React.useMemo(() => {
    if (preferences.theme.mode === 'auto') {
      return window.matchMedia('(prefers-color-scheme: light)').matches;
    }
    return preferences.theme.mode === 'light';
  }, [preferences.theme.mode]);

  // 监听系统主题变化
  React.useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handleChange = () => {
      if (preferences.theme.mode === 'auto') {
        // 触发重新渲染以更新 isLightMode
        usePreferencesStore.getState().setPreferences({ theme: { mode: 'auto' } });
      }
    };
    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, [preferences.theme.mode]);

  // MFA 待验证态收敛：mfaOperationId 非空表示密码已通过但需二次验证。
  // 此时用户无有效 token，唯一允许的页面是 MFA 挑战页；其余一律强制跳过去。
  // 这与 AuthGuard 中的同名收敛逻辑配合，确保半登录态被锁死在唯一出口。
  if (mfaOperationId && location.pathname !== '/auth/mfa-challenge') {
    return <Navigate to="/auth/mfa-challenge" replace />;
  }

  // 未登录保护
  if (requireAuth && !accessToken) {
    const redirect = encodeURIComponent(window.location.pathname + window.location.search);
    return <Navigate to={`/login?redirect=${redirect}`} replace />;
  }

  // 已登录跳转（用于登录页）
  if (!requireAuth && accessToken && !mfaOperationId) {
    return <Navigate to="/" replace />;
  }

  return (
    <ConfigProvider {...themeConfig}>
      <div
        className={`user-layout-wrapper${isLightMode ? ' light-mode' : ''}`}
        style={{
          display: 'flex',
          minHeight: '100vh',
          width: '100%',
          height: '100%'
        }}
      >
        {/* 右上角工具栏 */}
        <div className="user-toolbar">
          <Tooltip title={t('switchLanguage')}>
            <Button
              type="text"
              icon={<GlobalOutlined />}
              onClick={toggleLocale}
              className={isLightMode ? 'user-toolbar-btn-light' : 'user-toolbar-btn-dark'}
            >
              {locale === 'zh-CN' ? '中文' : 'English'}
            </Button>
          </Tooltip>
          <Tooltip
            title={preferences.theme.mode === 'light' ? t('switchToDarkMode') : t('switchToLightMode')}
          >
            <Button
              type="text"
              icon={preferences.theme.mode === 'light' ? <MoonOutlined /> : <SunOutlined />}
              onClick={() => {
                usePreferencesStore.getState().setPreferences({
                  theme: {
                    mode: preferences.theme.mode === 'light' ? 'dark' : 'light',
                  },
                });
              }}
              className={isLightMode ? 'user-toolbar-btn-light' : 'user-toolbar-btn-dark'}
            />
          </Tooltip>
        </div>

        {/* 左侧品牌展示区 */}
        <div className="user-brand-section">
          {/* 背景装饰 - 多层渐变 */}
          <div className="user-brand-overlay" />

          {/* 装饰圆形 */}
          <div className="user-brand-circle circle-1" />
          <div className="user-brand-circle circle-2" />

          {/* 品牌图标 */}
          <div className="user-brand-icon">
            <SloganIcon />
          </div>

          <h2 className="user-brand-title">{t('systemTitle')}</h2>
          <p className="user-brand-description">{t('systemDescription')}</p>
        </div>

        {/* 右侧表单区 */}
        <div className="user-form-section">
          <div className="user-form-content">
            {/* 路由出口 */}
            <Outlet />
          </div>

          {/* 底部版权信息 */}
          {preferences.copyright.enable && (
            <div className="user-copyright text-black/60 dark:text-zinc-500">
              Copyright © {preferences.copyright.date} {preferences.copyright.companyName}
            </div>
          )}
        </div>
      </div>
    </ConfigProvider>
  );
};

export default UserLayout;
