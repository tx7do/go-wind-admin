import { Button, Tooltip } from 'antd';
import {
  ColumnWidthOutlined,
  ColumnHeightOutlined,
  PushpinOutlined,
  PushpinFilled,
} from '@ant-design/icons';

interface ControlPanelProps {
  collapsed: boolean;
  isDark: boolean;
  expandOnHover: boolean;
  onToggleExpandOnHover: () => void;
  onToggleCollapse: () => void;
}

export const ControlPanel = ({
  collapsed,
  isDark,
  expandOnHover,
  onToggleExpandOnHover,
  onToggleCollapse,
}: ControlPanelProps) => {
  return (
    <div
      style={{
        borderTop: `1px solid ${isDark ? '#303030' : '#e5e7eb'}`,
        padding: '8px 4px',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        flexShrink: 0,
      }}
    >
      {/* 左侧：折叠/展开按钮（始终显示） */}
      <Tooltip title={collapsed ? '展开菜单' : '折叠菜单'}>
        <Button
          type="text"
          icon={collapsed ? <ColumnHeightOutlined /> : <ColumnWidthOutlined />}
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

      {/* 右侧：固定/自动按钮（始终显示） */}
      <Tooltip
        title={expandOnHover ? '切换为手动模式（固定菜单）' : '切换为自动模式（鼠标悬停展开）'}
      >
        <Button
          type="text"
          icon={!expandOnHover ? <PushpinFilled /> : <PushpinOutlined />}
          onClick={onToggleExpandOnHover}
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
    </div>
  );
};

export default ControlPanel;
