import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Form, Input, Button, App } from 'antd';
import { SafetyOutlined } from '@ant-design/icons';
import { useAuthStore } from '@go-wind/admin-react';
import { useSearchParams } from 'react-router-dom';

const MfaChallenge: React.FC = () => {
  const { t } = useTranslation('auth');
  const { completeMfaChallenge, loginLoading, mfaOperationId } = useAuthStore();
  const { message } = App.useApp();
  const [submitting, setSubmitting] = useState(false);
  const [searchParams] = useSearchParams();

  // 退出 MFA 挑战：清状态并回登录页
  const cancelMfa = () => {
    useAuthStore.setState({ mfaOperationId: null });
    window.location.href = '/auth/login';
  };

  const handleSubmit = async (values: { totpCode: string }) => {
    if (!mfaOperationId) {
      message.error(t('mfaNotRequired'));
      window.location.href = '/auth/login';
      return;
    }
    setSubmitting(true);
    // 验证成功后回到进入登录流程前的原目标页（由登录 store 整页跳转时透传）
    const redirect = searchParams.get('redirect') || '';
    const safeRedirect =
      typeof redirect === 'string' && redirect.startsWith('/') && !redirect.startsWith('//')
        ? redirect
        : '';
    try {
      // 有 redirect 才传 onSuccess（覆盖默认 homePath 跳转）；
      // 空时传 undefined，走 store 默认的 homePath 跳转
      await completeMfaChallenge(
        values.totpCode,
        safeRedirect ? () => { window.location.href = safeRedirect; } : undefined,
      );
    } catch (error: any) {
      message.error(error?.message || t('mfaVerifyFailed'));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-gray-900 via-gray-800 to-gray-900 p-4">
      <div className="w-full max-w-md rounded-xl border border-white/10 bg-white/5 p-8 shadow-[0_8px_32px_rgba(0,0,0,0.3)] backdrop-blur-md light:border-gray-100 light:bg-white light:shadow-lg">
        <h2 className="mb-2 text-center text-xl font-semibold text-white light:text-gray-900">
          {t('mfaChallengeTitle')}
        </h2>
        <p className="mb-6 text-center text-sm text-gray-300 light:text-gray-500">
          {t('mfaChallengeDesc')}
        </p>
        <Form name="mfa-challenge" onFinish={handleSubmit} size="large">
          <Form.Item
            name="totpCode"
            rules={[
              { required: true, message: t('mfaCodeRequired') },
              {
                pattern: /^\d{6}$/,
                message: t('mfaCodeFormatError'),
              },
            ]}
          >
            <Input
              prefix={<SafetyOutlined />}
              aria-label={t('mfaCodeLabel')}
              placeholder={t('mfaCodePlaceholder')}
              autoComplete="one-time-code"
              inputMode="numeric"
              maxLength={6}
            />
          </Form.Item>
          <Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              block
              loading={submitting || loginLoading}
            >
              {t('mfaVerify')}
            </Button>
          </Form.Item>
          <Form.Item>
            <Button block onClick={cancelMfa}>
              {t('mfaCancel')}
            </Button>
          </Form.Item>
        </Form>
      </div>
    </div>
  );
};

export default MfaChallenge;
