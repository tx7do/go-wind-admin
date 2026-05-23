import { useMemo, useCallback, useState } from 'react';
import { Menu } from 'antd';
import { useNavigate } from 'react-router-dom';

import { usePreferencesStore } from '@/core/preferences/store';
import { getIconFromName } from '../../utils/iconResolver';
import ControlPanel from './ControlPanel';
import './SiderMenu.style.less';

interface SiderMenuProps {
  menuData: any[];
  isMobile: boolean;
  isDark: boolean;
  openKeys: string[];
  selectedKeys: string[];
  onOpenChange: (keys: string[]) => void;
}

export const Index = ({
  menuData,
  isMobile,
  isDark,
  openKeys,
  selectedKeys,
  onOpenChange,
}: SiderMenuProps) => {
  const navigate = useNavigate();
  const preferences = usePreferencesStore((state) => state.preferences);
  const setPreferences = usePreferencesStore((state) => state.setPreferences);

  // 从 preferences 获取侧边栏配置
  const sidebarConfig = preferences.sidebar;
  const sidebarCollapsed = sidebarConfig?.collapsed ?? false;
  const collapsedShowTitle = sidebarConfig?.collapsedShowTitle ?? false;
  const expandOnHover = sidebarConfig?.expandOnHover ?? true;
  const sidebarEnable = sidebarConfig?.enable ?? true;
  const sidebarHidden = sidebarConfig?.hidden ?? false;

  // 鼠标悬停状态（用于 expandOnHover）
  const [isHovering, setIsHovering] = useState(false);

  // 转换菜单数据为 Ant Design Menu items 格式
  const menuItems = useMemo(() => {
    const transformItem = (items: any[]): any[] => {
      return items.map((item) => ({
        key: item.path || item.key,
        icon: getIconFromName(item.icon),
        label: item.name || item.label,
        children: item.children ? transformItem(item.children) : undefined,
      }));
    };
    return transformItem(menuData);
  }, [menuData]);

  // 菜单点击
  const handleMenuClick = useCallback(
    (info: { key: string }) => {
      if (info.key) {
        navigate(info.key);
        if (isMobile) {
          setPreferences({ sidebar: { collapsed: true } });
        }
      }
    },
    [navigate, isMobile, setPreferences],
  );

  // 完全隐藏侧边栏（sidebar.enable = false 或 sidebar.hidden = true）
  if (!sidebarEnable || sidebarHidden) return null;

  // 计算是否折叠：
  // - expandOnHover=true: 根据 hover 状态动态切换
  // - expandOnHover=false: 根据 collapsed 状态决定
  const isCollapsed = expandOnHover 
    ? (!isHovering)  // 自动模式：只看 hover 状态
    : sidebarCollapsed;  // 手动模式：看 collapsed 状态

  // 计算侧边栏宽度
  const sidebarWidth = isCollapsed ? 60 : (sidebarConfig?.width ?? 224);

  // 鼠标进入/离开事件处理（仅 expandOnHover=true 时生效）
  const handleMouseEnter = () => {
    if (expandOnHover) {
      setIsHovering(true);
    }
  };

  const handleMouseLeave = () => {
    if (expandOnHover) {
      setIsHovering(false);
    }
  };

  // 切换折叠状态
  const handleToggleCollapse = () => {
    setPreferences({ sidebar: { collapsed: !sidebarCollapsed } });
  };

  // 切换 expandOnHover 状态（固定/自动模式）
  const handleToggleExpandOnHover = () => {
    const newExpandOnHover = !expandOnHover;
    setPreferences({ sidebar: { expandOnHover: newExpandOnHover } });
    if (newExpandOnHover) {
      // 切换到自动模式时，如果当前是折叠状态，立即允许 hover 展开
      // 不需要额外操作，isHovering 会根据鼠标位置自动更新
    } else {
      // 切换到手动模式时，取消 hover 状态
      setIsHovering(false);
    }
  };

  return (
    <div
      style={{
        width: sidebarWidth,
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        backgroundColor: isDark ? '#141414' : '#ffffff',
        borderRight: `1px solid ${isDark ? '#303030' : '#e5e7eb'}`,
        transition: 'width 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
        flexShrink: 0,
        overflow: 'hidden',
        // Drawer 效果：expandOnHover=true 时，使用绝对定位悬浮在内容上方
        position: expandOnHover ? 'absolute' : 'relative',
        zIndex: expandOnHover ? 1000 : 1,
      }}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
    >
      {/* Logo 区域 */}
      <div
        style={{
          height: 56,
          display: 'flex',
          alignItems: 'center',
          justifyContent: isCollapsed && !collapsedShowTitle ? 'center' : 'flex-start',
          padding: isCollapsed && !collapsedShowTitle ? '0' : '0 16px',
          gap: 8,
          borderBottom: `1px solid ${isDark ? '#303030' : '#e5e7eb'}`,
          cursor: 'pointer',
          flexShrink: 0,
          overflow: 'hidden',
        }}
        onClick={() => navigate('/')}
      >
        {preferences.logo.enable && (
          <img
            src={preferences.logo.source}
            alt="logo"
            style={{ height: 32, width: 32, flexShrink: 0 }}
          />
        )}
        {(!isCollapsed || collapsedShowTitle) && preferences.app.dynamicTitle && (
          <span
            style={{
              fontWeight: 700,
              fontSize: 16,
              color: isDark ? '#ffffff' : '#262626',
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
            }}
          >
            {preferences.app.name}
          </span>
        )}
      </div>

      {/* 菜单树形列表 */}
      <div style={{ flex: 1, overflow: 'hidden' }}>
        <div style={{ height: '100%', overflowY: 'auto', overflowX: 'hidden', padding: '4px 0' }}>
          <Menu
            mode="inline"
            inlineCollapsed={isCollapsed && !collapsedShowTitle}
            selectedKeys={selectedKeys}
            openKeys={openKeys}
            onOpenChange={onOpenChange}
            onClick={handleMenuClick}
            items={menuItems}
            style={{
              background: 'transparent',
              borderInlineEnd: 'none',
              width: '100%',
            }}
            theme={isDark ? 'dark' : 'light'}
            // 当 collapsedShowTitle=true 时，覆盖默认的折叠样式
            className={collapsedShowTitle && isCollapsed ? 'menu-collapsed-show-title' : ''}
          />
        </div>
      </div>

      {/* 底部控制面板 */}
      <ControlPanel
        collapsed={sidebarCollapsed}
        isDark={isDark}
        expandOnHover={expandOnHover}
        onToggleExpandOnHover={handleToggleExpandOnHover}
        onToggleCollapse={handleToggleCollapse}
      />
    </div>
  );
};

export default Index;
