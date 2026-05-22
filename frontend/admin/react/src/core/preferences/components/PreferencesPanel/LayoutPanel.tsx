import React from 'react';
import {Button, InputNumber, Segmented, Space, Switch} from 'antd';
import {MinusOutlined, PlusOutlined, QuestionCircleOutlined} from '@ant-design/icons';
import {usePreferencesStore} from '../../store';
import type {ContentCompactType, LayoutType} from '../../types';
import './LayoutPanel.style.less';

/** 布局选项 */
const LAYOUT_OPTIONS = [
  {
    label: '垂直',
    value: 'sidebar-nav',
    icon: '📋',
  },
  {
    label: '双列菜单',
    value: 'sidebar-mixed-nav',
    icon: '📑',
  },
  {
    label: '水平',
    value: 'header-nav',
    icon: '',
  },
  {
    label: '混合菜单',
    value: 'mixed-nav',
    icon: '📊',
  },
  {
    label: '内容全屏',
    value: 'full-content',
    icon: '📱',
  },
];

/** 内容宽度选项 */
const CONTENT_COMPACT_OPTIONS = [
  { label: '流式', value: 'wide' },
  { label: '定宽', value: 'compact' },
];

export const LayoutPanel: React.FC = () => {
  const { preferences, setPreferences } = usePreferencesStore();

  const handleLayoutChange = (layout: LayoutType) => {
    setPreferences({ app: { layout } });
  };

  const handleContentCompactChange = (compact: ContentCompactType) => {
    setPreferences({ app: { contentCompact: compact } });
  };

  const handleSidebarWidthChange = (width: number | null) => {
    if (width && width >= 180 && width <= 320) {
      setPreferences({ sidebar: { width } });
    }
  };

  return (
    <div className="layout-panel">
      {/* 布局选择 */}
      <section className="layout-section">
        <h3 className="section-title">布局</h3>
        <div className="layout-grid">
          {LAYOUT_OPTIONS.map((option) => (
            <div
              key={option.value}
              className={`layout-item ${preferences.app.layout === option.value ? 'active' : ''}`}
              onClick={() => handleLayoutChange(option.value as LayoutType)}
            >
              <div className="layout-preview">
                <div className="layout-icon">{option.icon}</div>
              </div>
              <div className="layout-label">
                <span>{option.label}</span>
                <QuestionCircleOutlined className="help-icon" />
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* 内容宽度 */}
      <section className="layout-section">
        <h3 className="section-title">内容</h3>
        <div className="content-compact-grid">
          {CONTENT_COMPACT_OPTIONS.map((option) => (
            <div
              key={option.value}
              className={`content-compact-item ${preferences.app.contentCompact === option.value ? 'active' : ''}`}
              onClick={() => handleContentCompactChange(option.value as ContentCompactType)}
            >
              <div className="content-preview">
                <div className={`preview-bar ${option.value === 'compact' ? 'narrow' : 'wide'}`} />
              </div>
              <span>{option.label}</span>
            </div>
          ))}
        </div>
      </section>

      {/* 侧边栏设置 */}
      <section className="layout-section">
        <h3 className="section-title">侧边栏</h3>
        <div className="preference-item">
          <span>显示侧边栏</span>
          <Switch
            checked={preferences.sidebar.enable}
            onChange={(checked) => setPreferences({ sidebar: { enable: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>折叠菜单</span>
          <Switch
            checked={preferences.sidebar.collapsed}
            onChange={(checked) => setPreferences({ sidebar: { collapsed: checked } })}
          />
        </div>
        <div className={`preference-item ${preferences.sidebar.collapsed ? 'disabled' : ''}`}>
          <span>折叠显示菜单名</span>
          <Switch
            disabled={!preferences.sidebar.collapsed}
            checked={preferences.sidebar.collapsedShowTitle}
            onChange={(checked) => setPreferences({ sidebar: { collapsedShowTitle: checked } })}
          />
        </div>
        <div className="preference-item width-control">
          <span>宽度</span>
          <Space>
            <Button
              size="small"
              icon={<MinusOutlined/>}
              onClick={() => handleSidebarWidthChange(preferences.sidebar.width - 8)}
              disabled={preferences.sidebar.width <= 180}
            />
            <InputNumber
              min={180}
              max={320}
              value={preferences.sidebar.width}
              onChange={handleSidebarWidthChange}
              style={{width: 80}}
            />
            <Button
              size="small"
              icon={<PlusOutlined/>}
              onClick={() => handleSidebarWidthChange(preferences.sidebar.width + 8)}
              disabled={preferences.sidebar.width >= 320}
            />
          </Space>
        </div>
      </section>

      {/* 顶栏设置 */}
      <section className="layout-section">
        <h3 className="section-title">顶栏</h3>
        <div className="preference-item">
          <span>显示顶栏</span>
          <Switch
            checked={preferences.header.enable}
            onChange={(checked) => setPreferences({ header: { enable: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>模式</span>
          <Segmented
            options={[
              { label: '固定', value: 'fixed' },
              { label: '自动', value: 'auto' },
              { label: '静态', value: 'static' },
            ]}
            value={preferences.header.mode}
            onChange={(value) => setPreferences({ header: { mode: value as any } })}
          />
        </div>
      </section>

      {/* 导航菜单 */}
      <section className="layout-section">
        <h3 className="section-title">导航菜单</h3>
        <div className="preference-item">
          <span>导航菜单风格</span>
          <Segmented
            options={[
              { label: '圆润', value: 'rounded' },
              { label: '朴素', value: 'plain' },
            ]}
            value={preferences.navigation.styleType}
            onChange={(value) => setPreferences({ navigation: { styleType: value as any } })}
          />
        </div>
        <div className="preference-item">
          <span>导航菜单分离</span>
          <Switch
            checked={preferences.navigation.split}
            onChange={(checked) => setPreferences({ navigation: { split: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>侧边导航菜单手风琴模式</span>
          <Switch
            checked={preferences.navigation.accordion}
            onChange={(checked) => setPreferences({ navigation: { accordion: checked } })}
          />
        </div>
      </section>

      {/* 面包屑导航 */}
      <section className="preference-section">
        <h3 className="section-title">面包屑导航</h3>
        <div className="preference-item">
          <span>开启面包屑导航</span>
          <Switch
            checked={preferences.breadcrumb.enable}
            onChange={(checked) => setPreferences({ breadcrumb: { enable: checked } })}
          />
        </div>
      </section>
    </div>
  );
};
