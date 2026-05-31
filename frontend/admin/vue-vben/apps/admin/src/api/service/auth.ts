import {
  type authenticationservicev1_LoginRequest,
  type authenticationservicev1_RegisterUserRequest,
  createAuthenticationServiceClient,
} from '#/api/generated/admin/service/v1';
import { requestApi } from '#/transport/rest';

let _instance: null | ReturnType<typeof createAuthenticationServiceClient> =
  null;

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

export async function registerUser(
  request: authenticationservicev1_RegisterUserRequest,
) {
  return await getAuthService().RegisterUser(request);
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
