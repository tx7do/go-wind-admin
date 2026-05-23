import { useMemo } from 'react';
import React from 'react';
import { Avatar, Dropdown, Badge, Tooltip, Button, Breadcrumb, Input, Popover } from 'antd';
import type { MenuProps } from 'antd';
import * as Icons from '@ant-design/icons';
import { useMatches, useNavigate } from 'react-router-dom';
import { useI18n } from '@/core/i18n';
import { usePreferencesStore } from '@/core/preferences/store';
import type { SupportedLanguagesType } from '@/core/preferences/types/layout';

interface HeaderContentProps {
  userInfo: BasicUserInfo | null;
  collapsed: boolean;
  isFullscreen: boolean;
  onToggleCollapse: () => void;
  onRefresh: () => void;
  onToggleFullscreen: () => void;
  onLogout: () => void;
  isDark: boolean;
  onToggleTheme: () => void;
  onOpenSettings: () => void;
  widgetConfig: {
    fullscreen: boolean;
    globalSearch: boolean;
    languageToggle: boolean;
    notification: boolean;
    themeToggle: boolean;
    refresh: boolean;
    sidebarToggle: boolean;
  };
}

export const HeaderContent = ({
  userInfo,
  collapsed,
  isFullscreen,
  onToggleCollapse,
  onRefresh,
  onToggleFullscreen,
  onLogout,
  isDark,
  onToggleTheme,
  onOpenSettings,
  widgetConfig,
}: HeaderContentProps) => {
  const { t } = useI18n('common');
  const navigate = useNavigate();
  const matches = useMatches();

  // 计算面包屑
  const breadcrumbItems = useMemo(() => {
    type MatchWithHandle = { pathname: string; handle?: { title?: string; icon?: string } };
    const typedMatches = matches as MatchWithHandle[];

    const items = typedMatches
      .filter((match) => match.handle?.title)
      .map((match, index, arr) => {
        // 将图标字符串转换为 React 组件
        let icon: React.ReactNode = undefined;
        if (match.handle?.icon) {
          const IconComponent = (Icons as any)[match.handle.icon];
          if (IconComponent) {
            icon = React.createElement(IconComponent);
          }
        }

        return {
          key: match.pathname,
          title: match.handle?.title || '',
          icon, // 添加图标
          onClick:
            index < arr.length - 1
              ? () => {
                  navigate(match.pathname);
                }
              : undefined,
        };
      });

    // 如果第一个不是首页，添加首页
    if (items.length && items[0].key !== '/') {
      items.unshift({
        key: '/',
        title: t('home'),
        icon: React.createElement(Icons.HomeOutlined), // 首页图标
        onClick: () => navigate('/'),
      });
    }

    return items;
  }, [matches, navigate, t]);

  // 语言切换
  const toggleLocale = (newLocale: SupportedLanguagesType) => {
    console.log('[HeaderContent] toggleLocale called with:', newLocale);
    const { setPreferences } = usePreferencesStore.getState();
    console.log('[HeaderContent] calling setPreferences with:', { app: { locale: newLocale } });
    setPreferences({ app: { locale: newLocale } });
  };

  // 语言菜单
  const languageMenuItems: MenuProps['items'] = [
    {
      key: 'zh-CN',
      label: '简体中文',
      icon: <span style={{ fontSize: 16 }}>🇨🇳</span>,
      onClick: () => toggleLocale('zh-CN'),
    },
    {
      key: 'en-US',
      label: 'English',
      icon: <span style={{ fontSize: 16 }}>🇺🇸</span>,
      onClick: () => toggleLocale('en-US'),
    },
  ];

  // 用户菜单
  const userMenuItems: MenuProps['items'] = [
    {
      key: 'profile',
      icon: <Icons.UserOutlined />,
      label: t('header.profile'),
      onClick: () => {
        /* 跳转个人中心 */
      },
    },
    {
      key: 'settings',
      icon: <Icons.SettingOutlined />,
      label: t('header.settings'),
      onClick: onOpenSettings,
    },
    {
      type: 'divider',
    },
    {
      key: 'logout',
      icon: <Icons.LogoutOutlined style={{ color: '#ff4d4f' }} />,
      label: <span style={{ color: '#ff4d4f' }}>{t('header.logout')}</span>,
      onClick: onLogout,
    },
  ];

  // 通用按钮样式
  const btnStyle: React.CSSProperties = {
    color: isDark ? '#a6a6a6' : '#595959',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  };

  return (
    <div
      style={{
        width: '100%',
        height: '100%',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
      }}
    >
      {/* ========== 左侧区域：折叠按钮 + 刷新 + 面包屑 ========== */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 4, flex: 1, minWidth: 0 }}>
        {/* 隐藏/显示侧边栏 */}
        {widgetConfig.sidebarToggle && (
          <Tooltip title={collapsed ? t('header.expandSidebar') : t('header.collapseSidebar')}>
            <Button
              type="text"
              icon={collapsed ? <Icons.MenuUnfoldOutlined /> : <Icons.MenuFoldOutlined />}
              onClick={onToggleCollapse}
              size="small"
              style={btnStyle}
            />
          </Tooltip>
        )}

        {/* 刷新按钮 */}
        {widgetConfig.refresh && (
          <Tooltip title={t('header.refresh')}>
            <Button
              type="text"
              icon={<Icons.ReloadOutlined />}
              onClick={onRefresh}
              size="small"
              style={btnStyle}
            />
          </Tooltip>
        )}

        {/* 分割线 */}
        <div
          style={{
            width: 1,
            height: 20,
            backgroundColor: isDark ? '#303030' : '#e5e7eb',
            margin: '0 8px',
          }}
        />

        {/* 面包屑 */}
        <Breadcrumb
          items={breadcrumbItems.map((item) => ({
            key: item.key,
            title: item.onClick ? (
              <a
                onClick={(e) => {
                  e.preventDefault();
                  item.onClick?.();
                }}
                style={{
                  color: isDark ? '#a6a6a6' : '#595959',
                  fontSize: 13,
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: 4,
                }}
              >
                {item.icon}
                {item.title}
              </a>
            ) : (
              <span
                style={{
                  color: isDark ? '#ffffff' : '#262626',
                  fontSize: 13,
                  fontWeight: 500,
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: 4,
                }}
              >
                {item.icon}
                {item.title}
              </span>
            ),
          }))}
        />
      </div>

      {/* ========== 右侧区域：搜索 + 设置 + 主题 + 语言 + 全屏 + 通知 + 头像 ========== */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 4, flexShrink: 0 }}>
        {/* 搜索按钮（带快捷键提示） */}
        {widgetConfig.globalSearch && (
          <Popover
            trigger="click"
            placement="bottomRight"
            content={
              <div style={{ width: 320, padding: 8 }}>
                <Input.Search
                  placeholder={t('header.searchPlaceholder')}
                  size="large"
                  onSearch={(value) => {
                    console.log('Search:', value);
                    // 后续可实现全局搜索逻辑
                  }}
                  autoFocus
                />
              </div>
            }
          >
            <div
              className="search-trigger-btn"
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                padding: '6px 12px',
                borderRadius: 20,
                backgroundColor: isDark ? '#2a2a2a' : '#f5f5f5',
                border: `1px solid ${isDark ? '#404040' : '#d9d9d9'}`,
                cursor: 'pointer',
                transition: 'all 0.2s',
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.backgroundColor = isDark ? '#363636' : '#e8e8e8';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.backgroundColor = isDark ? '#2a2a2a' : '#f5f5f5';
              }}
            >
              <Icons.SearchOutlined
                style={{
                  color: isDark ? '#a6a6a6' : '#8c8c8c',
                  fontSize: 14,
                }}
              />
              <span
                style={{
                  color: isDark ? '#a6a6a6' : '#8c8c8c',
                  fontSize: 13,
                }}
              >
                {t('header.search')}
              </span>
              <kbd
                style={{
                  display: 'inline-block',
                  padding: '2px 6px',
                  fontSize: 11,
                  fontFamily: 'monospace',
                  lineHeight: 1.4,
                  color: isDark ? '#8c8c8c' : '#595959',
                  backgroundColor: isDark ? '#1f1f1f' : '#e8e8e8',
                  border: `1px solid ${isDark ? '#404040' : '#d9d9d9'}`,
                  borderRadius: 3,
                }}
              >
                {t('header.searchShortcut')}
              </kbd>
            </div>
          </Popover>
        )}

        {/* 设置按钮 */}
        <Tooltip title={t('header.settings')}>
          <Button
            type="text"
            icon={<Icons.SettingOutlined />}
            onClick={onOpenSettings}
            size="small"
            style={btnStyle}
          />
        </Tooltip>

        {/* 主题切换 */}
        {widgetConfig.themeToggle && (
          <Tooltip title={isDark ? t('header.switchToLight') : t('header.switchToDark')}>
            <Button
              type="text"
              icon={isDark ? <Icons.SunOutlined /> : <Icons.MoonOutlined />}
              onClick={onToggleTheme}
              size="small"
              style={btnStyle}
            />
          </Tooltip>
        )}

        {/* 语言切换 - 下拉菜单 */}
        {widgetConfig.languageToggle && (
          <Dropdown menu={{ items: languageMenuItems }} trigger={['click']} placement="bottomRight">
            <Tooltip title={t('header.switchLanguage')}>
              <Button type="text" icon={<Icons.GlobalOutlined />} size="small" style={btnStyle} />
            </Tooltip>
          </Dropdown>
        )}

        {/* 全屏切换 */}
        {widgetConfig.fullscreen && (
          <Tooltip title={isFullscreen ? t('header.exitFullscreen') : t('header.fullscreen')}>
            <Button
              type="text"
              icon={isFullscreen ? <Icons.FullscreenExitOutlined /> : <Icons.FullscreenOutlined />}
              onClick={onToggleFullscreen}
              size="small"
              style={btnStyle}
            />
          </Tooltip>
        )}

        {/* 通知 */}
        {widgetConfig.notification && (
          <Badge count={3} size="small" offset={[0, 4]}>
            <Tooltip title={t('header.notification')}>
              <Button type="text" icon={<Icons.BellOutlined />} size="small" style={btnStyle} />
            </Tooltip>
          </Badge>
        )}

        {/* 用户头像 + 下拉菜单 */}
        <Dropdown menu={{ items: userMenuItems }} trigger={['click']} placement="bottomRight">
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 6,
              cursor: 'pointer',
              borderRadius: 6,
              padding: '2px 8px',
              marginLeft: 4,
              transition: 'background-color 0.2s',
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.backgroundColor = isDark ? '#1f1f1f' : '#f5f5f5';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.backgroundColor = 'transparent';
            }}
          >
            <Avatar src={userInfo?.avatar} icon={<Icons.UserOutlined />} size="small" />
            <span
              className="hidden md:inline"
              style={{
                fontSize: 13,
                fontWeight: 500,
                color: isDark ? '#ffffff' : '#262626',
                maxWidth: 100,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}
            >
              {userInfo?.username || t('header.guest')}
            </span>
          </div>
        </Dropdown>
      </div>
    </div>
  );
};

export default HeaderContent;
