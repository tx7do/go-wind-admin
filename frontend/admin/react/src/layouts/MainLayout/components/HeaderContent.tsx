import { useEffect, useMemo } from 'react';
import React from 'react';
import { Avatar, Dropdown, Badge, Tooltip, Button, Breadcrumb, Input, Popover, List, Empty, Spin } from 'antd';
import type { MenuProps } from 'antd';
import {
  UserOutlined,
  SettingOutlined,
  LogoutOutlined,
  MenuUnfoldOutlined,
  MenuFoldOutlined,
  ReloadOutlined,
  SearchOutlined,
  SunOutlined,
  MoonOutlined,
  GlobalOutlined,
  FullscreenOutlined,
  FullscreenExitOutlined,
  BellOutlined,
} from '@ant-design/icons';
import { useMatches, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

import { getIconFromName } from '@/layouts/MainLayout/utils/iconResolver';

import { useI18n } from '@/core/i18n';
import { usePreferencesStore } from '@/core/preferences/store';
import type { SupportedLanguagesType } from '@/core/preferences/types/layout';
import { fetchListUserInbox } from '@/api/hooks/internal-message';
import { useQuery } from '@tanstack/react-query';
import { PaginationQuery, queryClient } from '@/core';
import { globalSSEClient } from '@/core/transport/sse';

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
  const { t: tRoutes, i18n } = useTranslation(); // 用于路由翻译
  const navigate = useNavigate();
  const matches = useMatches();

  // 面包屑偏好设置
  const breadcrumbPreferences = usePreferencesStore((state) => state.preferences.breadcrumb);
  const breadcrumbStyleType = breadcrumbPreferences?.styleType ?? 'normal';

  // 计算面包屑
  const breadcrumbItems = useMemo(() => {
    type MatchWithHandle = {
      pathname: string;
      handle?: { title?: string; icon?: string };
      id?: string; // React Router 内部 ID，用于识别 index 路由
    };
    const typedMatches = matches as MatchWithHandle[];
    const showIcon = breadcrumbPreferences?.showIcon ?? true;
    const showHome = breadcrumbPreferences?.showHome ?? true;

    const items = typedMatches
      .filter((match) => {
        // 过滤掉没有 title 的路由
        if (!match.handle?.title) return false;

        // 过滤掉 index 路由（它们通常是重定向，不应该出现在面包屑中）
        // index 路由的 id 通常包含 "-index"
        if (match.id?.includes('-index')) return false;

        return true;
      })
      .map((match, index, arr) => {
        // 将图标字符串转换为 React 组件（支持 Iconify 和 Ant Design）
        let icon: React.ReactNode = undefined;
        if (showIcon && match.handle?.icon) {
          icon = getIconFromName(match.handle.icon);
        }

        // 尝试通过路由 name 获取翻译标题
        let title = match.handle?.title || '';
        title = tRoutes(title, { defaultValue: title });

        return {
          key: match.pathname,
          title,
          icon,
          onClick:
            index < arr.length - 1
              ? () => {
                  navigate(match.pathname);
                }
              : undefined,
        };
      });

    // 始终在开头添加首页（如果 showHome 为 true）
    if (showHome) {
      // 检查是否已经存在根路径 '/' 的面包屑项，避免重复 key
      const hasHome = items.some((item) => item.key === '/');
      if (!hasHome) {
        items.unshift({
          key: '/',
          title: t('home'),
          icon: showIcon ? getIconFromName('lucide:home') : undefined,
          onClick: () => navigate('/'),
        });
      }
    }

    return items;
  }, [
    matches,
    navigate,
    t,
    tRoutes,
    i18n.language,
    breadcrumbPreferences?.showIcon,
    breadcrumbPreferences?.showHome,
  ]);

  // 语言切换
  const toggleLocale = (newLocale: SupportedLanguagesType) => {
    const { setPreferences } = usePreferencesStore.getState();
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
      icon: <UserOutlined />,
      label: t('header.profile'),
      onClick: () => {
        navigate('/opm/profile');
      },
    },
    {
      key: 'settings',
      icon: <SettingOutlined />,
      label: t('header.settings'),
      onClick: onOpenSettings,
    },
    {
      type: 'divider',
    },
    {
      key: 'logout',
      icon: <LogoutOutlined style={{ color: '#ff4d4f' }} />,
      label: <span style={{ color: '#ff4d4f' }}>{t('header.logout')}</span>,
      onClick: onLogout,
    },
  ];

  // 通用按钮样式
  const btnStyle: React.CSSProperties = {
    color: 'var(--ant-color-text-secondary)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  };

  // 通知弹出框数据
  const { t: tInbox } = useTranslation('inbox');
  const { data: inboxPreview, isLoading: inboxLoading } = useQuery({
    queryKey: ['inboxPreview', userInfo?.id],
    queryFn: async () => {
      const query = new PaginationQuery({
        paging: { page: 1, pageSize: 5 },
        // recipient_user_id 必传：不传只靠租户过滤，会查到同租户其他用户的收件记录
        formValues: { recipient_user_id: String(userInfo?.id), status: 'RECEIVED' },
      });
      return await fetchListUserInbox(query);
    },
    enabled: Boolean(userInfo?.id),
    // 兜底轮询：实时更新靠下方 SSE notification 事件触发 invalidate，这里只在 SSE 失联时补拉
    refetchInterval: 300_000,
    // 窗口不聚焦时不轮询
    refetchIntervalInBackground: false,
  });

  // 未读数用列表响应的服务端 total（独立 COUNT），不能用当前页条数——后者封顶在 pageSize
  const unreadCount = inboxPreview?.total ?? 0;

  // SSE 实时通知：收到新站内信即刷新预览。
  // 后端一直在推 notification 事件，此前前端建了连接却无人订阅，实时推送全部落空。
  useEffect(() => {
    if (!userInfo?.id) return undefined;
    const handler = () => {
      queryClient.invalidateQueries({ queryKey: ['inboxPreview', userInfo?.id] });
    };
    globalSSEClient.on('notification', handler);
    return () => {
      globalSSEClient.off('notification', handler);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [userInfo?.id]);

  const inboxContent = (
    <div style={{ width: 320 }}>
      <Spin spinning={inboxLoading}>
        {inboxPreview && (inboxPreview.items?.length ?? 0) > 0 ? (
          <List
            dataSource={(inboxPreview.items || []).slice(0, 5)}
            renderItem={(item) => (
              <List.Item
                style={{ padding: '8px 0', cursor: 'pointer' }}
                onClick={() => navigate('/internal-message/inbox')}
              >
                <List.Item.Meta
                  title={
                    <span style={{ fontSize: 13 }}>
                      {item.title || tInbox('noUnread')}
                    </span>
                  }
                  description={
                    <span style={{ fontSize: 12, color: 'var(--ant-color-text-secondary)' }}>
                      {item.createdAt}
                    </span>
                  }
                />
              </List.Item>
            )}
          />
        ) : (
          <Empty
            description={tInbox('noUnread')}
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            style={{ padding: '16px 0' }}
          />
        )}
      </Spin>
      <div
        style={{
          borderTop: '1px solid var(--ant-color-border)',
          textAlign: 'center',
          padding: '8px 0 0',
          marginTop: 4,
        }}
      >
        <Button
          type="link"
          size="small"
          onClick={() => navigate('/internal-message/inbox')}
        >
          {tInbox('goToInbox')}
        </Button>
      </div>
    </div>
  );

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
              icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
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
              icon={<ReloadOutlined />}
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
            backgroundColor: 'var(--ant-color-bg-container)',
            margin: '0 8px',
          }}
        />

        {/* 面包屑 */}
        <Breadcrumb
          separator={breadcrumbStyleType === 'background' ? '/' : '>'}
          style={
            breadcrumbStyleType === 'background'
              ? {
                  padding: '4px 8px',
                  borderRadius: 6,
                  backgroundColor: 'var(--ant-color-bg-container)',
                }
              : undefined
          }
          items={breadcrumbItems.map((item, _index) => ({
            key: item.key,
            title: item.onClick ? (
              <a
                onClick={(e) => {
                  e.preventDefault();
                  item.onClick?.();
                }}
                style={{
                  color: 'var(--ant-color-text-secondary)',
                  fontSize: 13,
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: 4,
                  ...(breadcrumbStyleType === 'background'
                    ? {
                        padding: '4px 8px',
                        borderRadius: 4,
                        backgroundColor: 'var(--ant-color-bg-container)',
                        border: '1px solid var(--ant-color-border)',
                      }
                    : {}),
                }}
              >
                {item.icon}
                {item.title}
              </a>
            ) : (
              <span
                style={{
                  color: 'var(--ant-color-text)',
                  fontSize: 13,
                  fontWeight: 500,
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: 4,
                  ...(breadcrumbStyleType === 'background'
                    ? {
                        padding: '4px 8px',
                        borderRadius: 4,
                        backgroundColor: 'var(--ant-color-bg-container)',
                        border: '1px solid var(--ant-color-border)',
                      }
                    : {}),
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
                backgroundColor: 'var(--ant-color-bg-container)',
                border: '1px solid var(--ant-color-border)',
                cursor: 'pointer',
                transition: 'all 0.2s',
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.backgroundColor = 'var(--ant-color-fill-tertiary)';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.backgroundColor = 'var(--ant-color-bg-container)';
              }}
            >
              <SearchOutlined
                style={{
                  color: 'var(--ant-color-text-secondary)',
                  fontSize: 14,
                }}
              />
              <span
                style={{
                  color: 'var(--ant-color-text-secondary)',
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
                  color: 'var(--ant-color-text-secondary)',
                  backgroundColor: 'var(--ant-color-bg-container)',
                  border: '1px solid var(--ant-color-border)',
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
            icon={<SettingOutlined />}
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
              icon={isDark ? <SunOutlined /> : <MoonOutlined />}
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
              <Button type="text" icon={<GlobalOutlined />} size="small" style={btnStyle} />
            </Tooltip>
          </Dropdown>
        )}

        {/* 全屏切换 */}
        {widgetConfig.fullscreen && (
          <Tooltip title={isFullscreen ? t('header.exitFullscreen') : t('header.fullscreen')}>
            <Button
              type="text"
              icon={isFullscreen ? <FullscreenExitOutlined /> : <FullscreenOutlined />}
              onClick={onToggleFullscreen}
              size="small"
              style={btnStyle}
            />
          </Tooltip>
        )}

        {/* 通知 */}
        {widgetConfig.notification && (
          <Popover
            content={inboxContent}
            trigger="click"
            placement="bottomRight"
            title={tInbox('pageTitle')}
          >
            <Badge count={unreadCount} size="small" offset={[0, 4]}>
              <Tooltip title={t('header.notification')}>
                <Button type="text" icon={<BellOutlined />} size="small" style={btnStyle} />
              </Tooltip>
            </Badge>
          </Popover>
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
              e.currentTarget.style.backgroundColor = 'var(--ant-color-bg-container)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.backgroundColor = 'transparent';
            }}
          >
            <Avatar src={userInfo?.avatar} icon={<UserOutlined />} size="small" />
            <span
              className="hidden md:inline"
              style={{
                fontSize: 13,
                fontWeight: 500,
                color: 'var(--ant-color-text)',
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
