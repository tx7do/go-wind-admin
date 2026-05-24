import { create } from 'zustand';
import { persist } from 'zustand/middleware';

import { encryptPassword } from '@/utils';
import {
  type authenticationservicev1_LoginRequest,
  fetchLogin,
  fetchLogout,
  fetchRefreshToken,
  fetchRegister,
  fetchUserProfile,
} from '@/api';

/**
 * 令牌载荷
 */
export interface TokenPayload {
  /**
   * 令牌值
   */
  value: string;
  /**
   * 令牌过期时间
   */
  expiresAt?: number;
}

export interface AuthState {
  // Token 状态（持久化）
  accessToken: string | null;
  refreshTokenValue: string | null;
  accessTokenExpireAt: number | null;
  refreshTokenExpireAt: number | null;

  // 用户状态（不持久化，避免脏数据）
  userInfo: UserInfo | null;

  // UI 状态
  loginLoading: boolean;
  registerLoading: boolean;
  error: string | null;

  // 动作
  login: (params: authenticationservicev1_LoginRequest, onSuccess?: () => void) => Promise<void>;
  register: (params: { username: string; password: string }) => Promise<void>;
  logout: (redirect?: boolean) => Promise<void>;
  refreshToken: () => Promise<string>;
  reauthenticate: () => void;
  setUserInfo: (info: UserInfo) => void;
  clearError: () => void;
  $reset: () => void;
}

// ========== 常量 ==========
const DEFAULT_ACCESS_EXPIRES_IN = 7200; // 2 小时
const DEFAULT_REFRESH_EXPIRES_IN = 2592000; // 30 天

// ========== Store 实现 ==========
export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      // 初始状态
      accessToken: null,
      refreshTokenValue: null,
      accessTokenExpireAt: null,
      refreshTokenExpireAt: null,
      userInfo: null,
      loginLoading: false,
      registerLoading: false,
      error: null,

      // 登录
      login: async (params, onSuccess) => {
        set({ loginLoading: true, error: null });

        try {
          // 1. 调用登录接口
          const response = await fetchLogin({
            ...params,
            password: encryptPassword(params.password || ''),
          });

          console.log('🔐 Login response:', {
            hasAccessToken: !!response.access_token,
            hasRefreshToken: !!response.refresh_token,
            expiresIn: response.expires_in,
          });

          const now = Date.now();

          // 2. 保存 Token
          const accessTokenPayload: TokenPayload = {
            value: response.access_token || '',
            expiresAt: now + (response.expires_in || DEFAULT_ACCESS_EXPIRES_IN) * 1000,
          };

          set({
            accessToken: accessTokenPayload.value,
            accessTokenExpireAt: accessTokenPayload.expiresAt,
          });

          console.log('💾 Access token saved:', {
            value: accessTokenPayload.value ? '***' + accessTokenPayload.value.slice(-8) : 'empty',
            expiresAt: accessTokenPayload.expiresAt
              ? new Date(accessTokenPayload.expiresAt).toISOString()
              : 'N/A',
          });

          if (response.refresh_token) {
            const refreshTokenPayload: TokenPayload = {
              value: response.refresh_token,
              expiresAt: now + (response.refresh_expires_in || DEFAULT_REFRESH_EXPIRES_IN) * 1000,
            };
            set({
              refreshTokenValue: refreshTokenPayload.value,
              refreshTokenExpireAt: refreshTokenPayload.expiresAt,
            });

            console.log('💾 Refresh token saved:', {
              value: refreshTokenPayload.value
                ? '***' + refreshTokenPayload.value.slice(-8)
                : 'empty',
              expiresAt: refreshTokenPayload.expiresAt
                ? new Date(refreshTokenPayload.expiresAt).toISOString()
                : 'N/A',
            });
          }

          // 3. 获取用户信息（交给 React Query 处理缓存，这里只更新 Zustand）
          console.log('👤 Fetching user info...');
          const userInfo = (await fetchUserProfile()) as unknown as UserInfo;
          set({ userInfo });
          console.log('✅ User info fetched:', userInfo);

          // 4. 执行成功回调或跳转
          if (onSuccess) {
            onSuccess();
          } else if (userInfo?.homePath) {
            window.location.href = userInfo.homePath;
          }
        } catch (err: any) {
          const errorMsg = err?.message || '登录失败';
          set({ error: errorMsg });
          throw err;
        } finally {
          set({ loginLoading: false });
        }
      },

      // 注册
      register: async (params) => {
        set({ registerLoading: true, error: null });

        try {
          // 调用注册 API（API 内部已处理密码加密）
          await fetchRegister(params.username, params.password);
        } catch (err: any) {
          const errorMsg = err?.message || '注册失败';
          set({ error: errorMsg });
          throw err;
        } finally {
          set({ registerLoading: false });
        }
      },

      // 登出
      logout: async (redirect = true) => {
        try {
          await fetchLogout().catch(() => {}); // 忽略接口错误
        } finally {
          // 清除 localStorage 中的持久化数据
          localStorage.removeItem('auth-storage');

          // 清除内存中的状态
          set({
            accessToken: null,
            refreshTokenValue: null,
            accessTokenExpireAt: null,
            refreshTokenExpireAt: null,
            userInfo: null,
            error: null,
            loginLoading: false,
            registerLoading: false,
          });

          // 跳转
          if (redirect && window.location.pathname !== '/login') {
            window.location.href = '/auth/login';
          }
        }
      },

      // 刷新 Token
      refreshToken: async () => {
        const { refreshTokenValue: refreshVal } = get();
        if (!refreshVal) {
          get().reauthenticate();
          return '';
        }

        try {
          const response = await fetchRefreshToken(refreshVal);

          const now = Date.now();
          set({
            accessToken: response.access_token,
            accessTokenExpireAt: now + (response.expires_in || DEFAULT_ACCESS_EXPIRES_IN) * 1000,
          });

          if (response.refresh_token) {
            set({
              refreshTokenValue: response.refresh_token,
              refreshTokenExpireAt:
                now + (response.refresh_expires_in || DEFAULT_REFRESH_EXPIRES_IN) * 1000,
            });
          }

          return response.access_token || '';
        } catch (err) {
          console.error('Refresh token failed:', err);
          get().reauthenticate();
          return '';
        }
      },

      // 重认证（兜底）
      reauthenticate: () => {
        console.warn('Token invalid, please re-login');
        set({ error: '认证已过期，请重新登录' });
        // 可选：自动跳转登录页
        // window.location.href = '/auth/login';
      },

      // 设置用户信息
      setUserInfo: (info) => set({ userInfo: info }),

      // 清除错误
      clearError: () => set({ error: null }),

      // 重置（用于测试/登出）
      $reset: () =>
        set({
          accessToken: null,
          refreshTokenValue: null,
          accessTokenExpireAt: null,
          refreshTokenExpireAt: null,
          userInfo: null,
          loginLoading: false,
          error: null,
        }),
    }),
    {
      name: 'auth-storage', // localStorage key
      partialize: (state) => {
        const persisted = {
          // ✅ 只持久化 Token 相关字段
          accessToken: state.accessToken,
          refreshTokenValue: state.refreshTokenValue,
          accessTokenExpireAt: state.accessTokenExpireAt,
          refreshTokenExpireAt: state.refreshTokenExpireAt,
          // ❌ userInfo/error 不持久化，避免脏数据
        };
        console.log('💿 Persisting auth state to localStorage:', {
          hasAccessToken: !!persisted.accessToken,
          hasRefreshToken: !!persisted.refreshTokenValue,
        });
        return persisted;
      },
    },
  ),
);
