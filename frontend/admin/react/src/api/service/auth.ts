import {
  type authenticationservicev1_LoginRequest,
  createAuthenticationServiceClient,
} from '@/api/generated/admin/service/v1';
import { requestApi } from '@/core';
import { encryptPassword } from '@/utils';

let _instance: ReturnType<typeof createAuthenticationServiceClient> | null = null;

/**
 * 获取认证服务单例（延迟初始化）
 */
export function getAuthService() {
  if (!_instance) {
    _instance = createAuthenticationServiceClient(requestApi);
  }
  return _instance;
}

export async function login(request: authenticationservicev1_LoginRequest) {
  return getAuthService().Login(request);
}

export async function logout() {
  return getAuthService().Logout({});
}

export async function register(username: string, password: string, tenantCode: string = '') {
  return await getAuthService().RegisterUser({
    username,
    password: encryptPassword(password),
    tenantCode: tenantCode,
  });
}

export async function generateCaptcha() {
  return await getAuthService().GenerateCaptcha({});
}

export async function refreshToken(refreshToken: string) {
  return getAuthService().RefreshToken({
    grant_type: 'refresh_token',
    refresh_token: refreshToken ?? '',
  });
}
