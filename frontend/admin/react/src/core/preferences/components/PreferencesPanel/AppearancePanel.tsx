import React from 'react';
import { Radio, Segmented, Switch } from 'antd';
import { BulbOutlined, LaptopOutlined, MoonOutlined, BgColorsOutlined } from '@ant-design/icons';
import { usePreferencesStore } from '../../store';
import type { BuiltinThemeType, ThemeModeType } from '../../types';
import './AppearancePanel.style.less';

/** 内置主题色配置 */
const BUILTIN_THEMES: { name: string; color: string; type: BuiltinThemeType }[] = [
  { name: '默认', color: '#1677ff', type: 'default' },
  { name: '紫罗兰', color: '#722ed1', type: 'violet' },
  { name: '樱花粉', color: '#eb2f96', type: 'pink' },
  { name: '柠檬黄', color: '#fadb14', type: 'yellow' },
  { name: '天蓝色', color: '#52c41a', type: 'sky-blue' },
  { name: '浅绿色', color: '#13c2c2', type: 'green' },
  { name: '锌色灰', color: '#595959', type: 'zinc' },
  { name: '深绿色', color: '#13a8a8', type: 'deep-green' },
  { name: '深蓝色', color: '#1677ff', type: 'deep-blue' },
  { name: '橙黄色', color: '#fa8c16', type: 'orange' },
  { name: '玫瑰红', color: '#f5222d', type: 'rose' },
  { name: '中性色', color: '#595959', type: 'neutral' },
  { name: '石板灰', color: '#54687a', type: 'slate' },
  { name: '中灰色', color: '#595959', type: 'gray' },
  { name: '自定义', color: 'custom', type: 'custom' },
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
        <h3 className="section-title">主题</h3>
        <Radio.Group
          value={preferences.theme.mode}
          onChange={(e) => handleThemeModeChange(e.target.value)}
          className="theme-mode-group"
        >
          <div className="theme-mode-options">
            <Radio.Button value="light" className="theme-mode-btn">
              <div className="theme-mode-content">
                <BulbOutlined className="theme-icon" />
                <span>浅色</span>
              </div>
            </Radio.Button>
            <Radio.Button value="dark" className="theme-mode-btn">
              <div className="theme-mode-content">
                <MoonOutlined className="theme-icon" />
                <span>深色</span>
              </div>
            </Radio.Button>
            <Radio.Button value="auto" className="theme-mode-btn">
              <div className="theme-mode-content">
                <LaptopOutlined className="theme-icon" />
                <span>跟随系统</span>
              </div>
            </Radio.Button>
          </div>
        </Radio.Group>
      </section>

      {/* 深色侧边栏和顶栏 */}
      <section className="appearance-section">
        <div className="preference-item">
          <span>深色侧边栏</span>
          <Switch
            checked={preferences.theme.semiDarkSidebar}
            onChange={(checked) => setPreferences({ theme: { semiDarkSidebar: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>深色顶栏</span>
          <Switch
            checked={preferences.theme.semiDarkHeader}
            onChange={(checked) => setPreferences({ theme: { semiDarkHeader: checked } })}
          />
        </div>
      </section>

      {/* 内置主题 */}
      <section className="appearance-section">
        <h3 className="section-title">内置主题</h3>
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
              <span className="theme-name">{theme.name}</span>
            </div>
          ))}
        </div>
      </section>

      {/* 圆角 */}
      <section className="appearance-section">
        <h3 className="section-title">圆角</h3>
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
        <h3 className="section-title">其它</h3>
        <div className="preference-item">
          <span>色弱模式</span>
          <Switch
            checked={preferences.app.colorWeakMode}
            onChange={(checked) => setPreferences({ app: { colorWeakMode: checked } })}
          />
        </div>
        <div className="preference-item">
          <span>灰色模式</span>
          <Switch
            checked={preferences.app.colorGrayMode}
            onChange={(checked) => setPreferences({ app: { colorGrayMode: checked } })}
          />
        </div>
      </section>
    </div>
  );
};
