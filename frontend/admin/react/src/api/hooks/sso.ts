import { useAuthStore } from '@go-wind/admin-react';
import { apiClient } from '@/api/client';

/**
 * OIDC SSO 登录（三步）：
 *  1. ssoEnabled() — 登录页判断是否显示「企业账号登录」按钮；
 *  2. startSsoLogin() — 取授权跳转 URL 并整页前往 IdP（state 由后端生成并存 Redis）；
 *  3. completeSsoLogin() — IdP 回调带回 code+state，换本系统 JWT 并走统一
 *     登录成功流程（applySuccessfulLogin：存 token/拉用户/跳转）。
 * refresh token 由后端经 HttpOnly Cookie 下发，与密码登录同形，前端无需处理。
 */
export async function ssoEnabled(): Promise<boolean> {
  try {
    const resp = await apiClient.authenticationService.GetSsoLoginInfo({});
    return !!resp.enabled;
  } catch {
    return false;
  }
}

export async function startSsoLogin(): Promise<void> {
  const resp = await apiClient.authenticationService.GetSsoLoginUrl({});
  if (!resp.authorizationUrl) {
    throw new Error('sso authorization url is empty');
  }
  // 整页跳转（IdP 在外部域，不能用 SPA 内部导航）
  window.location.assign(resp.authorizationUrl);
}

/** 回调页调用：code+state 换令牌并完成登录态初始化（走 auth store action，
 *  复用 applySuccessfulLogin 的存 token/拉用户/跳转流程）；onSuccess 里跳转首页 */
export async function completeSsoLogin(code: string, state: string, onSuccess?: () => void): Promise<void> {
  await useAuthStore.getState().completeSsoLogin(code, state, onSuccess);
}
