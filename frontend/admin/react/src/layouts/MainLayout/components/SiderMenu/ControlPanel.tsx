import { Button, Tooltip } from 'antd';
import {
  ColumnWidthOutlined,
  ColumnHeightOutlined,
  PushpinOutlined,
  PushpinFilled,
} from '@ant-design/icons';
import { useI18n } from '@go-wind/admin-react';

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
  void isDark;
  const { t } = useI18n('common');

  return (
    <div
      style={{
        borderTop: '1px solid var(--ant-color-border)',
        padding: '8px 4px',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        flexShrink: 0,
      }}
    >
      {/* 左侧：折叠/展开按钮（始终显示） */}
      <Tooltip title={collapsed ? t('sidebar.expandMenu') : t('sidebar.collapseMenu')}>
        <Button
          type="text"
          icon={collapsed ? <ColumnHeightOutlined /> : <ColumnWidthOutlined />}
          onClick={onToggleCollapse}
          size="small"
          style={{
            color: 'var(--ant-color-text-secondary)',
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
        title={expandOnHover ? t('sidebar.switchToManual') : t('sidebar.switchToAuto')}
      >
        <Button
          type="text"
          icon={!expandOnHover ? <PushpinFilled /> : <PushpinOutlined />}
          onClick={onToggleExpandOnHover}
          size="small"
          style={{
            color: 'var(--ant-color-text-secondary)',
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
