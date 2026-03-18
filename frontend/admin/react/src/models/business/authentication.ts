import {useState, useCallback} from 'react';
import {
  createAuthenticationServiceClient,
  createUserProfileServiceClient,
  authenticationservicev1_GrantType,
} from '@/api/admin/service/v1';
import {requestApi} from '@/transport/rest';
import {encryptByAES} from '@/utils/crypto';
import type {IUser} from '../types';

export interface LoginParams {
  username?: string;
  email?: string;
  mobile?: string;
  password: string;
  grant_type?: string;
}

const DEFAULT_HOME_PATH = '/';

/**
 * 认证管理 Model
 * 处理用户登录、登出、刷新令牌等认证相关操作
 */
export default function AuthenticationModel() {
  const [loginLoading, setLoginLoading] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  // 创建服务客户端
  const authService = createAuthenticationServiceClient(requestApi);
  const userProfileService = createUserProfileServiceClient(requestApi);

  /**
   * 加密密码
   */
  function encryptPassword(password: string): string {
    const key = process.env.NEXT_PUBLIC_AES_KEY || '';
    return encryptByAES(password, key);
  }

  /**
   * 获取用户信息
   */
  const fetchUserInfo = useCallback(async (): Promise<IUser> => {
    return (await userProfileService.GetUser({})) as unknown as IUser;
  }, []);

  /**
   * 用户登录
   */
  const login = useCallback(
    async (
      params: LoginParams,
      onSuccess?: () => Promise<void> | void,
    ): Promise<{userInfo: IUser | null}> => {
      let userInfo: IUser | null = null;

      try {
        setLoginLoading(true);
        setError(null);

        // 调用登录 API
        const response = await authService.Login({
          username: params.username,
          email: params.email,
          mobile: params.mobile,
          password: encryptPassword(params.password),
          grant_type: (params.grant_type || 'password') as authenticationservicev1_GrantType,
        });

        if (response.access_token) {
          // 注意：这里需要与 access model 配合使用
          // 在实际使用时，应该通过 useModel('auth.access') 来设置 token
          console.log('Login successful, access_token:', response.access_token);

          // 获取用户信息
          userInfo = await fetchUserInfo();

          // 执行成功回调
          if (onSuccess) {
            await onSuccess();
          } else if (userInfo?.homePage) {
            // 跳转到用户主页
            window.location.href = userInfo.homePage;
          }
        }
      } catch (err: any) {
        setError(err?.message || '登录失败');
        throw err;
      } finally {
        setLoginLoading(false);
      }

      return {userInfo};
    },
    [fetchUserInfo],
  );

  /**
   * 用户登出
   */
  const logout = useCallback(
    async (redirect: boolean = true): Promise<void> => {
      try {
        setLoading(true);
        await authService.Logout({});
      } catch {
        // 不做任何处理
      } finally {
        setLoading(false);

        // 清除 tokens（需要与 access model 配合）
        console.log('Logout successful, should clear tokens');

        // 跳转回登录页
        if (redirect) {
          window.location.href = '/user/login';
        }
      }
    },
    [],
  );

  /**
   * 刷新访问令牌
   */
  const refreshToken = useCallback(
    async (refreshTokenValue: string): Promise<string> => {
      try {
        setLoading(true);
        const resp = await authService.RefreshToken({
          grant_type: 'password',
          refresh_token: refreshTokenValue,
        });

        const newToken = resp.access_token || '';
        console.log('Token refreshed:', newToken);

        return newToken;
      } catch (err) {
        setError('刷新令牌失败');
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [],
  );

  /**
   * 重新认证（当 token 失效时）
   */
  const reauthenticate = useCallback(async (): Promise<void> => {
    console.warn('Access token or refresh token is invalid or expired.');
    setError('认证已过期，请重新登录');
  }, []);

  /**
   * 清除错误
   */
  const clearError = useCallback(() => {
    setError(null);
  }, []);

  /**
   * 重置认证状态
   */
  const resetAuth = useCallback(() => {
    setLoginLoading(false);
    setLoading(false);
    setError(null);
  }, []);

  return {
    // 状态
    loginLoading,
    loading,
    error,

    // 方法
    login,
    logout,
    refreshToken,
    reauthenticate,
    fetchUserInfo,
    clearError,
    resetAuth,
  };
}
