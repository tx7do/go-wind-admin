import React from 'react';
import { Switch } from 'antd';
import { useTranslation } from 'react-i18next';
import { usePreferencesStore } from '../../store';
import './ShortcutKeyPanel.style.less';

export const ShortcutKeyPanel: React.FC = () => {
  const { preferences, setPreferences } = usePreferencesStore();
  const { t } = useTranslation('preferences');

  return (
    <div className="shortcut-panel">
      <section className="shortcut-section">
        <h3 className="section-title">{t('shortcut.global')}</h3>

        <div className="preference-item">
          <span>{t('shortcut.shortcutKeys')}</span>
          <Switch
            checked={preferences.shortcutKeys.enable}
            onChange={(checked) => setPreferences({ shortcutKeys: { enable: checked } })}
          />
        </div>

        <div className={`preference-item ${!preferences.shortcutKeys.enable ? 'disabled' : ''}`}>
          <span>{t('shortcut.globalSearch')}</span>
          <div className="shortcut-key-row">
            <kbd className="kbd-badge">Ctrl</kbd>
            <kbd className="kbd-badge">K</kbd>
            <Switch
              disabled={!preferences.shortcutKeys.enable}
              checked={preferences.shortcutKeys.globalSearch}
              onChange={(checked) => setPreferences({ shortcutKeys: { globalSearch: checked } })}
            />
          </div>
        </div>

        <div className={`preference-item ${!preferences.shortcutKeys.enable ? 'disabled' : ''}`}>
          <span>{t('shortcut.logout')}</span>
          <div className="shortcut-key-row">
            <kbd className="kbd-badge">Alt</kbd>
            <kbd className="kbd-badge">Q</kbd>
            <Switch
              disabled={!preferences.shortcutKeys.enable}
              checked={preferences.shortcutKeys.globalLogout}
              onChange={(checked) => setPreferences({ shortcutKeys: { globalLogout: checked } })}
            />
          </div>
        </div>

        <div className={`preference-item ${!preferences.shortcutKeys.enable ? 'disabled' : ''}`}>
          <span>{t('shortcut.lockScreen')}</span>
          <div className="shortcut-key-row">
            <kbd className="kbd-badge">Alt</kbd>
            <kbd className="kbd-badge">L</kbd>
            <Switch
              disabled={!preferences.shortcutKeys.enable}
              checked={preferences.shortcutKeys.globalLockScreen}
              onChange={(checked) =>
                setPreferences({ shortcutKeys: { globalLockScreen: checked } })
              }
            />
          </div>
        </div>
      </section>
    </div>
  );
};
