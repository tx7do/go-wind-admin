import {GlobalOutlined, MoonOutlined, SunOutlined} from '@ant-design/icons';
import {LoginForm, ProFormCheckbox, ProFormText} from '@ant-design/pro-components';
import {Helmet, useIntl, useModel} from '@umijs/max';
import {App, Button, Tooltip} from 'antd';
import React from 'react';

import Settings from '../../../../config/defaultSettings';
import SloganIcon from './icons/SloganIcon';

const Login: React.FC = () => {
  const {message} = App.useApp();
  const intl = useIntl();
  const {login, loginLoading} = useModel('business.authentication');
  const access = useModel('auth.access');
  const {mode: themeMode, setMode: setThemeMode} = useModel('core.theme');
  const {locale: currentLocale, setLocale: setCurrentLocale} = useModel('core.language');

  // 切换主题
  const toggleTheme = () => {
    const newMode = themeMode === 'light' ? 'dark' : 'light';
    setThemeMode(newMode);
  };

  // 根据主题模式判断当前是否为亮色模式
  const isLightMode = React.useMemo(() => {
    if (themeMode === 'system') {
      return window.matchMedia('(prefers-color-scheme: light)').matches;
    }
    return themeMode === 'light';
  }, [themeMode]);

  // 监听系统主题变化
  React.useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handleChange = () => {
      if (themeMode === 'system') {
        // 强制重新渲染以更新 isLightMode
        setThemeMode('system');
      }
    };
    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, [themeMode]);

  // 切换语言
  const toggleLanguage = () => {
    setCurrentLocale(currentLocale === 'zh-CN' ? 'en-US' : 'zh-CN');
  };

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
    <div
      className="login-page-wrapper"
      style={{
        display: 'flex',
        minHeight: '100vh',
        background: isLightMode ? '#f5f7fa' : 'var(--login-page-bg, #0a0a0a)',
        overflow: 'hidden',
        position: 'relative',
      }}
    >
      {/* 右上角工具栏 */}
      <div
        style={{
          position: 'absolute',
          top: 20,
          right: 20,
          display: 'flex',
          gap: 8,
          zIndex: 10,
        }}
      >
        <Tooltip title={currentLocale === 'zh-CN' ? '切换语言' : 'Switch Language'}>
          <Button
            type="text"
            icon={<GlobalOutlined/>}
            onClick={toggleLanguage}
            style={{
              color: isLightMode ? 'rgba(0, 0, 0, 0.65)' : '#fff',
              background: isLightMode ? 'rgba(0, 0, 0, 0.04)' : 'rgba(255, 255, 255, 0.08)',
              border: isLightMode ? '1px solid rgba(0, 0, 0, 0.12)' : '1px solid rgba(255, 255, 255, 0.12)',
              borderRadius: 6,
            }}
          >
            {currentLocale === 'zh-CN' ? 'EN' : '中文'}
          </Button>
        </Tooltip>
        <Tooltip title={themeMode === 'light' ? '切换暗黑模式' : '切换亮色模式'}>
          <Button
            type="text"
            icon={themeMode === 'light' ? <MoonOutlined/> : <SunOutlined/>}
            onClick={toggleTheme}
            style={{
              color: isLightMode ? 'rgba(0, 0, 0, 0.65)' : '#fff',
              background: isLightMode ? 'rgba(0, 0, 0, 0.04)' : 'rgba(255, 255, 255, 0.08)',
              border: isLightMode ? '1px solid rgba(0, 0, 0, 0.12)' : '1px solid rgba(255, 255, 255, 0.12)',
              borderRadius: 6,
            }}
          />
        </Tooltip>
      </div>
      <Helmet>
        <title>
          {intl.formatMessage({
            id: 'menu.login',
            defaultMessage: '登录页',
          })}
          {Settings.title && ` - ${Settings.title}`}
        </title>
      </Helmet>

      {/* 左侧品牌展示区 */}
      <div
        style={{
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          background: isLightMode
            ? 'linear-gradient(135deg, #667eea 0%, #764ba2 50%, #f093fb 100%)'
            : 'linear-gradient(135deg, #0f2027 0%, #203a43 50%, #2c5364 100%)',
          position: 'relative',
          overflow: 'hidden',
          minWidth: 0,
        }}
      >
        {/* 背景装饰 - 多层渐变 */}
        <div
          style={{
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            background: isLightMode
              ? 'linear-gradient(180deg, rgba(255,255,255,0.1) 0%, transparent 50%, rgba(0,0,0,0.1) 100%)'
              : 'linear-gradient(180deg, rgba(0,0,0,0.2) 0%, transparent 50%, rgba(0,0,0,0.3) 100%)',
            pointerEvents: 'none',
          }}
        />
        
        {/* 装饰圆形 */}
        <div
          style={{
            position: 'absolute',
            top: '10%',
            right: '15%',
            width: 300,
            height: 300,
            borderRadius: '50%',
            background: isLightMode
              ? 'radial-gradient(circle, rgba(255,255,255,0.15) 0%, transparent 70%)'
              : 'radial-gradient(circle, rgba(59,130,246,0.15) 0%, transparent 70%)',
            pointerEvents: 'none',
          }}
        />
        
        <div
          style={{
            position: 'absolute',
            bottom: '15%',
            left: '10%',
            width: 200,
            height: 200,
            borderRadius: '50%',
            background: isLightMode
              ? 'radial-gradient(circle, rgba(255,255,255,0.1) 0%, transparent 70%)'
              : 'radial-gradient(circle, rgba(147,51,234,0.1) 0%, transparent 70%)',
            pointerEvents: 'none',
          }}
        />

        {/* 品牌图标 */}
        <div
          style={{
            width: 300,
            height: 300,
            marginBottom: 36,
            position: 'relative',
            zIndex: 1,
          }}
        >
          <SloganIcon/>
        </div>

        <h2
          style={{
            color: '#fff',
            fontSize: 26,
            fontWeight: 700,
            marginBottom: 14,
            textAlign: 'center',
            textShadow: '0 2px 20px rgba(0, 0, 0, 0.3)',
            letterSpacing: '0.5px',
            position: 'relative',
            zIndex: 1,
          }}
        >
          风行中后台管理系统
        </h2>
        <p
          style={{
            color: 'rgba(255, 255, 255, 0.95)',
            fontSize: 15,
            textAlign: 'center',
            textShadow: '0 1px 8px rgba(0, 0, 0, 0.2)',
            lineHeight: 1.6,
            position: 'relative',
            zIndex: 1,
          }}
        >
          开箱即用的企业级中后台管理系统
        </p>
      </div>

      {/* 右侧登录表单区 */}
      <div
        style={{
          width: '48%',
          minWidth: '500px',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'center',
          alignItems: 'center',
          padding: '64px 56px',
          background: isLightMode ? '#ffffff' : '#0d1117',
          borderLeft: isLightMode
            ? 'none'
            : '1px solid rgba(255, 255, 255, 0.06)',
          boxShadow: isLightMode ? '-20px 0 60px rgba(102, 126, 234, 0.15)' : 'none',
          position: 'relative',
        }}
      >
        <div style={{width: '100%', maxWidth: '420px'}}>
          <h1
            style={{
              color: isLightMode ? '#1a1a2e' : '#fff',
              fontSize: 34,
              fontWeight: 800,
              marginBottom: 10,
              letterSpacing: '-0.5px',
              background: isLightMode
                ? 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)'
                : 'none',
              WebkitBackgroundClip: isLightMode ? 'text' : 'unset',
              WebkitTextFillColor: isLightMode ? 'transparent' : 'unset',
            }}
          >
            欢迎回来
          </h1>
          <p
            style={{
              color: isLightMode ? 'rgba(0, 0, 0, 0.5)' : 'rgba(255, 255, 255, 0.6)',
              fontSize: 15,
              marginBottom: 44,
              paddingLeft: 2,
              lineHeight: 1.6,
            }}
          >
            请输入您的账户信息以登录系统
          </p>

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
                style: {
                  height: 48,
                  background: isLightMode ? 'rgba(102, 126, 234, 0.06)' : 'rgba(255, 255, 255, 0.08)',
                  border: isLightMode ? '1.5px solid rgba(102, 126, 234, 0.3)' : '1px solid rgba(255, 255, 255, 0.18)',
                  borderRadius: 8,
                  color: isLightMode ? '#1a1a2e' : '#fff',
                },
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
                style: {
                  height: 48,
                  background: isLightMode ? 'rgba(102, 126, 234, 0.06)' : 'rgba(255, 255, 255, 0.08)',
                  border: isLightMode ? '1.5px solid rgba(102, 126, 234, 0.3)' : '1px solid rgba(255, 255, 255, 0.18)',
                  borderRadius: 8,
                  color: isLightMode ? '#1a1a2e' : '#fff',
                },
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
                    color: isLightMode ? 'rgba(0, 0, 0, 0.65)' : 'rgba(255, 255, 255, 0.65)',
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
                  height: 50,
                  background: isLightMode
                    ? 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)'
                    : 'linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%)',
                  border: 'none',
                  borderRadius: 10,
                  color: '#fff',
                  fontSize: 16,
                  fontWeight: 600,
                  cursor: loginLoading ? 'not-allowed' : 'pointer',
                  opacity: loginLoading ? 0.7 : 1,
                  transition: 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
                  boxShadow: isLightMode
                    ? '0 8px 24px rgba(102, 126, 234, 0.35)'
                    : '0 8px 24px rgba(59, 130, 246, 0.3)',
                  letterSpacing: '2px',
                  transform: 'translateY(0)',
                }}
                disabled={loginLoading}
              >
                {loginLoading ? '登录中...' : '登 录'}
              </button>
            </div>
          </LoginForm>

          <div
            style={{
              textAlign: 'center',
              marginTop: 16,
            }}
          >
            <span style={{color: isLightMode ? 'rgba(0, 0, 0, 0.65)' : 'rgba(255, 255, 255, 0.65)', fontSize: 13}}>
              还没有账号？{' '}
            </span>
            <a
              href="/auth/register"
              style={{
                color: '#0066ff',
                fontSize: 13,
                textDecoration: 'none',
              }}
            >
              创建账号
            </a>
          </div>
        </div>

        {/* 底部版权信息 - 在右侧面板内 */}
        <div
          style={{
            position: 'absolute',
            bottom: 20,
            left: 0,
            right: 0,
            textAlign: 'center',
            color: isLightMode ? 'rgba(0, 0, 0, 0.45)' : 'rgba(255, 255, 255, 0.45)',
            fontSize: 12,
          }}
        >
          Copyright © {new Date().getFullYear()} GoWind
        </div>
      </div>
    </div>
  );
};

export default Login;
