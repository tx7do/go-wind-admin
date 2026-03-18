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

  const handleSubmit = async (values: { username: string; password: string }) => {
    try {
      await login(
        {
          username: values.username,
          password: values.password,
          grant_type: 'password',
        },
        () => {
          // 登录成功后的回调
          window.location.href = '/';
        },
      );
      message.success(intl.formatMessage({id: 'pages.login.success'}));
    } catch (error: any) {
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
