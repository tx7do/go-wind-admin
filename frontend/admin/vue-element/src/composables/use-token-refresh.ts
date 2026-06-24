import { refreshToken as refreshTokenService } from "@/api/composables";
import { LOGIN_PATH } from "@/constants";
import { preferences } from "@/core/preferences";
import { globalSSEClient } from "@/core/transport/sse";
import { queryClient } from "@/plugins/vue-query";
import { resetAllStores, useAccessStore } from "@/stores";
import { router } from "@/router";

// ==============================
// 常量
// ==============================

/** Access Token 刷新间隔（1.5 小时） */
const ACCESS_TOKEN_REFRESH_INTERVAL = 90 * 60 * 1000;
/** Refresh Token 刷新间隔（12 小时） */
const REFRESH_TOKEN_REFRESH_INTERVAL = 12 * 60 * 60 * 1000;

/** 在 access token 到期前多久开始刷新 */
const SAFETY_BUFFER_MS = 5 * 60 * 1000;
/** 最小刷新间隔（避免立即重试风暴） */
const MIN_INTERVAL_MS = 3 * 1000;

// ==============================
// 模块级状态（单例）
// ==============================

let refreshTimer: null | ReturnType<typeof setTimeout> = null;
let refreshCallback: null | RefreshTokenFunc = null;
let isReauthenticating = false;

type RefreshTokenFunc = () => Promise<string> | string;

// ==============================
// 核心：刷新 Access Token
// ==============================

/**
 * 刷新访问令牌
 * 使用 refresh_token 换取新的 access_token 和 refresh_token
 */
export async function refreshToken(): Promise<string> {
  const accessStore = useAccessStore();

  if (!accessStore.refreshToken) {
    await reauthenticate();
    return "";
  }

  try {
    const resp = await refreshTokenService(accessStore.refreshToken ?? "");

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

    // token 已更新，重连 SSE 以使用新凭证
    reconnectSSEServer();

    return newAccessToken ?? "";
  } catch (error) {
    console.error("刷新 access token 失败", error);
    await reauthenticate();
    return "";
  }
}

// ==============================
// 核心：重新认证
// ==============================

/**
 * 重新认证
 * 当 refresh token 失效时触发，根据配置决定弹窗或直接跳转登录页
 */
export async function reauthenticate(): Promise<void> {
  if (isReauthenticating) {
    return;
  }
  isReauthenticating = true;

  try {
    console.warn("Access token or refresh token is invalid or expired.");

    stopRefreshTimer();

    const accessStore = useAccessStore();
    accessStore.setAccessToken(null);
    accessStore.setRefreshToken(null);
    accessStore.setIsAccessChecked(false);
    accessStore.setAccessCodes([]);

    if (preferences.app.loginExpiredMode === "modal" && accessStore.isAccessChecked) {
      accessStore.setLoginExpired(true);
    } else {
      await logoutToLoginPage();
    }
  } finally {
    isReauthenticating = false;
  }
}

// ==============================
// 登出跳转
// ==============================

/**
 * 停止刷新定时器 → 重置所有 Store → 关闭 SSE → 跳转登录页
 * @param redirect 是否携带回跳地址
 */
export async function logoutToLoginPage(redirect: boolean = true): Promise<void> {
  console.log("logoutToLoginPage");

  stopRefreshTimer();

  resetAllStores();

  const accessStore = useAccessStore();
  accessStore.setLoginExpired(false);

  // 清除 queryClient 缓存，防止登出期间被缓存污染的查询结果
  // 导致重新登录时命中脏数据
  queryClient.clear();

  globalSSEClient.close();

  console.log("currentRoute", router.currentRoute.value);
  if (router.currentRoute.value.path === LOGIN_PATH) return;

  await router.replace({
    path: LOGIN_PATH,
    query: redirect
      ? {
          redirect: encodeURIComponent(router.currentRoute.value.fullPath),
        }
      : {},
  });
}

// ==============================
// 定时刷新管理
// ==============================

function computeNextInterval(): number {
  const accessStore = useAccessStore();
  const now = Date.now();

  const accessExpire = accessStore.accessTokenExpireTime ?? 0;
  const refreshExpire = accessStore.refreshTokenExpireTime ?? 0;

  const refreshRemaining = refreshExpire - now;
  if (refreshExpire && refreshRemaining <= SAFETY_BUFFER_MS) {
    return MIN_INTERVAL_MS;
  }

  const accessRemaining = accessExpire - now;
  if (!accessExpire || accessRemaining <= 0) {
    return MIN_INTERVAL_MS;
  }

  if (accessRemaining <= SAFETY_BUFFER_MS) {
    return MIN_INTERVAL_MS;
  }

  return Math.floor(
    Math.max(
      MIN_INTERVAL_MS,
      Math.min(ACCESS_TOKEN_REFRESH_INTERVAL, (accessRemaining - SAFETY_BUFFER_MS) * 0.8)
    )
  );
}

function _startRefreshTimer(cb?: RefreshTokenFunc): void {
  stopRefreshTimer();

  if (cb) refreshCallback = cb;
  if (!refreshCallback) return;

  const schedule = async () => {
    try {
      const accessStore = useAccessStore();
      const now = Date.now();
      const refreshExpire = accessStore.refreshTokenExpireTime ?? 0;
      if (!accessStore.refreshToken) {
        await reauthenticate();
        return;
      }
      if (refreshExpire && refreshExpire - now <= SAFETY_BUFFER_MS) {
        await reauthenticate();
        return;
      }

      await refreshCallback?.();
    } catch (error) {
      console.error("refreshToken 定时刷新失败", error);
    } finally {
      if (refreshCallback) {
        const next = computeNextInterval();
        refreshTimer = globalThis.setTimeout(schedule, next);
      }
    }
  };

  refreshTimer = globalThis.setTimeout(schedule, computeNextInterval());
}

export function stopRefreshTimer(): void {
  if (refreshTimer !== null) {
    globalThis.clearTimeout(refreshTimer);
    refreshTimer = null;
    refreshCallback = null;
  }
}

export function startRefreshTimer(): void {
  _startRefreshTimer(refreshToken);
}

// ==============================
// SSE 连接
// ==============================

export function connectSSEServer(): void {
  const accessStore = useAccessStore();

  const token = accessStore.accessToken ?? "";
  const targetSseUrl = `${import.meta.env.VITE_APP_SSE_URL}?stream=${encodeURIComponent(token)}`;

  globalSSEClient.setHeaders({ Authorization: `Bearer ${token}` });
  globalSSEClient.connect(targetSseUrl);
}

/**
 * 使用新 token 重连 SSE（关闭旧连接 → 更新凭证 → 重新连接）
 * 适用于 token 刷新后 SSE 连接携带的凭证已过期的场景
 */
function reconnectSSEServer(): void {
  const accessStore = useAccessStore();

  const token = accessStore.accessToken ?? "";
  const targetSseUrl = `${import.meta.env.VITE_APP_SSE_URL}?stream=${encodeURIComponent(token)}`;

  globalSSEClient.setHeaders({ Authorization: `Bearer ${token}` });
  globalSSEClient.reconnect(targetSseUrl);
}
