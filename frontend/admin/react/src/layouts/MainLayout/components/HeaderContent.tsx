import { useEffect, useMemo, useRef, useState } from 'react';
import React from 'react';
import { Avatar, Dropdown, Badge, Tooltip, Button, Breadcrumb, Input, Popover, Modal, Empty, Spin, App as AntdApp } from 'antd';
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
  CheckOutlined,
  CloseOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import 'dayjs/locale/zh-cn';
import { useMatches, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

import { getIconFromName } from '@/layouts/MainLayout/utils/iconResolver';

import { useI18n } from '@/core/i18n';
import { usePreferencesStore } from '@/core/preferences/store';
import type { SupportedLanguagesType } from '@/core/preferences/types/layout';
import { fetchListUserInbox, } from '@/api/hooks/internal-message';
import { apiClient } from '@/api/client';
import { useQuery } from '@tanstack/react-query';
import { PaginationQuery, queryClient } from '@/core';
import { globalSSEClient, SSE_EVENT } from '@/core/transport/sse';

dayjs.extend(relativeTime);

/** 键位提示条的小 kbd 徽标（与触发条上的 Ctrl K 同族样式） */
const searchHintKbdStyle: React.CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
  minWidth: 18,
  padding: '1px 5px',
  fontSize: 11,
  fontFamily: 'monospace',
  lineHeight: 1.4,
  color: 'var(--ant-color-text-secondary)',
  backgroundColor: 'var(--ant-color-bg-container)',
  border: '1px solid var(--ant-color-border)',
  borderRadius: 4,
};

/** 全局搜索结果行（形态对齐 vben SearchPanel：图标 + 标题，选中主色实底白字） */
const SearchRow = ({
  iconNode,
  title,
  active,
  onMouseEnter,
  onClick,
  onRemove,
}: {
  iconNode?: React.ReactNode;
  title: string;
  active: boolean;
  onMouseEnter: () => void;
  onClick: () => void;
  onRemove?: () => void;
}) => (
  <div
    className="semantic-search-item"
    onMouseEnter={onMouseEnter}
    onClick={onClick}
    style={{
      display: 'flex',
      alignItems: 'center',
      gap: 10,
      padding: '8px 10px',
      borderRadius: 6,
      cursor: 'pointer',
      ...(active ? { backgroundColor: 'var(--ant-color-primary)' } : {}),
    }}
  >
    {iconNode && (
      <span
        style={{
          display: 'inline-flex',
          fontSize: 14,
          flexShrink: 0,
          color: active ? 'var(--ant-color-white)' : 'var(--ant-color-text-secondary)',
        }}
      >
        {iconNode}
      </span>
    )}
    <span
      style={{
        flex: 1,
        minWidth: 0,
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
        fontSize: 13,
        ...(active ? { color: 'var(--ant-color-white)' } : {}),
      }}
    >
      {title}
    </span>
    {onRemove && (
      <span
        role="button"
        tabIndex={-1}
        onClick={(e) => {
          e.stopPropagation();
          onRemove();
        }}
        style={{
          display: 'inline-flex',
          padding: 3,
          fontSize: 11,
          lineHeight: 1,
          borderRadius: 4,
          color: active ? 'var(--ant-color-white)' : 'var(--ant-color-text-secondary)',
          backgroundColor: active ? 'transparent' : 'transparent',
        }}
        onMouseEnter={(e) => {
          e.currentTarget.style.backgroundColor = active
            ? 'color-mix(in srgb, var(--ant-color-white) 20%, transparent)'
            : 'var(--ant-color-fill-tertiary)';
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.backgroundColor = 'transparent';
        }}
      >
        <CloseOutlined />
      </span>
    )}
  </div>
);

/** 通知列表的相对时间（如"5 分钟前"）；跟随界面语言 */
const fromNow = (value?: string, locale?: string): string => {
  if (!value) return '-';
  const d = dayjs(value);
  if (!d.isValid()) return '-';
  return locale?.startsWith('zh') ? d.locale('zh-cn').fromNow() : d.fromNow();
};

