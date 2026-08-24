import { useMemo, useState, useEffect, useCallback } from 'react';
import { Outlet, useLocation, useMatches } from 'react-router-dom';
import { ConfigProvider } from 'antd';

// 内部组件
import HeaderContent from './components/HeaderContent';
import SiderMenu from './components/SiderMenu';
import TabsBar from './components/TabsBar';
import AppFooter from './components/Footer';
import { PageContainer } from '@/layouts/components/PageContainer';

// Hooks
import { useMenuData } from './hooks/useMenuData';
import { useLayoutState } from './hooks/useLayoutState';

// Stores & Preferences
import { useUserStore, useAuthStore, usePageRefreshStore } from '@/stores';
import { useAccess } from '@/core/access';
import { usePreferencesStore } from '@/core/preferences/store';
import { useThemeConfig } from '@/core/preferences/hooks/useThemeConfig';
import { PreferencesPanel } from '@/core/preferences/components';

import { allRoutes } from '@/router';
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

  // 权限
  const { getAllPermissions } = useAccess();

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
  const {
    collapsed: _collapsed,
    setCollapsed: _setCollapsed,
    isMobile,
    setIsMobile,
    openKeys,
    setOpenKeys,
  } = useLayoutState();

  // 使用 preferences.sidebar.hidden 控制侧边栏显示/隐藏
  const sidebarHidden = preferences.sidebar?.hidden ?? false;
  const setSidebarHidden = useCallback(
    (hidden: boolean) => {
      setPreferences({ sidebar: { hidden } });
    },
    [setPreferences],
  );

  // 菜单数据
  const permissions = useMemo(() => getAllPermissions(), [getAllPermissions]);
  const menuData = useMenuData({
    staticRoutes: allRoutes,
    dynamicRoutes,
    permissions,
  });

  // 全屏状态
  const [isFullscreen, setIsFullscreen] = useState(false);

  // 监听 fullscreenchange：isFullscreen 必须跟随浏览器真实全屏状态，
  // 而非仅由按钮点击乐观设置。用户按 Esc 或通过浏览器 UI 退出全屏时
  // 不会触发按钮点击，若不监听此事件，图标状态会与真实状态脱节。
  // requestFullscreen() 的 Promise 也可能 reject（如 iframe 未授权），
  // 监听事件可一并修正乐观置 true 的错误状态。
  useEffect(() => {
    const handler = () => setIsFullscreen(Boolean(document.fullscreenElement));
    document.addEventListener('fullscreenchange', handler);
    return () => document.removeEventListener('fullscreenchange', handler);
  }, []);

  // 设置面板
  const [settingsOpen, setSettingsOpen] = useState(false);

  // 页面刷新
  const triggerPageRefresh = usePageRefreshStore((state) => state.triggerRefresh);
  const refreshTrigger = usePageRefreshStore((state) => state.refreshTrigger);

  // 窗口大小监听
  useEffect(() => {
    const checkMobile = () => {
      const mobile = window.innerWidth < 768;
      setIsMobile(mobile);
      if (mobile) {
        setPreferences({ sidebar: { hidden: true } });
      }
    };

    checkMobile();
    window.addEventListener('resize', checkMobile);
    return () => window.removeEventListener('resize', checkMobile);
  }, [setIsMobile, setPreferences]);

  // 根据当前路径计算选中的菜单项
  const selectedKeys = useMemo(() => {
    return [location.pathname];
  }, [location.pathname]);

  // 根据当前路径和菜单数据计算应该展开的菜单项
  useEffect(() => {
    const calculateOpenKeys = (
      menuItems: any[],
      targetPath: string,
      parentKeys: string[] = [],
      parentPath: string = '',
    ): string[] => {
      for (const item of menuItems) {
        // 处理相对路径：拼接父路径
        const itemPath = item.path?.startsWith('/')
          ? item.path
          : `${parentPath}/${item.path}`.replace(/\/+/g, '/');
        const itemKey = item.key || itemPath;

        // 如果当前项就是目标路径，返回所有父级 key
        if (itemPath === targetPath || itemKey === targetPath) {
          return parentKeys;
        }
        // 如果有子菜单，递归查找
        if (item.children && item.children.length > 0) {
          const found = calculateOpenKeys(
            item.children,
            targetPath,
            [...parentKeys, itemPath],
            itemPath,
          );
          if (found.length > 0) {
            return found;
          }
          // 如果子项中直接匹配到目标路径，也要展开当前项
          if (
            item.children.some((child: any) => {
              const childPath = child.path?.startsWith('/')
                ? child.path
                : `${itemPath}/${child.path}`.replace(/\/+/g, '/');
              return childPath === targetPath || child.key === targetPath;
            })
          ) {
            return [...parentKeys, itemPath];
          }
        }
      }
      return [];
    };

    const openKeys = calculateOpenKeys(menuData, location.pathname);
    setOpenKeys(openKeys);
  }, [location.pathname, menuData, setOpenKeys]);

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
        collapsed={sidebarHidden}
        isFullscreen={isFullscreen}
        onToggleCollapse={() => setSidebarHidden(!sidebarHidden)}
        onRefresh={triggerPageRefresh}
        onToggleFullscreen={() => {
          // 不在此处乐观 setIsFullscreen：全屏状态的更新统一由上面的
          // fullscreenchange 监听器处理，确保始终与浏览器真实状态一致。
          // （requestFullscreen/exitFullscreen 的 Promise 若 reject，监听器也不会误置）
          if (!document.fullscreenElement) {
            document.documentElement.requestFullscreen().catch(() => {
              /* 进入全屏被拒绝（如 iframe 未授权），忽略 */
            });
          } else {
            document.exitFullscreen().catch(() => {});
          }
        }}
        onLogout={logout}
        isDark={isDark}
        onToggleTheme={toggleTheme}
        onOpenSettings={() => setSettingsOpen(true)}
        widgetConfig={{
          fullscreen: preferences.widget?.fullscreen ?? true,
          globalSearch: preferences.widget?.globalSearch ?? true,
          languageToggle: preferences.widget?.languageToggle ?? true,
          notification: preferences.widget?.notification ?? true,
          themeToggle: preferences.widget?.themeToggle ?? true,
          refresh: preferences.widget?.refresh ?? true,
          sidebarToggle: preferences.widget?.sidebarToggle ?? true,
        }}
      />
    );
  }, [userInfo, sidebarHidden, isFullscreen, logout, isDark, setPreferences, triggerPageRefresh, preferences.widget]);

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
          background: 'var(--ant-color-bg-layout)',
        }}
      >
        {/* ===== 左侧：侧边栏 ===== */}
        <SiderMenu
          menuData={menuData}
          isMobile={isMobile}
          isDark={isDark}
          openKeys={openKeys}
          selectedKeys={selectedKeys}
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
              backgroundColor: 'var(--app-bg-nav, var(--ant-color-bg-container))',
              borderBottom: '1px solid var(--ant-color-border)',
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
              overflow: 'hidden', // 禁止外层滚动，让内容区域自己管理滚动
              backgroundColor: 'var(--ant-color-bg-layout)',
              // padding: '8px', // 移除 padding，让内容占满整个区域
              display: 'flex',
              flexDirection: 'column',
            }}
          >
            <PageContainer
              ghost={true}
              route={{
                meta: {
                  title: matches.at(-1)?.handle?.title || '',
                  icon: undefined, // 明确禁止显示图标
                },
              }}
              breadcrumb={false} // 禁用面包屑（顶部栏已有）
              header={false}
              contentPadding={false}
              style={{
                flex: 1, // 让 PageContainer 占满父容器
                display: 'flex',
                flexDirection: 'column',
                minHeight: 0, // 防止 flex 子项溢出
                padding: '0',
                background: 'transparent',
              }}
            >
              {/* 使用 refreshTrigger 作为 key，触发页面重新渲染 */}
              <Outlet key={refreshTrigger} />
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
