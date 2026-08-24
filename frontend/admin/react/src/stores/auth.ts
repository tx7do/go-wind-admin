import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import i18next from 'i18next';

import { encryptPassword } from '@/utils';
import {
  type authenticationservicev1_LoginRequest,
  type authenticationservicev1_LoginResponse,
  fetchUserProfile,
  loginMutation,
  logoutMutation,
  refreshTokenMutation,
  registerMutation,
  verifyMfaMutation,
} from '@/api';
import { startRefreshTimer, stopRefreshTimer, disconnectSSEServer } from '@/hooks/useTokenRefresh';
import { queryClient } from '@/core/query-client';
import { setCaptchaHeaders } from '@/core/transport/rest';

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
  // Token 状态（仅存内存，不落 localStorage）
  accessToken: string | null;
  accessTokenExpireAt: number | null;

  // 用户状态（不持久化，避免脏数据）
  userInfo: UserInfo | null;

  // MFA 挑战状态（不持久化）：登录密码校验通过但需二次验证时，后端返回 operation_id。
  // 非空表示当前处于 MFA 挑战待验证阶段，前端据此跳挑战页。
  mfaOperationId: string | null;

  // UI 状态
  loginLoading: boolean;
  registerLoading: boolean;
  error: string | null;

  // 动作
  login: (
    params: authenticationservicev1_LoginRequest,
    onSuccess?: () => void,
    captcha?: { id: string; value: string },
  ) => Promise<void>;
  /**
   * 完成 MFA 挑战：用 operation_id + TOTP 码调后端验证，通过则后端返回真 token，
   * 复用登录成功流程（存 token / 拉用户信息 / 跳首页）。失败抛错。
   */
  completeMfaChallenge: (
    totpCode: string,
    onSuccess?: () => void,
  ) => Promise<void>;
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

