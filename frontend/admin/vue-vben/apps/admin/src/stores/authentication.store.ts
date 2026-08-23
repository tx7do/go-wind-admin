import type { Recordable, UserInfo } from '@vben/types';

import { ref } from 'vue';

import { DEFAULT_HOME_PATH, LOGIN_PATH } from '@vben/constants';
import { preferences } from '@vben/preferences';
import { resetAllStores, useAccessStore, useUserStore } from '@vben/stores';

import { notification } from 'ant-design-vue';
import CryptoJS from 'crypto-js';
import { defineStore } from 'pinia';

import { fetchMyPermissionCode, fetchUserProfile, loginMutation, logoutMutation, refreshTokenMutation, verifyMfaMutation } from '#/api/composables';
import { $t } from '#/locales';
import { queryClient } from '#/plugins/vue-query';
import { router } from '#/router';
import { setCaptchaHeaders } from '#/transport/rest';
import { globalSSEClient } from '#/transport/sse';

type RefreshTokenFunc = () => Promise<string> | string;

const ACCESS_TOKEN_REFRESH_INTERVAL = 90 * 60 * 1000; // 1.5 小时

let refreshTimer: null | ReturnType<typeof setTimeout> = null;
let refreshCallback: null | RefreshTokenFunc = null;

/**
 * 从 refresh_exp cookie 读取 refresh token 的过期时间戳（Unix 秒）。
 * refresh_exp 为非 HttpOnly cookie，仅含过期时间戳，无敏感信息。
 * 由后端在登录/刷新时写入。返回毫秒级时间戳或 null（cookie 不存在/已过期）。
 */
function getRefreshExpireAt(): number | null {
  const match = document.cookie
    .split('; ')
    .find((row) => row.startsWith('refresh_exp='));
  if (!match) return null;
  const parts = match.split('=');
  const raw = parts.length >= 2 ? parts[1] : undefined;
  if (raw === undefined) return null;
  const val = parseInt(raw, 10);
  if (!Number.isFinite(val) || val <= 0) return null;
  return val * 1000;
}

let isReauthenticating = false;

/**
 * 认证状态管理
 */
