import {LoginForm, ProFormCheckbox, ProFormText} from '@ant-design/pro-components';
import {useIntl, useModel} from '@umijs/max';
import {App} from 'antd';
import React from 'react';

import './login.style.less';
import AuthLayout from "@/components/AuthLayout";

const Login: React.FC = () => {
  const {message} = App.useApp();
  const intl = useIntl();
  const {login, loginLoading} = useModel('business.authentication');

  const {mode: themeMode} = useModel('core.theme');


  // 根据主题模式判断当前是否为亮色模式
  const isLightMode = React.useMemo(() => {
    if (themeMode === 'system') {
      return window.matchMedia('(prefers-color-scheme: light)').matches;
    }
    return themeMode === 'light';
  }, [themeMode]);

  const handleSubmit = async (values: { username: string; password: string }) => {
    console.log('[Login] Form submitted with values:', values);

    try {
      console.log('[Login] Calling login function...');
      await login(
        {
          username: values.username,
          password: values.password,
          grant_type: 'password',
        },
      );

      console.log('[Login] Login successful');

      message.success(intl.formatMessage({id: 'pages.login.success'}));

      // 等待一小段时间确保 localStorage 写入完成，然后跳转
      setTimeout(() => {
        const urlParams = new URL(window.location.href).searchParams;
        const redirect = urlParams.get('redirect') || '/';
        console.log('[Login] Redirecting to:', redirect);
        window.location.href = redirect;
      }, 300);
    } catch (error: any) {
      console.error('[Login] Error occurred:', error);
      // 错误已在 model 中处理
      message.error(error?.message || intl.formatMessage({id: 'pages.login.failure'}));
    }
  };

  return (
    <div className="login-page-wrapper">
      <AuthLayout
        title="欢迎回来"
        description="请输入您的账户信息以登录系统"
        pageKey="login"
        footerLink={{
          text: '还没有账号？',
          linkText: '创建账号',
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
            placeholder: '请输入用户名或邮箱',
            autoComplete: 'username',
          }}
          rules={[
            {
              required: true,
              message: '请输入用户名',
            },
          ]}
        />
        <ProFormText.Password
          name="password"
          fieldProps={{
            size: 'large',
            placeholder: '请输入密码',
            autoComplete: 'current-password',
          }}
          rules={[
            {
              required: true,
              message: '请输入密码',
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
                color: isLightMode ? 'rgba(0, 0, 0, 0.7)' : 'rgba(255, 255, 255, 0.75)', /* WCAG AA: 对比度至少 4.5:1 */
                fontSize: 13,
              },
            }}
          >
            记住账号
          </ProFormCheckbox>
        </div>
        <div style={{marginTop: 36}}>
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
            {loginLoading ? '登录中...' : '登录'}
          </button>
        </div>
      </LoginForm>
    </AuthLayout>
    </div>
  );
};

export default Login;
