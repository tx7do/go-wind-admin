import { LoginForm, ProFormCheckbox, ProFormText } from '@ant-design/pro-components';
import React from 'react';
import { useTranslation } from 'react-i18next';
import { useAuthStore } from '@/stores';
import { usePreferences } from '@/core/preferences';

import './login.style.less';
import AuthLayout from '@/components/bussiness/AuthLayout';

const Login: React.FC = () => {
  const { t } = useTranslation();
  const { login, loginLoading } = useAuthStore();
  const { theme } = usePreferences();

  // 根据主题模式判断当前是否为亮色模式
  const isLightMode = React.useMemo(() => {
    if (theme.mode === 'auto') {
      return window.matchMedia('(prefers-color-scheme: light)').matches;
    }
    return theme.mode === 'light';
  }, [theme.mode]);

  const handleSubmit = async (values: { username: string; password: string }) => {
    try {
      await login({
        username: values.username,
        password: values.password,
        grant_type: 'password',
      });

      // 等待一小段时间确保 localStorage 写入完成，然后跳转
      setTimeout(() => {
        const urlParams = new URL(window.location.href).searchParams;
        window.location.href = urlParams.get('redirect') || '/';
      }, 300);
    } catch (error: any) {
      // 错误已在 store 中处理，这里不需要再次弹出 message
    }
  };

  return (
    <AuthLayout
      title={t('auth:welcomeBack')}
      description={t('auth:loginDescription')}
      pageKey="login"
      footerLink={{
        text: t('auth:noAccount'),
        linkText: t('auth:createAccount'),
        href: '/auth/register',
      }}
    >
      <LoginForm
        loading={loginLoading}
        logo={false}
        title={false}
        subTitle={false}
        initialValues={{
          autoLogin: true,
        }}
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
            autoComplete: 'current-password',
          }}
          rules={[
            {
              required: true,
              message: t('auth:passwordRequired'),
            },
          ]}
        />
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginBottom: 24,
            marginTop: 8,
          }}
        >
          <ProFormCheckbox
            name="autoLogin"
            fieldProps={{
              style: {
                color: isLightMode ? 'rgba(0, 0, 0, 0.7)' : 'rgba(255, 255, 255, 0.75)',
                fontSize: 13,
              },
            }}
          >
            {t('auth:rememberAccount')}
          </ProFormCheckbox>
        </div>
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
              cursor: loginLoading ? 'not-allowed' : 'pointer',
              opacity: loginLoading ? 0.7 : 1,
              transition: 'all 0.3s ease',
              boxShadow: '0 2px 0 rgba(5, 145, 255, 0.1)',
              letterSpacing: '0.5px',
            }}
            disabled={loginLoading}
          >
            {loginLoading ? t('auth:loggingIn') : t('auth:loginButton')}
          </button>
        </div>
      </LoginForm>
    </AuthLayout>
  );
};

export default Login;