export const useAuthStore = defineStore('auth', () => {
  const accessStore = useAccessStore();
  const userStore = useUserStore();

  const loginLoading = ref(false);

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

      // 若表单携带验证码，先设置一次性 Header（由 transport.unary 消费）
      if (params.captchaId && params.captchaValue) {
        setCaptchaHeaders(params.captchaId, params.captchaValue);
      }

      const resp = await loginMutation.execute({
        username: params.username,
        password: encryptPassword(params.password),
        tenant_code: params.tenant_code,
        grant_type: 'password',
      });

      // ===== MFA 闸门：后端在密码校验通过、需二次验证时返回 mfa_operation_id，access_token 为空。
      // 不写任何 token，记录 operation_id 并跳转 MFA 挑战页（路由守卫亦据此强制跳转）。
      if ((resp as any).mfa_operation_id) {
        accessStore.mfaOperationId = (resp as any).mfa_operation_id as string;
        // 携带当前 redirect 到挑战页，验证通过后回到原目标页
        const redirect = (router.currentRoute.value.query.redirect as string) || '';
        await router.push({ name: 'MfaChallenge', query: redirect ? { redirect } : {} });
        return { userInfo: null };
        }

      userInfo = await applySuccessfulLogin(resp as any, onSuccess);
    } catch (error) {
      await _doLogout();

      // 处理登录错误
      // request() 方法可能抛出 Error 实例或纯响应数据对象（含 message 字段）
      const errorMsg =
        error instanceof Error
          ? error.message
          : (error as any)?.message || $t('authentication.loginFailedDesc');

      notification.error({
        message: $t('authentication.loginFailed'),
        description: errorMsg,
      });
      return null;
    } finally {
      loginLoading.value = false;
    }

    return {
      userInfo,
    };
  }

  // applySuccessfulLogin 处理"已拿到含真 token 的 LoginResponse"后的统一流程：
  // 存 token → 拉用户信息/权限码 → 跳转。
  // 登录成功与 MFA 验证成功都复用此函数。返回 userInfo（失败抛错）。
  async function applySuccessfulLogin(
    resp: any,
    onSuccess?: () => Promise<void> | void,
  ): Promise<UserInfo | null> {
    const accessToken = resp.access_token;
    let expiresAt = resp.expires_in;

    const expiresAtSec = Number(expiresAt);
    expiresAt =
      !Number.isFinite(expiresAtSec) || expiresAtSec <= 0
        ? Date.now() + ACCESS_TOKEN_REFRESH_INTERVAL
        : Date.now() + Math.floor(expiresAtSec * 1000);

    if (!accessToken) {
      return null;
    }

    accessStore.setAccessToken(accessToken);
    accessStore.setAccessTokenExpireTime(expiresAt);

    // refresh token 通过 HttpOnly Cookie 传输，前端不可读也不存内存。
    // 页面刷新后由 bootstrap 静默恢复（凭 cookie 调 /refresh-token 换新 access token）。
    startRefreshTimer();

    // 获取用户信息并存储到 accessStore 中
    const [fetchUserInfoResult, fetchAccessCodeResult] = await Promise.all([
      fetchUserInfo(),
      fetchAccessCodes(),
    ]);

    const userInfo = fetchUserInfoResult;
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

    return userInfo;
  }

  // completeMfaChallenge 用 operation_id + TOTP 码调后端验证，通过则复用 applySuccessfulLogin。
  async function completeMfaChallenge(
    totpCode: string,
    onSuccess?: () => Promise<void> | void,
  ): Promise<{ userInfo: null | UserInfo } | null> {
    let userInfo: null | UserInfo = null;
    const opId = accessStore.mfaOperationId;
    if (!opId) {
      return null;
    }
    try {
      loginLoading.value = true;
      const resp = await verifyMfaMutation.execute({
        operationId: opId,
        totpCode: totpCode,
      } as any);
      accessStore.mfaOperationId = null;
      userInfo = await applySuccessfulLogin(resp as any, onSuccess);
    } catch (error) {
      await _doLogout();

      const errorMsg =
        error instanceof Error
          ? error.message
          : (error as any)?.message || $t('authentication.loginFailedDesc');

      notification.error({
        message: $t('authentication.loginFailed'),
        description: errorMsg,
      });
      return null;
    } finally {
      loginLoading.value = false;
    }
    return { userInfo };
  }

  /**
   * 用户登出
   * @param redirect 是否重定向到登录页
   */
  async function logout(redirect: boolean = true) {
    try {
      if (accessStore.accessToken !== null && accessStore.accessToken !== '') {
        await logoutMutation.execute(undefined);
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

    // resetAllStores 可能从持久化中恢复 token，必须再次清除
    accessStore.setAccessToken(null);
    accessStore.setLoginExpired(false);

    // 清除 queryClient 缓存，防止登出期间被缓存污染的查询结果
    // （如 getMe 因 401 返回 null 被 fetchQuery 缓存）导致重新登录时命中脏数据
    queryClient.clear();

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
   * refresh token 以 HttpOnly Cookie 传输，刷新请求由浏览器自动携带 cookie，
   * 前端无需（也无法）读取 refresh token 值。
   */
  async function refreshToken(): Promise<string> {
    try {
      const resp = await refreshTokenMutation.execute(undefined);

      const newAccessToken = (resp as any).access_token;

      let expiresIn = (resp as any).expires_in;

      const expiresInSec = Number(expiresIn);
      expiresIn =
        !Number.isFinite(expiresInSec) || expiresInSec <= 0
          ? Date.now() + ACCESS_TOKEN_REFRESH_INTERVAL
          : Date.now() + Math.floor(expiresInSec * 1000);

      accessStore.setAccessTokenExpireTime(expiresIn);
      accessStore.setAccessToken(newAccessToken ?? null);

      // token 刷新成功后，使用新 token 重连 SSE
      _reconnectSSEServer();

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
      // 注意：setIsAccessChecked(false) 之前必须先读出原值用于下面的 modal 判定，
      // 否则下方 accessStore.isAccessChecked 永远是 false，modal 模式恒不触发。
      const wasAccessChecked = accessStore.isAccessChecked;
      accessStore.setIsAccessChecked(false);
      accessStore.setAccessCodes([]);

      if (preferences.app.loginExpiredMode === 'modal' && wasAccessChecked) {
        accessStore.setLoginExpired(true);
      } else {
        // 非 modal 模式直接清理并跳转登录页
        // 注意：不调用 logout()，因为 logout() 会调后端 logout API，
        // 但此时 token 已过期，后端会返回 401 又触发一轮拦截器循环
        await _doLogout();
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
      return (await fetchUserProfile()) as unknown as UserInfo;
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
    return await fetchMyPermissionCode();
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
      // refresh token 过期时间从 refresh_exp cookie 读取（非 HttpOnly）
      const refreshExpire = getRefreshExpireAt() ?? 0;

      // 如果 refresh token 已过期或快到期，优先走 reauthenticate（尽快处理）
      const refreshRemaining = refreshExpire - now;
      if (refreshExpire && refreshRemaining <= SAFETY_BUFFER_MS) {
        return MIN_INTERVAL_MS;
      }

      // 基于 access token 计算下一次刷新
      const accessRemaining = accessExpire - now;
      if (!accessExpire || accessRemaining <= 0) {
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
        const now = Date.now();
        // refresh token 过期时间从 refresh_exp cookie 读取
        const refreshExpire = getRefreshExpireAt();
        if (!refreshExpire) {
          await reauthenticate();
          return;
        }
        if (refreshExpire - now <= SAFETY_BUFFER_MS) {
          await reauthenticate();
          return;
        }

        // 尝试刷新 access token
        await refreshCallback?.();
      } catch (error) {
        console.error('refreshToken 定时刷新失败', error);
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
   * streamID 使用 userId，鉴权走 Authorization 头；两者缺一不可。
   */
  function _connectSSEServer() {
    const token = accessStore.accessToken ?? '';
    const userId = userStore.userInfo?.id;
    if (!token || userId == null) {
      console.warn('[SSE] No access token or userId, skip connection');
      return;
    }
    const targetSseUrl = `${import.meta.env.VITE_GLOB_SSE_URL}?stream=${userId}`;

    globalSSEClient.setHeaders({ Authorization: `Bearer ${token}` });
    globalSSEClient.connect(targetSseUrl);
  }

  /**
   * 使用新 token 重连 SSE（关闭旧连接 → 更新凭证 → 重新连接）
   * 适用于 token 刷新后 SSE 连接携带的凭证已过期的场景
   */
  function _reconnectSSEServer(): void {
    const token = accessStore.accessToken ?? '';
    const userId = userStore.userInfo?.id;
    if (!token || userId == null) {
      console.warn('[SSE] No access token or userId, skip reconnect');
      return;
    }
    const targetSseUrl = `${import.meta.env.VITE_GLOB_SSE_URL}?stream=${userId}`;

    globalSSEClient.setHeaders({ Authorization: `Bearer ${token}` });
    globalSSEClient.reconnect(targetSseUrl);
  }

  function $reset() {
    loginLoading.value = false;
    _stopRefreshTimer();
  }

  return {
    $reset,
    authLogin,
    completeMfaChallenge,
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
