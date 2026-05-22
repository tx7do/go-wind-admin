import { useMemo } from 'react';
import { Avatar, Dropdown, Badge, Tooltip, Button, Breadcrumb, Input, Popover } from 'antd';
import type { MenuProps } from 'antd';
import {
  LogoutOutlined,
  UserOutlined,
  SettingOutlined,
  BellOutlined,
  FullscreenOutlined,
  FullscreenExitOutlined,
  MoonOutlined,
  SunOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  ReloadOutlined,
  GlobalOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import { useMatches, useNavigate } from 'react-router-dom';
import { useI18n } from '@/core/i18n';
import { usePreferencesStore } from '@/core/preferences/store';

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
}: HeaderContentProps) => {
  const { t } = useI18n('common');
  const navigate = useNavigate();
  const matches = useMatches();
  const preferences = usePreferencesStore((state) => state.preferences);
  const locale = preferences.app.locale;

  // 计算面包屑
  const breadcrumbItems = useMemo(() => {
    type MatchWithHandle = { pathname: string; handle?: { title?: string; icon?: string } };
    const typedMatches = matches as MatchWithHandle[];

    const items = typedMatches
      .filter((match) => match.handle?.title)
      .map((match, index, arr) => ({
        key: match.pathname,
        title: match.handle?.title || '',
        onClick:
          index < arr.length - 1
            ? () => {
                navigate(match.pathname);
              }
            : undefined,
      }));

    // 如果第一个不是首页，添加首页
    if (items.length && items[0].key !== '/') {
      items.unshift({
        key: '/',
        title: t('home'),
        onClick: () => navigate('/'),
      });
    }

    return items;
  }, [matches, navigate, t]);

  // 语言切换
  const toggleLocale = () => {
    const newLocale = locale === 'zh-CN' ? 'en-US' : 'zh-CN';
    const { setPreferences } = usePreferencesStore.getState();
    setPreferences({ app: { locale: newLocale } });
  };

  // 用户菜单
  const userMenuItems: MenuProps['items'] = [
    {
      key: 'profile',
      icon: <UserOutlined />,
      label: t('header.profile'),
      onClick: () => {
        /* 跳转个人中心 */
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
        <Tooltip title={collapsed ? t('header.expandSidebar') : t('header.collapseSidebar')}>
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={onToggleCollapse}
            size="small"
            style={btnStyle}
          />
        </Tooltip>

        {/* 刷新按钮 */}
        <Tooltip title={t('header.refresh')}>
          <Button
            type="text"
            icon={<ReloadOutlined />}
            onClick={onRefresh}
            size="small"
            style={btnStyle}
          />
        </Tooltip>

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
                }}
              >
                {item.title}
              </a>
            ) : (
              <span
                style={{
                  color: isDark ? '#ffffff' : '#262626',
                  fontSize: 13,
                  fontWeight: 500,
                }}
              >
                {item.title}
              </span>
            ),
          }))}
        />
      </div>

      {/* ========== 右侧区域：搜索 + 设置 + 主题 + 语言 + 全屏 + 通知 + 头像 ========== */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 2, flexShrink: 0 }}>
        {/* 搜索框 */}
        <Tooltip title={t('header.search')}>
          <Popover
            trigger="click"
            placement="bottomRight"
            content={
              <div style={{ width: 280, padding: 4 }}>
                <Input.Search
                  placeholder={t('header.searchPlaceholder')}
                  onSearch={(value) => {
                    console.log('Search:', value);
                    // 后续可实现全局搜索逻辑
                  }}
                />
              </div>
            }
          >
            <Button
              type="text"
              icon={<SearchOutlined />}
              size="small"
              style={btnStyle}
            />
          </Popover>
        </Tooltip>

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
        <Tooltip title={isDark ? t('header.switchToLight') : t('header.switchToDark')}>
          <Button
            type="text"
            icon={isDark ? <SunOutlined /> : <MoonOutlined />}
            onClick={onToggleTheme}
            size="small"
            style={btnStyle}
          />
        </Tooltip>

        {/* 语言切换 */}
        <Tooltip title={t('header.switchLanguage')}>
          <Button
            type="text"
            icon={<GlobalOutlined />}
            onClick={toggleLocale}
            size="small"
            style={btnStyle}
          />
        </Tooltip>

        {/* 全屏切换 */}
        <Tooltip title={isFullscreen ? t('header.exitFullscreen') : t('header.fullscreen')}>
          <Button
            type="text"
            icon={isFullscreen ? <FullscreenExitOutlined /> : <FullscreenOutlined />}
            onClick={onToggleFullscreen}
            size="small"
            style={btnStyle}
          />
        </Tooltip>

        {/* 通知 */}
        <Badge count={3} size="small" offset={[0, 4]}>
          <Tooltip title={t('header.notification')}>
            <Button
              type="text"
              icon={<BellOutlined />}
              size="small"
              style={btnStyle}
            />
          </Tooltip>
        </Badge>

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
            <Avatar src={userInfo?.avatar} icon={<UserOutlined />} size="small" />
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
