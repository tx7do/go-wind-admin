import React from 'react';
import { Select, Switch } from 'antd';
import { useTranslation } from 'react-i18next';
import { usePreferencesStore } from '../../store';
import type { SupportedLanguagesType } from '../../types';
import './GeneralPanel.style.less';

/** 页面切换动画选项 */
const TRANSITION_OPTIONS = [
  { label: 'general.transitions.fade', value: 'fade', icon: '✨' },
  { label: 'general.transitions.fadeDown', value: 'fade-down', icon: '️' },
  { label: 'general.transitions.fadeSlide', value: 'fade-slide', icon: '↔️' },
  { label: 'general.transitions.fadeUp', value: 'fade-up', icon: '⬆️' },
];

/** 语言选项 */
const LANGUAGE_OPTIONS = [
  { label: 'general.languages.zhCN', value: 'zh-CN' },
  { label: 'general.languages.enUS', value: 'en-US' },
];

export const GeneralPanel: React.FC = () => {
  const { preferences, setPreferences } = usePreferencesStore();
  const { t } = useTranslation('preferences');

  const handleLanguageChange = (locale: SupportedLanguagesType) => {
    setPreferences({ app: { locale } });
  };

  return (
    <div className="general-panel">
      <section className="general-section">
        <h3 className="section-title">{t('general.general')}</h3>

        <div className="preference-item">
          <span>{t('general.language')}</span>
          <Select
            value={preferences.app.locale}
            options={LANGUAGE_OPTIONS.map((opt) => ({ ...opt, label: t(opt.label) }))}
            onChange={handleLanguageChange}
            style={{ width: 200 }}
          />
        </div>

        <div className="preference-item">
          <span>{t('general.dynamicTitle')}</span>
          <Switch
            checked={preferences.app.dynamicTitle}
            onChange={(checked) => setPreferences({ app: { dynamicTitle: checked } })}
          />
        </div>

        <div className="preference-item">
          <span>{t('general.watermark')}</span>
          <Switch
            checked={preferences.app.watermark}
            onChange={(checked) => setPreferences({ app: { watermark: checked } })}
          />
        </div>

        <div className="preference-item">
          <span>{t('general.checkUpdates')}</span>
          <Switch
            checked={preferences.app.enableCheckUpdates}
            onChange={(checked) => setPreferences({ app: { enableCheckUpdates: checked } })}
          />
        </div>
      </section>

      <section className="general-section">
        <h3 className="section-title">{t('general.animation')}</h3>

        <div className="preference-item">
          <span>{t('general.progressBar')}</span>
          <Switch
            checked={preferences.transition.progress}
            onChange={(checked) => setPreferences({ transition: { progress: checked } })}
          />
        </div>

        <div className="preference-item">
          <span>{t('general.loading')}</span>
          <Switch
            checked={preferences.transition.loading}
            onChange={(checked) => setPreferences({ transition: { loading: checked } })}
          />
        </div>

        <div className="preference-item">
          <span>{t('general.pageTransition')}</span>
          <Switch
            checked={preferences.transition.enable}
            onChange={(checked) => setPreferences({ transition: { enable: checked } })}
          />
        </div>

        <div
          className={`transition-animations ${!preferences.transition.enable ? 'disabled' : ''}`}
        >
          <div className="transition-grid">
            {TRANSITION_OPTIONS.map((option) => (
              <div
                key={option.value}
                className="transition-wrapper"
                onClick={() => {
                  if (preferences.transition.enable) {
                    setPreferences({ transition: { name: option.value } });
                  }
                }}
              >
                <div
                  className={`transition-item ${preferences.transition.name === option.value ? 'active' : ''}`}
                >
                  <div className="transition-preview">
                    <div className={`animation-box ${option.value}`} />
                  </div>
                </div>
                <span className="transition-label">{t(option.label)}</span>
              </div>
            ))}
          </div>
        </div>
      </section>
    </div>
  );
};