interface HeaderContentProps {
  userInfo: BasicUserInfo | null;
  collapsed: boolean;
  isFullscreen: boolean;
  onToggleCollapse: () => void;
  onRefresh: () => void;
  onToggleFullscreen: () => void;
  onLogout: () => void;
  isDark: boolean;
  onToggleTheme: (event?: React.MouseEvent<HTMLElement>) => void;
  onOpenSettings: () => void;
  menuData: any[];
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
  menuData,
  widgetConfig,
}: HeaderContentProps) => {
  const { t } = useI18n('common');
  const { t: tRoutes, i18n } = useTranslation('routes'); // 用于路由翻译（路由标题走 routes 命名空间，含后端菜单种子键 menu.*/page.*）
  const navigate = useNavigate();
  const matches = useMatches();

  // 面包屑偏好设置
  const breadcrumbPreferences = usePreferencesStore((state) => state.preferences.breadcrumb);
  const breadcrumbStyleType = breadcrumbPreferences?.styleType ?? 'normal';

  // 默认头像（与 ele/vben 三端统一，取 preferences.app.defaultAvatar）
  const defaultAvatar = usePreferencesStore((state) => state.preferences.app.defaultAvatar);

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
  const { message } = AntdApp.useApp();
  const [markAllReadLoading, setMarkAllReadLoading] = useState(false);
  const appLocale = usePreferencesStore((state) => state.preferences.app.locale);

  // 未读数（独立 COUNT）：badge 数字来源
  const { data: inboxUnread } = useQuery({
    queryKey: ['inboxPreview', userInfo?.id],
    queryFn: async () => {
      const query = new PaginationQuery({
        paging: { page: 1, pageSize: 1 },
        // recipient_user_id 必传：不传只靠租户过滤，会查到同租户其他用户的收件记录
        formValues: { recipient_user_id: String(userInfo?.id), status: 'RECEIVED' },
      });
      return await fetchListUserInbox(query);
    },
    enabled: Boolean(userInfo?.id),
    // 兜底轮询：实时更新靠下方 SSE notification 事件触发 invalidate，这里只在 SSE 失联时补拉
    refetchInterval: 300_000,
    refetchIntervalInBackground: false,
  });
  // 未读数用列表响应的服务端 total（独立 COUNT），不能用当前页条数——后者封顶在 pageSize
  const unreadCount = inboxUnread?.total ?? 0;

  // 弹层列表：最近 5 条（已读+未读混合），未读带圆点加粗
  const { data: inboxRecent, isLoading: recentLoading } = useQuery({
    queryKey: ['inboxPreviewList', userInfo?.id],
    queryFn: async () => {
      const query = new PaginationQuery({
        paging: { page: 1, pageSize: 5 },
        formValues: { recipient_user_id: String(userInfo?.id) },
      });
      return await fetchListUserInbox(query);
    },
    enabled: Boolean(userInfo?.id),
    refetchInterval: 300_000,
    refetchIntervalInBackground: false,
  });
  const recentItems = inboxRecent?.items ?? [];

  // SSE 实时通知：收到新站内信即刷新预览。
  // 后端一直在推 notification 事件，此前前端建了连接却无人订阅，实时推送全部落空。
  useEffect(() => {
    if (!userInfo?.id) return undefined;
    const handler = () => {
      queryClient.invalidateQueries({ queryKey: ['inboxPreview', userInfo?.id] });
      queryClient.invalidateQueries({ queryKey: ['inboxPreviewList', userInfo?.id] });
    };
    globalSSEClient.on(SSE_EVENT.Notification, handler);
    return () => {
      globalSSEClient.off(SSE_EVENT.Notification, handler);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [userInfo?.id]);

  const markAllRead = async () => {
    if (!userInfo?.id) return;
    setMarkAllReadLoading(true);
    try {
      // 后端约定：recipientIds 为空 = 该用户全部未读
      await apiClient.internalMessageRecipientService.MarkNotificationAsRead({
        userId: userInfo.id,
        recipientIds: [],
      });
      message.success(tInbox('markAllReadSuccess'));
      queryClient.invalidateQueries({ queryKey: ['inboxPreview', userInfo?.id] });
      queryClient.invalidateQueries({ queryKey: ['inboxPreviewList', userInfo?.id] });
      queryClient.invalidateQueries({ queryKey: ['listUserInbox'] });
    } catch (error) {
      message.error((error as Error)?.message || tInbox('markAllReadFailed'));
    } finally {
      setMarkAllReadLoading(false);
    }
  };

  const inboxContent = (
    <div className="notification-popover" style={{ width: 336 }}>
      {/* 头部：标题 + 未读数 + 全部已读 */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '2px 2px 10px',
          borderBottom: '1px solid var(--ant-color-border)',
          marginBottom: 4,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 14, fontWeight: 600 }}>
          {tInbox('pageTitle')}
          {unreadCount > 0 && (
            <span
              style={{
                fontSize: 12,
                fontWeight: 400,
                lineHeight: '18px',
                padding: '0 8px',
                borderRadius: 10,
                color: 'var(--ant-color-primary)',
                background: 'rgba(0, 107, 230, 0.12)',
              }}
            >
              {tInbox('unreadCount', { count: unreadCount })}
            </span>
          )}
        </div>
        <Button
          type="text"
          size="small"
          icon={<CheckOutlined />}
          loading={markAllReadLoading}
          disabled={unreadCount <= 0}
          onClick={markAllRead}
        >
          {tInbox('markAllRead')}
        </Button>
      </div>

      <Spin spinning={recentLoading}>
        {recentItems.length > 0 ? (
          <div>
            {recentItems.map((item, index) => {
              const unread = item.status === 'RECEIVED';
              const preview = (item.content || '')
                .replace(/<[^>]*>/g, ' ')
                .replace(/\s+/g, ' ')
                .trim()
                .slice(0, 46);
              return (
                <div
                  key={item.id ?? index}
                  className="notification-item"
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    padding: '9px 8px',
                    borderRadius: 8,
                    cursor: 'pointer',
                    gap: 10,
                  }}
                  onClick={() => navigate('/internal-message/inbox')}
                >
                  <span
                    className="notification-item__dot"
                    style={{
                      width: 6,
                      height: 6,
                      borderRadius: '50%',
                      flexShrink: 0,
                      marginTop: 6,
                      background: unread ? 'var(--ant-color-primary)' : 'transparent',
                    }}
                  />
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div
                      style={{
                        fontSize: 13,
                        fontWeight: unread ? 600 : 400,
                        color: unread
                          ? 'var(--ant-color-text)'
                          : 'var(--ant-color-text-secondary)',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                      }}
                    >
                      {item.title || '-'}
                    </div>
                    {preview && (
                      <div
                        style={{
                          fontSize: 12,
                          color: 'var(--ant-color-text-secondary)',
                          overflow: 'hidden',
                          textOverflow: 'ellipsis',
                          whiteSpace: 'nowrap',
                          marginTop: 2,
                        }}
                      >
                        {preview}
                      </div>
                    )}
                  </div>
                  <span
                    style={{
                      fontSize: 12,
                      color: 'var(--ant-color-text-secondary)',
                      flexShrink: 0,
                    }}
                  >
                    {fromNow(item.createdAt, appLocale)}
                  </span>
                </div>
              );
            })}
            </div>
        ) : (
          <Empty
            description={tInbox('noMessages')}
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
        <Button type="link" size="small" onClick={() => navigate('/internal-message/inbox')}>
          {tInbox('goToInbox')}
        </Button>
      </div>
    </div>
  );

  // ── 全局搜索（本地菜单匹配 + 语义搜索；视觉形态对齐 vben SearchPanel，
  // 规格见 docs/design-language.md「全局搜索面板」条）──
  const [semanticSearchOpen, setSemanticSearchOpen] = useState(false);
  const [semanticQuery, setSemanticQuery] = useState('');
  const [semanticResults, setSemanticResults] = useState<{title: string; route: string}[]>([]);
  const [semanticLoading, setSemanticLoading] = useState(false);
  const [activeIndex, setActiveIndex] = useState(-1);
  const [searchHistory, setSearchHistory] = useState<{title: string; path: string; icon?: string}[]>([]);
  const semanticTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  const SEARCH_HISTORY_KEY = 'menu_search_history';
  const SEARCH_HISTORY_MAX = 5;

  // 菜单标题翻译：与面包屑（tRoutes(title)）同语义——i18next 对 'routes:xxx' /
  // 'menu:xxx' 形态的 key 自动按「命名空间:键」解析，无需剥前缀
  const resolveMenuTitle = (label: string | undefined): string =>
    label ? tRoutes(label, { defaultValue: label }) : '';

  // 扁平化菜单树为可搜索的叶子页列表（i18n 相关：语言切换时重建）
  const searchableMenus = useMemo(() => {
    const excluded = ['/login', '/401', '/403', '/404', '/500'];
    const out: { title: string; path: string; icon?: string }[] = [];
    const walk = (items: any[]) => {
      for (const item of items) {
        if (item.children?.length) {
          walk(item.children);
          continue;
        }
        const raw = item.name || item.label;
        if (!item.path || !raw || excluded.includes(item.path)) continue;
        if (out.some((o) => o.path === item.path)) continue;
        out.push({
          title: resolveMenuTitle(raw),
          path: item.path,
          icon: typeof item.icon === 'string' ? item.icon : undefined,
        });
      }
    };
    walk(menuData);
    return out;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [menuData, i18n.language]);

  // 本地关键字匹配（contains，标题或路径）
  const localResults = useMemo(() => {
    const kw = semanticQuery.trim().toLowerCase();
    if (!kw) return [];
    return searchableMenus.filter(
      (item) => item.title.toLowerCase().includes(kw) || item.path.toLowerCase().includes(kw),
    );
  }, [semanticQuery, searchableMenus]);

  // 键盘导航作用于「与展示一致」的合并列表：有关键词=本地+语义，无关键词=历史
  const combinedResults = useMemo(
    () => (semanticQuery.trim() ? [...localResults, ...semanticResults] : searchHistory),
    [semanticQuery, localResults, semanticResults, searchHistory],
  );

  const loadSearchHistory = () => {
    try {
      const raw = localStorage.getItem(SEARCH_HISTORY_KEY);
      setSearchHistory(raw ? JSON.parse(raw) : []);
    } catch (error) {
      console.error('load search history failed:', error);
      setSearchHistory([]);
    }
  };

  useEffect(() => {
    loadSearchHistory();
  }, []);

  const saveSearchHistory = (list: {title: string; path: string; icon?: string}[]) => {
    localStorage.setItem(SEARCH_HISTORY_KEY, JSON.stringify(list));
    setSearchHistory(list);
  };

  const removeSearchHistory = (index: number) => {
    const next = [...searchHistory];
    next.splice(index, 1);
    saveSearchHistory(next);
  };

  const closeSearch = () => {
    setSemanticSearchOpen(false);
    setSemanticResults([]);
    setSemanticQuery('');
    setActiveIndex(-1);
  };

  const handleSemanticSearch = (query: string) => {
    setSemanticQuery(query);
    setActiveIndex(-1);
    clearTimeout(semanticTimer.current);
    if (!query.trim()) { setSemanticResults([]); return; }
    semanticTimer.current = setTimeout(async () => {
      setSemanticLoading(true);
      try {
        const resp = await apiClient.aiContentService.SemanticSearch({ query: query.trim(), limit: 8 });
        setSemanticResults(
          (resp.items ?? [])
            .filter((item) => item?.route)
            .map((item) => ({
              title: item.title ?? item.route ?? '',
              route: item.route ?? '',
            })),
        );
      } catch (error) {
        // 不吞错：带出原始错误对象；语义搜索失败降级为仅本地结果
        console.error('semantic search failed:', error);
        setSemanticResults([]);
      } finally { setSemanticLoading(false); }
    }, 500);
  };

  const goSearchItem = (item: { path?: string; route?: string; title?: string; icon?: string }) => {
    const path = item.path || item.route;
    if (!path) return;
    // 写入最近搜索（去重置顶，上限 5，语义命中按 route 收敛成菜单项形态）
    const matched = searchableMenus.find((m) => m.path === path);
    const entry = {
      title: item.title || matched?.title || path,
      path,
      icon: matched?.icon,
    };
    const next = [entry, ...searchHistory.filter((h) => h.path !== path)].slice(0, SEARCH_HISTORY_MAX);
    saveSearchHistory(next);
    closeSearch();
    navigate(path);
  };

  // Ctrl+K / Cmd+K 呼出全局搜索（偏好里可关：shortcutKeys.enable + shortcutKeys.globalSearch）
  const shortcutEnabled = usePreferencesStore((s) => s.preferences.shortcutKeys?.enable ?? true);
  const searchShortcutEnabled = usePreferencesStore((s) => s.preferences.shortcutKeys?.globalSearch ?? true);
  useEffect(() => {
    if (!widgetConfig.globalSearch || !shortcutEnabled || !searchShortcutEnabled) return undefined;
    const handler = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setSemanticSearchOpen((open) => {
          if (!open) {
            loadSearchHistory();
            setSemanticResults([]);
            setSemanticQuery('');
            setActiveIndex(-1);
          }
          return !open;
        });
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [widgetConfig.globalSearch, shortcutEnabled, searchShortcutEnabled]);

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
          <>
            <Modal
              open={semanticSearchOpen}
              onCancel={closeSearch}
              footer={null}
              width={600}
              closable={false}
              centered
              destroyOnHidden
              styles={{ body: { padding: 0 } }}
            >
              {/* 头部：搜索图标 + 无边框大输入 */}
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 10,
                  padding: '10px 16px',
                  borderBottom: '1px solid var(--ant-color-border-secondary)',
                }}
              >
                <SearchOutlined style={{ color: 'var(--ant-color-text-secondary)', fontSize: 16 }} />
                <Input
                  variant="borderless"
                  className="global-search-input"
                  placeholder={t('header.searchPlaceholder')}
                  value={semanticQuery}
                  onChange={(e) => handleSemanticSearch(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.nativeEvent.isComposing) return;
                    const list = combinedResults;
                    if (e.key === 'ArrowDown') {
                      e.preventDefault();
                      if (list.length) setActiveIndex((i) => (i >= list.length - 1 ? 0 : i + 1));
                    } else if (e.key === 'ArrowUp') {
                      e.preventDefault();
                      if (list.length) setActiveIndex((i) => (i <= 0 ? list.length - 1 : i - 1));
                    } else if (e.key === 'Enter') {
                      e.preventDefault();
                      const target = list[activeIndex >= 0 ? activeIndex : 0];
                      if (target) goSearchItem(target);
                    }
                  }}
                  autoFocus
                  allowClear
                  style={{ fontSize: 15 }}
                />
              </div>

              {/* 结果区（限高内滚） */}
              <div style={{ maxHeight: 450, overflowY: 'auto', padding: '8px' }}>
                {/* 无关键词：最近搜索 */}
                {!semanticQuery.trim() && searchHistory.length === 0 && (
                  <div style={{ textAlign: 'center', padding: '24px 0', color: 'var(--ant-color-text-secondary)', fontSize: 13 }}>
                    {t('header.noHistory')}
                  </div>
                )}
                {!semanticQuery.trim() && searchHistory.length > 0 && (
                  <>
                    <div style={{ fontSize: 12, color: 'var(--ant-color-text-secondary)', padding: '4px 10px 6px' }}>
                      {t('header.searchRecent')}
                    </div>
                    {searchHistory.map((item, i) => (
                      <SearchRow
                        key={item.path}
                        iconNode={item.icon ? getIconFromName(item.icon) : undefined}
                        title={item.title}
                        active={activeIndex === i}
                        onMouseEnter={() => setActiveIndex(i)}
                        onClick={() => goSearchItem(item)}
                        onRemove={() => removeSearchHistory(i)}
                      />
                    ))}
                  </>
                )}

                {/* 有关键词：本地菜单命中 */}
                {semanticQuery.trim() && !semanticLoading && localResults.length > 0 && (
                  <>
                    {localResults.map((item, i) => (
                      <SearchRow
                        key={item.path}
                        iconNode={item.icon ? getIconFromName(item.icon) : undefined}
                        title={item.title}
                        active={activeIndex === i}
                        onMouseEnter={() => setActiveIndex(i)}
                        onClick={() => goSearchItem(item)}
                      />
                    ))}
                  </>
                )}

                {/* 有关键词：语义搜索小节（独立分区，失败降级不弹错） */}
                {semanticQuery.trim() && (semanticLoading || semanticResults.length > 0) && (
                  <>
                    <div style={{ fontSize: 12, color: 'var(--ant-color-text-secondary)', padding: '6px 10px 6px', marginTop: localResults.length > 0 ? 6 : 0 }}>
                      {t('header.semanticTitle')}
                    </div>
                    {semanticLoading && (
                      <div style={{ textAlign: 'center', padding: '12px 0', color: 'var(--ant-color-text-secondary)', fontSize: 13 }}>
                        {t('header.searching')}
                      </div>
                    )}
                    {!semanticLoading && semanticResults.map((item, i) => (
                      <SearchRow
                        key={`${item.route}-${i}`}
                        title={item.title}
                        active={activeIndex === localResults.length + i}
                        onMouseEnter={() => setActiveIndex(localResults.length + i)}
                        onClick={() => goSearchItem(item)}
                      />
                    ))}
                  </>
                )}

                {/* 有关键词且无任何结果 */}
                {semanticQuery.trim() && !semanticLoading && combinedResults.length === 0 && (
                  <div style={{ textAlign: 'center', padding: '24px 0', color: 'var(--ant-color-text-secondary)', fontSize: 13 }}>
                    {t('header.noResults')}
                  </div>
                )}
              </div>

              {/* 底部键位提示条 */}
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 16,
                  padding: '8px 16px',
                  borderTop: '1px solid var(--ant-color-border-secondary)',
                  fontSize: 12,
                  color: 'var(--ant-color-text-secondary)',
                }}
              >
                <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5 }}>
                  <kbd style={searchHintKbdStyle}>↵</kbd>
                  {t('header.searchSelect')}
                </span>
                <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5 }}>
                  <kbd style={searchHintKbdStyle}>↑</kbd>
                  <kbd style={searchHintKbdStyle}>↓</kbd>
                  {t('header.searchSwitch')}
                </span>
                <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5 }}>
                  <kbd style={searchHintKbdStyle}>ESC</kbd>
                  {t('header.searchClose')}
                </span>
              </div>
            </Modal>

            <div
              className="search-trigger-btn"
              onClick={() => {
                loadSearchHistory();
                setActiveIndex(-1);
                setSemanticSearchOpen(true);
              }}
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
          </>
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
            styles={{ body: { padding: '10px 10px 6px' } } as any}
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
            <Avatar src={userInfo?.avatar || defaultAvatar || undefined} icon={<UserOutlined />} size="small" />
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
