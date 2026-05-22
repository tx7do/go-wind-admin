import { useMemo, useCallback, useState, useEffect } from 'react';
import { Menu } from 'antd';
import { useNavigate } from 'react-router-dom';
import { usePreferencesStore } from '@/core/preferences/store';
import { getIconFromName } from '../utils/iconResolver';
import ControlPanel from './ControlPanel';

interface SiderMenuProps {
  menuData: any[];
  collapsed: boolean;
  isMobile: boolean;
  isDark: boolean;
  openKeys: string[];
  selectedKeys: string[];
  onCollapse: (collapsed: boolean) => void;
  onOpenChange: (keys: string[]) => void;
}

export const SiderMenu = ({
  menuData,
  collapsed,
  isMobile,
  isDark,
  openKeys,
  selectedKeys,
  onCollapse,
  onOpenChange,
}: SiderMenuProps) => {
  const navigate = useNavigate();
  const preferences = usePreferencesStore((state) => state.preferences);

  // 固定状态：pinned=true 时始终显示完整列表，不允许折叠
  const [pinned, setPinned] = useState(false);

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
        if (isMobile) onCollapse(true);
      }
    },
    [navigate, isMobile, onCollapse],
  );

  // 固定菜单时强制展开
  useEffect(() => {
    if (pinned && collapsed) {
      onCollapse(false);
    }
  }, [pinned, collapsed, onCollapse]);

  // pinned 为 true 时不允许折叠
  const effectiveCollapsed = pinned ? false : collapsed;

  // collapsed 为 true 且未固定时完全隐藏侧边栏
  if (effectiveCollapsed) return null;

  const sidebarWidth = preferences.sidebar?.width || 224;

  return (
    <div
      style={{
        width: sidebarWidth,
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        backgroundColor: isDark ? '#141414' : '#ffffff',
        borderRight: `1px solid ${isDark ? '#303030' : '#e5e7eb'}`,
        transition: 'width 0.2s',
        flexShrink: 0,
        overflow: 'hidden',
      }}
    >
      {/* Logo 区域 */}
      <div
        style={{
          height: 56,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'flex-start',
          padding: '0 16px',
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
        {preferences.app.dynamicTitle && (
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
            inlineCollapsed={false}
            selectedKeys={selectedKeys}
            openKeys={openKeys}
            onOpenChange={onOpenChange}
            onClick={handleMenuClick}
            items={menuItems}
            style={{
              background: 'transparent',
              borderInlineEnd: 'none',
            }}
            theme={isDark ? 'dark' : 'light'}
          />
        </div>
      </div>

      {/* 底部控制面板 */}
      <ControlPanel
        collapsed={collapsed}
        isDark={isDark}
        onToggleCollapse={() => onCollapse(!collapsed)}
        pinned={pinned}
        onTogglePin={() => setPinned(!pinned)}
      />
    </div>
  );
};

export default SiderMenu;
