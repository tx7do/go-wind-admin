import React, { useMemo, useState, useCallback, useEffect } from 'react';
import { Tabs, Dropdown, Button, Space, type MenuProps, type TabsProps } from 'antd';
import { useLocation, useNavigate, useMatches } from 'react-router-dom';
import {
  CloseOutlined,
  PushpinOutlined,
  PushpinFilled,
  FullscreenOutlined,
  FullscreenExitOutlined,
  ReloadOutlined,
  ExportOutlined,
  LeftOutlined,
  RightOutlined,
  CloseCircleOutlined,
  DownOutlined,
} from '@ant-design/icons';
import { usePreferencesStore } from '@/core/preferences/store';
import { useTabsStore } from '@/stores/tabs';
import { useI18n } from '@/core/i18n';
import { usePageRefreshStore } from '@/stores/pageRefresh';
import { getIconFromName } from '../../utils/iconResolver';
import './tabsbar.css';

interface TabItem {
  key: string;
  label: React.ReactNode; // 支持图标 + 文本
  closable?: boolean;
}

/**
 * TabsBar 组件 - 多标签页导航栏
 * 支持 4 种风格：brisk | card | chrome | plain
 */
export const Index = () => {
  const location = useLocation();
  const navigate = useNavigate();
  const matches = useMatches();
  const preferences = usePreferencesStore((state) => state.preferences);
  const tabbarConfig = preferences.tabbar;
  const widgetConfig = preferences.widget;

  // 计算实际的暗色模式（支持 auto 模式）
  const [isDark, setIsDark] = useState(() => {
    const { theme } = preferences;
    if (theme.mode === 'dark') return true;
    if (theme.mode === 'light') return false;
    // auto 模式：跟随系统
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

  const { t } = useI18n('common');

  // 页面刷新
  const triggerPageRefresh = usePageRefreshStore((state) => state.triggerRefresh);

  // 全屏状态管理
  const [isFullscreen, setIsFullscreen] = useState(false);

  // 切换全屏
  const toggleFullscreen = useCallback(() => {
    if (!document.fullscreenElement) {
      document.documentElement.requestFullscreen();
      setIsFullscreen(true);
    } else {
      document.exitFullscreen();
      setIsFullscreen(false);
    }
  }, []);

  // 监听全屏状态变化
  useEffect(() => {
    const handleFullscreenChange = () => {
      setIsFullscreen(!!document.fullscreenElement);
    };
    document.addEventListener('fullscreenchange', handleFullscreenChange);
    return () => {
      document.removeEventListener('fullscreenchange', handleFullscreenChange);
    };
  }, []);

  // 右键菜单状态
  const [contextMenuTabKey, setContextMenuTabKey] = useState<string>('');

  // 标签页管理
  const {
    tabs,
    addTab,
    closeTab,
    closeLeftTabs,
    closeRightTabs,
    closeOtherTabs,
    closeAllTabs,
    togglePinTab,
    reloadTab,
    reorderTabs,
  } = useTabsStore();

  // 获取当前风格对应的 CSS 类名
  const tabbarClassName = useMemo(() => {
    const baseClass = 'tabsbar-container';
    const styleClass = `tabsbar-${tabbarConfig.styleType}`;
    return `${baseClass} ${styleClass}`;
  }, [tabbarConfig.styleType]);

  // 根据 styleType 映射 antd Tabs type
  const tabsType = useMemo(() => {
    switch (tabbarConfig.styleType) {
      case 'chrome':
        return 'editable-card';
      case 'card':
        return 'card';
      case 'brisk':
      case 'plain':
        return 'line';
      default:
        return 'editable-card';
    }
  }, [tabbarConfig.styleType]);

  // 当前标签
  const currentTab = useMemo(() => {
    const lastMatch = matches.at(-1) as any;
    const title = lastMatch?.handle?.title || lastMatch?.data?.title || '未知页面';
    
    // 从 handle 中获取图标字符串（已绎通过 transformMetaToHandle 转换）
    const icon = lastMatch?.handle?.icon || lastMatch?.data?.icon;
    
    console.log('TabsBar - currentTab:', { title, icon, handle: lastMatch?.handle });
    
    return {
      key: location.pathname,
      path: location.pathname,
      title,
      icon, // 保持为字符串，后续通过 getIconFromName 转换
      closable: location.pathname !== '/',
    };
  }, [location.pathname, matches]);

  // 自动添加当前标签
  useEffect(() => {
    addTab(currentTab);
  }, [currentTab, addTab]);

  // 根据 persist 配置决定是否清除持久化数据
  useEffect(() => {
    if (!tabbarConfig.persist) {
      // 如果不启用持久化，清除存储的标签页
      const stored = localStorage.getItem('app-tabs');
      if (stored) {
        localStorage.removeItem('app-tabs');
        // 重新初始化 store
        useTabsStore.setState({ tabs: [] });
      }
    }
  }, [tabbarConfig.persist]);

  // 构建标签列表（支持图标显示）
  const tabItems: TabItem[] = useMemo(() => {
    return tabs.map((tab) => ({
      key: tab.key,
      label: (
        <span className="tabsbar-tab-label">
          {tabbarConfig.showIcon && tab.icon && (
            <span className="tabsbar-tab-icon" style={{ marginRight: 4 }}>
              {getIconFromName(tab.icon)}
            </span>
          )}
          {tab.title}
        </span>
      ),
      closable: tab.closable,
    }));
  }, [tabs, tabbarConfig.showIcon]);

  // 处理标签切换
  const handleTabChange = useCallback(
    (key: string) => {
      navigate(key);
    },
    [navigate],
  );

  // 处理标签关闭
  const handleTabRemove = useCallback(
    (targetKey: string) => {
      closeTab(targetKey);

      // 如果关闭的是当前激活的标签，跳转到其他标签
      if (targetKey === location.pathname) {
        const currentIndex = tabs.findIndex((tab) => tab.key === targetKey);
        if (currentIndex > 0) {
          // 跳转到前一个标签
          navigate(tabs[currentIndex - 1].key);
        } else if (tabs.length > 1) {
          // 跳转到后一个标签
          navigate(tabs[1].key);
        } else {
          // 跳转到首页
          navigate('/');
        }
      }
    },
    [closeTab, location.pathname, tabs, navigate],
  );

  // 右键菜单
  const contextMenuItems: MenuProps['items'] = useMemo(() => {
    const currentTabData = tabs.find((tab) => tab.key === contextMenuTabKey);
    if (!currentTabData) return [];

    const isHome = contextMenuTabKey === '/';
    const isPinned = currentTabData.pinned;
    const currentIndex = tabs.findIndex((tab) => tab.key === contextMenuTabKey);

    const items: MenuProps['items'] = [
      {
        key: 'close',
        label: t('tabs.close'),
        icon: <CloseOutlined />,
        disabled: !currentTabData.closable,
        onClick: () => handleTabRemove(contextMenuTabKey),
      },
      {
        key: 'pin',
        label: isPinned ? t('tabs.unpin') : t('tabs.pin'),
        icon: isPinned ? <PushpinFilled /> : <PushpinOutlined />,
        onClick: () => togglePinTab(contextMenuTabKey),
      },
      {
        type: 'divider',
      },
      {
        key: 'reload',
        label: t('tabs.reload'),
        icon: <ReloadOutlined />,
        onClick: () => {
          reloadTab(contextMenuTabKey);
          // 触发页面刷新
          if (contextMenuTabKey === location.pathname) {
            // 刷新当前页面
            window.location.reload();
          }
        },
      },
      {
        key: 'maximize',
        label: t('tabs.maximize'),
        icon: <FullscreenOutlined />,
        // 最大化功能后续实现
      },
      {
        key: 'newWindow',
        label: t('tabs.openInNewWindow'),
        icon: <ExportOutlined />,
        onClick: () => {
          window.open(contextMenuTabKey, '_blank');
        },
      },
    ];

    // 添加分割线和批量关闭选项
    if (!isHome && tabs.length > 1) {
      items.push(
        { type: 'divider' },
        {
          key: 'closeLeft',
          label: t('tabs.closeLeft'),
          icon: <LeftOutlined />,
          disabled: currentIndex === 0,
          onClick: () => {
            closeLeftTabs(contextMenuTabKey);
            navigate(contextMenuTabKey);
          },
        },
        {
          key: 'closeRight',
          label: t('tabs.closeRight'),
          icon: <RightOutlined />,
          disabled: currentIndex === tabs.length - 1,
          onClick: () => {
            closeRightTabs(contextMenuTabKey);
            navigate(contextMenuTabKey);
          },
        },
        {
          key: 'closeOthers',
          label: t('tabs.closeOthers'),
          icon: <CloseCircleOutlined />,
          disabled: tabs.filter((t) => t.closable && t.key !== contextMenuTabKey).length === 0,
          onClick: () => {
            closeOtherTabs(contextMenuTabKey);
            navigate(contextMenuTabKey);
          },
        },
        {
          key: 'closeAll',
          label: t('tabs.closeAll'),
          icon: <CloseOutlined />,
          onClick: () => {
            closeAllTabs();
            navigate('/');
          },
        },
      );
    }

    return items;
  }, [
    contextMenuTabKey,
    tabs,
    t,
    handleTabRemove,
    togglePinTab,
    reloadTab,
    closeLeftTabs,
    closeRightTabs,
    closeOtherTabs,
    closeAllTabs,
    location.pathname,
    navigate,
  ]);

  // 更多操作菜单（包含所有操作）
  const moreMenuItems: MenuProps['items'] = useMemo(() => {
    const currentTabData = tabs.find((tab) => tab.key === location.pathname);
    if (!currentTabData) return [];

    const isPinned = currentTabData.pinned;
    const currentIndex = tabs.findIndex((tab) => tab.key === location.pathname);

    return [
      // 1. 关闭
      {
        key: 'close',
        label: t('tabs.close'),
        icon: <CloseOutlined />,
        disabled: !currentTabData.closable,
        onClick: () => handleTabRemove(location.pathname),
      },
      // 2. 固定/取消固定
      {
        key: 'pin',
        label: isPinned ? t('tabs.unpin') : t('tabs.pin'),
        icon: isPinned ? <PushpinFilled /> : <PushpinOutlined />,
        onClick: () => togglePinTab(location.pathname),
      },
      // 3. 最大化/退出最大化
      {
        key: 'maximize',
        label: isFullscreen ? t('tabs.exitMaximize') : t('tabs.maximize'),
        icon: isFullscreen ? <FullscreenExitOutlined /> : <FullscreenOutlined />,
        onClick: () => toggleFullscreen(),
      },
      // 4. 重新加载
      {
        key: 'reload',
        label: t('tabs.reload'),
        icon: <ReloadOutlined />,
        onClick: () => {
          reloadTab(location.pathname);
          triggerPageRefresh();
        },
      },
      // 5. 在新窗口打开
      {
        key: 'newWindow',
        label: t('tabs.openInNewWindow'),
        icon: <ExportOutlined />,
        onClick: () => {
          window.open(location.pathname, '_blank');
        },
      },
      // 分割线
      { type: 'divider' },
      // 6. 关闭左侧标签页
      {
        key: 'closeLeft',
        label: t('tabs.closeLeft'),
        icon: <LeftOutlined />,
        disabled: currentIndex === 0,
        onClick: () => {
          closeLeftTabs(location.pathname);
          navigate(location.pathname);
        },
      },
      // 7. 关闭右侧标签页
      {
        key: 'closeRight',
        label: t('tabs.closeRight'),
        icon: <RightOutlined />,
        disabled: currentIndex === tabs.length - 1,
        onClick: () => {
          closeRightTabs(location.pathname);
          navigate(location.pathname);
        },
      },
      // 8. 关闭其它标签页
      {
        key: 'closeOthers',
        label: t('tabs.closeOthers'),
        icon: <CloseCircleOutlined />,
        disabled: tabs.filter((tab) => tab.closable && tab.key !== location.pathname).length === 0,
        onClick: () => {
          closeOtherTabs(location.pathname);
          navigate(location.pathname);
        },
      },
      // 9. 关闭全部标签页
      {
        key: 'closeAll',
        label: t('tabs.closeAll'),
        icon: <CloseOutlined />,
        onClick: () => {
          closeAllTabs();
          navigate('/');
        },
      },
    ];
  }, [
    tabs,
    location.pathname,
    isFullscreen,
    t,
    handleTabRemove,
    togglePinTab,
    toggleFullscreen,
    reloadTab,
    triggerPageRefresh,
    closeLeftTabs,
    closeRightTabs,
    closeOtherTabs,
    closeAllTabs,
    navigate,
  ]);

  // 自定义渲染 TabBar，添加右键菜单和拖拽功能
  const renderTabBar: TabsProps['renderTabBar'] = useCallback(
    (props: any, DefaultTabBar: React.ComponentType<any>) => {
      // 如果启用了拖拽，添加拖拽相关的事件处理
      if (tabbarConfig.draggable) {
        return (
          <DefaultTabBar
            {...props}
            onDragStart={(e: React.DragEvent) => {
              const target = e.target as HTMLElement;
              const tabElement = target.closest('.ant-tabs-tab');
              if (tabElement) {
                const index = parseInt(tabElement.getAttribute('data-index') || '0', 10);
                e.dataTransfer.setData('text/plain', String(index));
                tabElement.classList.add('dragging');
              }
            }}
            onDragEnd={(e: React.DragEvent) => {
              const target = e.target as HTMLElement;
              const tabElement = target.closest('.ant-tabs-tab');
              if (tabElement) {
                tabElement.classList.remove('dragging');
              }
              // 清除所有 drag-over 样式
              document.querySelectorAll('.ant-tabs-tab.drag-over').forEach((el) => {
                el.classList.remove('drag-over');
              });
            }}
            onDragOver={(e: React.DragEvent) => {
              e.preventDefault();
              e.dataTransfer.dropEffect = 'move';

              const target = e.target as HTMLElement;
              const tabElement = target.closest('.ant-tabs-tab');
              if (tabElement) {
                // 清除其他元素的 drag-over 样式
                document.querySelectorAll('.ant-tabs-tab.drag-over').forEach((el) => {
                  if (el !== tabElement) {
                    el.classList.remove('drag-over');
                  }
                });
                tabElement.classList.add('drag-over');
              }
            }}
            onDrop={(e: React.DragEvent) => {
              e.preventDefault();
              const fromIndex = parseInt(e.dataTransfer.getData('text/plain'), 10);

              const target = e.target as HTMLElement;
              const tabElement = target.closest('.ant-tabs-tab');
              if (tabElement) {
                const toIndex = parseInt(tabElement.getAttribute('data-index') || '0', 10);
                if (fromIndex !== toIndex) {
                  reorderTabs(fromIndex, toIndex);
                }
              }

              // 清除所有 drag-over 样式
              document.querySelectorAll('.ant-tabs-tab.drag-over').forEach((el) => {
                el.classList.remove('drag-over');
              });
            }}
          />
        );
      }

      return <DefaultTabBar {...props} />;
    },
    [tabbarConfig.draggable, reorderTabs],
  );

  // 为每个 Tab 项添加右键菜单
  const tabItemsWithContextMenu: TabItem[] = useMemo(() => {
    return tabItems.map((tab) => ({
      ...tab,
      label: (
        <Dropdown
          menu={{ items: contextMenuItems }}
          trigger={['contextMenu']}
          onOpenChange={(visible) => {
            if (visible) {
              setContextMenuTabKey(tab.key);
            } else {
              setContextMenuTabKey('');
            }
          }}
        >
          <span style={{ display: 'inline-block', padding: '4px 0' }}>{tab.label}</span>
        </Dropdown>
      ),
    }));
  }, [tabItems, contextMenuItems]);

  if (!tabbarConfig.enable) return null;

  return (
    <div
      className={tabbarClassName}
      style={{
        height: tabbarConfig.height,
        borderBottom: `1px solid ${isDark ? '#303030' : '#e5e7eb'}`,
        backgroundColor: isDark ? '#141414' : '#ffffff',
        display: 'flex',
        alignItems: 'center',
        padding: '0 8px',
        flexShrink: 0,
      }}
    >
      <Tabs
        activeKey={location.pathname}
        size="small"
        type={tabsType}
        hideAdd
        items={tabItemsWithContextMenu}
        onChange={handleTabChange}
        onEdit={(_, action) => {
          if (action === 'remove' && typeof _ === 'string') {
            handleTabRemove(_);
          }
        }}
        renderTabBar={renderTabBar}
        style={{ width: '100%', margin: 0, flex: 1 }}
        tabBarStyle={{ margin: 0 }}
        // 如果启用了 keepAlive，可以在这里配置缓存逻辑
        // destroyInactiveTabPane={!tabbarConfig.keepAlive}
      />

      {/* 右侧操作按钮 */}
      <Space size={4} style={{ marginLeft: 8, flexShrink: 0 }}>
        {/* 更多按钮（下拉菜单） */}
        {tabbarConfig.showMore && (
          <Dropdown menu={{ items: moreMenuItems }} trigger={['click']} placement="bottomRight">
            <Button
              type="text"
              size="small"
              icon={<DownOutlined />}
              style={{
                color: isDark ? 'rgba(255, 255, 255, 0.65)' : 'rgba(0, 0, 0, 0.65)',
              }}
            />
          </Dropdown>
        )}

        {/* 最大化按钮 */}
        {tabbarConfig.showMaximize && widgetConfig.fullscreen && (
          <Button
            type="text"
            size="small"
            icon={isFullscreen ? <FullscreenExitOutlined /> : <FullscreenOutlined />}
            onClick={toggleFullscreen}
            style={{
              color: isDark ? 'rgba(255, 255, 255, 0.65)' : 'rgba(0, 0, 0, 0.65)',
            }}
            title={isFullscreen ? t('tabs.exitMaximize') : t('tabs.maximize')}
          />
        )}
      </Space>
    </div>
  );
};

export default Index;
