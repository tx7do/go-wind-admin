import type { Recordable, UserInfo } from '@vben/types';

import { ref } from 'vue';
import { useRouter } from 'vue-router';

import { DEFAULT_HOME_PATH, LOGIN_PATH } from '@vben/constants';
import { preferences } from '@vben/preferences';
import { resetAllStores, useAccessStore, useUserStore } from '@vben/stores';

import { notification } from 'ant-design-vue';
import CryptoJS from 'crypto-js';
import { defineStore } from 'pinia';

import {
  createAdminPortalServiceClient,
  createAuthenticationServiceClient,
  createUserProfileServiceClient,
} from '#/generated/api/admin/service/v1';
import { $t } from '#/locales';
import { globalSSEClient } from '#/transport/sse';
import { requestClientRequestHandler } from '#/utils/request';

type RefreshTokenFunc = () => Promise<string> | string;

const ACCESS_TOKEN_REFRESH_INTERVAL = 90 * 60 * 1000; // 1.5 小时
const REFRESH_TOKEN_REFRESH_INTERVAL = 12 * 60 * 60 * 1000; // 12 小时

let refreshTimer: null | ReturnType<typeof setTimeout> = null;
let refreshCallback: null | RefreshTokenFunc = null;

let isReauthenticating = false;

/**
 * 认证状态管理
 */
