import React, { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Form, Input, Button, Checkbox, App } from 'antd';
import { UserOutlined, LockOutlined, SafetyOutlined } from '@ant-design/icons';
import { useAuthStore } from '@go-wind/admin-react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { fetchGenerateCaptcha } from '@/api';
import { ssoEnabled, startSsoLogin } from '@/api/hooks/sso';
import { fetchTenantBranding } from '@/api/hooks/tenant-branding';

/**
 * 验证码控件：输入框与图片盒同排，组成单一受控组件挂进 Form.Item。
 * value/onChange 由 Form.Item 注入后透传给 Input；校验错误渲染在整行下方，
 * 验证码图不会被错误行挤偏（旧结构嵌套 Form.Item + items-center 会在报错时下坠）。
 */
const CaptchaField: React.FC<{
  value?: string;
  onChange?: (...event: any[]) => void;
  label?: string;
  placeholder?: string;
  imageEl?: React.ReactNode;
}> = ({ value, onChange, label, placeholder, imageEl }) => (
  <div className="flex items-center gap-2">
    <Input
      prefix={<SafetyOutlined />}
      aria-label={label}
      placeholder={placeholder}
      autoComplete="off"
      value={value}
      onChange={onChange}
    />
    {imageEl}
  </div>
);

const Login: React.FC = () => {
  const { t } = useTranslation('auth');
  const { login, loginLoading } = useAuthStore();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { message } = App.useApp();

  // 验证码状态
  const [captchaId, setCaptchaId] = useState<string>('');
  const [captchaImage, setCaptchaImage] = useState<string>('');
  const [captchaLoading, setCaptchaLoading] = useState(false);

  // SSO 开关（后端未配置 OIDC 时按钮不渲染）
  const [ssoOn, setSsoOn] = useState(false);

  // 租户白标：租户编号失焦后拉取（防抖交给 blur 语义），命中则替换站牌
  const [tenantBranding, setTenantBranding] = useState<{ found: boolean; name: string; logoUrl: string } | null>(null);
  const applyTenantBranding = useCallback(async (code: string) => {
    const trimmed = (code || '').trim();
    if (!trimmed) {
      setTenantBranding(null);
      return;
    }
    try {
      const branding = await fetchTenantBranding(trimmed);
      setTenantBranding(branding.found ? branding : null);
    } catch ( brandingError ) {
      // 白标是非关键路径：失败静默保持默认品牌，原始错误留控制台
      console.warn('fetch tenant branding failed', brandingError);
      setTenantBranding(null);
    }
  }, []);

  // 获取验证码
  const refreshCaptcha = useCallback(async () => {
    setCaptchaLoading(true);
    try {
      const resp = await fetchGenerateCaptcha();
      setCaptchaId(resp.captchaId ?? '');
      setCaptchaImage(resp.imageBase64 ?? '');
    } catch {
      // 验证码获取失败不阻断页面，登录时会再次校验
    } finally {
      setCaptchaLoading(false);
    }
  }, []);

  useEffect(() => {
    // React 18 StrictMode 下 effect 会执行两次，导致两个并发的
    // fetchGenerateCaptcha 请求；后到的响应会覆盖先到的 captchaId，
    // 而后端通常一次性消费 captcha，登录时用的 captchaId 可能对应已被
    // 先到请求作废的 captcha。这里用 cancelled 标志位丢弃首次（被 double-invoke
    // 的第一次）请求的结果，确保 state 始终是最后一次请求的值。
    let cancelled = false;
    const run = async () => {
      setCaptchaLoading(true);
      try {
        const resp = await fetchGenerateCaptcha();
        if (cancelled) return;
        setCaptchaId(resp.captchaId ?? '');
        setCaptchaImage(resp.imageBase64 ?? '');
      } catch {
        // 验证码获取失败不阻断页面，登录时会再次校验
      } finally {
        if (!cancelled) setCaptchaLoading(false);
      }
    };
    run();
    return () => {
      cancelled = true;
    };
  }, []);

  // SSO 开关探测：失败/未配置都不显示按钮（非关键路径，静默降级）
  useEffect(() => {
    ssoEnabled().then(setSsoOn).catch(() => setSsoOn(false));
  }, []);

  const handleSubmit = async (values: {
    username: string;
    password: string;
    tenant_code?: string;
    remember?: boolean;
    captcha?: string;
  }) => {
    try {
      // 跳转目标：redirect 参数 → 用户 homePath → 首页。
      // 同源相对路径校验保留，防开放重定向（如 ?redirect=https://evil.com 或 //evil.com）
      const resolveSafeRedirect = () => {
        const rawRedirect =
          searchParams.get('redirect') || useAuthStore.getState().userInfo?.homePath || '/';
        return typeof rawRedirect === 'string' &&
          rawRedirect.startsWith('/') &&
          !rawRedirect.startsWith('//')
          ? rawRedirect
          : '/';
      };

      await login(
        {
          username: values.username,
          password: values.password,
          tenant_code: values.tenant_code,
          grant_type: 'password',
        },
        // 传 onSuccess 后 store 跳过 window.location.href 整页跳转。必须在此同步
        // 导航、赶在 router 重建读取地址之前：isAuthenticated 翻转会让 AppRouter
        // effect 重建 createBrowserRouter（以当时 window.location 为初始地址），
        // 若像旧实现那样 setTimeout 后用旧 router 实例 navigate，只剩一次无感知的
        // pushState——地址栏落在 redirect、页面却渲染 homePath，两边劈叉。
        () => navigate(resolveSafeRedirect()),
        { id: captchaId, value: values.captcha ?? '' },
      );

      message.success(t('loginSuccess'));
    } catch (error: any) {
      // 登录失败后刷新验证码
      refreshCaptcha();
      // 弹出错误提示（与 CRUD 页面统一模式：优先用后端 message，兜底走 i18n）
      message.error(error?.message || t('loginFailed'));
    }
  };

  /**
   * 验证码图片盒：与输入框等高（h-11=44px），宽 132px 与后端 240×80（3:1）等比，
   * img object-cover 满高填充无留白（对齐 vue-element 观感）。点击刷新验证码。
   */
  const captchaImageEl = (
    <div
      className="flex items-center justify-center overflow-hidden w-[132px] h-11 shrink-0 rounded-lg cursor-pointer border border-solid bg-white light:border-black/10 light:bg-black/[0.03]"
      title={t('captchaRefresh')}
      onClick={() => !captchaLoading && refreshCaptcha()}
    >
      {captchaImage ? (
        <img
          src={captchaImage}
          alt="captcha"
          className="h-full w-full object-cover"
        />
      ) : (
        // 占位文案走 §2.1 的实底暗墨而非次要文字档：外层盒子两态都是浅底
        // （暗色 bg-white / 浅色 bg-black/[0.03]），实测次要档在浅底上只有 2.6:1。
        <span className="text-[color:var(--gowind-solid-ink)] text-xs">
          {captchaLoading ? '...' : t('captchaRefresh')}
        </span>
      )}
    </div>
  );

  return (
    <div className="w-full max-w-[420px]">
      {/* 标题（租户白标命中时替换为租户 Logo/名称） */}
      <div className="mb-11">
        {tenantBranding ? (
          <>
            {tenantBranding.logoUrl && (
              <img
                src={tenantBranding.logoUrl}
                alt={tenantBranding.name}
                className="h-12 mb-3 object-contain"
              />
            )}
            <h2 className="text-[34px] font-extrabold tracking-[-0.5px] mb-2.5 text-[color:var(--ant-color-text)]">
              {tenantBranding.name}
            </h2>
            <p className="text-[15px] leading-relaxed text-[color:var(--gowind-secondary-text)]">
              {t('welcomeBack')}
            </p>
          </>
        ) : (
          <>
            <h2 className="text-[34px] font-extrabold tracking-[-0.5px] mb-2.5 text-[color:var(--ant-color-text)]">
              {t('welcomeBack')}
            </h2>
            <p className="text-[15px] leading-relaxed text-[color:var(--gowind-secondary-text)]">
              {t('loginDescription')}
            </p>
          </>
        )}
      </div>

      {/* 登录表单卡片 —— 实底表面色 + 24px 大圆角 + 主色柔影（对齐 vben 认证面板） */}
      <div className="rounded-3xl border border-[color:var(--ant-color-border-secondary)] bg-[color:var(--ant-color-bg-container)] p-8 shadow-[0_12px_40px_-8px_rgba(0,107,230,0.18)]">
        <Form
          name="login"
          onFinish={handleSubmit}
          size="large"
          initialValues={{ remember: true }}
          className="login-form"
        >
          <Form.Item name="tenant_code" className="login-form-item">
            <Input
              prefix={<UserOutlined />}
              aria-label={t('tenantCodeLabel')}
              placeholder={t('tenantCodePlaceholder')}
              autoComplete="off"
              onBlur={(e: React.FocusEvent<HTMLInputElement>) => {
                applyTenantBranding(e.target.value);
              }}
            />
          </Form.Item>

          <Form.Item
            name="username"
            className="login-form-item"
            rules={[
              {
                required: true,
                message: t('usernameRequired'),
              },
            ]}
          >
            <Input
              prefix={<UserOutlined />}
              aria-label={t('usernameLabel')}
              placeholder={t('usernamePlaceholder')}
              autoComplete="username"
            />
          </Form.Item>

          <Form.Item
            name="password"
            className="login-form-item"
            rules={[
              {
                required: true,
                message: t('passwordRequired'),
              },
            ]}
          >
            <Input.Password
              prefix={<LockOutlined />}
              aria-label={t('passwordLabel')}
              placeholder={t('passwordPlaceholder')}
              autoComplete="current-password"
            />
          </Form.Item>

          {/* 验证码 —— 输入框与图片盒组成单一受控组件（CaptchaField），
              校验错误渲染在整行下方，图片盒与输入框始终保持等高对齐 */}
          <Form.Item
            name="captcha"
            rules={[
              {
                required: true,
                message: t('captchaRequired'),
              },
            ]}
            className="login-form-item"
          >
            <CaptchaField
              label={t('captchaLabel')}
              placeholder={t('captchaPlaceholder')}
              imageEl={captchaImageEl}
            />
          </Form.Item>

          <Form.Item className="login-remember-item">
            <Form.Item name="remember" valuePropName="checked" noStyle>
              <Checkbox>{t('rememberAccount')}</Checkbox>
            </Form.Item>
          </Form.Item>

          <Form.Item className="login-form-item">
            <Button
              type="primary"
              htmlType="submit"
              loading={loginLoading}
              block
              className="login-submit-btn"
            >
              {loginLoading ? t('loggingIn') : t('loginButton')}
            </Button>
          </Form.Item>
        </Form>

        {ssoOn && (
          <>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12, margin: '16px 0' }}>
              <div style={{ flex: 1, height: 1, background: 'var(--ant-color-split)' }} />
              <span style={{ fontSize: 12, color: 'var(--ant-color-text-secondary)' }}>或</span>
              <div style={{ flex: 1, height: 1, background: 'var(--ant-color-split)' }} />
            </div>
            <Button
              block
              size="large"
              onClick={() => {
                startSsoLogin().catch((err: any) => {
                  console.error('start sso login failed', err);
                  message.error(err?.message || t('ssoLoginFailed'));
                });
              }}
            >
              {t('ssoButton')}
            </Button>
          </>
        )}
      </div>

      {/* 底部链接 */}
      <div className="mt-6 text-center text-[13px]">
        <span className="text-[color:var(--ant-color-text-secondary)]">
          {t('noAccount')}{' '}
        </span>
        <a
          href="/auth/forgot-password"
          className="gowind-auth-link hover:opacity-80"
          style={{ fontSize: 13 }}
        >
          {t('forgotPassword')}
        </a>
        
      </div>
    </div>
  );
};

export default Login;
