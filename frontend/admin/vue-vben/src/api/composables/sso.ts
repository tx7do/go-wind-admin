import { apiClient } from '#/api/client';

/**
 * OIDC SSO 登录（react 基准的移植）：
 *  - ssoEnabled() — 登录页判断是否显示「企业账号登录」按钮；
 *  - startSsoLogin() — 取授权跳转 URL 并整页前往 IdP（state 由后端生成并存
 *    Redis，10 分钟单次有效）。
 * 回调落地走 authentication store 的 completeSsoLogin（见 sso-callback.vue）。
 * refresh token 由后端经 HttpOnly Cookie 下发，与密码登录同形。
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