export const useAuthStore = defineStore('auth', () => {
  const accessStore = useAccessStore();
  const userStore = useUserStore();
  const router = useRouter();

  const loginLoading = ref(false);

  const authnService = createAuthenticationServiceClient(
    requestClientRequestHandler,
  );
  const adminPortalService = createAdminPortalServiceClient(
    requestClientRequestHandler,
  );
  const userProfileService = createUserProfileServiceClient(
    requestClientRequestHandler,
  );

  /**
   * 加密数据
   * @param data 待加密数据
   * @param key 密钥
   * @param iv 初始向量
   */
  function encryptData(data: string, key: string, iv: string): string {
    const keyHex = CryptoJS.enc.Utf8.parse(key);
    const ivHex = CryptoJS.enc.Utf8.parse(iv);
    const encrypted = CryptoJS.AES.encrypt(data, keyHex, {
      iv: ivHex,
      mode: CryptoJS.mode.CBC,
      padding: CryptoJS.pad.Pkcs7,
    });
    return encrypted.toString();
  }

  /**
   * 加密密码
   * @param password 明文密码
   */
  function encryptPassword(password: string): string {
    const key = import.meta.env.VITE_AES_KEY;
    if (!key) {
      throw new Error('VITE_AES_KEY is not set in environment');
    }
    return encryptData(password, key, key);
  }

  /**
   * 异步处理登录操作
   * Asynchronously handle the login process
   * @param params 登录表单数据
   * @param onSuccess
   */
  async function authLogin(
    params: Recordable<any>,
    onSuccess?: () => Promise<void> | void,
  ): Promise<{ userInfo: null | UserInfo } | null> {
    // 异步处理用户登录操作并获取 accessToken
    let userInfo: null | UserInfo = null;
    try {
      loginLoading.value = true;

      const resp = await authnService.Login({
        username: params.username,
        password: encryptPassword(params.password),
        grant_type: 'password',
      });

      const accessToken = (resp as any).access_token;
      const refresh_token = (resp as any).refresh_token;
      let expiresAt = (resp as any).expires_in;
      let refreshExpiresAt = (resp as any).refresh_expires_in;

      const expiresAtSec = Number(expiresAt);
      expiresAt =
        !Number.isFinite(expiresAtSec) || expiresAtSec <= 0
          ? Date.now() + ACCESS_TOKEN_REFRESH_INTERVAL
          : Date.now() + Math.floor(expiresAtSec * 1000);

      const refreshExpiresAtSec = Number(refreshExpiresAt);
      refreshExpiresAt =
        !Number.isFinite(refreshExpiresAtSec) || refreshExpiresAtSec <= 0
          ? Date.now() + REFRESH_TOKEN_REFRESH_INTERVAL
          : Date.now() + Math.floor(refreshExpiresAtSec * 1000);

      // 如果成功获取到 accessToken
      if (accessToken) {
        accessStore.setAccessToken(accessToken);
        accessStore.setAccessTokenExpireTime(expiresAt);

        if (refresh_token) {
          accessStore.setRefreshToken(refresh_token);
          accessStore.setRefreshTokenExpireTime(refreshExpiresAt);
          startRefreshTimer();
        }

        // 获取用户信息并存储到 accessStore 中
        const [fetchUserInfoResult, fetchAccessCodeResult] = await Promise.all([
          fetchUserInfo(),
          fetchAccessCodes(),
        ]);

        // console.log('fetchUserInfoResult', fetchUserInfoResult);

        userInfo = fetchUserInfoResult;
        if (!userInfo) {
          throw new Error($t('authentication.loginFailedDesc'));
        }

        userStore.setUserInfo(userInfo);
        accessStore.setAccessCodes(fetchAccessCodeResult.codes ?? []);

        if (accessStore.loginExpired) {
          accessStore.setLoginExpired(false);
        } else {
          onSuccess
            ? await onSuccess?.()
            : await router.push(userInfo.homePath || DEFAULT_HOME_PATH);
        }

        if (userInfo?.realname) {
          notification.success({
            description: `${$t('authentication.loginSuccessDesc')}:${userInfo?.realname}`,
            duration: 3,
            message: $t('authentication.loginSuccess'),
          });
        }
      }
    } catch (error) {
      await _doLogout();

      // 处理登录错误
      if (error instanceof Error) {
        notification.error({
          message: $t('authentication.loginFailed'),
          description: error.message,
        });
      } else {
        notification.error({
          message: $t('authentication.loginFailed'),
          description: $t('authentication.loginFailedDesc'),
        });
      }
      return null;
    } finally {
      loginLoading.value = false;
    }

    return {
      userInfo,
    };
  }

  /**
   * 用户登出
   * @param redirect 是否重定向到登录页
   */
  async function logout(redirect: boolean = true) {
    try {
      if (accessStore.accessToken !== null && accessStore.accessToken !== '') {
        await authnService.Logout({});
      }
    } catch {
      // 忽略错误
    }

    await _doLogout(redirect);
  }

  /**
   * 执行登出操作
   * @param redirect 是否重定向到登录页
   */
  async function _doLogout(redirect: boolean = true) {
    console.log('_doLogout');

    // 停止定时刷新
    _stopRefreshTimer();

    resetAllStores();

    accessStore.setLoginExpired(false);

    globalSSEClient.close();

    loginLoading.value = false;

    console.log('currentRoute', router.currentRoute.value);
    // 如果当前页是登录页，则不处理
    if (router.currentRoute.value.path === LOGIN_PATH) return;

    // 回登录页带上当前路由地址
    await router.replace({
      path: LOGIN_PATH,
      query: redirect
        ? {
            redirect: encodeURIComponent(router.currentRoute.value.fullPath),
          }
        : {},
    });
  }

  /**
   * 刷新访问令牌
   */
  async function refreshToken(): Promise<string> {
    if (!accessStore.refreshToken) {
      await reauthenticate();
      return '';
    }

    try {
      const resp = await authnService.RefreshToken({
        grant_type: 'refresh_token',
        refresh_token: accessStore.refreshToken ?? '',
      });

      const newAccessToken = (resp as any).access_token;
      const newRefreshToken = (resp as any).refresh_token;

      let expiresIn = (resp as any).expires_in;
      let refreshExpiresIn = (resp as any).refresh_expires_in;

      const expiresInSec = Number(expiresIn);
      expiresIn =
        !Number.isFinite(expiresInSec) || expiresInSec <= 0
          ? Date.now() + ACCESS_TOKEN_REFRESH_INTERVAL
          : Date.now() + Math.floor(expiresInSec * 1000);

      const refreshExpiresInSec = Number(refreshExpiresIn);
      refreshExpiresIn =
        !Number.isFinite(refreshExpiresInSec) || refreshExpiresInSec <= 0
          ? Date.now() + REFRESH_TOKEN_REFRESH_INTERVAL
          : Date.now() + Math.floor(refreshExpiresInSec * 1000);

      accessStore.setAccessTokenExpireTime(expiresIn);
      accessStore.setRefreshTokenExpireTime(refreshExpiresIn);

      accessStore.setAccessToken(newAccessToken ?? null);
      accessStore.setRefreshToken(newRefreshToken ?? null);

      return newAccessToken ?? '';
    } catch (error) {
      console.error('刷新 access token 失败', error);
      await reauthenticate();
      return '';
    }
  }

  /**
   * 重新认证
   */
  async function reauthenticate(): Promise<void> {
    if (isReauthenticating) {
      return;
    }
    isReauthenticating = true;

    try {
      console.warn('Access token or refresh token is invalid or expired.');

      // 停止定时刷新并清理回调，防止继续触发刷新请求
      _stopRefreshTimer();

      accessStore.setAccessToken(null);
      accessStore.setRefreshToken(null);
      accessStore.setIsAccessChecked(false);
      accessStore.setAccessCodes([]);

      if (
        preferences.app.loginExpiredMode === 'modal' &&
        accessStore.isAccessChecked
      ) {
        accessStore.setLoginExpired(true);
      } else {
        // 非 modal 模式直接登出并跳转登录页
        await logout();
      }
    } finally {
      isReauthenticating = false;
    }
  }

  /**
   * 拉取用户信息
   */
  async function fetchUserInfo() {
    try {
      return (await userProfileService.GetUser({})) as unknown as UserInfo;
    } catch (error) {
      console.error('fetchUserInfo failed:', error);
      await _doLogout();
      return null;
    }
  }

  /**
   * 获取用户权限码
   */
  async function fetchAccessCodes() {
    return await adminPortalService.GetMyPermissionCode({});
  }

  /**
   * 启动定时刷新
   * @param cb 刷新回调函数
   */
  function _startRefreshTimer(cb?: RefreshTokenFunc): void {
    _stopRefreshTimer();

    if (cb) refreshCallback = cb;
    if (!refreshCallback) return;

    const SAFETY_BUFFER_MS = 5 * 60 * 1000; // 在 access 到期前 5 分钟开始刷新
    const MIN_INTERVAL_MS = 3 * 1000; // 最小 3s（避免立即重试风暴）
    const MAX_INTERVAL_MS = ACCESS_TOKEN_REFRESH_INTERVAL; // 上限

    const computeNextInterval = (): number => {
      const now = Date.now();

      const accessExpire = accessStore.accessTokenExpireTime ?? 0;
      const refreshExpire = accessStore.refreshTokenExpireTime ?? 0;

      // 如果 refresh token 已过期或快到期，优先走 reauthenticate（尽快处理）
      const refreshRemaining = refreshExpire - now;
      if (refreshExpire && refreshRemaining <= SAFETY_BUFFER_MS) {
        // 让 schedule 触发短延迟以处理 reauthenticate（或直接 reauthenticate）
        return MIN_INTERVAL_MS;
      }

      // 基于 access token 计算下一次刷新
      const accessRemaining = accessExpire - now;
      if (!accessExpire || accessRemaining <= 0) {
        // access 已过期或缺失 -> 立刻重认证
        return MIN_INTERVAL_MS;
      }

      // 如果 access 在安全缓冲内 (<= SAFETY_BUFFER_MS)，尽快刷新
      if (accessRemaining <= SAFETY_BUFFER_MS) {
        return MIN_INTERVAL_MS;
      }

      // 否则，选择在剩余时间的某个比例处触发（例如剩余时间去掉缓冲后 80% 的时间）
      return Math.floor(
        Math.max(
          MIN_INTERVAL_MS,
          Math.min(MAX_INTERVAL_MS, (accessRemaining - SAFETY_BUFFER_MS) * 0.8),
        ),
      );
    };

    const schedule = async () => {
      try {
        // 在每次触发前再做两个检查：
        // 1) 是否还有 refresh token（没有则 reauthenticate）
        // 2) 如果 refresh token 快过期，直接 reauthenticate
        const now = Date.now();
        const refreshExpire = accessStore.refreshTokenExpireTime ?? 0;
        if (!accessStore.refreshToken) {
          await reauthenticate();
          return;
        }
        if (refreshExpire && refreshExpire - now <= SAFETY_BUFFER_MS) {
          // refresh token 在缓冲期内或过期 -> 重新认证处理
          await reauthenticate();
          return;
        }

        // 尝试刷新 access token
        await refreshCallback?.();
      } catch (error) {
        console.error('refreshToken 定时刷新失败', error);
        // refresh 失败后，refreshToken() 内通常会调用 reauthenticate()
      } finally {
        if (refreshCallback) {
          const next = computeNextInterval();
          refreshTimer = globalThis.setTimeout(schedule, next);
        }
      }
    };

    // 首次 schedule（基于当前 access 到期时间）
    refreshTimer = globalThis.setTimeout(schedule, computeNextInterval());
  }

  /**
   * 停止定时器
   */
  function _stopRefreshTimer(): void {
    if (refreshTimer !== null) {
      globalThis.clearTimeout(refreshTimer);
      refreshTimer = null;
      // 清除回调，防止后续意外触发
      refreshCallback = null;
    }
  }

  function startRefreshTimer() {
    _startRefreshTimer(refreshToken);
  }

  async function getUserPermissionCodes() {
    let userPermissionCodes: string[] = [];

    if (userStore.userInfo === null || accessStore.accessCodes === null) {
      const [fetchUserInfoResult, fetchAccessCodeResult] = await Promise.all([
        fetchUserInfo(),
        fetchAccessCodes(),
      ]);
      if (fetchUserInfoResult === null || fetchAccessCodeResult === null) {
        console.warn(
          'setupAccessGuard failed fetch user info:',
          fetchUserInfoResult,
        );
        return false;
      }
      userStore.setUserInfo(fetchUserInfoResult);

      const roles = fetchUserInfoResult
        ? (fetchUserInfoResult.roles ?? [])
        : [];
      const codes = fetchAccessCodeResult
        ? (fetchAccessCodeResult.codes ?? [])
        : [];
      userPermissionCodes = [...roles, ...codes];
      accessStore.setAccessCodes(userPermissionCodes);
    } else {
      userPermissionCodes = [
        ...(userStore.userInfo.roles || []),
        ...accessStore.accessCodes,
      ];
    }

    startRefreshTimer();

    _connectSSEServer();

    return userPermissionCodes;
  }

  /**
   * 连接 SSE 服务器
   */
  function _connectSSEServer() {
    const targetSseUrl = `${import.meta.env.VITE_GLOB_SSE_URL}?stream=${encodeURIComponent(accessStore.accessToken ?? '')}`;
    globalSSEClient.connect(targetSseUrl);
  }

  function $reset() {
    loginLoading.value = false;
    _stopRefreshTimer();
  }

  return {
    $reset,
    authLogin,
    fetchUserInfo,
    fetchAccessCodes,
    loginLoading,
    logout,
    refreshToken,
    reauthenticate,
    startRefreshTimer,
    getUserPermissionCodes,
  };
});
