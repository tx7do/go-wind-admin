import { useMemo, useState, useEffect, useCallback } from 'react';
import { Outlet, useLocation, useMatches } from 'react-router-dom';
import { PageContainer } from '@ant-design/pro-components';
import { ConfigProvider } from 'antd';

// 内部组件
import HeaderContent from './components/HeaderContent';
import SiderMenu from './components/SiderMenu';
import TabsBar from './components/TabsBar';
import AppFooter from './components/Footer';

// Hooks
import { useMenuData } from './hooks/useMenuData';
import { useLayoutState } from './hooks/useLayoutState';

// Stores & Preferences
import { useUserStore, useAuthStore } from '@/stores';
import { usePreferencesStore } from '@/core/preferences/store';
import { useThemeConfig } from '@/core/preferences/hooks/useThemeConfig';
import { PreferencesPanel } from '@/core/preferences/components';

import { staticRoutes } from '@/router/config/static';
import type { AppRouteObject } from '@/core/router/types';

interface LayoutRouteHandle {
  title?: string;
  icon?: string;
}

interface LayoutRouteMatch {
  pathname: string;
  handle: LayoutRouteHandle;
}

interface MainLayoutProps {
  routes?: AppRouteObject[];
}

export const MainLayout = ({ routes: dynamicRoutes }: MainLayoutProps) => {
  const location = useLocation();
  const rawMatches = useMatches();
  const matches = rawMatches as LayoutRouteMatch[];

  // Stores
  const { userInfo } = useUserStore();
  const { logout } = useAuthStore();

  // Preferences
  const preferences = usePreferencesStore((state) => state.preferences);
  const setPreferences = usePreferencesStore((state) => state.setPreferences);

  // Theme
  const themeConfig = useThemeConfig();
  const [isDark, setIsDark] = useState(() => {
    const { theme } = preferences;
    if (theme.mode === 'dark') return true;
    if (theme.mode === 'light') return false;
    return window.matchMedia('(prefers-color-scheme: dark)').matches;
  });

  // 监听系统主题变化
  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handler = () => {
      const { theme } = preferences;
      if (theme.mode === 'auto') {
        setIsDark(mediaQuery.matches);
      }
    };
    mediaQuery.addEventListener('change', handler);
    return () => mediaQuery.removeEventListener('change', handler);
  }, [preferences.theme.mode]);

  // 监听偏好设置变化
  useEffect(() => {
    const { theme } = preferences;
    if (theme.mode === 'dark') setIsDark(true);
    else if (theme.mode === 'light') setIsDark(false);
    else setIsDark(window.matchMedia('(prefers-color-scheme: dark)').matches);
  }, [preferences.theme.mode]);

  // 布局状态
  const { collapsed, setCollapsed, isMobile, setIsMobile, openKeys, setOpenKeys } =
    useLayoutState();

  // 菜单数据
  const permissions = useMemo(() => userInfo?.permissions || [], [userInfo?.permissions]);
  const menuData = useMenuData({
    staticRoutes,
    dynamicRoutes,
    permissions,
  });

  // 全屏状态
  const [isFullscreen, setIsFullscreen] = useState(false);

  // 设置面板
  const [settingsOpen, setSettingsOpen] = useState(false);

  // 刷新 key（用于强制重渲染内容区）
  const [, setRefreshKey] = useState(0);

  const handleRefresh = useCallback(() => {
    setRefreshKey((k) => k + 1);
  }, []);

  // 窗口大小监听
  useEffect(() => {
    const checkMobile = () => {
      const mobile = window.innerWidth < 768;
      setIsMobile(mobile);
      if (mobile && !collapsed) setCollapsed(true);
    };

    checkMobile();
    window.addEventListener('resize', checkMobile);
    return () => window.removeEventListener('resize', checkMobile);
  }, [collapsed, setCollapsed, setIsMobile]);

  // 根据当前路径计算选中的菜单项
  const selectedKeys = useMemo(() => {
    return [location.pathname];
  }, [location.pathname]);

  // 顶栏右侧
  const headerContentRender = useCallback(() => {
    const toggleTheme = () => {
      setPreferences({
        theme: {
          mode: isDark ? 'light' : 'dark',
        },
      });
    };

    return (
      <HeaderContent
        userInfo={userInfo}
        collapsed={collapsed}
        isFullscreen={isFullscreen}
        onToggleCollapse={() => setCollapsed(!collapsed)}
        onRefresh={handleRefresh}
        onToggleFullscreen={() => {
          if (!document.fullscreenElement) {
            document.documentElement.requestFullscreen();
            setIsFullscreen(true);
          } else {
            document.exitFullscreen();
            setIsFullscreen(false);
          }
        }}
        onLogout={logout}
        isDark={isDark}
        onToggleTheme={toggleTheme}
        onOpenSettings={() => setSettingsOpen(true)}
      />
    );
  }, [userInfo, collapsed, isFullscreen, logout, isDark, setPreferences, handleRefresh]);

  // 主题切换
  useCallback(() => {
    setPreferences({
      theme: {
        mode: isDark ? 'light' : 'dark',
      },
    });
  }, [isDark, setPreferences]);
  return (
    <ConfigProvider theme={themeConfig}>
      <div
        style={{
          height: '100vh',
          display: 'flex',
          background: isDark ? '#000000' : '#f5f5f5',
        }}
      >
        {/* ===== 左侧：侧边栏 ===== */}
        <SiderMenu
          menuData={menuData}
          collapsed={collapsed}
          isMobile={isMobile}
          isDark={isDark}
          openKeys={openKeys}
          selectedKeys={selectedKeys}
          onCollapse={setCollapsed}
          onOpenChange={setOpenKeys}
        />

        {/* ===== 右侧：主内容区 ===== */}
        <div
          style={{
            flex: 1,
            display: 'flex',
            flexDirection: 'column',
            overflow: 'hidden',
          }}
        >
          {/* 顶部栏 */}
          <div
            style={{
              height: 56,
              backgroundColor: isDark ? '#141414' : '#ffffff',
              borderBottom: `1px solid ${isDark ? '#303030' : '#e5e7eb'}`,
              padding: '0 12px',
              flexShrink: 0,
              zIndex: 100,
            }}
          >
            {headerContentRender()}
          </div>

          {/* 标签栏 */}
          <TabsBar />

          {/* 内容区 */}
          <div
            style={{
              flex: 1,
              overflow: 'auto',
              backgroundColor: isDark ? '#000000' : '#f5f5f5',
            }}
          >
            <PageContainer
              ghost={true}
              route={{
                meta: {
                  title: matches.at(-1)?.handle?.title || '',
                },
              }}
              header={{
                title: matches.at(-1)?.handle?.title || '',
                breadcrumb: {},
              }}
              style={{
                padding: '24px',
                background: 'transparent',
              }}
            >
              <Outlet />
            </PageContainer>
          </div>

          {/* 页脚 */}
          {preferences.footer.enable && <AppFooter />}
        </div>
      </div>

      {/* 设置面板 */}
      <PreferencesPanel open={settingsOpen} onClose={() => setSettingsOpen(false)} />
    </ConfigProvider>
  );
};

export default MainLayout;