// ========== Store 实现 ==========
export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      // 初始状态
      accessToken: null,
      accessTokenExpireAt: null,
      userInfo: null,
      mfaOperationId: null,
      loginLoading: false,
      registerLoading: false,
      error: null,

      // applySuccessfulLogin 处理"已拿到含真 token 的 LoginResponse"后的统一流程：
      // 存 token → 拉用户信息 → 启动刷新定时器 → 跳转。
      // 登录成功与 MFA 验证成功都复用此函数。
      applySuccessfulLogin: async (
        response: authenticationservicev1_LoginResponse,
        now: number,
        onSuccess?: () => void,
      ) => {
        // 保存 Token
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

        // refresh token 通过 HttpOnly Cookie 传输，前端不可读也不存内存。
        // 页面刷新后由 bootstrap 静默恢复（凭 cookie 调 /refresh-token 换新 access token）。

        // 获取用户信息（交给 React Query 处理缓存，这里只更新 Zustand）
        console.log('👤 Fetching user info...');
        const userInfo = (await fetchUserProfile()) as unknown as UserInfo;
        set({ userInfo });
        console.log('✅ User info fetched:', userInfo);

        // 启动定时刷新 token
        startRefreshTimer();

        // 执行成功回调或跳转
        if (onSuccess) {
          onSuccess();
        } else if (userInfo?.homePath) {
          // 校验 homePath 必须为同源相对路径，防止服务端返回值导致开放重定向
          const rawHomePath = userInfo.homePath;
          if (
            typeof rawHomePath === 'string' &&
            rawHomePath.startsWith('/') &&
            !rawHomePath.startsWith('//')
          ) {
            window.location.href = rawHomePath;
          }
        }
      },

      // 登录
      login: async (params, onSuccess, captcha) => {
        set({ loginLoading: true, error: null });

        try {
          // 1. 调用登录接口（若有验证码，先设置一次性 Header，由 transport 消费）
          if (captcha) {
            setCaptchaHeaders(captcha.id, captcha.value);
          }
          const response = await loginMutation.execute({
            ...params,
            password: encryptPassword(params.password || ''),
          });

          console.log('🔐 Login response:', {
            hasAccessToken: !!response.access_token,
            expiresIn: response.expires_in,
          });

          // ===== MFA 闸门：后端在密码校验通过、需二次验证时返回 mfa_operation_id，=====
          // ===== access_token 为空。此时不写任何 token，跳转 MFA 挑战页。
          if (response.mfa_operation_id) {
            set({ mfaOperationId: response.mfa_operation_id });
            // 跳 MFA 挑战页（路由守卫亦会据此强制跳转）
            if (onSuccess) {
              onSuccess();
            }
            return;
          }

          const now = Date.now();
          await applySuccessfulLogin(response, now, onSuccess);
        } catch (err: any) {
          const errorMsg = err?.message || i18next.t('auth:loginFailed');
          // 登录失败时清除任何可能被部分写入的认证状态（含空 token / 未来 expiry / MFA 状态），
          // 避免脏数据残留导致后续鉴权误判。
          set({
            error: errorMsg,
            accessToken: null,
            accessTokenExpireAt: null,
            mfaOperationId: null,
          });
          throw err;
        } finally {
          set({ loginLoading: false });
        }
      },

      // 完成 MFA 挑战
      completeMfaChallenge: async (totpCode, onSuccess) => {
        set({ loginLoading: true, error: null });
        try {
          const opId = get().mfaOperationId;
          if (!opId) {
            throw new Error(i18next.t('auth:mfaNotRequired'));
          }
          // 调 MFA 验证接口；后端校验通过后返回含真 token 的 LoginResponse。
          const response = (await verifyMfaMutation.execute({
            operationId: opId,
            totpCode: totpCode,
          })) as authenticationservicev1_LoginResponse;

          // MFA 验证成功后清掉挑战态，复用登录成功流程存 token / 拉用户信息 / 跳转。
          set({ mfaOperationId: null });
          await applySuccessfulLogin(response, Date.now(), onSuccess);
        } catch (err: any) {
          const errorMsg = err?.message || i18next.t('auth:mfaVerifyFailed');
          set({ error: errorMsg, mfaOperationId: null });
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
          // 清除 queryClient 缓存，防止登出期间被缓存污染的查询结果
          // （如 getMe 因 401 返回 null 被 fetchQuery 缓存）导致重新登录时命中脏数据
          queryClient.clear();

          // 清除 localStorage 中的持久化数据
          localStorage.removeItem('auth-storage');
          localStorage.removeItem('user-storage');

          // 清除内存中的状态
          set({
            accessToken: null,
            accessTokenExpireAt: null,
            userInfo: null,
            error: null,
            loginLoading: false,
            registerLoading: false,
          });
        }
      },

      // 刷新 Token
      // refresh token 以 HttpOnly Cookie 传输，刷新请求由浏览器自动携带 cookie，
      // 前端无需（也无法）读取 refresh token 值。
      refreshToken: async () => {
        try {
          const response = await refreshTokenMutation.execute();

          const now = Date.now();
          set({
            accessToken: response.access_token,
            accessTokenExpireAt: now + (response.expires_in || DEFAULT_ACCESS_EXPIRES_IN) * 1000,
          });

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
        // 清除 queryClient 缓存，防止缓存污染导致重新登录失败
        queryClient.clear();
        localStorage.removeItem('auth-storage');
        localStorage.removeItem('user-storage');
        set({
          accessToken: null,
          accessTokenExpireAt: null,
          userInfo: null,
          mfaOperationId: null,
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
          accessTokenExpireAt: null,
          userInfo: null,
          mfaOperationId: null,
          loginLoading: false,
          error: null,
        }),
    }),
    {
      name: 'auth-storage', // localStorage key
      // Token（access/refresh 及其过期时间）全部仅存内存，不落 localStorage。
      // 刷新页面后 token 丢失 → AuthGuard 看到 accessToken 为 null → 跳登录页。
      // 登出时仍 localStorage.removeItem('auth-storage') 清理历史残留数据。
      partialize: () => ({}),
    },
  ),
);
