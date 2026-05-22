import React from 'react';
import { Radio, Segmented, Switch } from 'antd';
import { BulbOutlined, LaptopOutlined, MoonOutlined, BgColorsOutlined } from '@ant-design/icons';
import { useDynamicI18n } from '@/core/i18n';
import { usePreferencesStore } from '../../store';
import type { BuiltinThemeType, ThemeModeType } from '../../types';
import './AppearancePanel.style.less';

/** 内置主题色配置 */
const BUILTIN_THEMES: { nameKey: string; color: string; type: BuiltinThemeType }[] = [
  { nameKey: 'appearance.themes.default', color: '#1677ff', type: 'default' },
  { nameKey: 'appearance.themes.violet', color: '#722ed1', type: 'violet' },
  { nameKey: 'appearance.themes.pink', color: '#eb2f96', type: 'pink' },
  { nameKey: 'appearance.themes.yellow', color: '#fadb14', type: 'yellow' },
  { nameKey: 'appearance.themes.skyBlue', color: '#52c41a', type: 'sky-blue' },
  { nameKey: 'appearance.themes.green', color: '#13c2c2', type: 'green' },
  { nameKey: 'appearance.themes.zinc', color: '#595959', type: 'zinc' },
  { nameKey: 'appearance.themes.deepGreen', color: '#13a8a8', type: 'deep-green' },
  { nameKey: 'appearance.themes.deepBlue', color: '#1677ff', type: 'deep-blue' },
  { nameKey: 'appearance.themes.orange', color: '#fa8c16', type: 'orange' },
  { nameKey: 'appearance.themes.rose', color: '#f5222d', type: 'rose' },
  { nameKey: 'appearance.themes.neutral', color: '#595959', type: 'neutral' },
  { nameKey: 'appearance.themes.slate', color: '#54687a', type: 'slate' },
  { nameKey: 'appearance.themes.gray', color: '#595959', type: 'gray' },
  { nameKey: 'appearance.themes.custom', color: 'custom', type: 'custom' },
];

/** 圆角选项 */
const RADIUS_OPTIONS = [
  { label: '0', value: '0' },
  { label: '0.25', value: '0.25' },
  { label: '0.5', value: '0.5' },
  { label: '0.75', value: '0.75' },
  { label: '1', value: '1' },
];

export const AppearancePanel: React.FC = () => {
  const { preferences, setPreferences } = usePreferencesStore();
  const { t } = useDynamicI18n({ namespace: 'preferences' });

  // 主题模式切换
  const handleThemeModeChange = (mode: ThemeModeType) => {
    setPreferences({ theme: { mode } });
  };

  // 内置主题切换
  const handleBuiltinThemeChange = (type: BuiltinThemeType) => {
    const theme = BUILTIN_THEMES.find((t) => t.type === type);
    if (theme) {
      setPreferences({ theme: { builtinType: type, colorPrimary: theme.color } });
    }
  };

  // 圆角切换
  const handleRadiusChange = (radius: string) => {
    setPreferences({ theme: { radius } });
  };

  return (
    <div className="appearance-panel">
      {/* 主题模式 */}
      <section className="appearance-section">
        <h3 className="section-title">{t('appearance.theme')}</h3>
        <Radio.Group
          value={preferences.theme.mode}
          onChange={(e) => handleThemeModeChange(e.target.value)}
          className="theme-mode-group"
        >
          <div className="theme-mode-options">
            <Radio.Button value="light" className="theme-mode-btn">
              <div className="theme-mode-content">
                <BulbOutlined className="theme-icon" />
                <span>{t('appearance.lightMode')}</span>
              </div>
            </Radio.Button>
            <Radio.Button value="dark" className="theme-mode-btn">
              <div className="theme-mode-content">
                <MoonOutlined className="theme-icon" />
                <span>{t('appearance.darkMode')}</span>
              </div>
            </Radio.Button>
            <Radio.Button value="auto" className="theme-mode-btn">
              <div className="theme-mode-content">
                <LaptopOutlined className="theme-icon" />
                <span>{t('appearance.followSystem')}</span>
              </div>
            </Radio.Button>
          </div>
        </Radio.Group>
      </section>

      {/* 深色侧边栏和顶栏 */}
      <section className="appearance-section">
        <div className="preference-item">
          <span>{t('appearance.darkSidebar')}</span>
          <Switch
            checked={preferences.theme.semiDarkSidebar}
            onChange={(checked) => setPreferences({ theme: { semiDarkSidebar: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('appearance.darkHeader')}</span>
          <Switch
            checked={preferences.theme.semiDarkHeader}
            onChange={(checked) => setPreferences({ theme: { semiDarkHeader: checked } })}
          />
        </div>
      </section>

      {/* 内置主题 */}
      <section className="appearance-section">
        <h3 className="section-title">{t('appearance.builtinThemes')}</h3>
        <div className="builtin-themes-grid">
          {BUILTIN_THEMES.map((theme) => (
            <div
              key={theme.type}
              className={`builtin-theme-item ${preferences.theme.builtinType === theme.type ? 'active' : ''}`}
              onClick={() => handleBuiltinThemeChange(theme.type)}
            >
              <div
                className="theme-color-preview"
                style={{
                  backgroundColor: theme.type === 'custom' ? 'transparent' : theme.color,
                  border: theme.type === 'custom' ? '1px dashed #666' : 'none',
                }}
              >
                {theme.type === 'custom' && <BgColorsOutlined className="custom-icon" />}
              </div>
              <span className="theme-name">{t(theme.nameKey)}</span>
            </div>
          ))}
        </div>
      </section>

      {/* 圆角 */}
      <section className="appearance-section">
        <h3 className="section-title">{t('appearance.radius')}</h3>
        <Segmented
          options={RADIUS_OPTIONS}
          value={preferences.theme.radius}
          onChange={(value) => handleRadiusChange(value as string)}
          className="radius-segmented"
          block
        />
      </section>

      {/* 其它 */}
      <section className="appearance-section">
        <h3 className="section-title">{t('appearance.other')}</h3>
        <div className="preference-item">
          <span>{t('appearance.colorWeakMode')}</span>
          <Switch
            checked={preferences.app.colorWeakMode}
            onChange={(checked) => setPreferences({ app: { colorWeakMode: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>{t('appearance.colorGrayMode')}</span>
          <Switch
            checked={preferences.app.colorGrayMode}
            onChange={(checked) => setPreferences({ app: { colorGrayMode: checked } })}
          />
        </div>
      </section>
    </div>
  );
};
