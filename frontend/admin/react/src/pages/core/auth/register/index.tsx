import { LoginForm, ProFormText } from '@ant-design/pro-components';
import React from 'react';
import { useTranslation } from 'react-i18next';
import { useAuthStore } from '@/stores';
import { usePreferences } from '@/core/preferences';
import { message } from 'antd';

import './register.style.less';
import AuthLayout from '@/components/bussiness/AuthLayout';

const Register: React.FC = () => {
  const { t } = useTranslation();
  const { register, registerLoading } = useAuthStore();
  const { theme } = usePreferences();

  // 根据主题模式判断当前是否为亮色模式
  React.useMemo(() => {
    if (theme.mode === 'auto') {
      return window.matchMedia('(prefers-color-scheme: light)').matches;
    }
    return theme.mode === 'light';
  }, [theme.mode]);
  const handleSubmit = async (values: { username: string; password: string; confirmPassword: string }) => {
    // 验证密码一致性
    if (values.password !== values.confirmPassword) {
      message.error(t('auth:passwordMismatch'));
      return;
    }

    try {
      await register({
        username: values.username,
        password: values.password,
      });

      // 注册成功后跳转到登录页
      setTimeout(() => {
        window.location.href = '/login';
      }, 300);
    } catch (error: any) {
      // 错误已在 store 中处理
    }
  };

  return (
    <AuthLayout
      title={t('auth:registerTitle')}
      description={t('auth:registerDescription')}
      pageKey="register"
      footerLink={{
        text: t('auth:hasAccount'),
        linkText: t('auth:backToLogin'),
        href: '/login',
      }}
    >
      <LoginForm
        loading={registerLoading}
        logo={false}
        title={false}
        subTitle={false}
        onFinish={handleSubmit}
        submitter={false}
      >
        <ProFormText
          name="username"
          fieldProps={{
            size: 'large',
            placeholder: t('auth:usernamePlaceholder'),
            autoComplete: 'username',
          }}
          rules={[
            {
              required: true,
              message: t('auth:usernameRequired'),
            },
          ]}
        />
        <ProFormText.Password
          name="password"
          fieldProps={{
            size: 'large',
            placeholder: t('auth:passwordPlaceholder'),
            autoComplete: 'new-password',
          }}
          rules={[
            {
              required: true,
              message: t('auth:passwordRequired'),
            },
            {
              min: 6,
              message: t('auth:passwordMinLength'),
            },
          ]}
        />
        <ProFormText.Password
          name="confirmPassword"
          fieldProps={{
            size: 'large',
            placeholder: t('auth:confirmPasswordPlaceholder'),
            autoComplete: 'new-password',
          }}
          rules={[
            {
              required: true,
              message: t('auth:confirmPasswordRequired'),
            },
          ]}
        />
        <div style={{ marginTop: 36 }}>
          <button
            type="submit"
            style={{
              width: '100%',
              height: 40,
              background: '#1677ff',
              border: 'none',
              borderRadius: 6,
              color: '#fff',
              fontSize: 14,
              fontWeight: 500,
              cursor: registerLoading ? 'not-allowed' : 'pointer',
              opacity: registerLoading ? 0.7 : 1,
              transition: 'all 0.3s ease',
              boxShadow: '0 2px 0 rgba(5, 145, 255, 0.1)',
              letterSpacing: '0.5px',
            }}
            disabled={registerLoading}
          >
            {registerLoading ? t('auth:registering') : t('auth:registerButton')}
          </button>
        </div>
      </LoginForm>
    </AuthLayout>
  );
};

export default Register;
