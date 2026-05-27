import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import i18next from 'i18next';

import { encryptPassword } from '@/utils';
import {
  type authenticationservicev1_LoginRequest,
  fetchUserProfile,
  loginMutation,
  logoutMutation,
  refreshTokenMutation,
  registerMutation,
} from '@/api';
import { startRefreshTimer, stopRefreshTimer, disconnectSSEServer } from '@/hooks/useTokenRefresh';

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
  /** 强制登出：纯前端清除认证状态 + 跳转登录页，不调后端接口（用于 token 已失效场景） */
  forceLogout: () => void;
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
          const response = await loginMutation.execute({
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

          // 4. 启动定时刷新 token
          startRefreshTimer();

          // 5. 执行成功回调或跳转
          if (onSuccess) {
            onSuccess();
          } else if (userInfo?.homePath) {
            window.location.href = userInfo.homePath;
          }
        } catch (err: any) {
          const errorMsg = err?.message || i18next.t('auth:loginFailed');
          set({ error: errorMsg });
          throw err;
        } finally {
          set({ loginLoading: false });
        }
      },

      // 注册
      register: async (params) => {
        set({ registerLoading: true, error: null });

        const password = encryptPassword(params.password);

        try {
          // 调用注册 API（API 内部已处理密码加密）
          await registerMutation.execute({
            username: params.username,
            password: password,
            tenantCode: '',
          });
        } catch (err: any) {
          const errorMsg = err?.message || i18next.t('auth:registerFailed');
          set({ error: errorMsg });
          throw err;
        } finally {
          set({ registerLoading: false });
        }
      },

      // 登出（主动，调后端接口）
      // 清除状态后由 React 组件响应状态变化自然重定向
      logout: async (_redirect = true) => {
        stopRefreshTimer();
        disconnectSSEServer();
        try {
          await logoutMutation.execute({}).catch(() => {}); // 忽略接口错误
        } finally {
          // 清除 localStorage 中的持久化数据
          localStorage.removeItem('auth-storage');
          localStorage.removeItem('user-storage');

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
        }
      },

      // 刷新 Token
      refreshToken: async () => {
        const { refreshTokenValue: refreshVal } = get();
        if (!refreshVal) {
          get().forceLogout();
          return '';
        }

        try {
          const response = await refreshTokenMutation.execute(refreshVal);

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
          get().forceLogout();
          return '';
        }
      },

      // 重认证（兜底）
      reauthenticate: () => {
        console.warn('Token invalid, please re-login');
        set({ error: i18next.t('auth:sessionExpired') });
      },

      // 强制登出：纯前端操作，不调后端接口
      // 用于 token 已失效（401）场景，避免调 logout API 又触发 401 死循环
      // 只清除状态，不做页面跳转（让 React 组件响应状态变化自然重定向）
      forceLogout: () => {
        stopRefreshTimer();
        disconnectSSEServer();
        console.warn('Force logout: clearing auth state');
        localStorage.removeItem('auth-storage');
        localStorage.removeItem('user-storage');
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
