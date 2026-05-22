import {GlobalOutlined, MoonOutlined, SunOutlined} from '@ant-design/icons';
import {Button, Tooltip} from 'antd';
import React from 'react';
import {useTranslation} from 'react-i18next';

import SloganIcon from './icons/SloganIcon';
import './AuthLayout.style.less';
import {usePreferences} from '@/core/preferences/hooks/usePreferences';
import {useLocale} from '@/core/preferences/hooks/useLocale';

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
const AuthLayout: React.FC<AuthLayoutProps> = ({
                                                 title,
                                                 description,
                                                 children,
                                                 pageKey = 'auth',
                                                 footerLink,
                                               }) => {
  const {t} = useTranslation();
  const {theme, toggleTheme, setThemeMode, copyright} = usePreferences();
  const {locale, toggleLocale} = useLocale();

  // 切换主题
  const handleToggleTheme = () => {
    toggleTheme();
  };

  // 根据主题模式判断当前是否为亮色模式
  const isLightMode = React.useMemo(() => {
    if (theme.mode === 'auto') {
      return window.matchMedia('(prefers-color-scheme: light)').matches;
    }
    return theme.mode === 'light';
  }, [theme.mode]);

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
    <div className={`auth-layout-wrapper${isLightMode ? ' light-mode' : ''}`}>
      {/* 右上角工具栏 */}
      <div className="auth-toolbar">
        <Tooltip title={t('auth:switchLanguage')}>
          <Button
            type="text"
            icon={<GlobalOutlined/>}
            onClick={handleToggleLanguage}
            className={isLightMode ? 'auth-toolbar-btn-light' : 'auth-toolbar-btn-dark'}
          >
            {locale === 'zh-CN' ? 'EN' : '中文'}
          </Button>
        </Tooltip>
        <Tooltip title={theme.mode === 'light' ? t('auth:switchToDarkMode') : t('auth:switchToLightMode')}>
          <Button
            type="text"
            icon={theme.mode === 'light' ? <MoonOutlined/> : <SunOutlined/>}
            onClick={handleToggleTheme}
            className={isLightMode ? 'auth-toolbar-btn-light' : 'auth-toolbar-btn-dark'}
          />
        </Tooltip>
      </div>

      {/* 左侧品牌展示区 */}
      <div className="auth-brand-section">
        {/* 背景装饰 - 多层渐变 */}
        <div className="auth-brand-overlay" />

        {/* 装饰圆形 */}
        <div className="auth-brand-circle circle-1" />
        <div className="auth-brand-circle circle-2" />

        {/* 品牌图标 */}
        <div className="auth-brand-icon">
          <SloganIcon />
        </div>

        <h2 className="auth-brand-title">
          {t('auth:systemTitle')}
        </h2>
        <p className="auth-brand-description">
          {t('auth:systemDescription')}
        </p>
      </div>

      {/* 右侧表单区 */}
      <div className="auth-form-section">
        <div className="auth-form-content">
          {/* 页面标题 */}
          <h1 className="auth-form-title">
            {title}
          </h1>

          {/* 页面描述 */}
          <p className="auth-form-description">
            {description}
          </p>

          {/* 表单内容（由子页面传入） */}
          {children}

          {/* 底部链接 */}
          {footerLink && (
            <div className="auth-footer-link">
              <span className="auth-footer-text">
                {footerLink.text}{' '}
              </span>
              <a
                href={footerLink.href}
                className="auth-footer-anchor"
              >
                {footerLink.linkText}
              </a>
            </div>
          )}
        </div>

        {/* 底部版权信息 */}
        {copyright.enable && (
          <div className="auth-copyright">
            Copyright © {copyright.date} {copyright.companyName}
          </div>
        )}
      </div>
    </div>
  );
};

export default AuthLayout;
