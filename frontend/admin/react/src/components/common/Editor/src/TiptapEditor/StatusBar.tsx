import { useTranslation } from 'react-i18next';

interface StatusBarProps {
  words: number;
  chars: number;
  cursor: string;
  isDark: boolean;
}

export const StatusBar: React.FC<StatusBarProps> = ({ words, chars, cursor, isDark }) => {
  const { t } = useTranslation();

  return (
    <div className="tiptap-statusbar">
      <div className="status-info">
        <span className="status-item">
          {words} {t('editor:words', '词')}
        </span>
        <span className="status-divider">|</span>
        <span className="status-item">
          {chars} {t('editor:chars', '字符')}
        </span>
        <span className="status-divider">|</span>
        <span className="status-item">Ln {cursor}</span>
      </div>
      <div className="status-mode">
        <span className={`mode-badge${isDark ? ' mode-dark' : ''}`}>
          {isDark
            ? `🌙 ${t('preferences.theme.dark', '暗色')}`
            : `☀️ ${t('preferences.theme.light', '亮色')}`}
        </span>
      </div>
    </div>
  );
};
