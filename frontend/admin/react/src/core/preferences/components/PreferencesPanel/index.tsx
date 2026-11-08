import React, { useState } from 'react';
import { Button, Drawer, Segmented, message } from 'antd';
import { ReloadOutlined, CloseOutlined, CopyOutlined, LogoutOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

import { usePreferencesStore } from '../../store';
import { defaultPreferences } from '../../config/default';
import { diffDeep } from '../../utils/diff';
import { useAuthStore } from '@/stores';
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
  { label: 'appearance', value: 'appearance' },
  { label: 'layout', value: 'layout' },
  { label: 'shortcut', value: 'shortcut' },
  { label: 'general', value: 'general' },
];

const TAB_COMPONENTS = {
  appearance: AppearancePanel,
  layout: LayoutPanel,
  shortcut: ShortcutKeyPanel,
  general: GeneralPanel,
};

export const PreferencesPanel: React.FC<PreferencesPanelProps> = ({ open, onClose }) => {
  const { resetPreferences } = usePreferencesStore();
  const { t } = useTranslation('preferences');
  const [activeTab, setActiveTab] = useState<TabType>('appearance');

  const ActiveComponent = TAB_COMPONENTS[activeTab];

  const handleReset = () => {
    resetPreferences();
  };

  // 对齐 vben preferences-drawer 的 handleCopy：复制「与出厂默认的增量」而非全量快照，
  // 产物是可回贴的 overrides 对象（diffDeep 语义见 core/preferences/utils/diff.ts）。
  const handleCopy = async () => {
    const { preferences } = usePreferencesStore.getState();
    const overrides = diffDeep(defaultPreferences, preferences);
    try {
      await navigator.clipboard.writeText(JSON.stringify(overrides, null, 2));
      message.success(t('actions.copyPreferencesSuccess'));
    } catch (error) {
      console.error('[PreferencesPanel] 复制偏好设置失败:', error);
      message.error(t('actions.copyPreferencesFailed'));
    }
  };

  const handleLogout = async () => {
    try {
      await useAuthStore.getState().logout(true);
    } catch (error) {
      console.error('Logout failed:', error);
    }
  };

  return (
    <Drawer
      title={
        <div className="drawer-header">
          <div>
            <h2 className="drawer-title">{t('title')}</h2>
            <p className="drawer-subtitle">{t('subtitle')}</p>
          </div>
          <div className="drawer-actions">
            <Button type="text" icon={<ReloadOutlined />} onClick={handleReset} title={t('actions.reset')} />
            <Button type="text" icon={<CloseOutlined />} onClick={onClose} title={t('actions.close')} />
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
            {t('actions.copySettings')}
          </Button>
          <Button type="link" danger icon={<LogoutOutlined />} onClick={handleLogout}>
            {t('actions.clearCacheAndLogout')}
          </Button>
        </div>
      }
    >
      {/* Tab 切换 */}
      <div className="drawer-tabs">
        <Segmented
          options={TAB_OPTIONS.map(opt => ({ ...opt, label: t(`tabs.${opt.label}`) }))}
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
