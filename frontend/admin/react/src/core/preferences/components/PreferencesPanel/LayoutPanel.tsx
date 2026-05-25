import React from 'react';
import { Button, Input, InputNumber, Segmented, Select, Space, Switch } from 'antd';
import { MinusOutlined, PlusOutlined, QuestionCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { usePreferencesStore } from '../../store';
import type { ContentCompactType, LayoutType } from '../../types';
import './LayoutPanel.style.less';

/** 布局选项 */
const LAYOUT_OPTIONS = [
  {
    label: 'layout.modes.sidebarNav',
    value: 'sidebar-nav',
    icon: '📋',
  },
  {
    label: 'layout.modes.sidebarMixedNav',
    value: 'sidebar-mixed-nav',
    icon: '📑',
  },
  {
    label: 'layout.modes.headerNav',
    value: 'header-nav',
    icon: '',
  },
  {
    label: 'layout.modes.mixedNav',
    value: 'mixed-nav',
    icon: '📊',
  },
  {
    label: 'layout.modes.fullContent',
    value: 'full-content',
    icon: '📱',
  },
];

/** 内容宽度选项 */
const CONTENT_COMPACT_OPTIONS = [
  { label: 'layout.contentModes.wide', value: 'wide' },
  { label: 'layout.contentModes.compact', value: 'compact' },
];

export const LayoutPanel: React.FC = () => {
  const { preferences, setPreferences } = usePreferencesStore();
  const { t } = useTranslation('preferences');

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
              className="layout-wrapper"
              onClick={() => handleLayoutChange(option.value as LayoutType)}
            >
              <div
                className={`layout-item ${preferences.app.layout === option.value ? 'active' : ''}`}
              >
                <div className="layout-preview">
                {/* 渲染真实的布局示意图 */}
                {option.value === 'sidebar-nav' && (
                  <div className="layout-visual sidebar-nav">
                    <div className="sidebar" />
                    <div className="main-content">
                      <div className="header-bar gray" />
                      <div className="content-area">
                        <div className="block-row">
                          <div className="block" />
                          <div className="block" />
                        </div>
                        <div className="block" />
                      </div>
                    </div>
                  </div>
                )}
                {option.value === 'sidebar-mixed-nav' && (
                  <div className="layout-visual sidebar-mixed-nav">
                    <div className="sidebar thin primary" />
                    <div className="sidebar thin gray" />
                    <div className="main-content">
                      <div className="header-bar gray" />
                      <div className="content-area">
                        <div className="block-row">
                          <div className="block" />
                          <div className="block" />
                        </div>
                        <div className="block" />
                      </div>
                    </div>
                  </div>
                )}
                {option.value === 'header-nav' && (
                  <div className="layout-visual header-nav">
                    <div className="content-wrapper">
                      <div className="header-bar full-width">
                        <div className="menu-item" />
                        <div className="menu-item" />
                        <div className="menu-item" />
                        <div className="menu-item" />
                        <div className="menu-item" />
                      </div>
                      <div className="content-area">
                        <div className="block-row">
                          <div className="block" />
                          <div className="block" />
                        </div>
                        <div className="block" />
                      </div>
                    </div>
                  </div>
                )}
                {option.value === 'mixed-nav' && (
                  <div className="layout-visual mixed-nav">
                    <div className="content-wrapper">
                      <div className="header-bar full-width">
                        <div className="menu-item" />
                        <div className="menu-item" />
                        <div className="menu-item" />
                      </div>
                      <div className="main-content">
                        <div className="sidebar thin gray" />
                        <div className="content-area">
                          <div className="block-row">
                            <div className="block" />
                            <div className="block" />
                          </div>
                          <div className="block" />
                        </div>
                      </div>
                    </div>
                  </div>
                )}
                {option.value === 'full-content' && (
                  <div className="layout-visual full-content">
                    <div className="content-area full-width">
                      <div className="block-row">
                        <div className="block" />
                        <div className="block" />
                      </div>
                      <div className="block" />
                    </div>
                  </div>
                )}
                </div>
              </div>
              <div className="layout-label">
                <span>{t(option.label)}</span>
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
              className="content-compact-wrapper"
              onClick={() => handleContentCompactChange(option.value as ContentCompactType)}
            >
              <div
                className={`content-compact-item ${preferences.app.contentCompact === option.value ? 'active' : ''}`}
              >
                <div className="content-preview">
                <div className={`preview-bar ${option.value === 'compact' ? 'narrow' : 'wide'}`} />
                </div>
              </div>
              <span>{t(option.label)}</span>
            </div>
          ))}
        </div>
      </section>

      {/* 侧边栏设置 */}
      <section className="layout-section">
        <h3 className="section-title">{t('layout.sidebar')}</h3>
        <div className="preference-item">
          <span>{t('layout.showSidebar')}</span>
          <Switch
            checked={preferences.sidebar.enable}
            onChange={(checked) => setPreferences({ sidebar: { enable: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('layout.collapseMenu')}</span>
          <Switch
            checked={preferences.sidebar.collapsed}
            onChange={(checked) => setPreferences({ sidebar: { collapsed: checked } })}
          />
        </div>
        <div className={`preference-item ${!preferences.sidebar.collapsed ? 'disabled' : ''}`}>
          <span>{t('layout.collapseShowTitle')}</span>
          <Switch
            disabled={!preferences.sidebar.collapsed}
            checked={preferences.sidebar.collapsedShowTitle}
            onChange={(checked) => setPreferences({ sidebar: { collapsedShowTitle: checked } })}
          />
        </div>
        <div className="preference-item width-control">
          <span>{t('layout.width')}</span>
          <Space size={4}>
            <Button
              size="small"
              icon={<MinusOutlined />}
              onClick={() => handleSidebarWidthChange(preferences.sidebar.width - 8)}
              disabled={preferences.sidebar.width <= 180}
            />
            <InputNumber
              min={180}
              max={320}
              value={preferences.sidebar.width}
              onChange={handleSidebarWidthChange}
              style={{ width: 70 }} // 从80压缩到70
            />
            <Button
              size="small"
              icon={<PlusOutlined />}
              onClick={() => handleSidebarWidthChange(preferences.sidebar.width + 8)}
              disabled={preferences.sidebar.width >= 320}
            />
          </Space>
        </div>
      </section>

      {/* 顶栏设置 */}
      <section className="layout-section">
        <h3 className="section-title">{t('layout.header')}</h3>
        <div className="preference-item">
          <span>{t('layout.showHeader')}</span>
          <Switch
            checked={preferences.header.enable}
            onChange={(checked) => setPreferences({ header: { enable: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('layout.mode')}</span>
          <Segmented
            options={[
              { label: t('layout.headerModes.fixed'), value: 'fixed' },
              { label: t('layout.headerModes.auto'), value: 'auto' },
              { label: t('layout.headerModes.static'), value: 'static' },
            ]}
            value={preferences.header.mode}
            onChange={(value) => setPreferences({ header: { mode: value as any } })}
          />
        </div>
      </section>

      {/* 导航菜单 */}
      <section className="layout-section">
        <h3 className="section-title">{t('layout.navigationMenu')}</h3>
        <div className="preference-item">
          <span>{t('layout.menuStyle')}</span>
          <Segmented
            options={[
              { label: t('layout.menuStyles.rounded'), value: 'rounded' },
              { label: t('layout.menuStyles.plain'), value: 'plain' },
            ]}
            value={preferences.navigation.styleType}
            onChange={(value) => setPreferences({ navigation: { styleType: value as any } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('layout.menuSplit')}</span>
          <Switch
            checked={preferences.navigation.split}
            onChange={(checked) => setPreferences({ navigation: { split: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('layout.accordionMode')}</span>
          <Switch
            checked={preferences.navigation.accordion}
            onChange={(checked) => setPreferences({ navigation: { accordion: checked } })}
          />
        </div>
      </section>

      {/* 面包屑导航 */}
      <section className="layout-section">
        <h3 className="section-title">{t('layout.breadcrumb')}</h3>
        <div className="preference-item">
          <span>{t('layout.enableBreadcrumb')}</span>
          <Switch
            checked={preferences.breadcrumb.enable}
            onChange={(checked) => setPreferences({ breadcrumb: { enable: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('layout.hideWhenOnlyOne')}</span>
          <Switch
            checked={preferences.breadcrumb.hideOnlyOne}
            onChange={(checked) => setPreferences({ breadcrumb: { hideOnlyOne: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('layout.showIcon')}</span>
          <Switch
            checked={preferences.breadcrumb.showIcon}
            onChange={(checked) => setPreferences({ breadcrumb: { showIcon: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('layout.showHomeButton')}</span>
          <Switch
            checked={preferences.breadcrumb.showHome}
            onChange={(checked) => setPreferences({ breadcrumb: { showHome: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('layout.breadcrumbStyle')}</span>
          <Segmented
            options={[
              { label: t('layout.breadcrumbStyles.normal'), value: 'normal' },
              { label: t('layout.breadcrumbStyles.background'), value: 'background' },
            ]}
            value={preferences.breadcrumb.styleType}
            onChange={(value) => setPreferences({ breadcrumb: { styleType: value as any } })}
          />
        </div>
      </section>

      {/* 标签栏 */}
      <section className="layout-section">
        <h3 className="section-title">{t('layout.tabbar')}</h3>
        <div className="preference-item">
          <span>{t('layout.enableTabbar')}</span>
          <Switch
            checked={preferences.tabbar.enable}
            onChange={(checked) => setPreferences({ tabbar: { enable: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('layout.persistTabs')}</span>
          <Switch
            checked={preferences.tabbar.persist}
            onChange={(checked) => setPreferences({ tabbar: { persist: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('layout.draggableSort')}</span>
          <Switch
            checked={preferences.tabbar.draggable}
            onChange={(checked) => setPreferences({ tabbar: { draggable: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('layout.showTabbarIcon')}</span>
          <Switch
            checked={preferences.tabbar.showIcon}
            onChange={(checked) => setPreferences({ tabbar: { showIcon: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('layout.showMoreButton')}</span>
          <Switch
            checked={preferences.tabbar.showMore}
            onChange={(checked) => setPreferences({ tabbar: { showMore: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('layout.showMaximizeButton')}</span>
          <Switch
            checked={preferences.tabbar.showMaximize}
            onChange={(checked) => setPreferences({ tabbar: { showMaximize: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('layout.tabbarStyle')}</span>
          <Select
            style={{ width: 120 }}
            options={[
              { label: t('layout.tabbarStyles.chrome'), value: 'chrome' },
              { label: t('layout.tabbarStyles.card'), value: 'card' },
              { label: t('layout.tabbarStyles.brisk'), value: 'brisk' },
              { label: t('layout.tabbarStyles.plain'), value: 'plain' },
            ]}
            value={preferences.tabbar.styleType}
            onChange={(value) => setPreferences({ tabbar: { styleType: value as any } })}
          />
        </div>
      </section>

      {/* 小部件 */}
      <section className="layout-section">
        <h3 className="section-title">{t('layout.widgets')}</h3>
        <div className="preference-item">
          <span>{t('layout.globalSearch')}</span>
          <Switch
            checked={preferences.widget.globalSearch}
            onChange={(checked) => setPreferences({ widget: { globalSearch: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('layout.themeToggle')}</span>
          <Switch
            checked={preferences.widget.themeToggle}
            onChange={(checked) => setPreferences({ widget: { themeToggle: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('layout.languageToggle')}</span>
          <Switch
            checked={preferences.widget.languageToggle}
            onChange={(checked) => setPreferences({ widget: { languageToggle: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('layout.fullscreen')}</span>
          <Switch
            checked={preferences.widget.fullscreen}
            onChange={(checked) => setPreferences({ widget: { fullscreen: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('layout.notification')}</span>
          <Switch
            checked={preferences.widget.notification}
            onChange={(checked) => setPreferences({ widget: { notification: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('layout.lockScreen')}</span>
          <Switch
            checked={preferences.widget.lockScreen}
            onChange={(checked) => setPreferences({ widget: { lockScreen: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('layout.sidebarToggle')}</span>
          <Switch
            checked={preferences.widget.sidebarToggle}
            onChange={(checked) => setPreferences({ widget: { sidebarToggle: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('layout.refresh')}</span>
          <Switch
            checked={preferences.widget.refresh}
            onChange={(checked) => setPreferences({ widget: { refresh: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('layout.preferencesPosition')}</span>
          <Select
            style={{ width: 120 }}
            options={[
              { label: t('layout.positions.auto'), value: 'auto' },
              { label: t('layout.positions.fixed'), value: 'fixed' },
              { label: t('layout.positions.hidden'), value: 'hidden' },
            ]}
            value={preferences.app.preferencesButtonPosition}
            onChange={(value) =>
              setPreferences({ app: { preferencesButtonPosition: value as any } })
            }
          />
        </div>
      </section>

      {/* 底栏 */}
      <section className="layout-section">
        <h3 className="section-title">{t('layout.footer')}</h3>
        <div className="preference-item">
          <span>{t('layout.showFooter')}</span>
          <Switch
            checked={preferences.footer.enable}
            onChange={(checked) => setPreferences({ footer: { enable: checked } })}
          />
        </div>
        <div className={`preference-item ${!preferences.footer.enable ? 'disabled' : ''}`}>
          <span>{t('layout.fixedAtBottom')}</span>
          <Switch
            disabled={!preferences.footer.enable}
            checked={preferences.footer.fixed}
            onChange={(checked) => setPreferences({ footer: { fixed: checked } })}
          />
        </div>
      </section>

      {/* 版权 */}
      <section className="layout-section">
        <h3 className="section-title">{t('layout.copyright')}</h3>
        <div className="preference-item">
          <span>{t('layout.enableCopyright')}</span>
          <Switch
            checked={preferences.copyright.enable}
            onChange={(checked) => setPreferences({ copyright: { enable: checked } })}
          />
        </div>
        <div className={`preference-item ${!preferences.copyright.enable ? 'disabled' : ''}`}>
          <span>{t('layout.companyName')}</span>
          <Input
            style={{ width: 200 }}
            value={preferences.copyright.companyName}
            onChange={(e) => setPreferences({ copyright: { companyName: e.target.value } })}
            disabled={!preferences.copyright.enable}
          />
        </div>
        <div className={`preference-item ${!preferences.copyright.enable ? 'disabled' : ''}`}>
          <span>{t('layout.companySite')}</span>
          <Input
            style={{ width: 200 }}
            value={preferences.copyright.companySiteLink}
            onChange={(e) => setPreferences({ copyright: { companySiteLink: e.target.value } })}
            disabled={!preferences.copyright.enable}
          />
        </div>
        <div className={`preference-item ${!preferences.copyright.enable ? 'disabled' : ''}`}>
          <span>{t('layout.date')}</span>
          <Input
            style={{ width: 200 }}
            value={preferences.copyright.date}
            onChange={(e) => setPreferences({ copyright: { date: e.target.value } })}
            disabled={!preferences.copyright.enable}
          />
        </div>
        <div className={`preference-item ${!preferences.copyright.enable ? 'disabled' : ''}`}>
          <span>{t('layout.icpNumber')}</span>
          <Input
            style={{ width: 200 }}
            value={preferences.copyright.icp}
            onChange={(e) => setPreferences({ copyright: { icp: e.target.value } })}
            disabled={!preferences.copyright.enable}
          />
        </div>
        <div className={`preference-item ${!preferences.copyright.enable ? 'disabled' : ''}`}>
          <span>{t('layout.icpLink')}</span>
          <Input
            style={{ width: 200 }}
            value={preferences.copyright.icpLink}
            onChange={(e) => setPreferences({ copyright: { icpLink: e.target.value } })}
            disabled={!preferences.copyright.enable}
          />
        </div>
      </section>
    </div>
  );
};
