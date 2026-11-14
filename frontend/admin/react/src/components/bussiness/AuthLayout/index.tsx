import { GlobalOutlined, MoonOutlined, SunOutlined } from '@ant-design/icons';
import { Button, Tooltip } from 'antd';
import React from 'react';
import { useTranslation } from 'react-i18next';

import SloganIcon from './icons/SloganIcon';
import { usePreferences } from '@go-wind/admin-react';
import { useLocale } from '@go-wind/admin-react';

/**
 * 认证页面布局属性
 */
export interface AuthLayoutProps {
  /** 页面标题（如：欢迎回来、创建账号、找回密码） */
  title: string;
  /** 页面副标题描述 */
  description: string;
  /** 表单内容（由子页面传入） */
  children: React.ReactNode;
  /** 页面标识（用于 Helmet title） */
  pageKey?: string;
  /** 底部链接区域 */
  footerLink?: {
    text: string;
    linkText: string;
    href: string;
  };
}

/**
 * 认证页面通用布局组件
 * 用于登录、注册、找回密码等页面
 */
const AuthLayout: React.FC<AuthLayoutProps> = ({ title, description, children, footerLink }) => {
  const { t } = useTranslation('auth');
  const { theme, toggleTheme, setThemeMode, copyright } = usePreferences();
  const { locale, toggleLocale } = useLocale();

  // 切换主题
  const handleToggleTheme = () => {
    toggleTheme();
  };


  // 监听系统主题变化
  React.useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handleChange = () => {
      if (theme.mode === 'auto') {
        // 触发重新渲染以更新 isLightMode
        setThemeMode('auto');
      }
    };
    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, [theme.mode, setThemeMode]);

  // 切换语言
  const handleToggleLanguage = () => {
    toggleLocale();
  };

  return (
    <div className="flex min-h-screen overflow-hidden relative bg-[color:var(--ant-color-bg-layout)]">
      {/* 右上角工具栏 */}
      <div className="absolute top-5 right-5 flex gap-2 z-10">
        <Tooltip title={t('switchLanguage')}>
          <Button
            type="text"
            icon={<GlobalOutlined />}
            onClick={handleToggleLanguage}
          >
            {locale === 'zh-CN' ? t('langZhCN') : t('langEnUS')}
          </Button>
        </Tooltip>
        <Tooltip
          title={theme.mode === 'light' ? t('switchToDarkMode') : t('switchToLightMode')}
        >
          <Button
            type="text"
            icon={theme.mode === 'light' ? <MoonOutlined /> : <SunOutlined />}
            onClick={handleToggleTheme}
          />
        </Tooltip>
      </div>

      {/* 左侧品牌展示区 */}
      <div className="hidden lg:flex flex-1 flex-col items-center justify-center relative overflow-hidden min-w-0 bg-linear-to-br from-[var(--auth-panel-from)] via-[var(--auth-panel-via)] to-[var(--auth-panel-to)]">
        {/* 背景装饰：径向光晕，缓慢呼吸。亮色用浅白光晕，暗色用主色蓝光（--auth-panel-glow 分主题取值）。 */}
        <div
          className="absolute inset-0 pointer-events-none animate-breathe bg-[radial-gradient(circle_at_center,var(--auth-panel-glow),transparent_60%)]"
          aria-hidden="true"
        />
        {/* 3D 插图背后模糊渐变光晕：让插图融入背景，增强层次感 */}
        <div
          className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-96 h-96 rounded-full bg-linear-to-tr from-blue-50 to-indigo-50 blur-3xl opacity-70 light:opacity-70 dark:opacity-30 pointer-events-none"
          aria-hidden="true"
        />
        {/* 3D 插图 */}
        <div className="relative z-10 w-[440px] h-[440px] mb-9">
          <SloganIcon />
        </div>
        <h2 className="relative z-10 text-center text-[26px] font-bold tracking-wide text-[color:var(--ant-color-text)]">
          {t('systemTitle')}
        </h2>
        <p className="relative z-10 text-center text-[15px] leading-relaxed text-[color:var(--gowind-secondary-text)]">
          {t('systemDescription')}
        </p>
      </div>

      {/* 右侧表单区 */}
      <div className="w-full lg:w-[480px] flex flex-col justify-center items-center p-16 relative bg-[color:var(--ant-color-bg-container)] shadow-[-10px_0_40px_rgba(0,0,0,0.05)] dark:border-l dark:border-white/5 dark:shadow-none">
        <div className="w-full max-w-[420px]">
          {/* 页面标题 */}
          <h1 className="text-[34px] font-extrabold tracking-[-0.5px] mb-2.5 text-[color:var(--ant-color-text)]">
            {title}
          </h1>

          {/* 页面描述 */}
          <p className="text-[15px] leading-relaxed mb-11 text-[color:var(--gowind-secondary-text)]">
            {description}
          </p>

          {/* 表单内容（由子页面传入） */}
          {children}

          {/* 底部链接 */}
          {footerLink && (
            <div className="text-center mt-4 text-[13px]">
              <span className="text-[color:var(--ant-color-text-secondary)]">
                {footerLink.text}{' '}
              </span>
              <a
                href={footerLink.href}
                className="gowind-auth-link hover:opacity-80"
              >
                {footerLink.linkText}
              </a>
            </div>
          )}
        </div>

        {/* 底部版权信息 —— flex 列里 mt-auto 自动贴底 */}
        {copyright.enable && (
          <div className="mt-auto pt-8 text-center text-xs text-[color:var(--ant-color-text-secondary)]">
            Copyright © {copyright.date} {copyright.companyName}
          </div>
        )}
      </div>
    </div>
  );
};

export default AuthLayout;
