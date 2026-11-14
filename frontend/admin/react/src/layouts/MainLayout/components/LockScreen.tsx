import { useEffect, useState } from 'react';
import type { CSSProperties } from 'react';
import { Avatar, Button, Input } from 'antd';
import { LockOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';

import { useI18n } from '@go-wind/admin-react';
import { usePreferencesStore } from '@go-wind/admin-react';
import { useLockStore } from '@/stores';

interface LockScreenProps {
  avatar?: string;
  onToLogin: () => void;
}

/**
 * 全屏锁屏遮罩（行为基准：vue-vben lock-screen.vue）
 *
 * 时钟态与解锁表单态互斥：点「点击解锁」进表单，密码比对 lockScreenPassword，
 * 错了就地提示而非静默——静默失败会让人以为是输入框没生效。
 */
export const LockScreen = ({ avatar, onToLogin }: LockScreenProps) => {
  const { t } = useI18n('common');
  const locale = usePreferencesStore((state) => state.preferences.app.locale);
  const lockScreenEnabled = usePreferencesStore((state) => state.preferences.widget?.lockScreen ?? true);

  const isLockScreen = useLockStore((state) => state.isLockScreen);
  const lockScreenPassword = useLockStore((state) => state.lockScreenPassword);
  const unlockScreen = useLockStore((state) => state.unlockScreen);

  const [now, setNow] = useState<Dayjs>(() => dayjs());
  const [showUnlockForm, setShowUnlockForm] = useState(false);
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');

  const visible = lockScreenEnabled && isLockScreen;

  useEffect(() => {
    if (!visible) return;
    setNow(dayjs());
    const timer = window.setInterval(() => setNow(dayjs()), 1000);
    return () => window.clearInterval(timer);
  }, [visible]);

  useEffect(() => {
    if (!visible) return;
    const previous = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = previous;
    };
  }, [visible]);

  if (!visible) return null;

  const hour = now.format('HH');
  const minute = now.format('mm');
  const meridiem = now.format('A');
  const date = locale.startsWith('zh') ? now.locale('zh-cn').format('YYYY-MM-DD dddd') : now.format('YYYY-MM-DD dddd');

  const submit = () => {
    if (!password) return;
    if (password === lockScreenPassword) {
      unlockScreen();
      setPassword('');
      setError('');
    } else {
      setError(t('lockScreen.errorTip'));
    }
  };

  const toggleForm = () => {
    setShowUnlockForm((prev) => !prev);
    setPassword('');
    setError('');
  };

  const labelStyle: CSSProperties = {
    fontSize: 14,
    color: 'var(--ant-color-text-secondary)',
  };

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 2000,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 24,
        background: 'var(--ant-color-bg-layout)',
      }}
    >
      {!showUnlockForm ? (
        <>
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              gap: 4,
              cursor: 'pointer',
              fontSize: 16,
              fontWeight: 600,
              color: 'var(--ant-color-text)',
            }}
            onClick={toggleForm}
          >
            <LockOutlined style={{ fontSize: 20 }} />
            <span style={labelStyle}>{t('lockScreen.unlock')}</span>
          </div>
          <div style={{ position: 'relative', fontSize: 160, fontWeight: 700, lineHeight: 1, color: 'var(--ant-color-text)' }}>
            {hour}:{minute}
            <span style={{ position: 'absolute', top: 12, marginLeft: 16, fontSize: 24, fontWeight: 500, color: 'var(--ant-color-text-secondary)' }}>
              {meridiem}
            </span>
          </div>
        </>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', width: 300 }} onKeyDown={(e) => e.key === 'Enter' && submit()}>
          <Avatar src={avatar} size={80} />
          <Input.Password
            autoFocus
            value={password}
            status={error ? 'error' : undefined}
            style={{ width: '100%', marginTop: 24 }}
            placeholder={t('lockScreen.placeholder')}
            onChange={(e) => {
              setPassword(e.target.value);
              setError('');
            }}
          />
          {error && <div style={{ width: '100%', marginTop: 8, fontSize: 12, color: 'var(--ant-color-error)' }}>{error}</div>}
          <Button type="primary" block style={{ marginTop: 16 }} onClick={submit}>
            {t('lockScreen.entry')}
          </Button>
          <Button block style={{ marginTop: 8 }} onClick={onToLogin}>
            {t('lockScreen.backToLogin')}
          </Button>
          <Button type="text" block style={{ marginTop: 8 }} onClick={toggleForm}>
            {t('button.back')}
          </Button>
        </div>
      )}

      <div style={{ position: 'absolute', bottom: 24, width: '100%', textAlign: 'center', color: 'var(--ant-color-text-secondary)' }}>
        {showUnlockForm && (
          <div style={{ fontSize: 28, marginBottom: 4, color: 'var(--ant-color-text)' }}>
            {hour}:{minute} <span style={{ fontSize: 16 }}>{meridiem}</span>
          </div>
        )}
        <div style={{ fontSize: 24 }}>{date}</div>
      </div>
    </div>
  );
};

export default LockScreen;
