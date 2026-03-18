import {LockOutlined, UserOutlined} from '@ant-design/icons';
import {LoginForm, ProFormText} from '@ant-design/pro-components';
import {Helmet, useIntl, useModel} from '@umijs/max';
import {App} from 'antd';
import React from 'react';

import Settings from '../../../../config/defaultSettings';

const Login: React.FC = () => {
  const {message} = App.useApp();
  const intl = useIntl();
  const {login, loginLoading} = useModel('business.authentication');
  const access = useModel('auth.access');

  const handleSubmit = async (values: { username: string; password: string }) => {
    console.log('[Login] Form submitted with values:', values);
      
    try {
      console.log('[Login] Calling login function...');
      const result = await login(
        {
          username: values.username,
          password: values.password,
          grant_type: 'password',
        },
        // 不传 onSuccess 回调，避免提前跳转
      );
        
      console.log('[Login] Login function returned result:', result);
  
      // 保存令牌到 AccessModel
      if (result.accessToken || result.refreshToken) {
        console.log('[Login] Saving tokens to AccessModel');
        access.setTokens({
          accessToken: result.accessToken ? {
            value: result.accessToken,
            expiresAt: Date.now() + 7200 * 1000, // 默认 2 小时过期
          } : null,
          refreshToken: result.refreshToken ? {
            value: result.refreshToken,
            expiresAt: Date.now() + 30 * 24 * 60 * 60 * 1000, // 默认 30 天过期
          } : null,
        });
        console.log('[Login] Tokens saved to AccessModel');
      } else {
        console.warn('[Login] No tokens in result');
      }
        
      message.success(intl.formatMessage({id: 'pages.login.success'}));
      
      // 保存完令牌后再跳转
      setTimeout(() => {
        window.location.href = '/';
      }, 500);
    } catch (error: any) {
      console.error('[Login] Error occurred:', error);
      // 错误已在 model 中处理
      message.error(error?.message || intl.formatMessage({id: 'pages.login.failure'}));
    }
  };

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        minHeight: '100vh',
        background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
      }}
    >
      <Helmet>
        <title>
          {intl.formatMessage({
            id: 'menu.login',
            defaultMessage: '登录页',
          })}
          {Settings.title && ` - ${Settings.title}`}
        </title>
      </Helmet>
      <div
        style={{
          flex: 1,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          padding: '32px',
        }}
      >
        <LoginForm
          loading={loginLoading}
          logo={<img alt="logo" src="/logo.svg" style={{height: 44}}/>}
          title="Go Wind Admin"
          subTitle="企业级后台管理系统"
          initialValues={{
            autoLogin: true,
          }}
          onFinish={handleSubmit}
          submitter={{
            searchConfig: {
              submitText: intl.formatMessage({
                id: 'pages.login.submit',
                defaultMessage: '登录',
              }),
            },
            submitButtonProps: {
              size: 'large',
            },
          }}
        >
          <ProFormText
            name="username"
            fieldProps={{
              size: 'large',
              prefix: <UserOutlined/>,
              placeholder: intl.formatMessage({
                id: 'authentication.usernameTip',
                defaultMessage: '请输入用户名',
              }),
            }}
            placeholder={intl.formatMessage({
              id: 'authentication.usernameTip',
              defaultMessage: '请输入用户名',
            })}
            rules={[
              {
                required: true,
                message: intl.formatMessage({
                  id: 'authentication.usernameTip',
                  defaultMessage: '请输入用户名',
                }),
              },
            ]}
          />
          <ProFormText.Password
            name="password"
            fieldProps={{
              size: 'large',
              prefix: <LockOutlined/>,
              placeholder: intl.formatMessage({
                id: 'authentication.password',
                defaultMessage: '请输入密码',
              }),
            }}
            placeholder={intl.formatMessage({
              id: 'authentication.password',
              defaultMessage: '请输入密码',
            })}
            rules={[
              {
                required: true,
                message: intl.formatMessage({
                  id: 'authentication.passwordTip',
                  defaultMessage: '请输入密码',
                }),
              },
            ]}
          />
        </LoginForm>
      </div>
    </div>
  );
};

export default Login;
