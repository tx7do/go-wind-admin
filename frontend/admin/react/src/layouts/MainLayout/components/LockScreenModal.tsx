import { useEffect, useState } from 'react';
import { Avatar, Button, Input, Modal } from 'antd';
import { UserOutlined } from '@ant-design/icons';

import { useI18n } from '@go-wind/admin-react';

interface LockScreenModalProps {
  open: boolean;
  avatar?: string;
  text?: string;
  onCancel: () => void;
  onSubmit: (password: string) => void;
}

/** 锁定屏幕前的设密弹窗（行为基准：vue-vben lock-screen-modal.vue） */
export const LockScreenModal = ({ open, avatar, text, onCancel, onSubmit }: LockScreenModalProps) => {
  const { t } = useI18n('common');
  const [password, setPassword] = useState('');

  useEffect(() => {
    if (open) setPassword('');
  }, [open]);

  const submit = () => {
    if (!password) return;
    onSubmit(password);
  };

  return (
    <Modal
      open={open}
      title={t('lockScreen.title')}
      footer={null}
      width={360}
      destroyOnHidden
      onCancel={onCancel}
    >
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          padding: '8px 0 4px',
        }}
      >
        <Avatar src={avatar} icon={<UserOutlined />} size={64} />
        <div
          style={{
            marginTop: 16,
            marginBottom: 20,
            fontSize: 14,
            fontWeight: 500,
            color: 'var(--ant-color-text)',
          }}
        >
          {text}
        </div>
        <Input.Password
          autoFocus
          value={password}
          style={{ width: '100%' }}
          placeholder={t('lockScreen.placeholder')}
          onChange={(e) => setPassword(e.target.value)}
          onPressEnter={submit}
        />
        <Button type="primary" block style={{ marginTop: 12 }} disabled={!password} onClick={submit}>
          {t('lockScreen.lock')}
        </Button>
      </div>
    </Modal>
  );
};

export default LockScreenModal;
