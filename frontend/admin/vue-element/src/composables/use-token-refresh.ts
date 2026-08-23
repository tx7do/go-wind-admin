import { START_LOCATION } from "vue-router";

import { refreshToken as refreshTokenService } from "@/api/composables";
import { LOGIN_PATH } from "@/constants";
import { preferences } from "@/core/preferences";
import { globalSSEClient } from "@/core/transport/sse";
import { queryClient } from "@/plugins/vue-query";
import { resetAllStores, useAccessStore, useAppUserStore } from "@/stores";
import { router } from "@/router";

// ==============================
// 常量
// ==============================

/** Access Token 刷新间隔（1.5 小时） */
const ACCESS_TOKEN_REFRESH_INTERVAL = 90 * 60 * 1000;

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

/**
 * 从 refresh_exp cookie 读取 refresh token 的过期时间戳（Unix 秒）。
 * refresh_exp 为非 HttpOnly cookie，仅含过期时间戳，无敏感信息。
 * 由后端在登录/刷新时写入。返回毫秒级时间戳或 null（cookie 不存在/已过期）。
 */
function getRefreshExpireAt(): number | null {
  const match = document.cookie
    .split("; ")
    .find((row) => row.startsWith("refresh_exp="));
  if (!match) return null;
  const parts = match.split("=");
  if (parts.length < 2) return null;
  const val = parseInt(parts[1], 10);
  if (!Number.isFinite(val) || val <= 0) return null;
  return val * 1000;
}

// ==============================
// 核心：刷新 Access Token
// ==============================

/**
 * 刷新访问令牌
 * refresh token 以 HttpOnly Cookie 传输，刷新请求由浏览器自动携带 cookie，
 * 前端无需（也无法）读取 refresh token 值。
 */
export async function refreshToken(): Promise<string> {
  const accessStore = useAccessStore();

  try {
    const resp = await refreshTokenService();

    const newAccessToken = (resp as any).access_token;

    let expiresIn = (resp as any).expires_in;

    const expiresInSec = Number(expiresIn);
    expiresIn =
      !Number.isFinite(expiresInSec) || expiresInSec <= 0
        ? Date.now() + ACCESS_TOKEN_REFRESH_INTERVAL
        : Date.now() + Math.floor(expiresInSec * 1000);

    accessStore.setAccessTokenExpireTime(expiresIn);
    accessStore.setAccessToken(newAccessToken ?? null);

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
    // 注意：setIsAccessChecked(false) 之前必须先读出原值用于下面的 modal 判定，
    // 否则下方 accessStore.isAccessChecked 永远是 false，modal 模式恒不触发。
    const wasAccessChecked = accessStore.isAccessChecked;
    accessStore.setIsAccessChecked(false);
    accessStore.setAccessCodes([]);

    if (preferences.app.loginExpiredMode === "modal" && wasAccessChecked) {
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

  const cur = router.currentRoute.value;
  console.log("currentRoute", cur);
  if (cur.path === LOGIN_PATH) return;

  // 整页重载进行中（首次导航未完成，currentRoute 仍是 START_LOCATION）时，
  // 地址栏 URL（可能带着完好的 ?redirect=...）尚未被消费。此时若照常
  // replace 到登录页，redirect 会被写成 "%2F"，把原目标冲掉 → 下次登录
  // 成功落到首页。直接返回，交给待完成的首次导航/守卫自行落位。
  if (cur === START_LOCATION) return;

  // 已解析但未注册的路径（name 未定义，如 "/"）不携带有效回跳目标；
  // 写 redirect="%2F" 同样会冲掉登录页 URL 上原有的目标。直接去登录页。
  if (cur.name === undefined && (cur.path === "/" || cur.path === "")) {
    await router.replace({ path: LOGIN_PATH });
    return;
  }

  await router.replace({
    path: LOGIN_PATH,
    query: redirect
      ? {
          redirect: encodeURIComponent(cur.fullPath),
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

  // refresh token 过期时间从 refresh_exp cookie 读取（非 HttpOnly）
  const refreshExpire = getRefreshExpireAt() ?? 0;

  const refreshRemaining = refreshExpire - now;
  if (refreshExpire && refreshRemaining <= SAFETY_BUFFER_MS) {
    return MIN_INTERVAL_MS;
  }

  const accessExpire = accessStore.accessTokenExpireTime ?? 0;
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
  const userStore = useAppUserStore();

  // streamID 已改为 userId，鉴权仍走 Authorization 头。两者缺一不可。
  const token = accessStore.accessToken ?? "";
  const userId = userStore.userInfo?.id;
  if (!token || userId == null) {
    console.warn("[TokenRefresh] No access token or userId, skip SSE connection");
    return;
  }
  const targetSseUrl = `${import.meta.env.VITE_APP_SSE_URL}?stream=${userId}`;

  globalSSEClient.setHeaders({ Authorization: `Bearer ${token}` });
  globalSSEClient.connect(targetSseUrl);
}

/**
 * 使用新 token 重连 SSE（关闭旧连接 → 更新凭证 → 重新连接）
 * 适用于 token 刷新后 SSE 连接携带的凭证已过期的场景
 */
function reconnectSSEServer(): void {
  const accessStore = useAccessStore();
  const userStore = useAppUserStore();

  // streamID 已改为 userId，鉴权仍走 Authorization 头。两者缺一不可。
  const token = accessStore.accessToken ?? "";
  const userId = userStore.userInfo?.id;
  if (!token || userId == null) {
    console.warn("[TokenRefresh] No access token or userId, skip SSE reconnect");
    return;
  }
  const targetSseUrl = `${import.meta.env.VITE_APP_SSE_URL}?stream=${userId}`;

  globalSSEClient.setHeaders({ Authorization: `Bearer ${token}` });
  globalSSEClient.reconnect(targetSseUrl);
}
