import { useState } from 'react';
import { Listy, Button, Drawer, Form, Input, App } from 'antd';
import { MailOutlined, MobileOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/api/client';
import { useGetUserProfile } from '@/api/hooks/user-profile';
import { useDrawerCloseGuard } from '@/components/common/use-drawer-close-guard';

const ICON_STYLE = { fontSize: 40 };

interface AccountBindProps {
  onSwitchTab: (key: string) => void;
}

const maskEmail = (email: string) => {
  const at = email.lastIndexOf('@');
  if (at <= 0) return email;
  return `${email.slice(0, Math.min(2, at))}***${email.slice(at)}`;
};

const maskPhone = (phone: string) => {
  if (phone.length < 7) return phone;
  return `${phone.slice(0, 3)}****${phone.slice(-4)}`;
};

const AccountBind = ({ onSwitchTab }: AccountBindProps) => {
  const { t } = useTranslation(['profile', 'common']);
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const { data: user } = useGetUserProfile();

  const [bindOpen, setBindOpen] = useState(false);
  // 关闭守卫：遮罩/ESC/X/取消统一，脏数据先确认（docs/form_drawer_close_guard.md）
  const { guardClose, handleValuesChange } = useDrawerCloseGuard(bindOpen, () => setBindOpen(false));
  const [bindForm] = Form.useForm();
  const [codeSent, setCodeSent] = useState(false);
  const [sending, setSending] = useState(false);
  const [verifying, setVerifying] = useState(false);

  const bindList = [
    {
      key: 'email',
      title: t('accountBind.email'),
      description: user?.email
        ? t('accountBind.bound', { value: maskEmail(user.email) })
        : t('accountBind.unbound'),
      icon: <MailOutlined style={ICON_STYLE} />,
    },
    {
      key: 'phone',
      title: t('accountBind.phone'),
      description: user?.mobile
        ? t('accountBind.bound', { value: maskPhone(user.mobile) })
        : t('accountBind.unbound'),
      icon: <MobileOutlined style={ICON_STYLE} />,
    },
  ];

  const openBind = () => {
    bindForm.setFieldsValue({ email: user?.email ?? '' });
    bindForm.resetFields(['code']);
    setCodeSent(false);
    setBindOpen(true);
  };

  const handleSendCode = async () => {
    const values = await bindForm.validateFields(['email']);
    const email = values.email?.trim();
    if (!email) {
      message.error(t('accountBind.newEmailRequired'));
      return;
    }
    setSending(true);
    try {
      await apiClient.userProfileService.BindContact({ email: { email } });
      setCodeSent(true);
      message.success(t('accountBind.codeSent'));
    } catch (error: any) {
      console.error('Failed to send bind code:', error);
      message.error(error?.message || t('accountBind.vcodeSendFailed'));
    } finally {
      setSending(false);
    }
  };

  const handleVerify = async () => {
    const values = await bindForm.validateFields();
    const email = values.email?.trim();
    const code = values.code?.trim();
    if (!email || !code) {
      message.error(t('accountBind.vcodeRequired'));
      return;
    }
    setVerifying(true);
    try {
      await apiClient.userProfileService.VerifyContact({
        email: { email, code },
      });
      message.success(t('accountBind.bindSuccess'));
      setBindOpen(false);
      // 刷新用户信息以更新绑定状态展示（本页读 getMe，顶栏头像等也读 getMe）
      queryClient.invalidateQueries({ queryKey: ['getMe'] });
    } catch (error: any) {
      console.error('Failed to verify bind code:', error);
      message.error(error?.message || t('accountBind.bindFailed'));
    } finally {
      setVerifying(false);
    }
  };

  return (
    <>
      {/* antd 6 弃用 List（提示改用 Listy），这里用 Listy + 自绘行还原原
          List.Item.Meta(avatar) + actions 的观感：行距/配色按原 List 样式 token
          写死（item padding 8px 0、头像右距 16、标题下 4px、描述用
          colorTextDescription、actions 与内容间距 24px、按钮间 8px），
          分隔线只在行间（末行无线）。 */}
      <Listy
        items={bindList}
        rowKey="key"
        styles={{ item: { padding: '8px 0', background: 'transparent' } }}
        itemRender={(item, index) => (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 24,
              borderBottom:
                index < bindList.length - 1
                  ? '1px solid var(--ant-color-split)'
                  : undefined,
            }}
          >
            <div style={{ display: 'flex', alignItems: 'flex-start', flex: 1, minWidth: 0, gap: 16 }}>
              {item.icon}
              <div style={{ minWidth: 0 }}>
                <div
                  style={{
                    fontSize: 14,
                    color: 'var(--ant-color-text)',
                    marginBottom: 4,
                  }}
                >
                  {item.title}
                </div>
                <div style={{ fontSize: 14, color: 'var(--ant-color-text-description)' }}>
                  {item.description}
                </div>
              </div>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', flexShrink: 0, gap: 8 }}>
              {item.key === 'email' && (
                <Button type="link" size="small" onClick={openBind}>
                  {user?.email ? t('accountBind.rebind') : t('accountBind.bindNow')}
                </Button>
              )}
              <Button type="link" size="small" onClick={() => onSwitchTab('1')}>
                {t('accountBind.modify')}
              </Button>
            </div>
          </div>
        )}
      />

      <Drawer
        title={t('accountBind.bindEmailTitle')}
        open={bindOpen}
        onClose={guardClose}
        size={480}
        destroyOnHidden
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <Button onClick={guardClose}>{t('common:button.cancel')}</Button>
            <Button type="primary" loading={verifying} onClick={handleVerify}>
              {t('common:button.ok')}
            </Button>
          </div>
        }
      >
        <Form form={bindForm} layout="horizontal" labelCol={{ flex: '0 0 120px' }} wrapperCol={{ flex: 'auto' }} onValuesChange={handleValuesChange}>
          <Form.Item
            name="email"
            label={t('accountBind.newEmail')}
            rules={[{ required: true, message: t('accountBind.newEmailRequired') }]}
          >
            <Input placeholder={t('accountBind.newEmailPlaceholder')} />
          </Form.Item>
          <Form.Item
            name="code"
            label={t('accountBind.vcode')}
            rules={[{ required: true, message: t('accountBind.vcodeRequired') }]}
          >
            <div style={{ display: 'flex', gap: 8 }}>
              <Input placeholder={t('accountBind.vcodePlaceholder')} />
              <Button disabled={codeSent} loading={sending} onClick={handleSendCode}>
                {codeSent ? t('accountBind.codeSent') : t('accountBind.sendVcode')}
              </Button>
            </div>
          </Form.Item>
        </Form>
      </Drawer>
    </>
  );
};

export default AccountBind;
