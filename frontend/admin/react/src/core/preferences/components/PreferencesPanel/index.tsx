import React, { useState } from 'react';
import { Button, Drawer, Segmented } from 'antd';
import { ReloadOutlined, CloseOutlined, CopyOutlined, LogoutOutlined } from '@ant-design/icons';

import { usePreferencesStore } from '../../store';
import { AppearancePanel } from './AppearancePanel';
import { LayoutPanel } from './LayoutPanel';
import { ShortcutKeyPanel } from './ShortcutKeyPanel';
import { GeneralPanel } from './GeneralPanel';
import './PreferencesPanel.style.less';

interface PreferencesPanelProps {
  open: boolean;
  onClose: () => void;
}

type TabType = 'appearance' | 'layout' | 'shortcut' | 'general';

const TAB_OPTIONS = [
  { label: '外观', value: 'appearance' },
  { label: '布局', value: 'layout' },
  { label: '快捷键', value: 'shortcut' },
  { label: '通用', value: 'general' },
];

const TAB_COMPONENTS = {
  appearance: AppearancePanel,
  layout: LayoutPanel,
  shortcut: ShortcutKeyPanel,
  general: GeneralPanel,
};

export const PreferencesPanel: React.FC<PreferencesPanelProps> = ({ open, onClose }) => {
  const { resetPreferences } = usePreferencesStore();
  const [activeTab, setActiveTab] = useState<TabType>('appearance');

  const ActiveComponent = TAB_COMPONENTS[activeTab];

  const handleReset = () => {
    resetPreferences();
  };

  const handleCopy = () => {
    // TODO: 实现复制偏好设置功能
    console.log('复制偏好设置');
  };

  const handleLogout = () => {
    // TODO: 实现退出登录功能
    console.log('退出登录');
  };

  return (
    <Drawer
      title={
        <div className="drawer-header">
          <div>
            <h2 className="drawer-title">偏好设置</h2>
            <p className="drawer-subtitle">自定义偏好设置 & 实时预览</p>
          </div>
          <div className="drawer-actions">
            <Button type="text" icon={<ReloadOutlined />} onClick={handleReset} title="重置" />
            <Button type="text" icon={<CloseOutlined />} onClick={onClose} title="关闭" />
          </div>
        </div>
      }
      placement="right"
      size={360}
      open={open}
      onClose={onClose}
      closable={false}
      className="preferences-drawer"
      footer={
        <div className="drawer-footer">
          <Button type="primary" icon={<CopyOutlined />} onClick={handleCopy}>
            复制偏好设置
          </Button>
          <Button type="link" danger icon={<LogoutOutlined />} onClick={handleLogout}>
            清空缓存 & 退出登录
          </Button>
        </div>
      }
    >
      {/* Tab 切换 */}
      <div className="drawer-tabs">
        <Segmented
          options={TAB_OPTIONS}
          value={activeTab}
          onChange={(value) => setActiveTab(value as TabType)}
          block
        />
      </div>

      {/* 内容区域 */}
      <div className="drawer-content">
        <ActiveComponent />
      </div>
    </Drawer>
  );
};
