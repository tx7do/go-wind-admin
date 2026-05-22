import { Button, Tooltip } from 'antd';
import {
  MenuUnfoldOutlined,
  PushpinOutlined,
  PushpinFilled,
} from '@ant-design/icons';

interface ControlPanelProps {
  collapsed: boolean;
  isDark: boolean;
  onToggleCollapse: () => void;
  pinned: boolean;
  onTogglePin: () => void;
}

export const ControlPanel = ({
  collapsed,
  isDark,
  onToggleCollapse,
  pinned,
  onTogglePin,
}: ControlPanelProps) => {
  return (
    <div
      style={{
        borderTop: `1px solid ${isDark ? '#303030' : '#e5e7eb'}`,
        padding: '8px 4px',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        gap: 4,
        flexShrink: 0,
      }}
    >
      {/* 第一个按钮：折叠/展开列表 */}
      <Tooltip title={collapsed ? '展开菜单列表' : '折叠菜单列表'}>
        <Button
          type="text"
          icon={<MenuUnfoldOutlined />}
          onClick={onToggleCollapse}
          size="small"
          style={{
            color: isDark ? '#a6a6a6' : '#595959',
            width: 32,
            height: 32,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        />
      </Tooltip>

      {/* 第二个按钮：固定/取消固定 */}
      <Tooltip title={pinned ? '取消固定（允许折叠）' : '固定菜单（始终显示完整列表）'}>
        <Button
          type={pinned ? 'primary' : 'text'}
          icon={pinned ? <PushpinFilled /> : <PushpinOutlined />}
          onClick={onTogglePin}
          size="small"
          style={{
            color: isDark ? (pinned ? '#ffffff' : '#a6a6a6') : (pinned ? '#1677ff' : '#595959'),
            width: 32,
            height: 32,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        />
      </Tooltip>
    </div>
  );
};

export default ControlPanel;
